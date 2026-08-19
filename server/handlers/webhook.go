package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/kidandcat/magnethome/server/email"
	"github.com/kidandcat/magnethome/server/models"
)

// anyAddressMatchesDomain reports whether any recipient address has the given domain suffix.
func anyAddressMatchesDomain(addrs []string, domain string) bool {
	if domain == "" {
		return true
	}
	suffix := "@" + strings.ToLower(domain)
	for _, a := range addrs {
		// Resend may pass "Name <user@domain>" or just "user@domain".
		a = strings.ToLower(a)
		if i := strings.LastIndex(a, "<"); i >= 0 {
			if j := strings.Index(a[i:], ">"); j > 0 {
				a = a[i+1 : i+j]
			}
		}
		if strings.HasSuffix(strings.TrimSpace(a), suffix) {
			return true
		}
	}
	return false
}

// Resend forwards inbound emails through a webhook (Svix-style) with type "email.received".
type resendWebhook struct {
	Type string `json:"type"`
	Data struct {
		EmailID     string   `json:"email_id"`
		From        string   `json:"from"`
		To          []string `json:"to"`
		ReceivedFor []string `json:"received_for"`
		Subject     string   `json:"subject"`
		HTML        string   `json:"html"`
		Text        string   `json:"text"`
		// Resend email.received is metadata-only (no html/text). We still
		// accept those fields if a future payload includes them.
	} `json:"data"`
}

func (s *Server) ResendWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 5<<20)) // 5 MiB
	if err != nil {
		badRequest(w, "read body")
		return
	}
	defer r.Body.Close()

	if s.cfg.WebhookSecret != "" {
		msgID := r.Header.Get("svix-id")
		ts := r.Header.Get("svix-timestamp")
		sig := r.Header.Get("svix-signature")
		if err := email.VerifySvixSignature(s.cfg.WebhookSecret, msgID, ts, sig, body); err != nil {
			log.Printf("resend webhook: invalid signature: %v", err)
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	var p resendWebhook
	if err := json.Unmarshal(body, &p); err != nil {
		badRequest(w, "bad json")
		return
	}

	switch p.Type {
	case "email.received", "inbound.email":
		// Resend webhooks are account-wide — ignore events for other domains in the account.
		recipients := append([]string{}, p.Data.To...)
		recipients = append(recipients, p.Data.ReceivedFor...)
		if !anyAddressMatchesDomain(recipients, s.cfg.AcceptedDomain) {
			log.Printf("resend webhook: ignoring %s for non-magnethome recipients %v", p.Type, recipients)
			break
		}
		rec := &models.Email{
			Direction: "incoming",
			From:      p.Data.From,
			To:        strings.Join(p.Data.To, ", "),
			Subject:   p.Data.Subject,
			BodyHTML:  p.Data.HTML,
			BodyText:  p.Data.Text,
			ResendID:  p.Data.EmailID,
		}
		// Webhook payload has no body; fetch it from the Receiving API.
		if rec.BodyHTML == "" && rec.BodyText == "" && rec.ResendID != "" && s.cfg.Resend != nil {
			got, err := s.cfg.Resend.GetReceiving(rec.ResendID)
			if err != nil {
				log.Printf("resend webhook: fetch receiving %s: %v", rec.ResendID, err)
			} else if got != nil {
				rec.BodyHTML = got.HTML
				rec.BodyText = got.Text
				if rec.Subject == "" {
					rec.Subject = got.Subject
				}
			}
		}
		if _, err := s.cfg.Repo.Insert(rec); err != nil {
			serverError(w, err)
			return
		}
	default:
		// Other event types (sent, bounced, delivered) are acknowledged but ignored in v1.
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
