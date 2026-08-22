# Walk The City 🚶🏙️

A lightweight web application for generating and managing custom city travel itineraries, built with Go, GORM, SQLite, HTMX, and Pico.css.

## Tech Stack

- **Backend:** [Go 1.26+](https://go.dev/) with standard library `net/http` (Go 1.22+ routing patterns)
- **Database & ORM:** [GORM v2](https://gorm.io/) on SQLite via [`github.com/glebarez/sqlite`](https://github.com/glebarez/sqlite) (pure-Go, zero cgo requirements)
- **Frontend Interactivity:** [HTMX 4](https://four.htmx.org/) (vendored in embedded static assets)
- **UI Styling:** [Pico.css v2](https://picocss.com/) (classless/minimalist CSS framework, vendored)
- **Asset Packaging:** Go `embed.FS` compiled directly into the single application binary

---

## Project Structure

```
walk-the-city/
├── assets/                         # Embedded static files and HTML templates
│   ├── efs.go                      # embed.FS definitions (HTMLFiles, StaticFiles)
│   ├── html/                       # Go html/template hierarchy
│   │   ├── base.tmpl               # Main layout with Pico.css & HTMX
│   │   ├── pages/                  # Full-page content templates
│   │   │   ├── home.tmpl           # Trip planner wizard container & saved trips
│   │   │   └── trip_detail.tmpl    # Permanent rich itinerary detail view (/trips/{id})
│   │   └── partials/               # Reusable & HTMX swap partials
│   │       ├── step1_city.tmpl     # Step 1 destination lookup & autocomplete partial
│   │       ├── step2_preferences.tmpl # Step 2 preference questionnaire partial
│   │       ├── generation_loading.tmpl # Synthesis loading indicator
│   │       ├── trip_card.tmpl      # Unified highlight card component
│   │       ├── daily_route.tmpl    # Chronological daily walking/transit routes
│   │       ├── trip_row.tmpl       # Single trip card partial
│   │       ├── trip_list.tmpl      # Trip list container partial
│   │       └── form_errors.tmpl    # Validation error feedback partial
│   └── static/                     # Vendored assets & custom theme
│       ├── css/pico.min.css        # Pico CSS v2.1.1
│       ├── css/custom.css          # Custom badges, cards, and pill inputs
│       └── js/htmx.min.js          # HTMX 4.0.0-beta6
├── cmd/
│   └── web/                        # Web server binary entrypoint
│       ├── main.go                 # Server initialization, config, graceful shutdown
│       ├── handlers.go             # HTTP route handlers & middleware
│       ├── handlers_test.go        # HTTP integration test suite
│       ├── html.go                 # htmlRenderer engine with template caching & deep-link helpers
│       ├── html_test.go            # HTML renderer & helper unit tests
│       └── wizard_integration_test.go # Full end-to-end wizard flow integration tests
├── internal/                       # Internal application packages
│   ├── config/                     # Environment configuration loader
│   │   ├── config.go
│   │   └── config_test.go
│   ├── database/                   # SQLite connection & auto-migration
│   │   ├── database.go
│   │   └── database_test.go
│   ├── nominatim/                  # OpenStreetMap Nominatim search client & caching
│   │   ├── client.go
│   │   └── client_test.go
│   └── trip/                       # Trip domain entities, service seam & generation
│       ├── model.go                # Trip entity & typed Itinerary/DayPlan/Stop/Card models
│       ├── model_test.go           # Domain model serialization tests
│       ├── service.go              # TripService business interface & repo
│       ├── service_test.go         # Service unit test suite
│       ├── generator.go            # TripGenerator interface
│       ├── generator_mock.go       # Deterministic mock/placeholder generator
│       └── generator_test.go       # Generator unit tests
├── docs/plans/                     # Technical specifications & plans
├── .env.example                    # Sample environment variables
├── go.mod                          # Go module dependencies
└── README.md                       # Project documentation
```

---

## Prerequisites

- [Go](https://go.dev/doc/install) version **1.26** or higher.
- *(Optional)* [Air](https://github.com/air-verse/air) for live reloading during local development.

No C compiler (gcc/clang) or Docker containers are required to build, test, or run this project.

---

## Getting Started

### 1. Clone and Configure

```sh
# Copy sample environment configuration
cp .env.example .env
```

### 2. Run the Application

```sh
# Run the web server
go run ./cmd/web
```

The application will start on `http://localhost:8080`.

Open your browser at `http://localhost:8080` to view the city trip planner interface, create new trip itineraries, and view saved trips.

---

## Configuration

Settings are configured via environment variables or a local `.env` file:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server listening port |
| `DB_PATH` | `data/walkthecity.db` | File path for SQLite database |
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `ENV` | `development` | Environment name (`development`, `production`, `test`) |

---

## Testing & Quality Gates

The test suite runs entirely with in-memory SQLite instances and requires zero cgo dependencies.

```sh
# Run all tests
go test -v ./...

# Verify pure-Go / zero-cgo compilation and test execution
CGO_ENABLED=0 go test -v ./...

# Build binary
CGO_ENABLED=0 go build -o walkthecity ./cmd/web
```

---

## Architectural Notes

### 1. Alex Edwards HTMX & Templating Pattern
The application follows the [HTMX with Go pattern by Alex Edwards](https://www.alexedwards.net/blog/how-i-use-htmx-with-go):
- Templates and static assets are embedded via `assets.HTMLFiles` and `assets.StaticFiles`.
- The `htmlRenderer` in `cmd/web/html.go` pre-parses shared layout and partial templates on startup and clones the template set on demand to render full-page views or targeted HTMX partials.
- Responses set `Vary: HX-Request` to ensure caching layers differentiate between partial and full-page responses.
- Validation errors on HTMX form submissions issue `422 Unprocessable Entity` with `HX-Retarget: #form-errors` and `HX-Reswap: innerHTML` to render error feedback in place.

### 2. City Trip Planning Wizard & Agent Integration Seam
The application provides a 2-step HTMX trip planning flow:
1. **Step 1 (City Search):** Uses a server-proxied OpenStreetMap Nominatim client in `internal/nominatim` with in-memory TTL caching and rate limiting.
2. **Step 2 (Preferences):** Collects the travel month and 3 structured inputs:
   - Exploration Pace (Relaxed, Moderate, Packed)
   - Core Interests (Architecture, History, Local Food & Living, Hidden Gems, Parks)
   - Preferred Mobility (Walking + Transit, Walking Only, Bicycle, Accessible)
3. **Step 3 (Itinerary Generation & Permanent Link):** Synthesizes a hybrid itinerary with daily routes and unified visual cards, persists to SQLite, and displays at a permanent `/trips/{id}` URL with OpenStreetMap, Google Maps, and Wikipedia deep links.

The generation engine is decoupled behind the `TripGenerator` interface in `internal/trip/generator.go`:
```go
type TripGenerator interface {
    Generate(ctx context.Context, params CreateTripParams) (*Itinerary, error)
}
```
A deterministic `MockGenerator` is used by default for zero-setup local development and automated CI testing. Future LLM/agent implementations can implement `TripGenerator` to provide dynamic AI synthesis.
