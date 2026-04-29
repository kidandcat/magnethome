package models

import "time"

type Email struct {
	ID        int64
	Direction string // "incoming" | "outgoing"
	From      string
	To        string
	Subject   string
	BodyHTML  string
	BodyText  string
	ResendID  string
	InReplyTo string
	IsRead    bool
	IsArchived bool
	CreatedAt time.Time
}
