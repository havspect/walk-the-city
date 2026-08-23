---
title: Initial Scaffold and Trip Persistence - Plan
type: feat
date: 2026-08-22
artifact_contract: ce-unified-plan/v1
artifact_readiness: implementation-ready
product_contract_source: ce-plan-bootstrap
execution: code
---

## Goal Capsule

- **Objective:** Establish a clean, documented, and fully buildable Go 1.26 repository scaffold combining an Alex Edwards-style HTMX template architecture, GORM persistence on SQLite, and a minimal end-to-end trip creation slice that serves as a decoupled foundation for future LLM agent integration.
- **Means:** Complete the existing `cmd/web` and `assets` scaffold, implement an `embed.FS`-backed `htmlRenderer`, wire `glebarez/sqlite` via GORM, define an `internal/trip` service seam, and provide comprehensive README documentation and automated tests (KTD1, KTD2, KTD3, KTD5).
- **Authority:** Product Contract governs all requirements; Planning Contract governs architectural decisions; `CGO_ENABLED=0 go test ./...` and `go build ./...` serve as the primary verification gates.
- **Stop Conditions:** All Go files compile cleanly with `CGO_ENABLED=0`, automated test suites pass, database migrations run on startup, the home page renders and accepts trip creation via HTMX, and the repository is committed in a clean state.
- **Execution Profile:** Sequential unit implementation ordered by foundational dependencies: Hygiene & Config -> Embedded Templates -> Database & Models -> Service Seam -> Web Handlers & HTMX Slice -> Docs & Verification.

---

## Product Contract

### Summary

The initial repository setup for *walk-the-city* provides a Go web application foundation using HTMX, Pico.css, and SQLite via GORM. The repository delivers a complete web server with embedded static assets, an `htmlRenderer` handling both full-page and partial HTML responses, an `internal/trip` service layer, and an end-to-end slice allowing users to create and view city trips.

### Problem Frame

The project repository currently contains only an uncommitted skeleton with empty Go placeholder files (`cmd/web/main.go`, `cmd/web/html.go`, `cmd/web/handlers.go`, `assets/efs.go`), a conflicting root `main.go`, and no declared Go module dependencies. Before introducing complex AI or LLM agent workflows, the project requires a rock-solid, well-documented foundational scaffold with verified template rendering, clean persistence, and zero-cgo build ergonomics.

### Requirements

#### Repository & Configuration
- R1. Clean entrypoint architecture: Remove the conflicting root `main.go` and establish `cmd/web/main.go` as the sole application binary entrypoint.
- R2. Environment configuration: Load application settings (server port, SQLite database path, log level) from environment variables with safe defaults, supporting optional `.env` file loading for local development.
- R3. Git hygiene: Update `.gitignore` to ignore SQLite database artifacts (`*.db`, `*.sqlite`, `data/`) and local `.env` files.

#### Web & Templating
- R4. Embedded asset filesystem: Export sub-filesystems for HTML templates (`assets/html`) and static assets (`assets/static`) from an embedded filesystem root in `assets/efs.go`.
- R5. Reusable HTML renderer: Provide an `htmlRenderer` type that pre-parses shared templates (`base.tmpl`, `partials/*.tmpl`) on startup and clones the template set to execute named templates for full-page or partial responses.
- R6. HTMX response handling: Support progressive enhancement and HTMX partial swaps by inspecting the `HX-Request` header, setting `Vary: HX-Request` on responses, and issuing `HX-Redirect` headers when appropriate.
- R7. Styling and vendored assets: Serve Pico.css v2.1.1 and HTMX 4.0.0-beta6 from embedded static storage with correct MIME types and caching headers.

#### Persistence & Domain
- R8. Pure-Go SQLite persistence: Configure GORM using the pure-Go `github.com/glebarez/sqlite` driver, ensuring zero cgo requirements during compilation and automated testing.
- R9. Automatic schema migration: Automatically migrate the database schema on application startup, creating the SQLite file and parent directories if they do not exist.
- R10. Trip data model: Define a `Trip` domain entity with fields for ID, Destination, DurationDays, Notes, CreatedAt, and UpdatedAt.

