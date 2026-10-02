package httpapi_test

import (
	"context"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

type cookieDecoder struct{ Value string }

func (c cookieDecoder) DecodeCookie(name, value string) (string, error) {
	if name != "laravel_session" || value != "encrypted" {
		return "", errors.New("bad cookie")
	}
	return c.Value, nil
}
func TestLegacyBrowserCookieRestorationOriginAndLogout(t *testing.T) {
	db := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now()
	u, err := db.Create(ctx, domain.User{Name: "Legacy", Email: "legacy@example.com", CreatedAt: now, EmailVerifiedAt: &now})
	if err != nil {
		t.Fatal(err)
	}
	oldID := strings.Repeat("s", 40)
	if err = db.CreateSession(ctx, u.ID, application.Digest(oldID), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	auth := &application.Auth{Accounts: db, Sessions: db, Actions: db, Now: time.Now, SessionTTL: time.Hour, LegacySessionCookie: "laravel_session", LegacyCookies: cookieDecoder{oldID}}
	handler := httpapi.New(auth, nil, nil, "http://localhost:3000", false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	oldCookie := &http.Cookie{Name: "laravel_session", Value: "encrypted"}
	response := request(handler, "GET", "/api/account", "", "", oldCookie)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	fresh := response.Result().Cookies()
	if len(fresh) != 1 || len(fresh[0].Value) != 64 || !fresh[0].HttpOnly {
		t.Fatal("Go session cookie was not established")
	}
	if response = request(handler, "PUT", "/api/account", `{"name":"Legacy","email":"legacy@example.com"}`, "", oldCookie); response.Code != 403 {
		t.Fatal("source cookie bypassed Origin check")
	}
	if response = request(handler, "POST", "/api/logout", "", "http://localhost:3000", fresh[0]); response.Code != 200 {
		t.Fatal(response.Code)
	}
	if response = request(handler, "GET", "/api/account", "", "", oldCookie); response.Code != 401 {
		t.Fatal("old cookie resurrected logout")
	}
}
