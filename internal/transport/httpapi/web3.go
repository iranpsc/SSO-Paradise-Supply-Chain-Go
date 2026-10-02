package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

// Web3 wallet authentication, porting Web3AuthController as JSON-only
// endpoints under /api (browser redirect/session-error variants do not
// exist; failures preserve Laravel's status semantics with Persian messages).

func (s *Server) failWeb3(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrWeb3Signature):
		respond(w, 401, map[string]string{"message": err.Error()})
	case errors.Is(err, application.ErrWeb3Upstream):
		respond(w, 502, map[string]string{"message": err.Error()})
	case errors.Is(err, application.ErrWeb3Nonce),
		errors.Is(err, application.ErrWalletConnected),
		errors.Is(err, application.ErrWalletLinked):
		respond(w, 422, map[string]string{"message": err.Error()})
	default:
		s.fail(w, err)
	}
}

// optionalUser returns the session user when a valid token is presented and
// nil when the caller is a guest (missing or invalid token), mirroring how
// Laravel's session guard leaves $request->user() null.
func (s *Server) optionalUser(r *http.Request) *domain.User {
	u, err := s.auth.Authenticate(r.Context(), token(r))
	if err != nil {
		return nil
	}
	return &u
}

// GET /api/web3/nonce?address=0x... — guest only. Authenticated callers get
// the Laravel guest-middleware redirect to home.
func (s *Server) web3Nonce(w http.ResponseWriter, r *http.Request) {
	if s.optionalUser(r) != nil {
		http.Redirect(w, r, s.origin+"/home", http.StatusFound)
		return
	}
	nonce, err := s.web3.LoginNonce(r.Context(), r.URL.Query().Get("address"))
	if err != nil {
		s.failWeb3(w, err)
		return
	}
	respond(w, 200, map[string]string{"nonce": nonce})
}

// POST /api/web3/verify {address, signature} — guest login or, when the
// caller is already authenticated, wallet link with the login nonce.
func (s *Server) web3Verify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Address   string `json:"address"`
		Signature string `json:"signature"`
	}
	if !decode(w, r, &in) {
		return
	}
	out, err := s.web3.Verify(r.Context(), strings.TrimSpace(in.Address), strings.TrimSpace(in.Signature), s.optionalUser(r))
	if err != nil {
		s.failWeb3(w, err)
		return
	}
	if out.Linked {
		respond(w, 200, map[string]string{"message": "کیف پول با موفقیت متصل شد.", "redirect": out.Redirect})
		return
	}
	s.setSession(w, out.Token)
	respond(w, 200, map[string]string{"message": "ورود با کیف پول با موفقیت انجام شد.", "redirect": out.Redirect})
}

// GET /api/web3/link/nonce?address=0x... — authenticated, verified members.
func (s *Server) web3LinkNonce(w http.ResponseWriter, r *http.Request, u domain.User) {
	nonce, err := s.web3.LinkNonce(r.Context(), u, r.URL.Query().Get("address"))
	if err != nil {
		s.failWeb3(w, err)
		return
	}
	respond(w, 200, map[string]string{"nonce": nonce})
}

// POST /api/web3/link {address, signature} — authenticated, verified members.
func (s *Server) web3Link(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in struct {
		Address   string `json:"address"`
		Signature string `json:"signature"`
	}
	if !decode(w, r, &in) {
		return
	}
	result, err := s.web3.Link(r.Context(), u, strings.TrimSpace(in.Address), strings.TrimSpace(in.Signature))
	if err != nil {
		s.failWeb3(w, err)
		return
	}
	switch result {
	case domain.WalletLinkAlreadyConnected:
		s.failWeb3(w, application.ErrWalletConnected)
	case domain.WalletLinkAlreadyLinked:
		s.failWeb3(w, application.ErrWalletLinked)
	default:
		respond(w, 200, map[string]string{"message": "کیف پول با موفقیت متصل شد.", "redirect": s.origin + "/home"})
	}
}