#### Service Layer & Seams
- R11. Decoupled trip service seam: Define a `TripService` interface exposing methods to create, list, and retrieve trips, keeping HTTP handlers decoupled from database logic and providing a ready integration seam for future LLM agent generation.

#### End-to-End Slice
- R12. Trip creation and listing UI: Provide a responsive homepage with Pico.css styling that displays existing city trips and includes a form to submit a new trip.
- R13. HTMX interactivity: When submitted via HTMX, validate input, save the trip through the service layer, and dynamically append the new trip row or swap the list without a full-page reload.
- R14. Validation error feedback: Return a `422 Unprocessable Entity` response with inline error messages when trip form submissions fail validation.
- R15. Non-HTMX fallback: Ensure standard browser form POST requests without HTMX redirect gracefully to the homepage following the Post/Redirect/Get pattern.

#### Documentation & Testing
- R16. Comprehensive documentation: Write a complete `README.md` documenting prerequisites (Go 1.26+), repository layout, configuration options, running locally, and running tests.
- R17. Test coverage: Provide automated tests for configuration parsing, database initialization/migrations, service operations, and HTTP handler responses using in-memory SQLite and `httptest`.

### Key Decisions

- **Database Engine & Driver:** SQLite via pure-Go driver `github.com/glebarez/sqlite` `(session-settled: user-directed — chosen over Postgres and cgo-based mattn/go-sqlite3: eliminates external container dependencies and enables pure-Go CGO_ENABLED=0 builds)`. Governs R8, R9, R17.
- **Agent Integration Scope:** Exclude LLM/agent frameworks from this initial setup `(session-settled: user-directed — chosen over wiring a mock/placeholder agent library: keeps initial setup clean, robust, and focused on core web/data plumbing)`. Governs R11.
- **Setup Depth:** Include a minimal end-to-end trip creation slice `(session-settled: user-directed — chosen over empty skeleton: proves the template rendering, HTMX interaction, and GORM persistence pipeline end-to-end)`. Governs R12, R13, R14, R15.
- **HTMX Version Policy:** Retain vendored HTMX `4.0.0-beta6` and align template attributes with HTMX 4 semantics `(session-settled: user-approved — chosen over downgrading to 2.0.x: respects scaffold state, relies on stable core hypermedia attributes, and simplifies forward migration when 4.0 reaches GA)`. Governs R6, R7.

### Scope Boundaries

#### In Scope
- Completing the `cmd/web` server, `assets/efs.go`, and HTML template hierarchy.
- Adding GORM v2 and `github.com/glebarez/sqlite` dependencies to `go.mod`.
- Configuration loading package in `internal/config`.
- Trip model and GORM repository in `internal/trip`.
- Service interface and implementation in `internal/trip`.
- HTTP handlers for `GET /`, `POST /trips`, and static file serving.
- Full-page and partial HTML templates for trip listing and creation.
- Table-driven unit and integration tests using in-memory SQLite.
- Clear `README.md` and `.env.example` documentation.

#### Deferred to Follow-Up Work
- LLM agent service integration (e.g., OpenAI API, tool-calling loop, itinerary generation prompt engineering).
- User authentication, session management, and authorization.
- Interactive map rendering (Leaflet/MapLibre) and geolocation lookup.
- Background worker queues for long-running trip generation.
- Production deployment configurations (Dockerfiles, Kamal, Fly.io/render configs, CI workflows).

---

## Planning Contract

### Key Technical Decisions

