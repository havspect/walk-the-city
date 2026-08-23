package nominatim

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSearch_Success(t *testing.T) {
	mockResponse := []nominatimRawResult{
		{
			PlaceID:     1001,
			DisplayName: "Rome, Roma Capitale, Lazio, 00187, Italy",
			Lat:         "41.8933203",
			Lon:         "12.4829321",
			Address: nominatimAddress{
				City:        "Rome",
				State:       "Lazio",
				Country:     "Italy",
				CountryCode: "it",
			},
		},
	}

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		// Assert compliant User-Agent
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("expected non-empty User-Agent header")
		}

		// Assert query parameters
		q := r.URL.Query().Get("q")
		if q != "Rome" {
			t.Errorf("expected query 'Rome', got '%s'", q)
		}
		if r.URL.Query().Get("format") != "jsonv2" {
			t.Errorf("expected format 'jsonv2', got '%s'", r.URL.Query().Get("format"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithUserAgent("walk-the-city-test/1.0"),
		WithCacheTTL(5*time.Minute),
	)

	results, err := client.Search(context.Background(), "Rome")
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if res.PlaceID != 1001 {
		t.Errorf("expected PlaceID 1001, got %d", res.PlaceID)
	}
	if res.City != "Rome" {
		t.Errorf("expected City 'Rome', got '%s'", res.City)
	}
	if res.State != "Lazio" {
		t.Errorf("expected State 'Lazio', got '%s'", res.State)
	}
	if res.Country != "Italy" {
		t.Errorf("expected Country 'Italy', got '%s'", res.Country)
	}
	if res.Lat != 41.8933203 || res.Lon != 12.4829321 {
		t.Errorf("expected coords (41.8933203, 12.4829321), got (%f, %f)", res.Lat, res.Lon)
	}
}

func TestSearch_Caching(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		mockResponse := []nominatimRawResult{
			{
				PlaceID:     2002,
				DisplayName: "Paris, Île-de-France, France",
				Lat:         "48.8534951",
				Lon:         "2.3483915",
				Address: nominatimAddress{
					City:    "Paris",
					Country: "France",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithCacheTTL(1*time.Hour),
	)

	// First request -> hits server
	res1, err := client.Search(context.Background(), "Paris")
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}
	if len(res1) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res1))
	}

	// Second request with same query -> hits cache
	res2, err := client.Search(context.Background(), "Paris")
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}
	if len(res2) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res2))
	}

	if count := atomic.LoadInt32(&requestCount); count != 1 {
		t.Errorf("expected exactly 1 server request due to cache, got %d", count)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	client := NewClient()
	results, err := client.Search(context.Background(), "   ")
	if err != nil {
		t.Fatalf("unexpected error for empty query: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestSearch_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	_, err := client.Search(context.Background(), "Berlin")
	if err == nil {
		t.Errorf("expected error on 500 response, got nil")
	}
}
