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
	dsn := fmt.Sprintf("file:wizard_e2e_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&trip.Trip{}, &trip.Itinerary{}, &trip.Stop{}, &trip.Segment{}); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	renderer, err := newHTMLRenderer(assets.HTMLFiles, "base.tmpl", "partials/*.tmpl")
	if err != nil {
		t.Fatalf("failed to create htmlRenderer: %v", err)
	}
	nomServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		if strings.EqualFold(q, "Rome") {
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"place_id": 1001, "display_name": "Rome, Roma Capitale, Lazio, 00187, Italy",
				"lat": "41.8933203", "lon": "12.4829321",
				"address": map[string]string{"city": "Rome", "state": "Lazio", "country": "Italy", "country_code": "it"},
			}})
			return
		}
		_ = json.NewEncoder(w).Encode([]any{})
	}))
	defer nomServer.Close()
	nomClient := nominatim.NewClient(nominatim.WithBaseURL(nomServer.URL), nominatim.WithRateLimit(0))
	app := &application{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		config: &config.Config{Port: "8080", DBPath: ":memory:", LogLevel: "debug", Env: "test"},
		html: renderer, staticFS: assets.StaticFiles,
		tripRepo: trip.NewService(db), nominatimClient: nomClient, tripGenerator: trip.NewMockGenerator(),
	}
	ts := httptest.NewServer(app.routes())
	defer ts.Close()
	client := &http.Client{}

	respHome, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer respHome.Body.Close()
	if respHome.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for homepage, got %d", respHome.StatusCode)
	}
	homeBody, _ := io.ReadAll(respHome.Body)
	if !strings.Contains(string(homeBody), "Step 1: Choose Your Destination") {
		t.Errorf("expected Step 1 container on homepage")
	}

	respSearch, err := client.Get(ts.URL + "/api/cities/search?q=Rome")
	if err != nil {
		t.Fatalf("GET /api/cities/search failed: %v", err)
	}
	defer respSearch.Body.Close()
	if respSearch.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for city search, got %d", respSearch.StatusCode)
	}
	searchBody, _ := io.ReadAll(respSearch.Body)
	if !strings.Contains(string(searchBody), "Rome") || !strings.Contains(string(searchBody), "Italy") {
		t.Errorf("expected search result to contain Rome, Italy, got: %s", string(searchBody))
	}

	prefURL := fmt.Sprintf("%s/wizard/preferences?destination=%s&city=%s&country=%s&lat=41.8933203&lon=12.4829321", ts.URL, url.QueryEscape("Rome, Roma Capitale, Lazio, 00187, Italy"), url.QueryEscape("Rome"), url.QueryEscape("Italy"))
	respPref, err := client.Get(prefURL)
	if err != nil {
		t.Fatalf("GET /wizard/preferences failed: %v", err)
	}
	defer respPref.Body.Close()
	if respPref.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for preferences form, got %d", respPref.StatusCode)
	}
	prefBody, _ := io.ReadAll(respPref.Body)
	if !strings.Contains(string(prefBody), "Trip Preferences for Rome") {
		t.Errorf("expected Step 2 heading for Rome, got: %s", string(prefBody))
	}

	genForm := url.Values{
		"destination": {"Rome, Roma Capitale, Lazio, 00187, Italy"}, "city": {"Rome"}, "country": {"Italy"},
		"lat": {"41.8933203"}, "lon": {"12.4829321"}, "month": {"September"}, "duration_days": {"3"},
		"pace": {"Moderate"}, "interests": {"Architecture", "History", "Local Food & Living"},
		"mobility": {"Walking + Public Transit"}, "notes": {"Love historic cafes and gelato"},
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

	expectedElements := []string{
		"Rome", "September", "3 Days", "Moderate",
		"Classic Highlights",
		"Neighborhood &amp; Food Immersion",
		"Colosseum",
		"OpenStreetMap", "Google Maps", "Wikipedia",
	}
	for _, elem := range expectedElements {
		if !strings.Contains(tripStr, elem) {
			t.Errorf("expected trip page to contain %q, but was missing", elem)
		}
	}
	// Check stacked sections render segment chips and images
	if !strings.Contains(tripStr, "segment-chip") {
		t.Errorf("expected segment chips in trip page")
	}
	if !strings.Contains(tripStr, "<img") {
		t.Errorf("expected stop images in trip page")
	}
	// N-1 invariant: page should have at least one itinerary's stops interleaved with segments
	// Count occurrences of stop-card to verify at least 4 stops rendered
	if strings.Count(tripStr, "stop-card") < 4 {
		t.Errorf("expected at least 4 stop cards, got %d", strings.Count(tripStr, "stop-card"))
	}

	respSaved, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / saved check failed: %v", err)
	}
	defer respSaved.Body.Close()
	savedBody, _ := io.ReadAll(respSaved.Body)
	if !strings.Contains(string(savedBody), "Rome") || !strings.Contains(string(savedBody), redirectHeader) {
		t.Errorf("expected saved trip list on homepage to include Rome with link %s", redirectHeader)
	}
}
