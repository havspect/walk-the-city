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

func newTestApplication(t *testing.T) *application {
	t.Helper()
	dsn := fmt.Sprintf("file:memtest_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&trip.Trip{}, &trip.Itinerary{}, &trip.Stop{}, &trip.Segment{}); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	renderer, err := newHTMLRenderer(assets.HTMLFiles, "base.tmpl", "partials/*.tmpl")
	if err != nil {
		t.Fatalf("failed to create test htmlRenderer: %v", err)
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
	t.Cleanup(nomServer.Close)
	nomClient := nominatim.NewClient(nominatim.WithBaseURL(nomServer.URL), nominatim.WithRateLimit(0))
	app := &application{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		config: &config.Config{Port: "8080", DBPath: ":memory:", LogLevel: "debug", Env: "test"},
		html: renderer, staticFS: assets.StaticFiles,
		tripRepo: trip.NewService(db), nominatimClient: nomClient, tripGenerator: trip.NewMockGenerator(),
	}
	return app
}

func TestHandlers_Home(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()
	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("failed GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Walk The City") {
		t.Errorf("expected body to contain 'Walk The City'")
	}
	if !strings.Contains(bodyStr, "Step 1: Choose Your Destination") {
		t.Errorf("expected body to contain Step 1 heading")
	}
	if !strings.Contains(bodyStr, "Saved City Trips") {
		t.Errorf("expected body to contain Saved City Trips")
	}
}

func TestHandlers_SearchCities(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()
	resp, err := http.Get(server.URL + "/api/cities/search?q=Rome")
	if err != nil {
		t.Fatalf("failed search request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Rome") || !strings.Contains(bodyStr, "Italy") {
		t.Errorf("expected search results to contain Rome Italy, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "hx-get=\"/wizard/preferences") {
		t.Errorf("expected result item to carry hx-get attribute to /wizard/preferences")
	}
	respEmpty, err := http.Get(server.URL + "/api/cities/search?q=")
	if err != nil {
		t.Fatalf("failed empty search: %v", err)
	}
	defer respEmpty.Body.Close()
	if respEmpty.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for empty query, got %d", respEmpty.StatusCode)
	}
}

func TestHandlers_WizardPreferences(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()
	reqURL := fmt.Sprintf("%s/wizard/preferences?destination=%s&city=%s&country=%s&lat=41.8933&lon=12.4829", server.URL, url.QueryEscape("Rome, Lazio, Italy"), url.QueryEscape("Rome"), url.QueryEscape("Italy"))
	resp, err := http.Get(reqURL)
	if err != nil {
		t.Fatalf("failed GET /wizard/preferences: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Trip Preferences for Rome") {
		t.Errorf("expected preferences heading for Rome, got: %s", bodyStr)
	}
}

func TestHandlers_GenerateTrip_HTMX_Success(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()
	formData := url.Values{
		"destination": {"Rome, Lazio, Italy"}, "city": {"Rome"}, "country": {"Italy"},
		"lat": {"41.8933"}, "lon": {"12.4829"}, "month": {"September"}, "duration_days": {"3"},
		"pace": {"Moderate"}, "interests": {"Architecture", "History", "Local Food & Living"},
		"mobility": {"Walking + Public Transit"}, "notes": {"Staying near Monti"},
	}
	req, _ := http.NewRequest("POST", server.URL+"/trips/generate", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed POST /trips/generate: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
	hxRedirect := resp.Header.Get("HX-Redirect")
	if !strings.HasPrefix(hxRedirect, "/trips/") {
		t.Errorf("expected HX-Redirect header to start with '/trips/', got %q", hxRedirect)
	}
	respDetail, err := http.Get(server.URL + hxRedirect)
	if err != nil {
		t.Fatalf("failed to fetch trip detail: %v", err)
	}
	defer respDetail.Body.Close()
	if respDetail.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK from detail page, got %d", respDetail.StatusCode)
	}
	detailBody, _ := io.ReadAll(respDetail.Body)
	detailStr := string(detailBody)
	// Stacked itineraries: two themed sections
	if !strings.Contains(detailStr, "Classic Highlights") {
		t.Errorf("expected detail to contain 'Classic Highlights' itinerary")
	}
	if !strings.Contains(detailStr, "Neighborhood") {
		t.Errorf("expected detail to contain Neighborhood itinerary")
	}
	// Stops
	if !strings.Contains(detailStr, "Colosseum") {
		t.Errorf("expected detail page to contain Colosseum stop")
	}
	// Segment chip should render mode/distance/duration
	if !strings.Contains(detailStr, "segment-chip") && !strings.Contains(detailStr, "segment-") {
		// fallback: check for walk/transit label
		if !strings.Contains(detailStr, "walk") && !strings.Contains(detailStr, "transit") {
			t.Errorf("expected segment chip or mode label in detail page")
		}
	}
	// Every stop has image
	if !strings.Contains(detailStr, "<img") {
		t.Errorf("expected detail page to contain stop images")
	}
	// Deep links
	if !strings.Contains(detailStr, "OpenStreetMap") || !strings.Contains(detailStr, "Google Maps") {
		t.Errorf("expected deep links in detail page")
	}
}

func TestHandlers_GenerateTrip_ValidationError(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()
	formData := url.Values{"destination": {""}, "month": {""}, "duration_days": {"0"}}
	req, _ := http.NewRequest("POST", server.URL+"/trips/generate", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed POST /trips/generate: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Destination city is required") {
		t.Errorf("expected destination error in response, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Travel month is required") {
		t.Errorf("expected month error in response, got: %s", bodyStr)
	}
}

func TestHandlers_ShowTrip_NotFound(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()
	resp, err := http.Get(server.URL + "/trips/99999")
	if err != nil {
		t.Fatalf("failed GET /trips/99999: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

func TestHandlers_StaticAssets(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()
	respCSS, err := http.Get(server.URL + "/static/css/pico.min.css")
	if err != nil {
		t.Fatalf("failed to fetch pico.min.css: %v", err)
	}
	defer respCSS.Body.Close()
	if respCSS.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for pico.min.css, got %d", respCSS.StatusCode)
	}
	respCustomCSS, err := http.Get(server.URL + "/static/css/custom.css")
	if err != nil {
		t.Fatalf("failed to fetch custom.css: %v", err)
	}
	defer respCustomCSS.Body.Close()
	if respCustomCSS.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for custom.css, got %d", respCustomCSS.StatusCode)
	}
	respJS, err := http.Get(server.URL + "/static/js/htmx.min.js")
	if err != nil {
		t.Fatalf("failed to fetch htmx.min.js: %v", err)
	}
	defer respJS.Body.Close()
	if respJS.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for htmx.min.js, got %d", respJS.StatusCode)
	}
}
