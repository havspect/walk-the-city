package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/havspect/walk-the-city/assets"
	"github.com/havspect/walk-the-city/internal/config"
	"github.com/havspect/walk-the-city/internal/nominatim"
	"github.com/havspect/walk-the-city/internal/trip"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestWizardEndToEndJourney(t *testing.T) {
	// 1. Setup isolated test database & dependencies
	dsn := fmt.Sprintf("file:wizard_e2e_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&trip.Trip{}); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}

	renderer, err := newHTMLRenderer(assets.HTMLFiles, "base.tmpl", "partials/*.tmpl")
	if err != nil {
		t.Fatalf("failed to create htmlRenderer: %v", err)
	}

	// Mock Nominatim Server
	nomServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		if strings.EqualFold(q, "Rome") {
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"place_id":     1001,
					"display_name": "Rome, Roma Capitale, Lazio, 00187, Italy",
					"lat":          "41.8933203",
					"lon":          "12.4829321",
					"address": map[string]string{
						"city":         "Rome",
						"state":        "Lazio",
						"country":      "Italy",
						"country_code": "it",
					},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode([]any{})
	}))
	defer nomServer.Close()

	nomClient := nominatim.NewClient(
		nominatim.WithBaseURL(nomServer.URL),
		nominatim.WithRateLimit(0),
	)

	app := &application{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		config: &config.Config{
			Port:     "8080",
			DBPath:   ":memory:",
			LogLevel: "debug",
			Env:      "test",
		},
		html:            renderer,
		staticFS:        assets.StaticFiles,
		tripService:     trip.NewService(db),
		nominatimClient: nomClient,
		tripGenerator:   trip.NewMockGenerator(),
	}

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	client := &http.Client{}

	// --- Phase 1: Load Homepage (Step 1) ---
	respHome, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer respHome.Body.Close()

	if respHome.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for homepage, got %d", respHome.StatusCode)
	}
	homeBody, _ := io.ReadAll(respHome.Body)
	homeStr := string(homeBody)
	if !strings.Contains(homeStr, "Step 1: Choose Your Destination") {
		t.Errorf("expected Step 1 container on homepage")
	}

	// --- Phase 2: Autocomplete City Search ---
	respSearch, err := client.Get(ts.URL + "/api/cities/search?q=Rome")
	if err != nil {
		t.Fatalf("GET /api/cities/search failed: %v", err)
	}
	defer respSearch.Body.Close()

	if respSearch.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for city search, got %d", respSearch.StatusCode)
	}
	searchBody, _ := io.ReadAll(respSearch.Body)
	searchStr := string(searchBody)
	if !strings.Contains(searchStr, "Rome") || !strings.Contains(searchStr, "Italy") {
		t.Errorf("expected search result to contain Rome, Italy, got: %s", searchStr)
	}

	// --- Phase 3: Transition to Preferences Form (Step 2) ---
	prefURL := fmt.Sprintf("%s/wizard/preferences?destination=%s&city=%s&country=%s&lat=41.8933203&lon=12.4829321",
		ts.URL,
		url.QueryEscape("Rome, Roma Capitale, Lazio, 00187, Italy"),
		url.QueryEscape("Rome"),
		url.QueryEscape("Italy"),
	)
	respPref, err := client.Get(prefURL)
	if err != nil {
		t.Fatalf("GET /wizard/preferences failed: %v", err)
	}
	defer respPref.Body.Close()

	if respPref.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for preferences form, got %d", respPref.StatusCode)
	}
	prefBody, _ := io.ReadAll(respPref.Body)
	prefStr := string(prefBody)
	if !strings.Contains(prefStr, "Trip Preferences for Rome") {
		t.Errorf("expected Step 2 heading for Rome, got: %s", prefStr)
	}

	// --- Phase 4: Submit Preferences & Generate Trip (Step 3) ---
	genForm := url.Values{
		"destination":   {"Rome, Roma Capitale, Lazio, 00187, Italy"},
		"city":          {"Rome"},
		"country":       {"Italy"},
		"lat":           {"41.8933203"},
		"lon":           {"12.4829321"},
		"month":         {"September"},
		"duration_days": {"3"},
		"pace":          {"Moderate"},
		"interests":     {"Architecture", "History", "Local Food & Living"},
		"mobility":      {"Walking + Public Transit"},
		"notes":         {"Love historic cafes and gelato"},
	}

	reqGen, _ := http.NewRequest("POST", ts.URL+"/trips/generate", strings.NewReader(genForm.Encode()))
	reqGen.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqGen.Header.Set("HX-Request", "true")

	respGen, err := client.Do(reqGen)
	if err != nil {
		t.Fatalf("POST /trips/generate failed: %v", err)
	}
	defer respGen.Body.Close()

	if respGen.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for HTMX generation response, got %d", respGen.StatusCode)
	}

	redirectHeader := respGen.Header.Get("HX-Redirect")
	if !strings.HasPrefix(redirectHeader, "/trips/") {
		t.Fatalf("expected HX-Redirect to /trips/{id}, got %q", redirectHeader)
	}

	// --- Phase 5: View Permanent Generated Trip (/trips/{id}) ---
	respTrip, err := client.Get(ts.URL + redirectHeader)
	if err != nil {
		t.Fatalf("GET %s failed: %v", redirectHeader, err)
	}
	defer respTrip.Body.Close()

	if respTrip.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for trip page, got %d", respTrip.StatusCode)
	}

	tripBody, _ := io.ReadAll(respTrip.Body)
	tripStr := string(tripBody)

	// Assertions on the permanent trip detail page
	expectedElements := []string{
		"Rome",
		"September",
		"3 Days",
		"Moderate",
		"Public Transit",
		"Colosseum &amp; Ludus Magnus",
		"Pantheon Concrete Dome",
		"Trastevere Artisan Bakeries",
		"Nasone Drinking Fountains",
		"OpenStreetMap",
		"Google Maps",
		"Wikipedia",
	}

	for _, elem := range expectedElements {
		if !strings.Contains(tripStr, elem) {
			t.Errorf("expected trip page to contain %q, but was missing", elem)
		}
	}

	// --- Phase 6: Return to Homepage and verify Saved Trips list ---
	respSaved, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / saved check failed: %v", err)
	}
	defer respSaved.Body.Close()

	savedBody, _ := io.ReadAll(respSaved.Body)
	savedStr := string(savedBody)
	if !strings.Contains(savedStr, "Rome") || !strings.Contains(savedStr, redirectHeader) {
		t.Errorf("expected saved trip list on homepage to include Rome with link %s", redirectHeader)
	}
}
