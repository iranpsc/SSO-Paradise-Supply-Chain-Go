package httpapi

import (
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) confirmPassword(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in struct {
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if u.PasswordHash == "" || !s.auth.Passwords.Matches(u.PasswordHash, in.Password) {
		s.fail(w, domain.Validation{"password": "رمز عبور صحیح نیست."})
		return
	}
	attributes, ok := s.auth.Sessions.(application.SessionAttributes)
	if !ok {
		s.fail(w, domain.ErrNotFound)
		return
	}
	now := s.auth.Now()
	if err := attributes.SetSessionAttribute(r.Context(), application.Digest(token(r)), "password_confirmed_at", strconv.FormatInt(now.Unix(), 10), now.Add(3*time.Hour)); err != nil {
		s.fail(w, err)
		return
	}
	respond(w, 200, map[string]any{"confirmed": true, "expires_in": 10800})
}
