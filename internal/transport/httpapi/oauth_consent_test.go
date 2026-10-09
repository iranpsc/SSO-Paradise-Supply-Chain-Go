package httpapi_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

func TestOAuthConsentOwnershipApprovalDenialAndReplay(t *testing.T) {
	db := mysqltest.Open(t)
	ctx := context.Background()
	db.ConfigureRateLimit(1000)
	passwords := security.Bcrypt{Cost: 4}
	u, err := db.Create(ctx, domain.User{Name: "Member", Email: "consent@example.com", CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, Passwords: passwords, Now: time.Now, PublicURL: "http://localhost:3000", SessionTTL: time.Hour}
	a.OAuth = &application.OAuth{Store: db, Passwords: passwords, Now: time.Now}
	session, err := a.StartSession(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: "paradise_session", Value: session}
	h := httpapi.New(a, nil, nil, a.PublicURL, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	raw, err := sql.Open("mysql", db.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	for i, tc := range []struct{ firstParty, confidential, approve bool }{
		{true, true, true}, {false, true, true}, {false, true, false}, {true, false, true},
	} {
		hash := ""
		if tc.confidential {
			hash, _ = passwords.Hash("client-secret")
		}
		id, err := db.CreateOAuthClient(ctx, "Client", hash, []string{"https://client.example/callback?existing=1"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := raw.Exec(`UPDATE oauth_clients SET first_party=? WHERE id=?`, tc.firstParty, id); err != nil {
			t.Fatal(err)
		}
		query := url.Values{"client_id": {fmt.Sprint(id)}, "response_type": {"code"}, "state": {"bound-state"}}
		if !tc.confidential {
			query.Set("code_challenge", application.PKCE(strings.Repeat("a", 43)))
			query.Set("code_challenge_method", "S256")
		}
		w := request(h, "GET", "/oauth/authorize?"+query.Encode(), "", "", cookie)
		if w.Code != 302 {
			t.Fatal(i, w.Code, w.Body.String())
		}
		if tc.firstParty && tc.confidential {
			if !strings.Contains(w.Header().Get("Location"), "code=") {
				t.Fatal("internal client did not skip consent")
			}
			continue
		}
		if w.Header().Get("Location") != a.PublicURL+"/authorize" {
			t.Fatal("consent bypassed", w.Header())
		}
		w = request(h, "GET", "/api/oauth/consent", "", "", cookie)
		var pending struct {
			Token string `json:"auth_token"`
		}
		if json.Unmarshal(w.Body.Bytes(), &pending) != nil || pending.Token == "" {
			t.Fatal("missing pending request", w.Body.String())
		}
		// A forged decision must not consume the valid pending request.
		w = request(h, "POST", "/api/oauth/consent", `{"auth_token":"wrong","approve":true}`, a.PublicURL, cookie)
		if w.Code != 403 {
			t.Fatal("forged token accepted", w.Code)
		}
		body := fmt.Sprintf(`{"auth_token":%q,"approve":%t}`, pending.Token, tc.approve)
		w = request(h, "POST", "/api/oauth/consent", body, a.PublicURL, cookie)
		var result struct {
			Redirect string `json:"redirect"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil {
			t.Fatal(w.Code, w.Body.String())
		}
		target, _ := url.Parse(result.Redirect)
		if target.Query().Get("state") != "bound-state" || target.Query().Get("existing") != "1" {
			t.Fatal("callback state lost")
		}
		if tc.approve && target.Query().Get("code") == "" {
			t.Fatal("no authorization code")
		}
		if !tc.approve && (target.Query().Get("error") != "access_denied" || target.Query().Get("code") != "") {
			t.Fatal("denial issued a code")
		}
		if w = request(h, "POST", "/api/oauth/consent", body, a.PublicURL, cookie); w.Code != 403 {
			t.Fatal("decision replayed", w.Code)
		}
	}
}
