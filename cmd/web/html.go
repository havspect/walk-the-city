package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/havspect/walk-the-city/internal/trip"
)

type htmlRenderer struct {
	templateFS      fs.FS
	sharedTemplates *template.Template
}

// newHTMLRenderer constructs an htmlRenderer that pre-parses shared layout and partial templates.
func newHTMLRenderer(templateFS fs.FS, sharedTemplateFiles ...string) (*htmlRenderer, error) {
	funcs := template.FuncMap{
		"now": time.Now,
		"formatDate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("Jan 02, 2006")
		},
		"osmLink": func(name string, lat, lon float64) template.URL {
			if lat != 0 && lon != 0 {
				return template.URL(fmt.Sprintf("https://www.openstreetmap.org/?mlat=%.6f&mlon=%.6f#map=16/%.6f/%.6f", lat, lon, lat, lon))
			}
			return template.URL(fmt.Sprintf("https://www.openstreetmap.org/search?query=%s", url.QueryEscape(name)))
		},
		"googleMapsLink": func(name string, lat, lon float64) template.URL {
			if lat != 0 && lon != 0 {
				return template.URL(fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%.6f,%.6f", lat, lon))
			}
			return template.URL(fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", url.QueryEscape(name)))
		},
		"wikiLink": func(args ...string) template.URL {
			var q string
			for _, a := range args {
				trimmed := strings.TrimSpace(a)
				if trimmed != "" {
					q = trimmed
					break
				}
			}
			if q == "" {
				return template.URL("https://en.wikipedia.org")
			}
			return template.URL(fmt.Sprintf("https://en.wikipedia.org/wiki/Special:Search?search=%s", url.QueryEscape(q)))
		},
		"categoryBadgeClass": func(category string) string {
			switch strings.ToLower(strings.ReplaceAll(category, "_", "")) {
			case "landmark", "architecture":
				return "badge-architecture"
			case "history":
				return "badge-history"
			case "fooddrink", "food & living", "food", "dining", "foodanddrink":
				return "badge-food"
			case "hiddengem", "hidden gem", "culture":
				return "badge-gem"
			case "parknature", "park & nature", "nature", "park":
				return "badge-nature"
			case "neighborhood":
				return "badge-neighborhood"
			case "trivia", "cool fact":
				return "badge-trivia"
			case "transit", "walk":
				return "badge-transit"
			default:
				return "badge-default"
			}
		},
		"categoryIcon": func(category string) string {
			switch strings.ToLower(strings.ReplaceAll(category, "_", "")) {
			case "landmark", "architecture":
				return "🏛️"
			case "history":
				return "📜"
			case "fooddrink", "food & living", "food", "dining", "foodanddrink":
				return "🍝"
			case "hiddengem", "hidden gem", "culture":
				return "💎"
			case "parknature", "park & nature", "nature", "park":
				return "🌳"
			case "neighborhood":
				return "🏘️"
			case "trivia", "cool fact":
				return "💡"
			case "transit":
				return "🚌"
			case "walk":
				return "🚶"
			case "bicycle":
				return "🚲"
			case "drive":
				return "🚗"
			default:
				return "📍"
			}
		},
		"hasItinerary": func(t *trip.Trip) bool {
			if t == nil {
				return false
			}
			return len(t.Itineraries) > 0
		},
		"hasItineraries": func(t *trip.Trip) bool {
			if t == nil {
				return false
			}
			return len(t.Itineraries) > 0
		},
		"formatDistance": func(meters int) string {
			if meters >= 1000 {
				return fmt.Sprintf("%.1f km", float64(meters)/1000)
			}
			return fmt.Sprintf("%d m", meters)
		},
		"modeIcon": func(mode string) string {
			switch strings.ToLower(mode) {
			case "walk":
				return "🚶"
			case "transit":
				return "🚌"
			case "bicycle":
				return "🚲"
			case "drive":
				return "🚗"
			default:
				return "➡️"
			}
		},
		"segmentLabel": func(s trip.Segment) string {
			var dist string
			if s.DistanceMeters >= 1000 {
				dist = fmt.Sprintf("%.1f km", float64(s.DistanceMeters)/1000)
			} else {
				dist = fmt.Sprintf("%d m", s.DistanceMeters)
			}
			return fmt.Sprintf("%s · %s · %d min", s.Mode, dist, s.DurationMinutes)
		},
	}

	sharedTemplates, err := template.New("").Funcs(funcs).ParseFS(templateFS, sharedTemplateFiles...)
	if err != nil {
		return nil, err
	}

	return &htmlRenderer{
		templateFS:      templateFS,
		sharedTemplates: sharedTemplates,
	}, nil
}

// render clones the shared template set, parses any additional page templates,
// executes into a buffer, sets the Vary header, and writes the response.
func (h *htmlRenderer) render(w http.ResponseWriter, status int, data any, templateName string, additionalTemplateFiles ...string) error {
	ts, err := h.sharedTemplates.Clone()
	if err != nil {
		return err
	}

	if len(additionalTemplateFiles) > 0 {
		ts, err = ts.ParseFS(h.templateFS, additionalTemplateFiles...)
		if err != nil {
			return err
		}
	}

	buf := new(bytes.Buffer)
	err = ts.ExecuteTemplate(buf, templateName, data)
	if err != nil {
		return err
	}

	w.Header().Set("Vary", "HX-Request")
	w.WriteHeader(status)
	_, err = buf.WriteTo(w)
	return err
}

// isHTMXRequest returns true if the incoming HTTP request was initiated by HTMX.
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
