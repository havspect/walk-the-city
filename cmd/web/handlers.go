package main

import (
	"errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/havspect/walk-the-city/internal/trip"
)

type homeData struct {
	Trips      []trip.Trip
	FormValues trip.CreateTripParams
	FormErrors []string
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

func (app *application) createTrip(w http.ResponseWriter, r *http.Request) {
	// Limit form body size to 4KB to prevent oversized payloads
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

			// Non-HTMX validation error fallback: re-render full page with 422
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

	// Standard browser form submission redirect (PRG pattern)
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