- KTD1. **Project Directory Organization:** Structure the codebase into `cmd/web` for the HTTP server application, `assets/` for embedded static files and templates, and `internal/` (`internal/config`, `internal/database`, `internal/trip`) for application packages. Delete root `main.go`.
- KTD2. **Pure-Go SQLite Driver (`glebarez/sqlite`):** `(session-settled: user-directed — chosen over gorm.io/driver/sqlite: enables cross-platform compilation without cgo toolchain requirements)`. Use connection string format `file:data/walkthecity.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)` for file databases, and isolated in-memory connection strings (e.g. `file:memdb_test?mode=memory&cache=shared`) or a single-connection in-memory database for isolated unit tests.
- KTD3. **Alex Edwards `htmlRenderer` Architecture:** Construct an `htmlRenderer` in `cmd/web/html.go` that parses shared templates (`base.tmpl`, `partials/*.tmpl`) at startup. Handlers call `render(w, status, data, templateName, additionalFiles...)`, which clones the shared template set, executes into a `bytes.Buffer`, sets `Vary: HX-Request`, and writes the response.
- KTD4. **HTMX 4 Configuration & Semantics:** In `base.tmpl`, include the HTMX 4 script tag. Rely on HTMX 4 explicit inheritance, native fetch API, and default swap behaviors. For validation errors (422 Unprocessable Entity), handlers set `HX-Retarget: #form-errors` and `HX-Reswap: innerHTML` so validation error fragments render in the form error container rather than corrupting the trip list.
- KTD5. **Service Layer Seam (`TripService`):** Implement `TripService` in `internal/trip/service.go` defining `CreateTrip(ctx, params)`, `ListTrips(ctx)`, and `GetTripByID(ctx, id)`. Handlers interact exclusively with the service interface.
- KTD6. **Configuration Management:** Define `Config` struct in `internal/config/config.go` with fields `Port` (default `8080`), `DBPath` (default `data/walkthecity.db`), `LogLevel` (default `info`), and `Env` (default `development`). Attempt loading `.env` via `godotenv.Load()` if present, ignoring `os.ErrNotExist`.

### High-Level Technical Design

```mermaid
flowchart TB
    subgraph Browser["Client Browser (Pico.css + HTMX 4)"]
        Page["Full Page (GET /)"]
        HTMXReq["HTMX Swap (POST /trips)"]
    end

    subgraph WebLayer["cmd/web (HTTP Layer)"]
        Mux["http.ServeMux (Go 1.22+ Routing)"]
        Renderer["htmlRenderer (embed.FS)"]
        Handlers["HTTP Handlers (app context)"]
    end

    subgraph InternalLayer["internal/ (Domain & Persistence)"]
        TripSvc["TripService Interface"]
        GormRepo["GORM Repository"]
        DBConn["GORM DB (glebarez/sqlite)"]
        ConfigPkg["internal/config"]
    end

    subgraph Storage["Storage & Assets"]
        Assets["assets/ (embed.FS: html, static)"]
        SQLiteFile["data/walkthecity.db (WAL mode)"]
    end

    Page -->|GET /| Mux
    HTMXReq -->|POST /trips (HX-Request: true)| Mux
    Mux --> Handlers
    Handlers -->|Render Full Page or Partial| Renderer
    Renderer -->|Reads Templates| Assets
    Handlers -->|Invokes Seam| TripSvc
    TripSvc --> GormRepo
    GormRepo --> DBConn
    DBConn --> SQLiteFile
    Handlers -.-> ConfigPkg
```

### Output Structure

```
walk-the-city/
├── .gitignore                      # Updated: ignores *.db, data/, .env
├── .env.example                    # Sample environment variables
├── README.md                       # Full documentation: setup, run, test
├── go.mod                          # Go module with GORM + glebarez/sqlite + godotenv
├── go.sum                          # Checksums
├── assets/
│   ├── efs.go                      # embed.FS definitions for HTMLFiles and StaticFiles
│   ├── html/
│   │   ├── base.tmpl               # Base layout with Pico.css and HTMX 4
│   │   ├── pages/
│   │   │   └── home.tmpl           # Homepage with trip list and creation form
│   │   └── partials/
│   │       ├── trip_row.tmpl       # Partial row template for single trip
│   │       └── trip_list.tmpl      # Partial list template for updated trips
│   └── static/
│       ├── css/
│       │   └── pico.min.css        # Pico CSS v2.1.1 (already vendored)
│       └── js/
│           └── htmx.min.js         # HTMX 4.0.0-beta6 (already vendored)
├── cmd/
│   └── web/
│       ├── main.go                 # Server initialization, routing, startup
│       ├── handlers.go             # HTTP handlers (home, createTrip, static)
│       ├── handlers_test.go        # HTTP handler tests using httptest
│       └── html.go                 # htmlRenderer implementation over embed.FS
└── internal/
    ├── config/
    │   ├── config.go               # Env-based configuration loader
    │   └── config_test.go          # Config parsing tests
    ├── database/
    │   ├── database.go             # GORM SQLite connection and auto-migration
    │   └── database_test.go        # Database connection & migration tests
    └── trip/
        ├── model.go                # Trip domain model struct
        ├── service.go              # TripService interface and implementation
        └── service_test.go         # Service logic unit tests
```

