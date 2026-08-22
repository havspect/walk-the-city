package main

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/havspect/walk-the-city/internal/nominatim"
	"github.com/havspect/walk-the-city/internal/trip"
)

type homeData struct {
	Trips      []trip.Trip
	FormValues trip.CreateTripParams
	FormErrors []string
}

type searchData struct {
	Query   string
	Results []nominatim.SearchResult
}

type preferencesData struct {
	Destination  string
	City         string
	Country      string
	Lat          float64
	Lon          float64
	Month        string
	DurationDays int
	Pace         string
	Mobility     string
	Notes        string
	Errors       []string
}

type tripDetailData struct {
	Trip      *trip.Trip
	Itinerary *trip.Itinerary
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	trips, err := app.tripService.ListTrips(r.Context())
	if err != nil {
		app.logger.Error("failed to list trips", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data := homeData{
		Trips: trips,
		FormValues: trip.CreateTripParams{
			DurationDays: 3,
		},
	}

	if err := app.html.render(w, http.StatusOK, data, "base", "pages/home.tmpl"); err != nil {
		app.logger.Error("failed to render home page", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (app *application) searchCities(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		_ = app.html.render(w, http.StatusOK, searchData{}, "city_search_results")
		return
	}

	results, err := app.nominatimClient.Search(r.Context(), q)
	if err != nil {
		app.logger.Warn("nominatim search error", "query", q, "error", err)
		_ = app.html.render(w, http.StatusOK, searchData{Query: q}, "city_search_results")
		return
	}

	data := searchData{
		Query:   q,
		Results: results,
	}

	if err := app.html.render(w, http.StatusOK, data, "city_search_results"); err != nil {
		app.logger.Error("failed to render search results partial", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (app *application) wizardPreferences(w http.ResponseWriter, r *http.Request) {
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, _ := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)

	data := preferencesData{
		Destination:  r.URL.Query().Get("destination"),
		City:         r.URL.Query().Get("city"),
		Country:      r.URL.Query().Get("country"),
		Lat:          lat,
		Lon:          lon,
		Month:        "September",
		DurationDays: 3,
		Pace:         "Moderate",
		Mobility:     "Walking + Public Transit",
	}

	if err := app.html.render(w, http.StatusOK, data, "step2_preferences"); err != nil {
		app.logger.Error("failed to render wizard preferences partial", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (app *application) generateTrip(w http.ResponseWriter, r *http.Request) {
	// Limit form payload size to 16KB
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	lat, _ := strconv.ParseFloat(r.PostFormValue("lat"), 64)
	lon, _ := strconv.ParseFloat(r.PostFormValue("lon"), 64)
	duration, _ := strconv.Atoi(r.PostFormValue("duration_days"))
	if duration <= 0 {
		duration = 3
	}

	interests := r.PostForm["interests"]
	month := strings.TrimSpace(r.PostFormValue("month"))
	destination := strings.TrimSpace(r.PostFormValue("destination"))
	city := strings.TrimSpace(r.PostFormValue("city"))
	country := strings.TrimSpace(r.PostFormValue("country"))
	pace := strings.TrimSpace(r.PostFormValue("pace"))
	mobility := strings.TrimSpace(r.PostFormValue("mobility"))
	notes := strings.TrimSpace(r.PostFormValue("notes"))

	// Validation checks
	var validationErrors []string
	if destination == "" {
		validationErrors = append(validationErrors, "Destination city is required")
	}
	if month == "" {
		validationErrors = append(validationErrors, "Travel month is required")
	}
	if duration < 1 || duration > 30 {
		validationErrors = append(validationErrors, "Duration must be between 1 and 30 days")
	}
	if len(interests) == 0 {
		validationErrors = append(validationErrors, "Please select at least one core interest")
	}

	if len(validationErrors) > 0 {
		prefData := preferencesData{
			Destination:  destination,
			City:         city,
			Country:      country,
			Lat:          lat,
			Lon:          lon,
			Month:        month,
			DurationDays: duration,
			Pace:         pace,
			Mobility:     mobility,
			Notes:        notes,
			Errors:       validationErrors,
		}

		if isHTMXRequest(r) {
			if err := app.html.render(w, http.StatusUnprocessableEntity, prefData, "step2_preferences"); err != nil {
				app.logger.Error("failed to render preferences validation errors", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		http.Error(w, strings.Join(validationErrors, ", "), http.StatusUnprocessableEntity)
		return
	}

	params := trip.CreateTripParams{
		Destination:  destination,
		City:         city,
		Country:      country,
		Lat:          lat,
		Lon:          lon,
		Month:        month,
		DurationDays: duration,
		Pace:         pace,
		Interests:    interests,
		Mobility:     mobility,
		Notes:        notes,
	}

	createdTrip, err := app.tripService.GenerateAndSaveTrip(r.Context(), params, app.tripGenerator)
	if err != nil {
		app.logger.Error("failed to generate and save trip", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tripURL := fmt.Sprintf("/trips/%d", createdTrip.ID)

	if isHTMXRequest(r) {
		w.Header().Set("HX-Redirect", tripURL)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, tripURL, http.StatusSeeOther)
}

func (app *application) showTrip(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	idVal, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || idVal == 0 {
		http.NotFound(w, r)
		return
	}

	t, err := app.tripService.GetTripByID(r.Context(), uint(idVal))
	if err != nil {
		if errors.Is(err, trip.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		app.logger.Error("failed to retrieve trip", "id", idVal, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	it, err := t.GetItinerary()
	if err != nil {
		app.logger.Warn("failed to parse trip itinerary json", "trip_id", t.ID, "error", err)
	}

	data := tripDetailData{
		Trip:      t,
		Itinerary: it,
	}

	if err := app.html.render(w, http.StatusOK, data, "base", "pages/trip_detail.tmpl"); err != nil {
		app.logger.Error("failed to render trip detail page", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (app *application) createTrip(w http.ResponseWriter, r *http.Request) {
	// Limit form body size to 4KB
	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	duration, _ := strconv.Atoi(r.PostFormValue("duration_days"))
	params := trip.CreateTripParams{
		Destination:  r.PostFormValue("destination"),
		DurationDays: duration,
		Notes:        r.PostFormValue("notes"),
	}

	created, err := app.tripService.CreateTrip(r.Context(), params)
	if err != nil {
		var valErr *trip.ValidationError
		if errors.As(err, &valErr) {
			keys := make([]string, 0, len(valErr.FieldErrors))
			for field := range valErr.FieldErrors {
				keys = append(keys, field)
			}
			sort.Strings(keys)

			var errMsgs []string
			for _, field := range keys {
				errMsgs = append(errMsgs, valErr.FieldErrors[field])
			}

			if isHTMXRequest(r) {
				w.Header().Set("HX-Retarget", "#form-errors")
				w.Header().Set("HX-Reswap", "innerHTML")
				if renderErr := app.html.render(w, http.StatusUnprocessableEntity, errMsgs, "form_errors"); renderErr != nil {
					app.logger.Error("failed to render validation errors", "error", renderErr)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				return
			}

			// Non-HTMX validation error fallback
			trips, listErr := app.tripService.ListTrips(r.Context())
			if listErr != nil {
				app.logger.Error("failed to list trips for error page", "error", listErr)
			}
			data := homeData{
				Trips:      trips,
				FormValues: params,
				FormErrors: errMsgs,
			}
			if renderErr := app.html.render(w, http.StatusUnprocessableEntity, data, "base", "pages/home.tmpl"); renderErr != nil {
				app.logger.Error("failed to render home page with errors", "error", renderErr)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		app.logger.Error("unexpected error creating trip", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Success response
	if isHTMXRequest(r) {
		if err := app.html.render(w, http.StatusOK, created, "trip_row"); err != nil {
			app.logger.Error("failed to render trip row", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	// Standard browser form submission redirect
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// routes builds the application http.Handler with static file serving and route endpoints.
func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// Static asset serving from embedded filesystem
	staticServer := http.FileServerFS(app.staticFS)
	mux.Handle("GET /static/", http.StripPrefix("/static", staticServer))

	// Application routes
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /api/cities/search", app.searchCities)
	mux.HandleFunc("GET /wizard/preferences", app.wizardPreferences)
	mux.HandleFunc("POST /trips/generate", app.generateTrip)
	mux.HandleFunc("GET /trips/{id}", app.showTrip)
	mux.HandleFunc("POST /trips", app.createTrip)

	return app.logRequest(app.recoverPanic(mux))
}

// Middleware: log incoming requests
func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.logger.Debug("http request", "method", r.Method, "url", r.URL.String(), "remote", r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// Middleware: recover from panics gracefully
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.logger.Error("panic recovered", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
