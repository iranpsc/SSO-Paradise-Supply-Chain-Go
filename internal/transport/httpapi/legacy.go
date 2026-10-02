package httpapi

import (
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"net/http"
)

func (s *Server) authenticate(r *http.Request, w http.ResponseWriter) (domain.User, error) {
	if token(r) != "" {
		return s.auth.Authenticate(r.Context(), token(r))
	}
	for _, name := range []string{s.auth.LegacySessionCookie, s.auth.LegacyRememberCookie} {
		if name == "" {
			continue
		}
		cookie, err := r.Cookie(name)
		if err != nil {
			continue
		}
		u, session, err := s.auth.RestoreLegacyCookie(r.Context(), name, cookie.Value)
		if errors.Is(err, domain.ErrCredentials) {
			continue
		}
		if err != nil {
			return u, err
		}
		ttl := s.auth.SessionTTL
		if name == s.auth.LegacyRememberCookie {
			ttl = s.auth.RememberSessionTTL()
		}
		restored := &http.Cookie{Name: cookieName, Value: session, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: int(ttl.Seconds())}
		http.SetCookie(w, restored)
		r.AddCookie(restored)
		return u, nil
	}
	return domain.User{}, domain.ErrCredentials
}
func (s *Server) hasBrowserCookie(r *http.Request) bool {
	for _, name := range []string{cookieName, s.auth.LegacySessionCookie, s.auth.LegacyRememberCookie} {
		if name != "" {
			if _, err := r.Cookie(name); err == nil {
				return true
			}
		}
	}
	return false
}
