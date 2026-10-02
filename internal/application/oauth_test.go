package application_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
)

func TestOAuthPKCERefreshReplayAndJWT(t *testing.T) {
	a, db, m := fixture(t)
	ctx := context.Background()
	u := verify(t, a, m, register(t, a, "oauth@example.com"))
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	o := &application.OAuth{Store: db, Passwords: a.Passwords, Now: time.Now, Signer: security.JWT{Key: key}}
	client, err := db.CreateOAuthClient(ctx, "Browser", "", []string{"https://client.example/callback?existing=1"})
	if err != nil {
		t.Fatal(err)
	}
	verifier := strings.Repeat("a", 43)
	in := application.Authorization{ClientID: client, RedirectURI: "https://client.example/callback?existing=1", ResponseType: "code", State: "bound-state", Challenge: application.PKCE(verifier), Method: "S256", Approve: true}
	evil := in
	evil.RedirectURI = "https://client.example/callback/evil"
	if _, _, err = o.ValidateAuthorization(ctx, evil); err == nil {
		t.Fatal("redirect prefix accepted")
	}
	noPKCE := in
	noPKCE.Challenge = ""
	noPKCE.Method = ""
	if _, _, err = o.ValidateAuthorization(ctx, noPKCE); err == nil {
		t.Fatal("public client accepted without PKCE")
	}
	target, err := o.Authorize(ctx, u, in, true)
	if err != nil {
		t.Fatal(err)
	}
	redirect, _ := url.Parse(target)
	code := redirect.Query().Get("code")
	if redirect.Query().Get("state") != "bound-state" || redirect.Query().Get("wallet_login") != "true" || redirect.Query().Get("existing") != "1" {
		t.Fatalf("callback flags missing: %s", target)
	}
	if _, err = o.Token(ctx, client, "", "authorization_code", code, in.RedirectURI, strings.Repeat("b", 43), ""); !errors.Is(err, application.OAuthInvalidGrant) {
		t.Fatalf("wrong verifier accepted: %v", err)
	}
	tokens, err := o.Token(ctx, client, "", "authorization_code", code, in.RedirectURI, verifier, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Split(tokens.AccessToken, ".")) != 3 {
		t.Fatal("access token is not JWT")
	}
	if got, err := o.AccessUser(ctx, tokens.AccessToken, "profile"); err != nil || got.ID != u.ID {
		t.Fatalf("JWT authentication: %v", err)
	}
	refreshed, err := o.Token(ctx, client, "", "refresh_token", "", "", "", tokens.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.AccessUser(ctx, tokens.AccessToken, "profile"); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("old generation survived refresh")
	}
	if _, err = o.AccessUser(ctx, refreshed.AccessToken, "profile"); err != nil {
		t.Fatal(err)
	}
	if _, err = o.Token(ctx, client, "", "refresh_token", "", "", "", tokens.RefreshToken); !errors.Is(err, application.OAuthInvalidGrant) {
		t.Fatal("refresh replay accepted")
	}
	if _, err = o.AccessUser(ctx, refreshed.AccessToken, "profile"); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("replay did not revoke token family")
	}
	// A new grant is independent; reusing its code revokes only that grant.
	target, err = o.Authorize(ctx, u, in, false)
	if err != nil {
		t.Fatal(err)
	}
	redirect, _ = url.Parse(target)
	code = redirect.Query().Get("code")
	next, err := o.Token(ctx, client, "", "authorization_code", code, in.RedirectURI, verifier, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.Token(ctx, client, "", "authorization_code", code, in.RedirectURI, verifier, ""); !errors.Is(err, application.OAuthInvalidGrant) {
		t.Fatal("code replay accepted")
	}
	if _, err = o.AccessUser(ctx, next.AccessToken, "profile"); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("code replay did not revoke access")
	}
}
func TestOAuthConfidentialClientLegacyFlowAndRevocation(t *testing.T) {
	a, db, m := fixture(t)
	ctx := context.Background()
	u := verify(t, a, m, register(t, a, "confidential@example.com"))
	hash, err := a.Passwords.Hash("client-secret")
	if err != nil {
		t.Fatal(err)
	}
	id, err := db.CreateOAuthClient(ctx, "Laravel client", hash, []string{"https://client.example/callback"})
	if err != nil {
		t.Fatal(err)
	}
	o := &application.OAuth{Store: db, Passwords: a.Passwords, Now: time.Now}
	in := application.Authorization{ClientID: id, RedirectURI: "https://client.example/callback", ResponseType: "code", Approve: true}
	target, err := o.Authorize(ctx, u, in, false)
	if err != nil {
		t.Fatal(err)
	}
	redirect, _ := url.Parse(target)
	code := redirect.Query().Get("code")
	if _, err = o.Token(ctx, id, "wrong", "authorization_code", code, in.RedirectURI, "", ""); !errors.Is(err, application.OAuthInvalidClient) {
		t.Fatal("wrong secret accepted")
	}
	tokens, err := o.Token(ctx, id, "client-secret", "authorization_code", code, in.RedirectURI, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err = o.Revoke(ctx, id, tokens.AccessToken); err != nil {
		t.Fatal(err)
	}
	if _, err = o.AccessUser(ctx, tokens.AccessToken, "profile"); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("revoked token accepted")
	}
}
func TestRememberMeAndRegistrationCallback(t *testing.T) {
	a, db, m := fixture(t)
	ctx := context.Background()
	a.OAuth = &application.OAuth{Store: db, Passwords: a.Passwords, Now: time.Now}
	hash, _ := a.Passwords.Hash("client-secret")
	id, err := db.CreateOAuthClient(ctx, "client", hash, []string{"https://metarang.com/callback"})
	if err != nil {
		t.Fatal(err)
	}
	in := application.Registration{Username: "callback_user", Name: "Name", Email: "callback@example.com", Password: password, Confirmation: password, ClientID: &id, RedirectURI: "https://wrong.example/callback", BackURL: "https://metarang.com/app?existing=1"}
	if _, err = a.Register(ctx, in); err == nil {
		t.Fatal("invalid registration callback accepted")
	}
	in.RedirectURI = "https://metarang.com/callback"
	u, err := a.Register(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	u = verify(t, a, m, u)
	target, err := a.VerificationRedirect(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if target != "https://metarang.com/app?existing=1&verified=1" {
		t.Fatalf("bad redirect: %s", target)
	}
	if target, err = a.VerificationRedirect(ctx, u.ID); err != nil || target != a.PublicURL+"/home" {
		t.Fatal("callback replayed")
	}
	_, token, err := a.LoginRemember(ctx, u.Email, password, true)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	a.Now = func() time.Time { return now.Add(29 * 24 * time.Hour) }
	if _, err = a.Authenticate(ctx, token); err != nil {
		t.Fatal("remember-me expired early")
	}
	a.Now = func() time.Time { return now.Add(31 * 24 * time.Hour) }
	if _, err = a.Authenticate(ctx, token); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("remember-me never expires")
	}
}
