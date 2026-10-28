package server

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
)

//go:embed templates
var templateFS embed.FS

// render executes the specified template with the given data and writes the result to the response writer. If an error
// occurs, it logs the error and returns a 500 status code.
func (server *Server) render(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	tmpl, ok := server.templates[name]
	if !ok {
		server.logger.Error("template not found", "name", name)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(&buf, "base.html", data); err != nil {
		server.logger.Error("failed to execute template", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

// parseTemplates parses all page and partial templates and returns a map of template names to parsed templates.
func parseTemplates() (map[string]*template.Template, error) {
	pages, err := fs.Glob(templateFS, "templates/pages/*.html")
	if err != nil {
		return nil, fmt.Errorf("glob page templates: %w", err)
	}

	partials, err := fs.Glob(templateFS, "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("glob partial templates: %w", err)
	}

	templates := make(map[string]*template.Template)
	for _, page := range pages {
		files := []string{"templates/base.html"}
		files = append(files, partials...)
		files = append(files, page)
		tmpl, err := template.New("base.html").Funcs(templateFuncs()).ParseFS(templateFS, files...)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", page, err)
		}
		templates[path.Base(page)] = tmpl
	}

	return templates, nil
}