### Implementation Constraints

- **Pure-Go Compilation:** `CGO_ENABLED=0` must remain fully supported for all builds and tests. Do not import `mattn/go-sqlite3` or `gorm.io/driver/sqlite`.
- **Go Standard Library Routing:** Use standard library Go 1.22+ method-and-path pattern routing (`mux.HandleFunc("GET /{$}", ...)` and `mux.HandleFunc("POST /trips", ...)`).
- **Template Execution Safety:** Templates must always be executed into an intermediate `bytes.Buffer` before writing to `http.ResponseWriter` so template execution errors result in a 500 status rather than a half-written 200 response.
- **Directory Auto-Creation:** Database initialization must verify that the directory containing the SQLite file (e.g., `data/`) exists, creating it with `0755` permissions if absent.

### Sources & Research

- **Alex Edwards HTMX + Go Reference:** `https://www.alexedwards.net/blog/how-i-use-htmx-with-go` — pattern for `htmlRenderer`, `embed.FS`, `isHTMXRequest`, `Vary: HX-Request`, and partial vs full-page rendering.
- **HTMX 4 Documentation:** `https://four.htmx.org/docs` — explicit attribute inheritance, native fetch API, `HX-Request: true` header verification, `meta name="htmx-config"` options.
- **GORM v2 & SQLite:** `https://github.com/glebarez/sqlite` and `https://gorm.io/docs/connecting_to_the_database.html` — pure-Go SQLite driver, PRAGMA settings for WAL mode and busy timeouts, in-memory connection strings for unit testing.

---

## Implementation Units

### U1. Repository Hygiene and Configuration Layer

- **Goal:** Clean up root repository entrypoint confusion, update git ignore rules, define `.env.example`, and implement the environment-based configuration package.
- **Requirements:** R1, R2, R3, KTD1, KTD6.
- **Dependencies:** None.
- **Files:**
  - Delete: `main.go`
  - Modify: `.gitignore`
  - Create: `.env.example`
  - Create: `internal/config/config.go`
  - Create: `internal/config/config_test.go`
- **Approach:**
  1. Delete the root `main.go` placeholder so `cmd/web/main.go` becomes the singular entrypoint.
  2. Update `.gitignore` to explicitly ignore `*.db`, `*.db-journal`, `*.db-wal`, `data/`, and `.env`.
  3. Create `.env.example` documenting `PORT=8080`, `DB_PATH=data/walkthecity.db`, `LOG_LEVEL=info`, and `ENV=development`.
  4. Implement `internal/config/config.go` with `Load()` function that reads from environment variables, falls back to defaults, and safely attempts `godotenv.Load()` if a `.env` file is present.
- **Test Scenarios:**
  - *Happy path defaults:* Calling `config.Load()` with empty environment returns `Config` with default port `8080`, db path `data/walkthecity.db`, log level `info`, and env `development`.
  - *Environment override:* Setting `PORT=9090` and `DB_PATH=/tmp/test.db` overrides defaults in returned `Config`.
  - *Missing .env file:* `config.Load()` succeeds without error when no `.env` file exists on disk.
- **Verification:** `go test ./internal/config/...` passes.

---

### U2. Embedded Asset Filesystem and HTML Renderer

- **Goal:** Wire `assets/efs.go` to embed static files and templates, and build the `htmlRenderer` in `cmd/web/html.go` to support cached template execution and partial renders.
- **Requirements:** R4, R5, R6, R7, KTD3, KTD4.
- **Dependencies:** U1.
- **Files:**
  - Modify: `assets/efs.go`
  - Modify: `assets/html/base.tmpl`
  - Create: `cmd/web/html.go`
