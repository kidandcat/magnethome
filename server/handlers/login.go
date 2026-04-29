package handlers

import "net/http"

func (s *Server) LoginPage(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Auth.IsAuthed(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	s.renderer.render(w, "login", map[string]any{"Error": ""})
}

func (s *Server) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		badRequest(w, "bad form")
		return
	}
	user := r.PostFormValue("username")
	pass := r.PostFormValue("password")
	if !s.cfg.Auth.Verify(user, pass) {
		w.WriteHeader(http.StatusUnauthorized)
		s.renderer.render(w, "login", map[string]any{"Error": "Credenciales inválidas"})
		return
	}
	s.cfg.Auth.Login(w)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	s.cfg.Auth.Logout(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}
