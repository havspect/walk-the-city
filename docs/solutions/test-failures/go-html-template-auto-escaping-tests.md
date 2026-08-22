---
title: "Go html/template Contextual Auto-Escaping in HTML Bodies and Test Assertions"
date: 2026-08-22
category: test-failures
module: cmd/web
problem_type: test_failure
component: frontend
severity: medium
symptoms:
  - "TestHandlers_GenerateTrip_HTMX_Success failed expecting '2,500+ Nasone Drinking Fountains' because html/template rendered '2,500&#43;'"
  - "TestWizardEndToEndJourney failed expecting 'Walking + Public Transit' because '+' was escaped to '&#43;'"
  - "TestTemplateHelpers failed when testing URL query string helpers due to '&' escaping to '&amp;'"
root_cause: wrong_api
resolution_type: code_fix
tags:
  - go
  - html-template
  - testing
  - htmx
  - escaping
---

# Go html/template Contextual Auto-Escaping in HTML Bodies and Test Assertions

## Problem
In Go's `html/template` package, context-aware auto-escaping converts characters like `+` into HTML numeric entities (`&#43;`) inside text nodes and `&` into `&amp;` in HTML attribute values. When writing template helpers that construct external URLs or writing Go HTTP handler/integration tests that assert on raw HTML string content, naive string assertions and helper return types cause subtle test failures and malformed links.

## Symptoms
- HTTP handler test `TestHandlers_GenerateTrip_HTMX_Success` in `cmd/web/handlers_test.go` failed with:
  ```
  expected detail page to contain Nasone cool fact
  ```
  because the title `"2,500+ Nasone Drinking Fountains"` rendered into HTML as `"2,500&#43; Nasone Drinking Fountains"`.
- End-to-end integration test `TestWizardEndToEndJourney` in `cmd/web/wizard_integration_test.go` failed with:
  ```
  expected trip page to contain "Walking + Public Transit", but was missing
  ```
  because the mobility tag rendered as `Walking &#43; Public Transit`.
- Custom template helpers (`osmLink`, `googleMapsLink`, `wikiLink`) returning `string` caused `html/template` to apply default contextual string escaping on URL attributes or query parameters.

## What Didn't Work
- **Raw string matching with special characters in tests:** Asserting `strings.Contains(detailStr, "2,500+ Nasone Drinking Fountains")` or `strings.Contains(tripStr, "Walking + Public Transit")` failed because Go's `html/template` auto-escapes `+` to `&#43;` in text elements.
- **Returning plain `string` from URL template helper functions:** Returning `string` rather than `template.URL` in `cmd/web/html.go` triggered context-aware sanitization and URL escaping on `href` attributes, interfering with formatted query strings.

## Solution

### 1. Return `template.URL` from URL Helper Functions
In `cmd/web/html.go:30-58`, return `template.URL` instead of `string` for custom URL generator functions so Go's template engine recognizes them as pre-sanitized safe URLs:

```go
// cmd/web/html.go
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
```

### 2. Update Test Assertions to Match Escaped HTML Content
In `cmd/web/handlers_test.go:250-265` and `cmd/web/wizard_integration_test.go:200-220`, assert on unique text tokens that exclude punctuation auto-escaped by `html/template`, or explicitly expect the escaped HTML entity (e.g. `Colosseum &amp; Ludus Magnus` and `&#43;`):

```go
// cmd/web/wizard_integration_test.go
expectedElements := []string{
    "Rome",
    "September",
    "3 Days",
    "Moderate",
    "Public Transit",
    "Colosseum &amp; Ludus Magnus",
    "Pantheon Concrete Dome",
    "Trastevere Artisan Bakeries",
    "Nasone Drinking Fountains",
    "OpenStreetMap",
    "Google Maps",
    "Wikipedia",
}

for _, elem := range expectedElements {
    if !strings.Contains(tripStr, elem) {
        t.Errorf("expected trip page to contain %q, but was missing", elem)
    }
}
```

## Why This Works
Go's `html/template` package parses HTML contextually (text content, attribute values, URIs, CSS, and JS) to guarantee security against cross-site scripting (XSS).
1. Inside regular HTML text nodes, characters such as `+`, `&`, `<`, `>`, `"`, and `'` are automatically encoded (e.g., `+` becomes `&#43;` and `&` becomes `&amp;`). Direct substring checks on raw HTTP response bodies must account for these entity transformations.
2. In URL attribute contexts (like `href="..."`), any untrusted `string` is subjected to contextual filtering. Explicitly returning `template.URL` indicates to the template engine that the helper has already properly structured and escaped the URL query parameters using `url.QueryEscape`.

## Prevention
- **Use `template.URL` and `template.HTML` intentionally:** Return typed values from `template.FuncMap` helpers whenever generating URLs or raw HTML fragments.
- **Isolate assertions to stable keywords in HTML tests:** In HTTP integration tests, assert on specific alphanumeric identifiers or account for standard HTML escaping (`&amp;`, `&#43;`) rather than expecting verbatim raw input strings in rendered HTML.
- **Run pure-Go test suites across all packages:** Keep unit and integration tests executing with `CGO_ENABLED=0 go test ./...` in CI to catch template escaping regressions across packages immediately.

## Related Issues
- Initial scaffold plan: `docs/plans/2026-08-22-1746-feat-initial-scaffold-and-trip-persistence-plan.md`
- Wizard implementation plan: `docs/plans/2026-08-22-1838-feat-city-trip-wizard-generator-plan.md`
