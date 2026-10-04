package httpapi_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPassportJWTLoginCallbackFlagConsumptionAndLogout(t *testing.T) {
	db := mysqltest.Open(t)
	ctx := context.Background()
	passwords := security.Bcrypt{Cost: 4}
	hash, _ := passwords.Hash("SecurePass!2026")
	u, err := db.Create(ctx, domain.User{Name: "Test", Username: "jwt_user", Email: "jwt@example.com", PasswordHash: hash, CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, Passwords: passwords, Now: time.Now, PublicURL: "http://localhost:3000", SessionTTL: time.Hour, SigningKey: []byte(strings.Repeat("k", 32))}
	a.OAuth = &application.OAuth{Store: db, Passwords: passwords, Now: time.Now, Signer: security.JWT{Key: key}}
	web3 := &application.Web3{Attributes: db}
	h := httpapi.New(a, web3, &application.Profile{Store: db}, a.PublicURL, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ip := 0
	do := func(method, path, body, contentType string, cookie *http.Cookie, bearer string) *httptest.ResponseRecorder {
		ip++
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = fmt.Sprintf("198.51.100.%d:1234", ip)
		if contentType != "" {
			r.Header.Set("Content-Type", contentType)
		}
		if cookie != nil {
			r.AddCookie(cookie)
			r.Header.Set("Origin", a.PublicURL)
		}
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	login := do("POST", "/api/login", `{"login":"jwt_user","password":"SecurePass!2026","remember":true}`, "application/json", nil, "")
	if login.Code != 200 {
		t.Fatal(login.Code, login.Body.String())
	}
	var result struct {
		Message string `json:"message"`
		Token   string `json:"token"`
	}
	json.Unmarshal(login.Body.Bytes(), &result)
	if result.Message != "Login successful" {
		t.Fatal("login message differs from Laravel", result.Message)
	}
	if len(strings.Split(result.Token, ".")) != 3 {
		t.Fatal("API login did not issue JWT")
	}
	cookie := login.Result().Cookies()[0]
	if cookie.MaxAge != 30*24*60*60 {
		t.Fatal("remember cookie lifetime differs")
	}
	if w := do("GET", "/api/user", "", "", nil, result.Token); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	secretHash, _ := passwords.Hash("client-secret")
	id, err := db.CreateOAuthClient(ctx, "Laravel client", secretHash, []string{"https://client.example/callback"})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SetSessionAttribute(ctx, application.Digest(cookie.Value), "wallet_login", "true", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	query := url.Values{"client_id": {strconv.FormatInt(id, 10)}, "redirect_uri": {"https://client.example/callback"}, "response_type": {"code"}, "scope": {""}, "state": {"bound-state"}}
	guest := do("GET", "/oauth/authorize?"+query.Encode(), "", "", nil, "")
	if guest.Code != 302 || guest.Header().Get("Location") != a.PublicURL+"/login" {
		t.Fatal("guest authorization redirect differs", guest.Code, guest.Header())
	}
	var flashCookie *http.Cookie
	for _, c := range guest.Result().Cookies() {
		if c.Name == "paradise_flash" {
			flashCookie = c
		}
	}
	if flashCookie == nil {
		t.Fatal("intended authorization URL lost")
	}
	flash := do("GET", "/api/legacy/flash", "", "", flashCookie, "")
	var flashBody map[string]string
	if err := json.Unmarshal(flash.Body.Bytes(), &flashBody); err != nil || flashBody["intended"] != "/oauth/authorize?"+query.Encode() {
		t.Fatal("signed intended URL lost", flash.Body.String())
	}
	callback := do("GET", "/oauth/authorize?"+query.Encode(), "", "", cookie, "")
	if callback.Code != 302 {
		t.Fatal(callback.Code, callback.Body.String())
	}
	target, _ := url.Parse(callback.Header().Get("Location"))
	if target.Query().Get("wallet_login") != "true" || target.Query().Get("state") != "bound-state" || target.Query().Get("code") == "" {
		t.Fatal("callback contract differs")
	}
	if _, err = db.SessionAttribute(ctx, application.Digest(cookie.Value), "wallet_login", time.Now()); err != domain.ErrNotFound {
		t.Fatal("wallet flag not consumed")
	}
	second := do("GET", "/oauth/authorize?"+query.Encode(), "", "", cookie, "")
	if strings.Contains(second.Header().Get("Location"), "wallet_login") {
		t.Fatal("wallet callback flag replayed")
	}
	// Password confirmation is tied to the browser session.
	confirm := do("POST", "/api/password/confirm", `{"password":"SecurePass!2026"}`, "application/json", cookie, "")
	if confirm.Code != 200 {
		t.Fatal(confirm.Code, confirm.Body.String())
	}
	if _, err = db.SessionAttribute(ctx, application.Digest(cookie.Value), "password_confirmed_at", time.Now()); err != nil {
		t.Fatal(err)
	}
	// Native Laravel signed URLs authenticate the same account and assign its code.
	link := security.SignedVerificationURL(a.PublicURL, u.ID, u.Email, time.Now().Add(time.Hour), a.SigningKey)
	verified := do("GET", link, "", "", cookie, "")
	if verified.Code != 302 {
		t.Fatal(verified.Code, verified.Body.String())
	}
	if fresh, err := db.ByID(ctx, u.ID); err != nil || fresh.EmailVerifiedAt == nil || fresh.Code == nil {
		t.Fatal("signed verification did not persist")
	}
	logout := do("POST", "/api/logout", "", "application/json", nil, result.Token)
	if logout.Code != 200 {
		t.Fatal(logout.Code, logout.Body.String())
	}
	var logoutResult map[string]string
	if err := json.Unmarshal(logout.Body.Bytes(), &logoutResult); err != nil || len(logoutResult) != 1 || logoutResult["message"] != "Logged out successfully" {
		t.Fatal("logout response differs from Laravel", logout.Body.String())
	}
	if w := do("GET", "/api/user", "", "", nil, result.Token); w.Code != 401 {
		t.Fatal("JWT survived logout")
	}
	if w := do("GET", "/api/account", "", "", cookie, ""); w.Code != 401 {
		t.Fatal("cookie survived logout")
	}
}
