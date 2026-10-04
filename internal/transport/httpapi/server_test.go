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
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
)

type mailer struct{}

func (mailer) Send(context.Context, string, string, string) error { return nil }
func server(t *testing.T, enableOAuth ...bool) http.Handler {
	t.Helper()
	db := mysqltest.Open(t)
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, Passwords: security.Bcrypt{Cost: 4}, Mailer: mailer{}, Now: time.Now, PublicURL: "http://localhost:3000", SessionTTL: time.Hour}
	if len(enableOAuth) > 0 && enableOAuth[0] {
		a.OAuth = &application.OAuth{Store: db, Passwords: a.Passwords, Now: time.Now}
	}
	p := &application.Profile{Store: db}
	w := &application.Web3{Wallets: db, Challenges: db, Attributes: db, Registry: &stubRegistry{}, Sessions: db, Now: time.Now, SessionTTL: time.Hour, AppName: "Laravel", PublicURL: "http://localhost:3000"}
	return httpapi.New(a, w, p, a.PublicURL, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
}
func request(h http.Handler, method, path, body, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestCookieSessionCSRFAndProtectedRoutes(t *testing.T) {
	h := server(t)
	w := request(h, "POST", "/api/register", `{"username":"tester","name":"Test","email":"a@example.com","password":"Secure!2026","password_confirmation":"Secure!2026"}`, "http://localhost:3000", nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("unsafe session cookie")
	}
	cookie := cookies[0]
	if strings.Contains(w.Body.String(), "password_hash") || strings.Contains(w.Body.String(), "Secure!") {
		t.Fatal("secret in response")
	}
	if w := request(h, "GET", "/api/account", "", "", cookie); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(h, "PUT", "/api/account", `{"name":"New","email":"b@example.com"}`, "http://localhost:3000", cookie); w.Code != 403 {
		t.Fatal("unverified edit allowed")
	}
	for _, origin := range []string{"https://evil.example", ""} {
		if w := request(h, "POST", "/api/logout", "{}", origin, cookie); w.Code != 403 {
			t.Fatal("CSRF allowed", origin)
		}
	}
	if w := request(h, "POST", "/api/logout", "{}", "http://localhost:3000", cookie); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := request(h, "GET", "/api/account", "", "", cookie); w.Code != 401 {
		t.Fatal("revoked cookie allowed")
	}
}
func TestLaravelLoginInvalidInputAndThrottles(t *testing.T) {
	h := server(t)
	for _, body := range []string{`{"email":"x","admin":true}`, `{} {}`, `{`} {
		if w := request(h, "POST", "/api/login", body, "", nil); w.Code != 422 {
			t.Fatal("invalid body accepted", w.Code)
		}
	}
	for i := 0; i < 7; i++ {
		request(h, "POST", "/api/login", `{"email":"a@example.com","password":"wrong"}`, "", nil)
	}
	if w := request(h, "POST", "/api/login", `{}`, "", nil); w.Code != 429 {
		t.Fatal("login not throttled", w.Code)
	}
}
func TestLimitSharedAcrossRoutesAndMethods(t *testing.T) {
	h := server(t)
	for i := 0; i < 10; i++ {
		path, method := "/api/account", "GET"
		if i%2 == 0 {
			path, method = "/api/login", "POST"
		}
		if w := request(h, method, path, `{}`, "", nil); w.Code == 429 {
			t.Fatal("blocked before ten requests")
		}
	}
	w := request(h, "GET", "/healthz", "", "", nil)
	if w.Code != 429 || w.Header().Get("Retry-After") != "60" {
		t.Fatal("routes do not share budget", w.Code, w.Header())
	}
	r := httptest.NewRequest("GET", "/healthz", nil)
	r.RemoteAddr = "192.0.2.2:1234"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal("independent IP blocked")
	}
}
func TestCrossOriginLoginBlocked(t *testing.T) {
	h := server(t)
	if w := request(h, "POST", "/api/login", `{"email":"a@example.com","password":"Secure!2026"}`, "https://evil.example", nil); w.Code != 403 {
		t.Fatal("cross-origin login accepted")
	}
}

func TestLaravelAPIResponseShapes(t *testing.T) {
	h := server(t)
	w := request(h, "POST", "/api/register", `{"username":"tester","name":"Test","email":"a@example.com","password":"Secure!2026","password_confirmation":"Secure!2026"}`, "", nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	w = request(h, "POST", "/api/login", `{"email":"a@example.com","password":"Secure!2026"}`, "", nil)
	var body map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || len(body) != 2 || body["token"] == nil || body["message"] == nil {
		t.Fatal("login contract", w.Body.String())
	}
	w = request(h, "GET", "/api/user", "", "", cookie)
	body = nil
	json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != 200 || body["data"] != nil || body["id"] == nil || body["password"] != nil {
		t.Fatal("user contract", w.Body.String())
	}
	w = request(h, "GET", "/api/users/1", "", "", nil)
	var resource struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resource)
	if w.Code != 200 || len(resource.Data) != 4 || resource.Data["avatar"] == nil || resource.Data["email"] != nil {
		t.Fatal("public resource", w.Body.String())
	}
	w = request(h, "POST", "/api/login", `{}`, "", nil)
	var validation struct {
		Errors map[string][]string `json:"errors"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &validation); err != nil || w.Code != 422 || len(validation.Errors) == 0 {
		t.Fatal("validation contract", w.Body.String(), err)
	}
}
