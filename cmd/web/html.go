package main

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"
	"time"
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
