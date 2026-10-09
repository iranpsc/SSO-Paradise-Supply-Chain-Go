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

func (s *Server) oauthError(w http.ResponseWriter, err error, requests ...*http.Request) {
	var request *http.Request
	if len(requests) > 0 {
		request = requests[0]
	}
	var detailed passportError
	if errors.As(err, &detailed) {
		passportRespond(w, request, detailed.kind, detailed.parameter)
		return
	}
	var oauthErr application.OAuthError
	if errors.As(err, &oauthErr) {
		parameter := ""
		if request != nil && oauthErr == application.OAuthInvalidScope {
			parameter = request.FormValue("scope")
		}
		passportRespond(w, request, oauthErr, parameter)
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
	in.Laravel = r.URL.Path == "/oauth/authorize"
	if in.Laravel {
		if in.ResponseType == "" {
			s.oauthError(w, passportError{application.OAuthInvalidRequest, "response_type"}, r)
			return
		}
		if in.ResponseType != "code" {
			s.oauthError(w, application.OAuthError("unsupported_grant_type"), r)
			return
		}
		if in.ClientID == 0 && r.URL.Query().Get("client_id") == "" {
			s.oauthError(w, passportError{application.OAuthInvalidRequest, "client_id"}, r)
			return
		}
	}
	client, scopes, err := s.auth.OAuth.ValidateAuthorization(r.Context(), in)
	if err != nil {
		if in.Laravel && errors.Is(err, application.OAuthInvalidRequest) {
			parameter := "redirect_uri"
			if in.Challenge != "" {
				parameter = "code_challenge_method"
			} else if in.Method != "" || client.SecretHash == "" {
				parameter = "code_challenge"
			}
			s.oauthError(w, passportError{application.OAuthInvalidRequest, parameter}, r)
			return
		}
		s.oauthError(w, err, r)
		return
	}
	// OAuth authorization accepts first-party sessions only, never an access token.
	u, err := s.authenticate(r, w)
	if err != nil {
		if r.URL.Path == "/oauth/authorize" {
			s.flashErrors(w, mustJSON(map[string]string{"intended": r.URL.RequestURI()}))
			http.Redirect(w, r, s.origin+"/login", http.StatusFound)
		} else {
			s.fail(w, err)
		}
		return
	}
	if r.URL.Path == "/oauth/authorize" {
		if !client.SkipsAuthorization() || r.URL.Query().Get("prompt") == "consent" {
			if err := s.storeConsent(r, in); err != nil {
				s.fail(w, err)
				return
			}
			http.Redirect(w, r, s.origin+"/authorize", http.StatusFound)
			return
		}
		in.Approve = true
		target, err := s.auth.OAuth.Authorize(r.Context(), u, in, false)
		if err != nil {
			s.oauthError(w, err, r)
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
	if r.URL.Path == "/oauth/authorize" {
		in, ok := laravelInput(w, r)
		if !ok {
			return
		}
		supplied := inputString(in, "auth_token")
		target, err := s.completeConsent(r, u, supplied, r.Method != http.MethodDelete)
		if err != nil {
			s.consentError(w, err)
			return
		}
		http.Redirect(w, r, target, http.StatusFound)
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
			s.oauthError(w, application.OAuthInvalidRequest, r)
			return
		}
		in = parseAuthorization(r.PostForm)
	}
	if r.Method == http.MethodDelete {
		in.Approve = false
	}
	target, err := s.auth.OAuth.Authorize(r.Context(), u, in, false)
	if err != nil {
		s.oauthError(w, err, r)
		return
	}
	target, err = s.walletCallback(r, target)
	if err != nil {
		s.fail(w, err)
		return
	}
	if r.URL.Path == "/oauth/authorize" {
		http.Redirect(w, r, target, http.StatusFound)
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
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Refreshed."))
}
func oauthForm(w http.ResponseWriter, r *http.Request) bool {
	in, ok := laravelInput(w, r)
	if !ok {
		return false
	}
	r.PostForm = make(url.Values)
	for key, value := range in {
		if text, ok := value.(string); ok {
			r.PostForm.Set(key, text)
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
	grant := r.PostForm.Get("grant_type")
	if grant == "" {
		s.oauthError(w, passportError{application.OAuthInvalidRequest, "grant_type"}, r)
		return
	}
	if grant != "authorization_code" && grant != "refresh_token" {
		s.oauthError(w, application.OAuthError("unsupported_grant_type"), r)
		return
	}
	if r.PostForm.Get("client_id") == "" && r.Header.Get("Authorization") == "" {
		s.oauthError(w, passportError{application.OAuthInvalidRequest, "client_id"}, r)
		return
	}
	id, secret, err := oauthCredentials(r)
	if err != nil {
		s.oauthError(w, err, r)
		return
	}
	client, err := s.auth.OAuth.AuthenticateClient(r.Context(), id, secret)
	if err != nil {
		s.oauthError(w, err, r)
		return
	}
	if !client.SupportsGrant(grant) {
		s.oauthError(w, application.OAuthError("unauthorized_client"), r)
		return
	}
	parameter := "code"
	if grant == "refresh_token" {
		parameter = "refresh_token"
	}
	if r.PostForm.Get(parameter) == "" {
		s.oauthError(w, passportError{application.OAuthInvalidRequest, parameter}, r)
		return
	}
	result, err := s.auth.OAuth.Token(r.Context(), id, secret, r.PostForm.Get("grant_type"), r.PostForm.Get("code"), r.PostForm.Get("redirect_uri"), r.PostForm.Get("code_verifier"), r.PostForm.Get("refresh_token"))
	if err != nil {
		s.oauthError(w, err, r)
		return
	}
	passportTokens(w, result)
}

func passportTokens(w http.ResponseWriter, result application.OAuthTokens) {
	w.Header().Set("Pragma", "no-cache")
	respond(w, 200, map[string]any{"token_type": result.TokenType, "expires_in": result.ExpiresIn, "access_token": result.AccessToken, "refresh_token": result.RefreshToken})
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
		s.oauthError(w, err, r)
		return
	}
	if _, err = s.auth.OAuth.AuthenticateClient(r.Context(), id, secret); err != nil {
		s.oauthError(w, err, r)
		return
	}
	if err = s.auth.OAuth.Revoke(r.Context(), id, r.PostForm.Get("token")); err != nil {
		s.oauthError(w, err, r)
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
