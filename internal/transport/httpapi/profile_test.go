package httpapi_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
)

const png1x1 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

type mailboxCapture struct{ body string }

func (m *mailboxCapture) Send(_ context.Context, _, _, body string) error { m.body = body; return nil }

func profileServer(t *testing.T) (http.Handler, *application.Auth, *mailboxCapture) {
	t.Helper()
	db := mysqltest.Open(t)
	m := &mailboxCapture{}
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, Passwords: security.Bcrypt{Cost: 4}, Mailer: m, Now: time.Now, PublicURL: "http://localhost:3000", SessionTTL: time.Hour}
	p := &application.Profile{Store: db}
	w := &application.Web3{Wallets: db, Challenges: db, Attributes: db, Registry: &stubRegistry{}, Sessions: db, Now: time.Now, SessionTTL: time.Hour, AppName: "Laravel", PublicURL: "http://localhost:3000"}
	h := httpapi.New(a, w, p, a.PublicURL, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return h, a, m
}

func registerVerified(t *testing.T, h http.Handler, a *application.Auth, m *mailboxCapture, username, email string) *http.Cookie {
	t.Helper()
	w := request(h, "POST", "/api/register", `{"username":"`+username+`","name":"Test User","email":"`+email+`","password":"Secure!2026","password_confirmation":"Secure!2026"}`, "", nil)
	if w.Code != 201 {
		t.Fatalf("register %d %s", w.Code, w.Body.String())
	}
	// Extract token from last mail.
	body := m.body
	idx := strings.LastIndex(body, "http")
	if idx < 0 {
		t.Fatal("no verification mail")
	}
	link := body[idx:]
	token := ""
	if i := strings.Index(link, "token="); i >= 0 {
		rest := link[i+6:]
		if j := strings.Index(rest, "&"); j >= 0 {
			token = rest[:j]
		} else {
			token = strings.Fields(rest)[0]
		}
	}
	if token == "" {
		t.Fatal("no token in mail")
	}
	// Need user ID: login to get cookie first (unverified can still login? yes, login has no verified check).
	// Verify via application layer: find user by email then Verify.
	ctx := context.Background()
	u, err := a.Accounts.ByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	// Authenticate with register cookie to call verify endpoint.
	cookie := w.Result().Cookies()[0]
	r := httptest.NewRequest("POST", "/api/email/verify", strings.NewReader(`{"token":"`+token+`"}`))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(cookie)
	r.Header.Set("Origin", "http://localhost:3000")
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 200 {
		t.Fatalf("verify %d %s user=%d", rw.Code, rw.Body.String(), u.ID)
	}
	// Login fresh to get verified session.
	w2 := request(h, "POST", "/api/login", `{"email":"`+email+`","password":"Secure!2026"}`, "", nil)
	if w2.Code != 200 {
		t.Fatalf("login %d %s", w2.Code, w2.Body.String())
	}
	return w2.Result().Cookies()[0]
}

