package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	cookieName = "mh_admin"
	maxAge     = 7 * 24 * time.Hour
)

type Auth struct {
	username string
	password string
	secret   []byte
}

func New(username, password, secret string) *Auth {
	return &Auth{username: username, password: password, secret: []byte(secret)}
}

func (a *Auth) Verify(user, pass string) bool {
	uok := subtle.ConstantTimeCompare([]byte(user), []byte(a.username)) == 1
	pok := subtle.ConstantTimeCompare([]byte(pass), []byte(a.password)) == 1
	return uok && pok
}

func (a *Auth) Login(w http.ResponseWriter) {
	exp := time.Now().Add(maxAge).Unix()
	payload := strconv.FormatInt(exp, 10)
	mac := a.sign(payload)
	value := payload + "." + mac
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(maxAge),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *Auth) Logout(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})
}

func (a *Auth) IsAuthed(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return false
	}
	parts := strings.SplitN(c.Value, ".", 2)
	if len(parts) != 2 {
		return false
	}
	expected := a.sign(parts[0])
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return false
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return true
}

func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.IsAuthed(r) {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Auth) sign(payload string) string {
	h := hmac.New(sha256.New, a.secret)
	h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

var ErrUnauthorized = errors.New("unauthorized")