- **Approach:**
  1. Implement `assets/efs.go` using `//go:embed "html" "static"` and `fs.Sub()` to expose `assets.HTMLFiles` and `assets.StaticFiles`.
  2. Update `assets/html/base.tmpl` with application branding ("Walk The City"), Pico.css stylesheet link, and HTMX 4 script tag with `defer`.
  3. Implement `newHTMLRenderer(templateFS fs.FS, sharedTemplateFiles ...string)` in `cmd/web/html.go` with template function map (including date/time helpers).
  4. Implement `render(w http.ResponseWriter, status int, data any, templateName string, additionalFiles ...string)` method on `htmlRenderer` that clones the shared template, parses additional page files if provided, renders to a `bytes.Buffer`, sets `w.Header().Set("Vary", "HX-Request")`, and flushes to `w`.
  5. Add `isHTMXRequest(r *http.Request) bool` helper checking `r.Header.Get("HX-Request") == "true"`.
- **Test Scenarios:**
  - *Renderer initialization:* Initializing renderer with `assets.HTMLFiles`, `base.tmpl`, and `partials/*.tmpl` succeeds without error.
  - *Full page render:* Rendering a page template executes the `base` layout with page title and page content blocks filled.
  - *Buffer protection:* Template syntax or execution error in data rendering does not write partial 200 HTTP headers to the response writer.
- **Verification:** `cmd/web/html.go` and `assets/efs.go` compile cleanly with `go build ./cmd/web/...`.

---

### U3. SQLite Database Connection and Trip Domain Model

- **Goal:** Set up GORM with `github.com/glebarez/sqlite` pure-Go driver, define the `Trip` domain model, and handle automatic schema migration on startup.
- **Requirements:** R8, R9, R10, KTD2.
- **Dependencies:** U1.
- **Files:**
  - Modify: `go.mod` (add `gorm.io/gorm`, `github.com/glebarez/sqlite`, `github.com/joho/godotenv`)
  - Create: `internal/trip/model.go`
  - Create: `internal/database/database.go`
  - Create: `internal/database/database_test.go`
- **Approach:**
  1. Run `go get` for `gorm.io/gorm`, `github.com/glebarez/sqlite`, and `github.com/joho/godotenv`.
  2. Define `Trip` struct in `internal/trip/model.go` with fields: `ID` (uint primary key), `Destination` (string, not null), `DurationDays` (int), `Notes` (string), `CreatedAt` (time.Time), `UpdatedAt` (time.Time).
  3. Implement `database.Open(dbPath string) (*gorm.DB, error)` in `internal/database/database.go`:
     - If path is not an in-memory database (`:memory:` or starting with `file::memory:`), extract parent directory via `filepath.Dir(dbPath)` and create via `os.MkdirAll(dir, 0755)`.
     - Construct DSN with WAL mode and busy timeout pragmas (`fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", dbPath)`).
     - Open GORM database connection with `glebarez/sqlite.Open(dsn)`.
     - Execute `db.AutoMigrate(&trip.Trip{})`.
     - Provide `database.OpenDSN(dsn string) (*gorm.DB, error)` helper for testing with custom/in-memory DSNs.
- **Execution note:** Use pure-Go `github.com/glebarez/sqlite` only; verify builds pass with `CGO_ENABLED=0`.
- **Test Scenarios:**
  - *In-memory connection:* `database.Open("file::memory:?cache=shared")` connects and auto-migrates the `trips` table.
  - *Directory auto-creation:* Calling `database.Open("temp_test_dir/test.db")` creates `temp_test_dir/` and initializes the SQLite file.
  - *Model persistence:* Inserting a `Trip` record sets auto-incremented ID and timestamps.
- **Verification:** `CGO_ENABLED=0 go test ./internal/database/...` passes.

---

### U4. Trip Service Layer (Agent Integration Seam)

