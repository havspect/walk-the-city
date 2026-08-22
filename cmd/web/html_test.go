package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
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
