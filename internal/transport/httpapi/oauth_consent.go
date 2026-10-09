package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

type pendingAuthorization struct {
	Token   string                    `json:"auth_token"`
	Request application.Authorization `json:"request"`
}

func (s *Server) storeConsent(r *http.Request, in application.Authorization) error {
	attrs, ok := s.auth.Sessions.(application.SessionAttributes)
	if !ok {
		return errors.New("OAuth session attributes unavailable")
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	data, err := json.Marshal(pendingAuthorization{Token: hex.EncodeToString(b), Request: in})
	if err != nil {
		return err
	}
	return attrs.SetSessionAttribute(r.Context(), application.Digest(token(r)), "oauth_authorization", string(data), s.auth.Now().Add(10*time.Minute))
}

func (s *Server) readConsent(r *http.Request) (pendingAuthorization, error) {
	var pending pendingAuthorization
	attrs, ok := s.auth.Sessions.(application.SessionAttributes)
	if !ok {
		return pending, domain.ErrToken
	}
	raw, err := attrs.SessionAttribute(r.Context(), application.Digest(token(r)), "oauth_authorization", s.auth.Now())
	if err != nil {
		return pending, err
	}
	err = json.Unmarshal([]byte(raw), &pending)
	pending.Request.Laravel = true
	return pending, err
}

func (s *Server) pendingConsent(w http.ResponseWriter, r *http.Request, u domain.User) {
	pending, err := s.readConsent(r)
	if err != nil {
		s.consentError(w, err)
		return
	}
	client, scopes, err := s.auth.OAuth.ValidateAuthorization(r.Context(), pending.Request)
	if err != nil {
		s.oauthError(w, err, r)
		return
	}
	respond(w, 200, map[string]any{"client": client, "scopes": scopes, "auth_token": pending.Token})
}

func (s *Server) completeConsent(r *http.Request, u domain.User, supplied string, approve bool) (string, error) {
	pending, err := s.readConsent(r)
	if err != nil || supplied == "" || subtle.ConstantTimeCompare([]byte(pending.Token), []byte(supplied)) != 1 {
		return "", domain.ErrToken
	}
	attrs, ok := s.auth.Sessions.(application.SessionAttributeConsumer)
	if !ok {
		return "", domain.ErrToken
	}
	raw, err := attrs.PullSessionAttribute(r.Context(), application.Digest(token(r)), "oauth_authorization", s.auth.Now())
	if err != nil {
		return "", domain.ErrToken
	}
	var consumed pendingAuthorization
	if json.Unmarshal([]byte(raw), &consumed) != nil || subtle.ConstantTimeCompare([]byte(consumed.Token), []byte(supplied)) != 1 {
		return "", domain.ErrToken
	}
	consumed.Request.Laravel = true
	consumed.Request.Approve = approve
	target, err := s.auth.OAuth.Authorize(r.Context(), u, consumed.Request, false)
	if err != nil {
		return "", err
	}
	return s.walletCallback(r, target)
}

func (s *Server) consentError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrToken) || errors.Is(err, domain.ErrNotFound) {
		respond(w, 403, map[string]string{"message": "The provided auth token for the request is different from the session auth token."})
		return
	}
	s.oauthError(w, err)
}

func (s *Server) decideConsent(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in struct {
		Token   string `json:"auth_token"`
		Approve bool   `json:"approve"`
	}
	if !decode(w, r, &in) {
		return
	}
	target, err := s.completeConsent(r, u, in.Token, in.Approve)
	if err != nil {
		s.consentError(w, err)
		return
	}
	respond(w, 200, map[string]string{"redirect": target})
}
