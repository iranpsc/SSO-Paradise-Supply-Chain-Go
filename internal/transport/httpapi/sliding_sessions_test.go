package httpapi_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
)

func TestActivityRenewsBrowserCookieAndLogoutDeletesIt(t *testing.T) {
	db := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	u, err := db.Create(ctx, domain.User{Name: "member", Email: "idle-http@example.com", CreatedAt: now, EmailVerifiedAt: &now})
	if err != nil {
		t.Fatal(err)
	}
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, SessionIdle: true, SessionTTL: time.Hour, Now: func() time.Time { return now }}
	h := httpapi.New(a, nil, nil, "http://localhost:3000", false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	token, err := a.StartSession(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: "paradise_session", Value: token}
	now = now.Add(45 * time.Minute)
	response := request(h, "GET", "/api/account", "", "", cookie)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != 3600 || !cookies[0].HttpOnly {
		t.Fatal("browser deadline was not renewed", cookies)
	}
	fixed := strings.Repeat("f", 64)
	if err := db.CreateSession(ctx, u.ID, application.Digest(fixed), now.Add(5*time.Hour)); err != nil {
		t.Fatal(err)
	}
	cookie.Value = fixed
	response = request(h, "GET", "/api/account", "", "", cookie)
	cookies = response.Result().Cookies()
	if response.Code != 200 || len(cookies) != 1 || cookies[0].MaxAge != 18000 {
		t.Fatal("fixed remember deadline was shortened or extended", response.Code, cookies)
	}
	cookie.Value = token
	response = request(h, "POST", "/api/logout", "", "http://localhost:3000", cookie)
	cookies = response.Result().Cookies()
	if response.Code != 200 || len(cookies) == 0 || cookies[len(cookies)-1].MaxAge != -1 {
		t.Fatal("renewal overrode logout cookie", response.Code, cookies)
	}
}
