package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

func (s *Server) oauthError(w http.ResponseWriter, err error) {
	var oauthErr application.OAuthError
	if errors.As(err, &oauthErr) {
		status := 400
		if oauthErr == application.OAuthInvalidClient {
			status = 401
			w.Header().Set("WWW-Authenticate", `Basic realm="oauth"`)
		}
		respond(w, status, map[string]string{"error": string(oauthErr), "error_description": oauthErr.PersianMessage(), "message": oauthErr.PersianMessage()})
		return
	}
	s.fail(w, err)
}
func parseAuthorization(values url.Values) application.Authorization {
	id, _ := strconv.ParseInt(values.Get("client_id"), 10, 64)
	return application.Authorization{ClientID: id, RedirectURI: values.Get("redirect_uri"), ResponseType: values.Get("response_type"), Scope: values.Get("scope"), State: values.Get("state"), Challenge: values.Get("code_challenge"), Method: values.Get("code_challenge_method"), Approve: values.Get("approve") == "true"}
}

func (s *Server) oauthConsent(w http.ResponseWriter, r *http.Request) {
	if s.auth.OAuth == nil {
		s.fail(w, domain.ErrNotFound)
		return
	}
	in := parseAuthorization(r.URL.Query())
	client, scopes, err := s.auth.OAuth.ValidateAuthorization(r.Context(), in)
	if err != nil {
		s.oauthError(w, err)
		return
	}
	// OAuth authorization accepts first-party sessions only, never an access token.
	u, err := s.authenticate(r, w)
	if err != nil {
		if r.URL.Path == "/oauth/authorize" {
			http.Redirect(w, r, s.origin+"/login?"+url.Values{"return_to": {r.URL.RequestURI()}}.Encode(), http.StatusFound)
		} else {
			s.fail(w, err)
		}
		return
	}
	if r.URL.Path == "/oauth/authorize" {
		in.Approve = true
		target, err := s.auth.OAuth.Authorize(r.Context(), u, in, false)
		if err != nil {
			s.oauthError(w, err)
			return
		}
		target, err = s.walletCallback(r, target)
		if err != nil {
			s.fail(w, err)
			return
		}
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	respond(w, 200, map[string]any{"client": client, "scopes": scopes})
}
func (s *Server) oauthAuthorize(w http.ResponseWriter, r *http.Request, u domain.User) {
	if s.auth.OAuth == nil {
		s.fail(w, domain.ErrNotFound)
		return
	}
	var in application.Authorization
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		if !decode(w, r, &in) {
			return
		}
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
		if err := r.ParseForm(); err != nil {
			s.oauthError(w, application.OAuthInvalidRequest)
			return
		}
		in = parseAuthorization(r.PostForm)
	}
	if r.Method == http.MethodDelete {
		in.Approve = false
	}
	target, err := s.auth.OAuth.Authorize(r.Context(), u, in, false)
	if err != nil {
		s.oauthError(w, err)
		return
	}
	target, err = s.walletCallback(r, target)
	if err != nil {
		s.fail(w, err)
		return
	}
	if r.URL.Path == "/oauth/authorize" {
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	respond(w, 200, map[string]string{"redirect": target})
}

// Go's transient browser authentication is its revocable HttpOnly session.
// Renew only after authenticating the existing session and checking Origin.
func (s *Server) refreshBrowserToken(w http.ResponseWriter, r *http.Request, u domain.User) {
	session, err := s.auth.StartSession(r.Context(), u.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.setSession(w, session)
	respond(w, http.StatusOK, map[string]string{"message": "نشست کاربری با موفقیت تمدید شد."})
}
func oauthForm(w http.ResponseWriter, r *http.Request) bool {
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/x-www-form-urlencoded" {
		respond(w, 415, map[string]string{"error": "invalid_request", "message": application.OAuthInvalidRequest.PersianMessage(), "error_description": application.OAuthInvalidRequest.PersianMessage()})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if err := r.ParseForm(); err != nil {
		respond(w, 400, map[string]string{"error": "invalid_request", "message": application.OAuthInvalidRequest.PersianMessage(), "error_description": application.OAuthInvalidRequest.PersianMessage()})
		return false
	}
	for _, values := range r.PostForm {
		if len(values) != 1 {
			respond(w, 400, map[string]string{"error": "invalid_request", "message": application.OAuthInvalidRequest.PersianMessage(), "error_description": application.OAuthInvalidRequest.PersianMessage()})
			return false
		}
	}
	return true
}
func oauthCredentials(r *http.Request) (int64, string, error) {
	raw, secret := r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
	if name, password, ok := r.BasicAuth(); ok {
		if raw != "" || secret != "" {
			return 0, "", application.OAuthInvalidRequest
		}
		var err error
		raw, err = url.QueryUnescape(name)
		if err != nil {
			return 0, "", application.OAuthInvalidClient
		}
		secret, err = url.QueryUnescape(password)
		if err != nil {
			return 0, "", application.OAuthInvalidClient
		}
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, "", application.OAuthInvalidClient
	}
	return id, secret, nil
}
func (s *Server) oauthToken(w http.ResponseWriter, r *http.Request) {
	if s.auth.OAuth == nil {
		s.fail(w, domain.ErrNotFound)
		return
	}
	if !oauthForm(w, r) {
		return
	}
	id, secret, err := oauthCredentials(r)
	if err != nil {
		s.oauthError(w, err)
		return
	}
	result, err := s.auth.OAuth.Token(r.Context(), id, secret, r.PostForm.Get("grant_type"), r.PostForm.Get("code"), r.PostForm.Get("redirect_uri"), r.PostForm.Get("code_verifier"), r.PostForm.Get("refresh_token"))
	if err != nil {
		s.oauthError(w, err)
		return
	}
	w.Header().Set("Pragma", "no-cache")
	respond(w, 200, result)
}
func (s *Server) oauthRevoke(w http.ResponseWriter, r *http.Request) {
	if s.auth.OAuth == nil {
		s.fail(w, domain.ErrNotFound)
		return
	}
	if !oauthForm(w, r) {
		return
	}
	id, secret, err := oauthCredentials(r)
	if err != nil {
		s.oauthError(w, err)
		return
	}
	if _, err = s.auth.OAuth.AuthenticateClient(r.Context(), id, secret); err != nil {
		s.oauthError(w, err)
		return
	}
	if err = s.auth.OAuth.Revoke(r.Context(), id, r.PostForm.Get("token")); err != nil {
		s.oauthError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) walletCallback(r *http.Request, target string) (string, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	if parsed.Query().Get("code") == "" || s.web3 == nil {
		return target, nil
	}
	consumer, ok := s.web3.Attributes.(application.SessionAttributeConsumer)
	if !ok {
		return target, nil
	}
	value, err := consumer.PullSessionAttribute(r.Context(), application.Digest(token(r)), "wallet_login", s.auth.Now())
	if errors.Is(err, domain.ErrNotFound) {
		return target, nil
	}
	if err != nil {
		return "", err
	}
	if value == "true" {
		query := parsed.Query()
		query.Set("wallet_login", "true")
		parsed.RawQuery = query.Encode()
		return parsed.String(), nil
	}
	return target, nil
}
