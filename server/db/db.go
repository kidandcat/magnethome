package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/kidandcat/colmena"
)

type Store struct {
	Node *colmena.Node
	DB   *sql.DB
}

func Open(dataDir, bind string) (*Store, error) {
	cfg := colmena.Config{
		NodeID:    "magnethome-1",
		DataDir:   dataDir,
		Bind:      bind,
		Bootstrap: true,
	}
	node, err := colmena.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("colmena.New: %w", err)
	}
	if err := node.WaitForLeader(15 * time.Second); err != nil {
		node.Close()
		return nil, fmt.Errorf("waiting for leader: %w", err)
	}
	s := &Store{Node: node, DB: node.DB()}
	if err := s.migrate(); err != nil {
		node.Close()
		return nil, err
	}
	return s, nil
}

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