- **Goal:** Implement the `TripService` business logic layer and repository operations, establishing a clean, decoupled interface that future LLM agent trip generation can call directly.
- **Requirements:** R10, R11, KTD5.
- **Dependencies:** U3.
- **Files:**
  - Create: `internal/trip/service.go`
  - Create: `internal/trip/service_test.go`
- **Approach:**
  1. Define input parameters and validation:
     ```go
     type CreateTripParams struct {
         Destination  string
         DurationDays int
         Notes        string
     }
     ```
  2. Define `TripService` interface:
     - `CreateTrip(ctx context.Context, params CreateTripParams) (*Trip, error)`
     - `ListTrips(ctx context.Context) ([]Trip, error)`
     - `GetTripByID(ctx context.Context, id uint) (*Trip, error)`
  3. Implement `tripService` struct holding `*gorm.DB`.
  4. Implement validation in `CreateTrip`: Destination must not be blank; DurationDays must be between 1 and 30. Return custom validation error type if invalid.
- **Test Scenarios:**
  - *Create trip success:* Valid `CreateTripParams` creates and returns persisted `Trip` with matching fields.
  - *Validation failure:* Blank destination or zero duration returns descriptive validation error without inserting into DB.
  - *List trips ordering:* `ListTrips` returns all created trips ordered by `created_at DESC`.
  - *Get trip by ID:* Returns requested trip when present; returns `ErrNotFound` for non-existent ID.
- **Verification:** `go test ./internal/trip/...` passes against in-memory SQLite.

---

### U5. Web Handlers, Routes, and HTMX Templates (Minimal End-to-End Slice)

- **Goal:** Build the HTTP handlers and HTML templates for the minimal end-to-end trip slice, wiring static asset serving, homepage rendering, HTMX form submissions, and validation error responses.
- **Requirements:** R5, R6, R7, R12, R13, R14, R15, KTD3, KTD4, KTD5.
- **Dependencies:** U2, U4.
- **Files:**
  - Modify: `cmd/web/main.go`
  - Modify: `cmd/web/handlers.go`
  - Create: `cmd/web/handlers_test.go`
  - Modify: `assets/html/pages/home.tmpl`
  - Create: `assets/html/partials/trip_row.tmpl`
  - Create: `assets/html/partials/trip_list.tmpl`
  - Delete: `assets/html/partials/images.tmpl` (remove tutorial leftover)
- **Approach:**
  1. Define `application` struct in `cmd/web/main.go` holding `logger *slog.Logger`, `config *config.Config`, `html *htmlRenderer`, and `tripService trip.TripService`.
  2. Implement `cmd/web/main.go` initialization: load config, initialize logger, connect to database, create trip service, create `htmlRenderer`, setup `http.ServeMux`, and start `http.Server` with graceful shutdown and timeouts.
  3. Register routes:
     - `GET /static/` -> `http.FileServerFS(assets.StaticFiles)` with prefix stripped.
     - `GET /{$}` -> `app.home` (fetches trips list, renders `base` with `pages/home.tmpl`).
     - `POST /trips` -> `app.createTrip` (parses form; if HTMX request: on success renders `partials/trip_row.tmpl` targeting `#trip-list` with `afterbegin`, on validation error sets `HX-Retarget: #form-errors` and `HX-Reswap: innerHTML` with status 422 returning `partials/form_errors.tmpl`; if non-HTMX: redirects to `/` with 303 See Other on success or re-renders full page with status 422 and validation errors).
  4. Write `assets/html/pages/home.tmpl` using Pico.css container, heading, `<div id="form-errors"></div>`, trip creation `<form hx-post="/trips" hx-target="#trip-list" hx-swap="afterbegin">`, and `<div id="trip-list">` embedding `trip_list.tmpl`.
- **Test Scenarios:**
  - *GET / returns 200 OK:* Requesting `/` returns 200 OK and contains the HTML page layout and trip creation form.
  - *POST /trips via HTMX (happy path):* POST with `HX-Request: true` and valid fields creates trip and returns 200 OK with `trip_row` HTML fragment containing the destination name.
  - *POST /trips via HTMX (validation error):* POST with empty destination and `HX-Request: true` returns 422 Unprocessable Entity with error message markup.
  - *POST /trips standard browser (happy path):* POST without `HX-Request` header creates trip and returns 303 See Other redirecting to `/`.
  - *Static assets served:* `GET /static/css/pico.min.css` returns 200 OK with `text/css` MIME type.
