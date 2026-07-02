package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mentasystems/colmena"
	"github.com/mentasystems/colmena/backup/s3"
)

// Backup enables continuous backup when Bucket and the credentials are set.
type Backup struct {
	Endpoint   string
	Region     string
	Bucket     string
	AccessKey  string
	SecretKey  string
	AlertEmail string
	ResendKey  string
}

func (b Backup) enabled() bool {
	return b.Bucket != "" && b.AccessKey != "" && b.SecretKey != ""
}

func (b Backup) backend(db string) (colmena.BackupBackend, error) {
	return s3.NewBackend(s3.Config{
		Endpoint:  b.Endpoint,
		Region:    b.Region,
		Bucket:    b.Bucket,
		Prefix:    "magnethome/" + db,
		AccessKey: b.AccessKey,
		SecretKey: b.SecretKey,
	})
}

type Store struct {
	Node *colmena.Node
	DB   *sql.DB
}

// Open boots the colmena store (single node) and applies the schema. With
// backups enabled it restores from the newest backup when the database does
// not exist yet, and alerts by email when the backup engine fails.
func Open(dataDir string, backup Backup) (*Store, error) {
	cfg := colmena.Config{DataDir: dataDir}
	if backup.enabled() {
		if err := restoreIfMissing(dataDir, backup); err != nil {
			return nil, err
		}
		cfg.Backup = &colmena.BackupConfig{
			NewBackend: backup.backend,
			OnError: colmena.NewResendAlerter(backup.ResendKey,
				"Magnethome <alerts@mentasystems.com>", backup.AlertEmail, "magnethome"),
		}
	}
	node, err := colmena.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("colmena.New: %w", err)
	}
	s := &Store{Node: node, DB: node.DB()}
	if err := s.migrate(); err != nil {
		node.Close()
		return nil, err
	}
	return s, nil
}

// restoreIfMissing rebuilds default.db from the newest backup when the data
// dir has no database yet — the disaster-recovery boot path.
func restoreIfMissing(dataDir string, backup Backup) error {
	if _, statErr := os.Stat(filepath.Join(dataDir, "default.db")); statErr == nil {
		return nil
	}
	backend, err := backup.backend("default")
	if err != nil {
		return fmt.Errorf("backup backend: %w", err)
	}
	defer backend.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	gens, err := backend.Generations(ctx)
	if err != nil || len(gens) == 0 {
		log.Printf("db: no backups to restore (fresh install): %v", err)
		return nil
	}
	log.Printf("db: default.db missing — restoring from backup generation %s", gens[0].ID)
	if err := colmena.Restore(ctx, backend, dataDir); err != nil {
		return fmt.Errorf("restore from backup: %w", err)
	}
	log.Printf("db: restore complete")
	return nil
}

// BackupStatus exposes the backup engine state (health endpoint).
func (s *Store) BackupStatus() map[string]colmena.BackupStatus { return s.Node.BackupStatus() }

func (s *Store) Close() error {
	return s.Node.Close()
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			direction TEXT NOT NULL,
			"from" TEXT NOT NULL,
			"to" TEXT NOT NULL,
			subject TEXT NOT NULL DEFAULT '',
			body_html TEXT NOT NULL DEFAULT '',
			body_text TEXT NOT NULL DEFAULT '',
			resend_id TEXT NOT NULL DEFAULT '',
			in_reply_to TEXT NOT NULL DEFAULT '',
			is_read INTEGER NOT NULL DEFAULT 0,
			is_archived INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_direction_created ON emails(direction, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_resend_id ON emails(resend_id)`,
	}
	for _, q := range stmts {
		if _, err := s.DB.Exec(q); err != nil {
			return fmt.Errorf("migrate: %w\n%s", err, q)
		}
	}
	return nil
}
