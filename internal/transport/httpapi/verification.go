package httpapi

import (
	"crypto/subtle"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"net/http"
	"strconv"
)

func (s *Server) verifySignedEmail(w http.ResponseWriter, r *http.Request, u domain.User) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	expected := security.EmailFingerprint(u.Email)
	if !security.ValidLaravelSignature(s.origin, r.URL.EscapedPath(), r.URL.RawQuery, s.auth.Now(), s.auth.SigningKey) {
		respond(w, 403, map[string]string{"message": "Invalid signature."})
		return
	}
	if err != nil || id != u.ID || subtle.ConstantTimeCompare([]byte(expected), []byte(r.PathValue("hash"))) != 1 {
		respond(w, 403, map[string]string{"message": "This action is unauthorized."})
		return
	}
	if u.EmailVerifiedAt != nil {
		if expectsJSON(r) {
			w.WriteHeader(204)
		} else {
			http.Redirect(w, r, s.origin+"/home", 302)
		}
		return
	}
	actions, ok := s.auth.Actions.(application.SignedEmailActions)
	if !ok {
		s.fail(w, domain.ErrNotFound)
		return
	}
	if err = actions.VerifySignedEmail(r.Context(), u.ID, u.Email, s.auth.Now()); err != nil {
		s.fail(w, err)
		return
	}
	target, err := s.auth.VerificationRedirect(r.Context(), u.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}
