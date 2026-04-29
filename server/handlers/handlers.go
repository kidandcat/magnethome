package handlers

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/kidandcat/magnethome/server/auth"
	"github.com/kidandcat/magnethome/server/email"
	"github.com/kidandcat/magnethome/server/models"
)

//go:embed all:static
var staticFS embed.FS

type Config struct {
	Auth          *auth.Auth
	Repo          *models.EmailRepo
	Resend        *email.Client
	WebhookSecret string
	FromOptions   []string // e.g. ["hola@magnethome.es", "info@magnethome.es"]
	DefaultFrom   string
}

type Server struct {
	cfg      Config
	renderer *tmplRenderer
}

func NewServer(cfg Config) (*Server, error) {
	r, err := newRenderer()
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, renderer: r}, nil
}

func (s *Server) Routes(mux *http.ServeMux, landingDir string) {
	// Admin static assets
	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /admin/static/", http.StripPrefix("/admin/static/", http.FileServerFS(staticSub)))

	// Auth
	mux.HandleFunc("GET /admin/login", s.LoginPage)
	mux.HandleFunc("POST /admin/login", s.LoginSubmit)
	mux.HandleFunc("POST /admin/logout", s.Logout)

	// Protected admin pages — wrap each with auth middleware
	protect := func(h http.HandlerFunc) http.Handler { return s.cfg.Auth.Require(h) }
	mux.Handle("GET /admin/{$}", protect(s.Inbox))
	mux.Handle("GET /admin/sent", protect(s.Sent))
	mux.Handle("GET /admin/compose", protect(s.ComposeForm))
	mux.Handle("POST /admin/compose", protect(s.ComposeSubmit))
	mux.Handle("GET /admin/email/{id}", protect(s.EmailDetail))
	mux.Handle("POST /admin/email/{id}/reply", protect(s.Reply))
	mux.Handle("POST /admin/email/{id}/archive", protect(s.ToggleArchive))

	// Webhook (no auth, signature-verified)
	mux.HandleFunc("POST /webhooks/resend", s.ResendWebhook)

	// Static landing site (catch-all)
	mux.Handle("/", http.FileServer(http.Dir(landingDir)))
}

func (s *Server) baseData(section string, extra map[string]any) map[string]any {
	unread, _ := s.cfg.Repo.UnreadCount()
	d := map[string]any{
		"Section":     section,
		"UnreadCount": unread,
		"FromOptions": s.cfg.FromOptions,
		"Title":       sectionTitle(section),
	}
	for k, v := range extra {
		d[k] = v
	}
	return d
}

func sectionTitle(s string) string {
	switch s {
	case "inbox":
		return "Bandeja"
	case "sent":
		return "Enviados"
	case "compose":
		return "Nuevo"
	default:
		return "Email"
	}
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func badRequest(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusBadRequest)
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func splitAddrs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
