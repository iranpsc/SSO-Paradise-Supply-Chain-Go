package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
)

func TestOAuthLoginReturnAcrossSeparateHostsPreservesPKCE(t *testing.T) {
	db := mysqltest.Open(t)
	db.ConfigureRateLimit(1000)
	ctx := context.Background()
	passwords := security.Bcrypt{Cost: 4}
	hash, err := passwords.Hash("SyntheticPass!2026")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	_, err = db.Create(ctx, domain.User{Name: "OAuth member", Email: "return@example.test", PasswordHash: hash, CreatedAt: now, EmailVerifiedAt: &now})
	if err != nil {
		t.Fatal(err)
	}
	callback := "https://origin.example/callback?existing=1"
	clientID, err := db.CreateOAuthClient(ctx, "Public PKCE client", "", []string{callback})
	if err != nil {
		t.Fatal(err)
	}
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, Passwords: passwords, Now: time.Now, PublicURL: "http://accounts.example.test", SessionTTL: time.Hour, SigningKey: []byte(strings.Repeat("k", 32))}
	a.OAuth = &application.OAuth{Store: db, Passwords: passwords, Now: time.Now}
	h := httpapi.New(a, nil, nil, a.PublicURL, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	jar, _ := cookiejar.New(nil)
	apiHost, _ := url.Parse("http://api.example.test")
	frontHost, _ := url.Parse(a.PublicURL)
	do := func(host *url.URL, method, path, body, contentType string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, host.String()+path, strings.NewReader(body))
		for _, c := range jar.Cookies(host) {
			r.AddCookie(c)
		}
		if contentType != "" {
			r.Header.Set("Content-Type", contentType)
		}
		if method != "GET" && host.Host == frontHost.Host {
			r.Header.Set("Origin", a.PublicURL)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		jar.SetCookies(host, w.Result().Cookies())
		return w
	}
	verifier := strings.Repeat("v", 43)
	state := "origin-state&encoded=+value"
	query := url.Values{"client_id": {fmt.Sprint(clientID)}, "redirect_uri": {callback}, "response_type": {"code"}, "scope": {""}, "state": {state}, "code_challenge": {application.PKCE(verifier)}, "code_challenge_method": {"S256"}}
	intended := "/oauth/authorize?" + query.Encode()
	guest := do(apiHost, "GET", intended, "", "")
	loginURL, err := url.Parse(guest.Header().Get("Location"))
	if err != nil || guest.Code != 302 || loginURL.Scheme+"://"+loginURL.Host != a.PublicURL || loginURL.Path != "/login" || loginURL.Query().Get("return_to") != intended {
		t.Fatalf("authorization request lost across login redirect: status %d", guest.Code)
	}
	if len(jar.Cookies(frontHost)) != 0 || len(jar.Cookies(apiHost)) == 0 {
		t.Fatal("fixture must reproduce host-only API cookie isolation")
	}
	login := do(frontHost, "POST", "/api/login", `{"login":"return@example.test","password":"SyntheticPass!2026"}`, "application/json")
	if login.Code != 200 {
		t.Fatalf("login failed: %d %s", login.Code, login.Body.String())
	}
	resume := do(frontHost, "GET", loginURL.Query().Get("return_to"), "", "")
	if resume.Code != 302 || resume.Header().Get("Location") != a.PublicURL+"/authorize" {
		t.Fatalf("public client must require consent: %d", resume.Code)
	}
	pending := do(frontHost, "GET", "/api/oauth/consent", "", "")
	var consent struct {
		Token string `json:"auth_token"`
	}
	if json.Unmarshal(pending.Body.Bytes(), &consent) != nil || consent.Token == "" {
		t.Fatal("pending consent missing")
	}
	decision := do(frontHost, "POST", "/api/oauth/consent", fmt.Sprintf(`{"auth_token":%q,"approve":true}`, consent.Token), "application/json")
	var result struct {
		Redirect string `json:"redirect"`
	}
	if decision.Code != 200 || json.Unmarshal(decision.Body.Bytes(), &result) != nil {
		t.Fatalf("consent failed: %d", decision.Code)
	}
	target, err := url.Parse(result.Redirect)
	if err != nil || target.Host != "origin.example" || target.Path != "/callback" || target.Query().Get("state") != state || target.Query().Get("existing") != "1" || target.Query().Get("code") == "" {
		t.Fatal("registered callback, state or code lost")
	}
	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {fmt.Sprint(clientID)}, "code": {target.Query().Get("code")}, "redirect_uri": {callback}, "code_verifier": {verifier}}
	tokens := do(apiHost, "POST", "/oauth/token", form.Encode(), "application/x-www-form-urlencoded")
	var tokenBody struct {
		Access string `json:"access_token"`
	}
	if tokens.Code != 200 || json.Unmarshal(tokens.Body.Bytes(), &tokenBody) != nil || tokenBody.Access == "" {
		t.Fatalf("PKCE exchange failed: %d", tokens.Code)
	}
	// Like Passport, a valid previously issued access token covers a repeated
	// request, but prompt=consent still forces a new explicit decision.
	for _, prompt := range []string{"", "consent", "select_account consent"} {
		repeated, _ := url.ParseQuery(query.Encode())
		if prompt != "" {
			repeated.Set("prompt", prompt)
		}
		next := do(frontHost, "GET", "/oauth/authorize?"+repeated.Encode(), "", "")
		if next.Code != 302 {
			t.Fatalf("repeat authorization failed: %d", next.Code)
		}
		if prompt == "" {
			callbackURL, _ := url.Parse(next.Header().Get("Location"))
			if callbackURL.Host != "origin.example" || callbackURL.Query().Get("code") == "" || callbackURL.Query().Get("state") != state {
				t.Fatal("active previous authorization did not skip consent")
			}
		} else if next.Header().Get("Location") != a.PublicURL+"/authorize" {
			t.Fatal("prompt=consent did not force consent")
		}
	}
	replay := do(apiHost, "POST", "/oauth/token", form.Encode(), "application/x-www-form-urlencoded")
	if replay.Code == 200 {
		t.Fatal("authorization code replay accepted")
	}
	if next := do(frontHost, "GET", intended, "", ""); next.Header().Get("Location") != a.PublicURL+"/authorize" {
		t.Fatal("revoked grant must not bypass consent")
	}
}
