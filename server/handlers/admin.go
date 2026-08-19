package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/kidandcat/magnethome/server/email"
	"github.com/kidandcat/magnethome/server/models"
)

func (s *Server) Inbox(w http.ResponseWriter, r *http.Request) {
	showArchived := r.URL.Query().Get("archived") == "1"
	emails, err := s.cfg.Repo.ListIncoming(showArchived, 200)
	if err != nil {
		serverError(w, err)
		return
	}
	// In archived view we only want archived ones; ListIncoming with includeArchived=true returns both.
	if showArchived {
		filtered := emails[:0]
		for _, e := range emails {
			if e.IsArchived {
				filtered = append(filtered, e)
			}
		}
		emails = filtered
	}
	s.renderer.render(w, "inbox", s.baseData("inbox", map[string]any{
		"Emails":       emails,
		"ShowArchived": showArchived,
	}))
}

func (s *Server) Sent(w http.ResponseWriter, r *http.Request) {
	emails, err := s.cfg.Repo.ListOutgoing(200)
	if err != nil {
		serverError(w, err)
		return
	}
	s.renderer.render(w, "sent", s.baseData("sent", map[string]any{"Emails": emails}))
}

func (s *Server) EmailDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		badRequest(w, "bad id")
		return
	}
	e, err := s.cfg.Repo.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if e.Direction == "incoming" && e.ResendID != "" && e.BodyHTML == "" && e.BodyText == "" && s.cfg.Resend != nil {
		if got, ferr := s.cfg.Resend.GetReceiving(e.ResendID); ferr != nil {
			log.Printf("email detail: fetch receiving %s: %v", e.ResendID, ferr)
		} else if got != nil && (got.HTML != "" || got.Text != "") {
			if uerr := s.cfg.Repo.UpdateBody(e.ID, got.HTML, got.Text); uerr != nil {
				log.Printf("email detail: update body %d: %v", e.ID, uerr)
			} else {
				e.BodyHTML = got.HTML
				e.BodyText = got.Text
			}
		}
	}
	if e.Direction == "incoming" && !e.IsRead {
		_ = s.cfg.Repo.MarkRead(id)
		e.IsRead = true
	}
	subject := e.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}
	s.renderer.render(w, "detail", s.baseData("inbox", map[string]any{
		"Email":        e,
		"ReplySubject": subject,
	}))
}

func (s *Server) ToggleArchive(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		badRequest(w, "bad id")
		return
	}
	e, err := s.cfg.Repo.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.cfg.Repo.SetArchived(id, !e.IsArchived); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) Preview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		badRequest(w, "bad form")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(email.RenderHTML(r.PostFormValue("body"))))
}

func (s *Server) ComposeForm(w http.ResponseWriter, r *http.Request) {
	s.renderer.render(w, "compose", s.baseData("compose", map[string]any{
		"To": "", "Subject": "", "Body": "", "From": s.cfg.DefaultFrom, "Error": "",
	}))
}

func (s *Server) ComposeSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		badRequest(w, "bad form")
		return
	}
	from := r.PostFormValue("from")
	to := strings.TrimSpace(r.PostFormValue("to"))
	subject := r.PostFormValue("subject")
	body := r.PostFormValue("body")
	if from == "" || to == "" || body == "" {
		s.renderer.render(w, "compose", s.baseData("compose", map[string]any{
			"To": to, "Subject": subject, "Body": body, "From": from,
			"Error": "Faltan campos obligatorios",
		}))
		return
	}
	if err := s.send(from, splitAddrs(to), subject, body, ""); err != nil {
		s.renderer.render(w, "compose", s.baseData("compose", map[string]any{
			"To": to, "Subject": subject, "Body": body, "From": from,
			"Error": "Error al enviar: " + err.Error(),
		}))
		return
	}
	http.Redirect(w, r, "/admin/sent", http.StatusSeeOther)
}

func (s *Server) Reply(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		badRequest(w, "bad id")
		return
	}
	orig, err := s.cfg.Repo.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		badRequest(w, "bad form")
		return
	}
	from := r.PostFormValue("from")
	to := strings.TrimSpace(r.PostFormValue("to"))
	subject := r.PostFormValue("subject")
	body := r.PostFormValue("body")
	if from == "" || to == "" || body == "" {
		s.renderer.render(w, "detail", s.baseData("inbox", map[string]any{
			"Email": orig, "ReplySubject": subject,
			"ReplyError": "Faltan campos obligatorios",
		}))
		return
	}
	if err := s.send(from, splitAddrs(to), subject, body, orig.ResendID); err != nil {
		s.renderer.render(w, "detail", s.baseData("inbox", map[string]any{
			"Email": orig, "ReplySubject": subject,
			"ReplyError": "Error al enviar: " + err.Error(),
		}))
		return
	}
	http.Redirect(w, r, "/admin/email/"+r.PathValue("id"), http.StatusSeeOther)
}

func (s *Server) send(from string, to []string, subject, body, inReplyTo string) error {
	req := email.SendRequest{
		From:    from,
		To:      to,
		Subject: subject,
		Text:    body,
		HTML:    email.RenderHTML(body),
	}
	resp, err := s.cfg.Resend.Send(req)
	if err != nil {
		return err
	}
	rec := &models.Email{
		Direction: "outgoing",
		From:      from,
		To:        strings.Join(to, ", "),
		Subject:   subject,
		BodyHTML:  req.HTML,
		BodyText:  body,
		ResendID:  resp.ID,
		InReplyTo: inReplyTo,
		IsRead:    true,
	}
	_, err = s.cfg.Repo.Insert(rec)
	return err
}

func (s *Server) EmailHTML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		badRequest(w, "bad id")
		return
	}
	e, err := s.cfg.Repo.Get(id)
	if err != nil || e.BodyHTML == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src data: https: http:; style-src 'unsafe-inline'; font-src data: https:")
	_, _ = w.Write([]byte(e.BodyHTML))
}
