package main

import (
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
	"github.com/havspect/walk-the-city/internal/trip"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestApplication(t *testing.T) *application {
	t.Helper()

	dsn := fmt.Sprintf("file:memtest_%d?mode=memory&cache=shared", time.Now().UnixNano())
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
		t.Fatalf("failed to create test htmlRenderer: %v", err)
	}

	app := &application{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		config: &config.Config{
			Port:     "8080",
			DBPath:   ":memory:",
			LogLevel: "debug",
			Env:      "test",
		},
		html:        renderer,
		staticFS:    assets.StaticFiles,
		tripService: trip.NewService(db),
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
	if !strings.Contains(bodyStr, "Plan a Custom City Trip") {
		t.Errorf("expected body to contain 'Plan a Custom City Trip'")
	}
	if !strings.Contains(bodyStr, "No city trips planned yet") {
		t.Errorf("expected body to contain empty state message")
	}
}

func TestHandlers_CreateTrip_HTMX_Success(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()

	formData := url.Values{
		"destination":   {"Amsterdam"},
		"duration_days": {"4"},
		"notes":         {"Canal tour, Jordaan, Van Gogh Museum"},
	}

	req, _ := http.NewRequest("POST", server.URL+"/trips", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed POST /trips: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Amsterdam") {
		t.Errorf("expected partial to contain 'Amsterdam', got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "4 days") {
		t.Errorf("expected partial to contain '4 days', got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Canal tour") {
		t.Errorf("expected partial to contain notes, got: %s", bodyStr)
	}
	// Verify it's a partial (does not contain html/base layout tags)
	if strings.Contains(bodyStr, "<!doctype html>") {
		t.Errorf("expected partial response, but got full page layout")
	}
}

func TestHandlers_CreateTrip_HTMX_ValidationError(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()

	formData := url.Values{
		"destination":   {""}, // missing destination
		"duration_days": {"0"},
	}

	req, _ := http.NewRequest("POST", server.URL+"/trips", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed POST /trips: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 Unprocessable Entity, got %d", resp.StatusCode)
	}

	if retarget := resp.Header.Get("HX-Retarget"); retarget != "#form-errors" {
		t.Errorf("expected HX-Retarget: #form-errors, got %q", retarget)
	}
	if reswap := resp.Header.Get("HX-Reswap"); reswap != "innerHTML" {
		t.Errorf("expected HX-Reswap: innerHTML, got %q", reswap)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Destination is required") {
		t.Errorf("expected error message in response, got: %s", bodyStr)
	}
}

func TestHandlers_CreateTrip_StandardBrowser_Success(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()

	formData := url.Values{
		"destination":   {"Barcelona"},
		"duration_days": {"3"},
		"notes":         {"Sagrada Familia, Park Guell"},
	}

	// Disable auto-following redirects so we can inspect the 303 status
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.PostForm(server.URL+"/trips", formData)
	if err != nil {
		t.Fatalf("failed POST /trips: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("expected 303 See Other, got %d", resp.StatusCode)
	}

	if loc := resp.Header.Get("Location"); loc != "/" {
		t.Errorf("expected redirect to '/', got %q", loc)
	}
}

func TestHandlers_StaticAssets(t *testing.T) {
	app := newTestApplication(t)
	server := httptest.NewServer(app.routes())
	defer server.Close()

	// Check Pico CSS
	respCSS, err := http.Get(server.URL + "/static/css/pico.min.css")
	if err != nil {
		t.Fatalf("failed to fetch pico.min.css: %v", err)
	}
	defer respCSS.Body.Close()

	if respCSS.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for pico.min.css, got %d", respCSS.StatusCode)
	}
	if !strings.Contains(respCSS.Header.Get("Content-Type"), "text/css") {
		t.Errorf("expected text/css content type, got %s", respCSS.Header.Get("Content-Type"))
	}

	// Check HTMX JS
	respJS, err := http.Get(server.URL + "/static/js/htmx.min.js")
	if err != nil {
		t.Fatalf("failed to fetch htmx.min.js: %v", err)
	}
	defer respJS.Body.Close()

	if respJS.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for htmx.min.js, got %d", respJS.StatusCode)
	}
}