- **Verification:** `go test ./cmd/web/...` passes with all handler tests.

---

### U6. Documentation and Smoke Verification Suite

- **Goal:** Provide comprehensive project documentation in `README.md` and verify the entire build, test, and run workflow from scratch.
- **Requirements:** R16, R17.
- **Dependencies:** U1, U2, U3, U4, U5.
- **Files:**
  - Modify: `README.md`
- **Approach:**
  1. Rewrite `README.md` to document:
     - Project overview: Walk The City architecture, tech stack (Go 1.26, GORM, SQLite, HTMX 4, Pico.css).
     - Prerequisites: Go 1.26+ installed.
     - Quick start instructions: `go run ./cmd/web`.
     - Testing instructions: `go test -v ./...` and `CGO_ENABLED=0 go test ./...`.
     - Configuration environment variables table (`PORT`, `DB_PATH`, `LOG_LEVEL`, `ENV`).
     - Project directory structure walkthrough.
     - Future agent integration notes (highlighting `TripService` seam in `internal/trip/service.go`).
  2. Execute full repository verification: run `go mod tidy`, run all tests with `CGO_ENABLED=0`, compile binary to temporary location, and run a smoke test.
- **Test Scenarios:**
  - *Go mod consistency:* `go mod tidy` produces no unexpected changes.
  - *CGO-free build:* `CGO_ENABLED=0 go build -o /tmp/walkthecity ./cmd/web` builds successfully.
  - *Test suite pass:* `CGO_ENABLED=0 go test -v ./...` passes all tests.
- **Verification:** Clean `git status` (except planned untracked/modified changes), all tests pass, and binary runs.

---

## Verification Contract

### Automated Verification Matrix

| Target | Command | Purpose | Quality Gate |
|---|---|---|---|
| Module Hygiene | `go mod tidy && git diff --exit-code go.mod go.sum` | Ensure dependency manifests are clean and locked | Clean exit (0) |
| Pure-Go Compilation | `CGO_ENABLED=0 go build ./...` | Verify zero cgo requirement across all packages | Binary builds with exit 0 |
| Unit & Integration Tests | `CGO_ENABLED=0 go test -v ./...` | Execute all config, database, service, and handler tests | All tests pass |
| Static Asset Integrity | `go test ./cmd/web -run TestStaticAssets` | Verify embedded static files (Pico, HTMX) exist and are servable | 200 OK with valid Content-Type |
| End-to-End Handler Flow | `go test ./cmd/web -run TestTripHandlers` | Verify HTMX and non-HTMX trip creation flows | 200 partial / 303 redirect / 422 error |

---

## Definition of Done

- [ ] All placeholder 0-byte Go files (`cmd/web/*.go`, `assets/efs.go`) are fully implemented and free of EOF syntax errors.
- [ ] Conflicting root `main.go` is deleted.
- [ ] `go.mod` includes `gorm.io/gorm`, `github.com/glebarez/sqlite`, and `github.com/joho/godotenv` with clean `go.sum`.
- [ ] `CGO_ENABLED=0 go build ./cmd/web` produces a working binary without requiring external cgo compilers.
- [ ] Database automatically creates SQLite storage directory (`data/`) and runs schema migrations on startup.
- [ ] `TripService` seam is defined and tested under `internal/trip/service.go`.
- [ ] Visiting `GET /` serves the homepage styled with Pico.css.
- [ ] Submitting a trip via HTMX returns a 200 partial row update without full page reload.
- [ ] Submitting invalid trip data via HTMX returns a 422 status with inline validation error markup.
- [ ] Submitting a trip via standard browser form returns a 303 redirect to `/`.
- [ ] `README.md` provides clear, accurate setup, run, configuration, and testing instructions.
- [ ] All automated tests pass with `CGO_ENABLED=0 go test ./...`.
- [ ] No temporary or dead-end scaffolding files remain.
