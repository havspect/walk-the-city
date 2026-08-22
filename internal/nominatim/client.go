package nominatim

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL   = "https://nominatim.openstreetmap.org/search"
	defaultUserAgent = "walk-the-city/1.0 (+https://github.com/havspect/walk-the-city)"
	defaultCacheTTL  = 24 * time.Hour
	defaultMinDelay  = 1000 * time.Millisecond // 1 req/sec per Nominatim usage policy
)

// SearchResult represents a normalized geographic destination from Nominatim.
type SearchResult struct {
	PlaceID     int64   `json:"place_id"`
	DisplayName string  `json:"display_name"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
}

type nominatimAddress struct {
	City         string `json:"city"`
	Town         string `json:"town"`
	Village      string `json:"village"`
	Municipality string `json:"municipality"`
	County       string `json:"county"`
	State        string `json:"state"`
	Country      string `json:"country"`
	CountryCode  string `json:"country_code"`
}

type nominatimRawResult struct {
	PlaceID     int64            `json:"place_id"`
	DisplayName string           `json:"display_name"`
	Lat         string           `json:"lat"`
	Lon         string           `json:"lon"`
	Type        string           `json:"type"`
	Class       string           `json:"class"`
	Address     nominatimAddress `json:"address"`
}

type cacheEntry struct {
	results   []SearchResult
	expiresAt time.Time
}

// Client provides access to the OpenStreetMap Nominatim search API with caching and rate limiting.
type Client struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client
	cacheTTL   time.Duration
	minDelay   time.Duration

	cacheMu sync.RWMutex
	cache   map[string]cacheEntry

	rateMu   sync.Mutex
	lastReq  time.Time
}

// Option configures Client instances.
type Option func(*Client)

// WithBaseURL sets the base URL for Nominatim queries.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithUserAgent sets the User-Agent header for HTTP requests.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithCacheTTL configures the in-memory cache TTL.
func WithCacheTTL(ttl time.Duration) Option {
	return func(c *Client) {
		c.cacheTTL = ttl
	}
}

// WithRateLimit sets the minimum delay between consecutive outbound requests.
func WithRateLimit(delay time.Duration) Option {
	return func(c *Client) {
		c.minDelay = delay
	}
}

// NewClient returns a configured Nominatim Client with default rate limiting and caching.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		userAgent:  defaultUserAgent,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cacheTTL:   defaultCacheTTL,
		minDelay:   defaultMinDelay,
		cache:      make(map[string]cacheEntry),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Search queries Nominatim for matching cities or urban destinations.
func (c *Client) Search(ctx context.Context, query string) ([]SearchResult, error) {
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return []SearchResult{}, nil
	}

	cacheKey := strings.ToLower(cleanQuery)

	// Check cache
	c.cacheMu.RLock()
	if entry, ok := c.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		c.cacheMu.RUnlock()
		return entry.results, nil
	}
	c.cacheMu.RUnlock()

	// Enforce rate limiter
	c.rateMu.Lock()
	if !c.lastReq.IsZero() {
		elapsed := time.Since(c.lastReq)
		if elapsed < c.minDelay {
			sleepTime := c.minDelay - elapsed
			c.rateMu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(sleepTime):
			}
			c.rateMu.Lock()
		}
	}
	c.lastReq = time.Now()
	c.rateMu.Unlock()

	// Build request URL
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("q", cleanQuery)
	q.Set("format", "jsonv2")
	q.Set("addressdetails", "1")
	q.Set("featuretype", "city")
	q.Set("limit", "5")
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create nominatim request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nominatim request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nominatim api returned status %d", resp.StatusCode)
	}

	var rawResults []nominatimRawResult
	if err := json.NewDecoder(resp.Body).Decode(&rawResults); err != nil {
		return nil, fmt.Errorf("failed to decode nominatim response: %w", err)
	}

	results := make([]SearchResult, 0, len(rawResults))
	for _, raw := range rawResults {
		lat, _ := strconv.ParseFloat(raw.Lat, 64)
		lon, _ := strconv.ParseFloat(raw.Lon, 64)

		cityName := raw.Address.City
		if cityName == "" {
			cityName = raw.Address.Town
		}
		if cityName == "" {
			cityName = raw.Address.Municipality
		}
		if cityName == "" {
			cityName = raw.Address.Village
		}
		if cityName == "" {
			// Fallback: extract leading label before first comma in DisplayName
			parts := strings.Split(raw.DisplayName, ",")
			if len(parts) > 0 {
				cityName = strings.TrimSpace(parts[0])
			}
		}

		results = append(results, SearchResult{
			PlaceID:     raw.PlaceID,
			DisplayName: raw.DisplayName,
			City:        cityName,
			State:       raw.Address.State,
			Country:     raw.Address.Country,
			CountryCode: strings.ToUpper(raw.Address.CountryCode),
			Lat:         lat,
			Lon:         lon,
		})
	}

	// Store in cache
	c.cacheMu.Lock()
	c.cache[cacheKey] = cacheEntry{
		results:   results,
		expiresAt: time.Now().Add(c.cacheTTL),
	}
	c.cacheMu.Unlock()

	return results, nil
}
