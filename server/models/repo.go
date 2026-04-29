package models

import (
	"database/sql"
	"errors"
	"time"
)

type EmailRepo struct{ DB *sql.DB }

func NewEmailRepo(db *sql.DB) *EmailRepo { return &EmailRepo{DB: db} }

func (r *EmailRepo) Insert(e *Email) (int64, error) {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	res, err := r.DB.Exec(
		`INSERT INTO emails (direction, "from", "to", subject, body_html, body_text, resend_id, in_reply_to, is_read, is_archived, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Direction, e.From, e.To, e.Subject, e.BodyHTML, e.BodyText,
		e.ResendID, e.InReplyTo, boolToInt(e.IsRead), boolToInt(e.IsArchived), e.CreatedAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *EmailRepo) Get(id int64) (*Email, error) {
	row := r.DB.QueryRow(
		`SELECT id, direction, "from", "to", subject, body_html, body_text, resend_id, in_reply_to, is_read, is_archived, created_at
		 FROM emails WHERE id = ?`, id)
	return scanEmail(row)
}

func (r *EmailRepo) ListIncoming(includeArchived bool, limit int) ([]Email, error) {
	q := `SELECT id, direction, "from", "to", subject, body_html, body_text, resend_id, in_reply_to, is_read, is_archived, created_at
	      FROM emails WHERE direction = 'incoming'`
	if !includeArchived {
		q += ` AND is_archived = 0`
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	return r.queryList(q, limit)
}

func (r *EmailRepo) ListOutgoing(limit int) ([]Email, error) {
	return r.queryList(
		`SELECT id, direction, "from", "to", subject, body_html, body_text, resend_id, in_reply_to, is_read, is_archived, created_at
		 FROM emails WHERE direction = 'outgoing' ORDER BY created_at DESC LIMIT ?`, limit)
}

func (r *EmailRepo) MarkRead(id int64) error {
	_, err := r.DB.Exec(`UPDATE emails SET is_read = 1 WHERE id = ?`, id)
	return err
}

func (r *EmailRepo) SetArchived(id int64, archived bool) error {
	_, err := r.DB.Exec(`UPDATE emails SET is_archived = ? WHERE id = ?`, boolToInt(archived), id)
	return err
}

func (r *EmailRepo) UnreadCount() (int, error) {
	var n int
	err := r.DB.QueryRow(`SELECT COUNT(*) FROM emails WHERE direction = 'incoming' AND is_read = 0 AND is_archived = 0`).Scan(&n)
	return n, err
}

func (r *EmailRepo) queryList(q string, args ...any) ([]Email, error) {
	rows, err := r.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Email
	for rows.Next() {
		e, err := scanEmail(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEmail(r rowScanner) (*Email, error) {
	var e Email
	var isRead, isArchived int
	err := r.Scan(&e.ID, &e.Direction, &e.From, &e.To, &e.Subject, &e.BodyHTML, &e.BodyText,
		&e.ResendID, &e.InReplyTo, &isRead, &isArchived, &e.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	e.IsRead = isRead != 0
	e.IsArchived = isArchived != 0
	return &e, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