func multipartProfile(t *testing.T, fields map[string]string, files map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	raw, _ := base64.StdEncoding.DecodeString(png1x1)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	for kind, filename := range files {
		fw, err := w.CreateFormFile(kind, filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(raw); err != nil {
			t.Fatal(err)
		}
	}
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

func TestPersonalInfoFlow(t *testing.T) {
	h, a, m := profileServer(t)
	cookie := registerVerified(t, h, a, m, "profileuser", "p@example.com")

	// GET empty personal-info requires verified (we are verified) -> 200 with empty names.
	r := httptest.NewRequest("GET", "/api/personal-info", nil)
	r.AddCookie(cookie)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 200 {
		t.Fatalf("get empty %d %s", rw.Code, rw.Body.String())
	}

	// PUT without docs -> 422.
	r = httptest.NewRequest("PUT", "/api/personal-info", strings.NewReader(`{"is_company":false,"first_name":"Ali","last_name":"Rezaei","mobile":"09123456789","telephone":"02112345678","national_code":"1111111111","address":"Tehran"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://localhost:3000")
	r.AddCookie(cookie)
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 422 {
		t.Fatalf("expected docs required, got %d %s", rw.Code, rw.Body.String())
	}

	// PUT multipart with 3 docs -> 200.
	fields := map[string]string{
		"is_company": "false", "first_name": "Ali", "last_name": "Rezaei",
		"mobile": "09123456789", "telephone": "02112345678", "national_code": "1111111111", "address": "Tehran",
	}
	files := map[string]string{
		"melli_card_scan": "melli.png", "certificate_scan": "cert.png", "bank_card_scan": "bank.png",
	}
	buf, ct := multipartProfile(t, fields, files)
	r = httptest.NewRequest("PUT", "/api/personal-info", buf)
	r.Header.Set("Content-Type", ct)
	r.Header.Set("Origin", "http://localhost:3000")
	r.AddCookie(cookie)
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 200 {
		t.Fatalf("put multipart %d %s", rw.Code, rw.Body.String())
	}

	// GET document.
	r = httptest.NewRequest("GET", "/api/personal-info/documents/melli_card_scan", nil)
	r.AddCookie(cookie)
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 200 || !strings.HasPrefix(rw.Header().Get("Content-Type"), "image/") {
		t.Fatalf("get doc %d %s", rw.Code, rw.Body.String())
	}

	// JSON text-only update now allowed (docs exist).
	r = httptest.NewRequest("PUT", "/api/personal-info", strings.NewReader(`{"is_company":false,"first_name":"Ali2","last_name":"Rezaei","mobile":"09123456789","telephone":"02112345678","national_code":"1111111111","address":"Tehran 2"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://localhost:3000")
	r.AddCookie(cookie)
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 200 || !strings.Contains(rw.Body.String(), "Ali2") {
		t.Fatalf("json update after docs %d %s", rw.Code, rw.Body.String())
	}

	// Invalid national code rejected.
	r = httptest.NewRequest("PUT", "/api/personal-info", strings.NewReader(`{"is_company":false,"first_name":"A","last_name":"B","mobile":"09123456789","telephone":"02112345678","national_code":"1234567890","address":"x"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://localhost:3000")
	r.AddCookie(cookie)
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 422 {
		t.Fatalf("invalid national accepted %d", rw.Code)
	}
}

func TestAvatarFlow(t *testing.T) {
	h, a, m := profileServer(t)
	cookie := registerVerified(t, h, a, m, "avataruser", "av@example.com")

	raw, _ := base64.StdEncoding.DecodeString(png1x1)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("avatar", "avatar.png")
	_, _ = fw.Write(raw)
	_ = w.Close()
	r := httptest.NewRequest("PUT", "/api/account/avatar", &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.Header.Set("Origin", "http://localhost:3000")
	r.AddCookie(cookie)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 200 {
		t.Fatalf("put avatar %d %s", rw.Code, rw.Body.String())
	}

	// Own avatar.
	r = httptest.NewRequest("GET", "/api/account/avatar", nil)
	r.AddCookie(cookie)
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 200 {
		t.Fatalf("get own avatar %d", rw.Code)
	}

	// Public avatar: user id 1.
	r = httptest.NewRequest("GET", "/api/users/1/avatar", nil)
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 200 {
		t.Fatalf("public avatar %d %s", rw.Code, rw.Body.String())
	}

	// /api/me includes avatar URL.
	r = httptest.NewRequest("POST", "/api/me", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://localhost:3000")
	r.AddCookie(cookie)
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 200 || !strings.Contains(rw.Body.String(), "/api/users/1/avatar") {
		t.Fatalf("me avatar %d %s", rw.Code, rw.Body.String())
	}

	// Reject non-image.
	var buf2 bytes.Buffer
	w2 := multipart.NewWriter(&buf2)
	fw2, _ := w2.CreateFormFile("avatar", "evil.php")
	_, _ = fw2.Write([]byte("<?php echo 1;"))
	_ = w2.Close()
	r = httptest.NewRequest("PUT", "/api/account/avatar", &buf2)
	r.Header.Set("Content-Type", w2.FormDataContentType())
	r.Header.Set("Origin", "http://localhost:3000")
	r.AddCookie(cookie)
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	if rw.Code != 422 {
		t.Fatalf("evil avatar accepted %d", rw.Code)
	}
}
