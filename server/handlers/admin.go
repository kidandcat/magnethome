package handlers

import (
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
		HTML:    plainToHTML(body),
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

func plainToHTML(s string) string {
	// Minimal: escape via the template package would be cleaner, but we keep <br>-based formatting.
	var b strings.Builder
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			b.WriteString("<br>")
		}
		b.WriteString(htmlEscape(line))
	}
	return b.String()
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;")
	return r.Replace(s)
}
