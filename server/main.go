package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kidandcat/magnethome/server/auth"
	"github.com/kidandcat/magnethome/server/db"
	"github.com/kidandcat/magnethome/server/email"
	"github.com/kidandcat/magnethome/server/handlers"
	"github.com/kidandcat/magnethome/server/models"
)

func main() {
	cfg := loadConfig()

	store, err := db.Open(cfg.DataDir, cfg.RaftBind)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer store.Close()

	repo := models.NewEmailRepo(store.DB)
	a := auth.New(cfg.AdminUser, cfg.AdminPass, cfg.SessionSecret)
	resend := email.New(cfg.ResendKey)

	srv, err := handlers.NewServer(handlers.Config{
		Auth:          a,
		Repo:          repo,
		Resend:        resend,
		WebhookSecret: cfg.ResendWebhookSecret,
		FromOptions:   cfg.FromOptions,
		DefaultFrom:   cfg.DefaultFrom,
	})
	if err != nil {
		log.Fatalf("handlers init: %v", err)
	}

	mux := http.NewServeMux()
	srv.Routes(mux, cfg.LandingDir)

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           logMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s (landing=%s)", cfg.HTTPAddr, cfg.LandingDir)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}

type config struct {
	HTTPAddr            string
	RaftBind            string
	DataDir             string
	LandingDir          string
	AdminUser           string
	AdminPass           string
	SessionSecret       string
	ResendKey           string
	ResendWebhookSecret string
	FromOptions         []string
	DefaultFrom         string
}

func loadConfig() config {
	cfg := config{
		HTTPAddr:            envOr("HTTP_ADDR", ":8080"),
		RaftBind:            envOr("RAFT_BIND", "127.0.0.1:9000"),
		DataDir:             envOr("DATA_DIR", "./data"),
		LandingDir:          envOr("LANDING_DIR", "."),
		AdminUser:           os.Getenv("ADMIN_USER"),
		AdminPass:           os.Getenv("ADMIN_PASS"),
		SessionSecret:       os.Getenv("SESSION_SECRET"),
		ResendKey:           os.Getenv("RESEND_API_KEY"),
		ResendWebhookSecret: os.Getenv("RESEND_WEBHOOK_SECRET"),
		DefaultFrom:         envOr("DEFAULT_FROM", "hola@magnethome.es"),
	}
	froms := envOr("FROM_OPTIONS", "hola@magnethome.es,info@magnethome.es")
	for _, f := range strings.Split(froms, ",") {
		if f = strings.TrimSpace(f); f != "" {
			cfg.FromOptions = append(cfg.FromOptions, f)
		}
	}
	if cfg.AdminUser == "" || cfg.AdminPass == "" {
		log.Fatal("ADMIN_USER and ADMIN_PASS env vars are required")
	}
	if cfg.SessionSecret == "" {
		log.Print("SESSION_SECRET not set — generating ephemeral secret (sessions will be invalidated on restart)")
		cfg.SessionSecret = randomSecret()
	}
	if cfg.ResendKey == "" {
		log.Print("warning: RESEND_API_KEY not set — sending will fail")
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func randomSecret() string {
	var b [32]byte
	_, _ = rand.Read(b[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}
