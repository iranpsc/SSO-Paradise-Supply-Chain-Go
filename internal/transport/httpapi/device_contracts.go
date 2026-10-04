package httpapi

import (
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"net/http"
	"net/url"
	"strconv"
)

// The inspected Laravel app registers these headless Passport routes, but
// its legacy client model advertises no device_code grant and registers no
// device views. Preserve that configuration rather than inventing clients.
func (s *Server) registerDeviceRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /oauth/device", func(w http.ResponseWriter, r *http.Request) {
		if code := r.URL.Query().Get("user_code"); code != "" {
			http.Redirect(w, r, s.origin+"/oauth/device/authorize?"+url.Values{"user_code": {code}}.Encode(), 302)
			return
		}
		respond(w, 500, map[string]string{"message": "Server Error"})
	})
	m.HandleFunc("POST /oauth/device/code", func(w http.ResponseWriter, r *http.Request) {
		if !oauthForm(w, r) {
			return
		}
		raw := r.PostForm.Get("client_id")
		if raw == "" {
			if name, _, ok := r.BasicAuth(); ok {
				raw = name
			}
		}
		if raw == "" {
			s.oauthError(w, passportError{application.OAuthInvalidRequest, "client_id"}, r)
			return
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || s.auth.OAuth == nil {
			s.oauthError(w, application.OAuthInvalidClient, r)
			return
		}
		client, err := s.auth.OAuth.Store.OAuthClient(r.Context(), id)
		if err != nil || client.Revoked {
			s.oauthError(w, application.OAuthInvalidClient, r)
			return
		}
		s.oauthError(w, application.OAuthError("unauthorized_client"), r)
	})
	m.HandleFunc("GET /oauth/device/authorize", func(w http.ResponseWriter, r *http.Request) {
		if !s.legacyGuard(w, r, false) {
			return
		}
		if code := r.URL.Query().Get("user_code"); code != "" {
			s.flashErrors(w, mustJSON(map[string]any{"errors": map[string][]string{"user_code": {"Incorrect code."}}}))
		}
		http.Redirect(w, r, s.origin+"/oauth/device", 302)
	})
	for _, method := range []string{"POST", "DELETE"} {
		m.HandleFunc(method+" /oauth/device/authorize", func(w http.ResponseWriter, r *http.Request) {
			in, ok := laravelInput(w, r)
			if !ok || !s.legacyCSRF(w, r, in) || !s.legacyGuard(w, r, false) {
				return
			}
			respond(w, 403, map[string]string{"message": "The provided auth token for the request is different from the session auth token."})
		})
	}
}
