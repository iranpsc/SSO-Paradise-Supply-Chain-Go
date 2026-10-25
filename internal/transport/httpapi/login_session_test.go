package httpapi_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
)

func TestAuthenticatedAPILoginReturnsJSONWhileLegacyFormRedirects(t *testing.T) {
	db := mysqltest.Open(t)
	ctx := context.Background()
	user, err := db.Create(ctx, domain.User{Name: "Existing session", Email: "existing-session@example.test", CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, Now: time.Now, PublicURL: "http://localhost:3000", SessionTTL: time.Hour}
	session, err := a.StartSession(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	h := httpapi.New(a, nil, nil, a.PublicURL, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	cookie := &http.Cookie{Name: "paradise_session", Value: session}
	// The guest guard ignores submitted credentials when a session exists;
	// it must not replace the account or return an HTML redirect to API callers.
	response := request(h, "POST", "/api/login", `{"login":"another-account","password":"not-used"}`, a.PublicURL, cookie)
	var body struct {
		Message string `json:"message"`
	}
	if response.Code != http.StatusOK || response.Header().Get("Location") != "" || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.Message != "Login successful" {
		t.Fatalf("authenticated API login did not return JSON: %d", response.Code)
	}
	account := request(h, "GET", "/api/account", "", "", cookie)
	var result struct {
		Data domain.User `json:"data"`
	}
	if account.Code != 200 || json.Unmarshal(account.Body.Bytes(), &result) != nil || result.Data.ID != user.ID {
		t.Fatal("existing session was lost or replaced")
	}
	csrf := request(h, "GET", "/sanctum/csrf-cookie", "", "", cookie)
	legacyRequest := httptest.NewRequest("POST", "/login", strings.NewReader(`{"email":"another-account","password":"not-used"}`))
	legacyRequest.Header.Set("Content-Type", "application/json")
	legacyRequest.Header.Set("Origin", a.PublicURL)
	legacyRequest.AddCookie(cookie)
	for _, c := range csrf.Result().Cookies() {
		legacyRequest.AddCookie(c)
		if c.Name == "XSRF-TOKEN" {
			legacyRequest.Header.Set("X-XSRF-TOKEN", c.Value)
		}
	}
	legacy := httptest.NewRecorder()
	h.ServeHTTP(legacy, legacyRequest)
	if legacy.Code != http.StatusFound || legacy.Header().Get("Location") != a.PublicURL+"/home" {
		t.Fatalf("legacy form redirect changed: %d", legacy.Code)
	}
}
