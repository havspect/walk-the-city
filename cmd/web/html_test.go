package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/havspect/walk-the-city/internal/trip"
)

func TestHTMLRenderer_Render(t *testing.T) {
	mockFS := fstest.MapFS{
		"base.tmpl": &fstest.MapFile{
			Data: []byte(`{{define "base"}}<title>{{template "page:title" .}}</title><body>{{template "page:content" .}}</body>{{end}}`),
		},
		"partials/item.tmpl": &fstest.MapFile{
			Data: []byte(`{{define "item"}}<div>{{.}}</div>{{end}}`),
		},
		"pages/home.tmpl": &fstest.MapFile{
			Data: []byte(`{{define "page:title"}}Home{{end}}{{define "page:content"}}<h1>Hello {{.}}</h1>{{end}}`),
		},
	}

	renderer, err := newHTMLRenderer(mockFS, "base.tmpl", "partials/*.tmpl")
	if err != nil {
		t.Fatalf("failed to initialize htmlRenderer: %v", err)
	}

	// Test full page render
	rec := httptest.NewRecorder()
	err = renderer.render(rec, http.StatusOK, "World", "base", "pages/home.tmpl")
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if vary := rec.Header().Get("Vary"); vary != "HX-Request" {
		t.Errorf("expected Vary: HX-Request header, got %q", vary)
	}

	body := rec.Body.String()
	expected := `<title>Home</title><body><h1>Hello World</h1></body>`
	if body != expected {
		t.Errorf("expected body %q, got %q", expected, body)
	}

	// Test partial render
	recPartial := httptest.NewRecorder()
	err = renderer.render(recPartial, http.StatusOK, "Partial Content", "item")
	if err != nil {
		t.Fatalf("unexpected partial render error: %v", err)
	}

	if recPartial.Body.String() != `<div>Partial Content</div>` {
		t.Errorf("expected partial body %q, got %q", `<div>Partial Content</div>`, recPartial.Body.String())
	}
}

func TestTemplateHelpers(t *testing.T) {
	mockFS := fstest.MapFS{
		"base.tmpl": &fstest.MapFile{
			Data: []byte(`{{define "base"}}{{end}}`),
		},
		"partials/test.tmpl": &fstest.MapFile{
			Data: []byte(`{{define "test"}}<a href="{{osmLink "Rome" 41.8902 12.4922}}">OSM</a>|<a href="{{googleMapsLink "Pantheon" 41.8986 12.4769}}">Google</a>|<a href="{{wikiLink "Roman cuisine"}}">Wiki</a>{{end}}`),
		},
	}

	renderer, err := newHTMLRenderer(mockFS, "base.tmpl", "partials/*.tmpl")
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}

	rec := httptest.NewRecorder()
	if err := renderer.render(rec, http.StatusOK, nil, "test"); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	out := rec.Body.String()
	if !strings.Contains(out, "mlat=41.890200") || !strings.Contains(out, "mlon=12.492200") {
		t.Errorf("expected osm link in output, got: %s", out)
	}
	if !strings.Contains(out, "google.com/maps/search") || !strings.Contains(out, "41.898600,12.476900") {
		t.Errorf("expected google maps link in output, got: %s", out)
	}
	if !strings.Contains(out, "wikipedia.org/wiki/Special:Search") || !strings.Contains(out, "Roman") {
		t.Errorf("expected wiki link in output, got: %s", out)
	}
}

func TestHasItineraryHelper(t *testing.T) {
	tripWithoutIt := &trip.Trip{}
	tripWithIt := &trip.Trip{
		ItineraryJSON: `{"days":[{"day_number":1,"theme":"Historic Rome"}]}`,
	}

	mockFS := fstest.MapFS{
		"base.tmpl": &fstest.MapFile{Data: []byte(`{{define "base"}}{{end}}`)},
		"partials/check.tmpl": &fstest.MapFile{
			Data: []byte(`{{define "check"}}{{if hasItinerary .}}has_it{{else}}no_it{{end}}{{end}}`),
		},
	}

	renderer, err := newHTMLRenderer(mockFS, "base.tmpl", "partials/*.tmpl")
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}

	rec1 := httptest.NewRecorder()
	_ = renderer.render(rec1, http.StatusOK, tripWithoutIt, "check")
	if rec1.Body.String() != "no_it" {
		t.Errorf("expected 'no_it', got '%s'", rec1.Body.String())
	}

	rec2 := httptest.NewRecorder()
	_ = renderer.render(rec2, http.StatusOK, tripWithIt, "check")
	if rec2.Body.String() != "has_it" {
		t.Errorf("expected 'has_it', got '%s'", rec2.Body.String())
	}
}

func TestFormatDateHelper(t *testing.T) {
	d := time.Date(2026, time.September, 15, 10, 0, 0, 0, time.UTC)
	mockFS := fstest.MapFS{
		"base.tmpl": &fstest.MapFile{Data: []byte(`{{define "base"}}{{end}}`)},
		"partials/date.tmpl": &fstest.MapFile{
			Data: []byte(`{{define "date"}}{{formatDate .}}{{end}}`),
		},
	}
	renderer, err := newHTMLRenderer(mockFS, "base.tmpl", "partials/*.tmpl")
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}

	rec := httptest.NewRecorder()
	_ = renderer.render(rec, http.StatusOK, d, "date")
	if rec.Body.String() != "Sep 15, 2026" {
		t.Errorf("expected 'Sep 15, 2026', got '%s'", rec.Body.String())
	}
}

func TestIsHTMXRequest(t *testing.T) {
	reqHTMX, _ := http.NewRequest("GET", "/", nil)
	reqHTMX.Header.Set("HX-Request", "true")
	if !isHTMXRequest(reqHTMX) {
		t.Errorf("expected isHTMXRequest to return true for HX-Request: true")
	}

	reqStandard, _ := http.NewRequest("GET", "/", nil)
	if isHTMXRequest(reqStandard) {
		t.Errorf("expected isHTMXRequest to return false without header")
	}
}
