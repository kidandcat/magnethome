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

// Pages render layout.html + their own file as a separate tree, so each can
// define its own {{define "content"}} block without colliding with the others.
var pageFiles = map[string]string{
	"inbox":   "inbox.html",
	"sent":    "sent.html",
	"detail":  "detail.html",
	"compose": "compose.html",
}

type tmplRenderer struct {
	pages map[string]*template.Template
	login *template.Template // standalone, no shared layout
}

func newRenderer() (*tmplRenderer, error) {
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, err
	}
	r := &tmplRenderer{pages: map[string]*template.Template{}}
	for name, file := range pageFiles {
		t, err := template.New("layout.html").Funcs(funcs).ParseFS(sub, "layout.html", file)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", file, err)
		}
		r.pages[name] = t
	}
	login, err := template.New("login.html").Funcs(funcs).ParseFS(sub, "login.html")
	if err != nil {
		return nil, err
	}
	r.login = login
	return r, nil
}

func (r *tmplRenderer) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var t *template.Template
	var entry string
	if name == "login" {
		t, entry = r.login, "login"
	} else {
		t = r.pages[name]
		entry = "layout"
	}
	if t == nil {
		http.Error(w, "unknown template "+name, http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, entry, data); err != nil {
		http.Error(w, fmt.Sprintf("template %s: %v", name, err), http.StatusInternalServerError)
	}
}
