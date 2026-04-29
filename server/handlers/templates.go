package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"
)

//go:embed all:templates
var templatesFS embed.FS

var funcs = template.FuncMap{
	"fmtTime": func(t time.Time) string { return t.Local().Format("2006-01-02 15:04") },
	"truncate": func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		return s[:n] + "…"
	},
}

func loadTemplates() (*template.Template, error) {
	root := template.New("").Funcs(funcs)
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, err
	}
	return root.ParseFS(sub, "*.html")
}

type tmplRenderer struct{ t *template.Template }

func newRenderer() (*tmplRenderer, error) {
	t, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	return &tmplRenderer{t: t}, nil
}

func (r *tmplRenderer) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := r.t.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("template %s: %v", name, err), http.StatusInternalServerError)
	}
}
