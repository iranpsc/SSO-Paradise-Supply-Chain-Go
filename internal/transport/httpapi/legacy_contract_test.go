package httpapi_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLaravelLoginValidationAndUnknownFields(t *testing.T) {
	h := server(t)
	data, err := os.ReadFile("testdata/laravel-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures struct{ Login map[string]json.RawMessage }
	if err = json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"empty": "{}", "invalid": `{"email":"invalid","password":12}`} {
		w := request(h, "POST", "/api/login", body, "", nil)
		if w.Code != 422 {
			t.Fatal(w.Code, w.Body.String())
		}
		var want, got any
		json.Unmarshal(fixtures.Login[name], &want)
		json.Unmarshal(w.Body.Bytes(), &got)
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("validation differs from PHP: %s", w.Body.String())
		}
	}
	for _, tc := range []struct{ body, ct string }{{`{"email":"valid@example.com","password":"test","admin":true}`, "application/json"}, {"email=valid%40example.com&password=test&admin=1", "application/x-www-form-urlencoded"}} {
		r := httptest.NewRequest("POST", "/api/login", strings.NewReader(tc.body))
		r.Header.Set("Content-Type", tc.ct)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 401 || strings.TrimSpace(w.Body.String()) != `{"message":"Invalid credentials"}` {
			t.Fatal("unknown fields/form contract differs", w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/api/user", "/api/me", "/api/logout"} {
		method := "POST"
		if path == "/api/user" {
			method = "GET"
		}
		w := request(h, method, path, "{}", "", nil)
		if w.Code != 401 || strings.TrimSpace(w.Body.String()) != `{"message":"Unauthenticated."}` {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
}

func TestLaravelUserAndResourcesOverHTTP(t *testing.T) {
	db := mysqltest.Open(t)
	ctx := context.Background()
	stamp, _ := time.Parse(time.RFC3339, "2026-01-01T23:34:05Z")
	u, err := db.Create(ctx, domain.User{Name: "Test", Username: "extension_only", Email: "test@example.com", CreatedAt: stamp})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("mysql", db.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err = raw.Exec(`UPDATE users SET code='hm-2000042',email_verified_at=?,updated_at=? WHERE id=?`, stamp, stamp, u.ID); err != nil {
		t.Fatal(err)
	}
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, Passwords: security.Bcrypt{Cost: 4}, Now: time.Now, SessionTTL: time.Hour, PublicURL: "http://localhost:3000"}
	session, err := a.StartSession(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	h := httpapi.New(a, nil, &application.Profile{Store: db}, a.PublicURL, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	data, err := os.ReadFile("testdata/laravel-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]map[string]any // routes/oauth have other shapes, so read only the model/resource values
	var rawFixture map[string]json.RawMessage
	json.Unmarshal(data, &rawFixture)
	fixture = map[string]map[string]any{}
	for _, key := range []string{"user", "resource", "verified_resource"} {
		var value map[string]any
		json.Unmarshal(rawFixture[key], &value)
		fixture[key] = value
		fixture[key]["id"] = float64(u.ID)
	}
	compare := func(method, path, key string, wrapped bool) {
		t.Helper()
		r := httptest.NewRequest(method, path, nil)
		r.Header.Set("Authorization", "Bearer "+session)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var body map[string]any
		json.Unmarshal(w.Body.Bytes(), &body)
		if wrapped {
			body = body["data"].(map[string]any)
		}
		if !reflect.DeepEqual(body, fixture[key]) {
			t.Fatalf("%s differs from Laravel: %s", path, w.Body.String())
		}
	}
	compare("GET", "/api/user", "user", false)
	compare("POST", "/api/me", "resource", true)
	compare("GET", fmt.Sprintf("/api/users/%d", u.ID), "resource", true)
	if _, err = raw.Exec(`UPDATE personal_infos SET is_verified=1,first_name='First',last_name='Last' WHERE user_id=?`, u.ID); err != nil {
		t.Fatal(err)
	}
	compare("POST", "/api/me", "verified_resource", true)
	compare("GET", fmt.Sprintf("/api/users/%d", u.ID), "verified_resource", true)
}

func TestPassportPasswordGrantAndDeviceClientPolicy(t *testing.T) {
	db := mysqltest.Open(t)
	ctx := context.Background()
	passwords := security.Bcrypt{Cost: 4}
	hash, err := passwords.Hash("SecurePass!2026")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Create(ctx, domain.User{Name: "Password grant", Email: "password@example.com", PasswordHash: hash, CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	secret, err := passwords.Hash("client-secret")
	if err != nil {
		t.Fatal(err)
	}
	id, err := db.CreateOAuthClient(ctx, "Legacy password client", secret, []string{"https://client.example/callback"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("mysql", db.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err = raw.Exec(`UPDATE oauth_clients SET grant_types='["password","refresh_token"]' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, Passwords: passwords, Now: time.Now, SessionTTL: time.Hour, PublicURL: "http://localhost:3000"}
	a.OAuth = &application.OAuth{Store: db, Passwords: passwords, Now: time.Now, AccessTTL: time.Hour, RefreshTTL: 2 * time.Hour}
	h := httpapi.New(a, nil, nil, a.PublicURL, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	base := fmt.Sprintf("client_id=%d&client_secret=client-secret", id)
	send := func(path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	bad := send("/oauth/token", base+"&grant_type=password&username=password%40example.com&password=wrong")
	if bad.Code != 400 || strings.TrimSpace(bad.Body.String()) != `{"error":"invalid_grant","error_description":"The user credentials were incorrect."}` {
		t.Fatal(bad.Code, bad.Body.String())
	}
	result := send("/oauth/token", base+"&grant_type=password&username=password%40example.com&password=SecurePass%212026")
	if result.Code != 200 {
		t.Fatal(result.Code, result.Body.String())
	}
	var body map[string]any
	if err = json.Unmarshal(result.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 4 || body["token_type"] != "Bearer" || body["expires_in"] != float64(3600) || body["access_token"] == "" || body["refresh_token"] == "" {
		t.Fatal("Passport token response differs", body)
	}
	rotated := send("/oauth/token", base+"&grant_type=refresh_token&refresh_token="+body["refresh_token"].(string))
	if rotated.Code != 200 {
		t.Fatal(rotated.Code, rotated.Body.String())
	}
	replay := send("/oauth/token", base+"&grant_type=refresh_token&refresh_token="+body["refresh_token"].(string))
	if replay.Code != 400 {
		t.Fatal("refresh replay accepted", replay.Code)
	}
	device := send("/oauth/device/code", fmt.Sprintf("client_id=%d", id))
	if device.Code != 400 || !strings.Contains(device.Body.String(), `"error":"unauthorized_client"`) {
		t.Fatal("device grant policy differs from Laravel legacy clients", device.Code, device.Body.String())
	}
}

func TestLegacyWeb3CSRFAndRedirects(t *testing.T) {
	h := server(t)
	r := httptest.NewRequest("GET", "/web3/nonce?address=0x"+strings.Repeat("a", 40), nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"nonce"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	var nonce map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &nonce); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/laravel-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Web3Nonce map[string]string `json:"web3_nonce"`
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	actual := nonce["nonce"]
	if index := strings.LastIndex(actual, "Nonce: "); index >= 0 {
		actual = actual[:index] + "Nonce: " + strings.Repeat("a", 32)
	}
	if actual != fixture.Web3Nonce["nonce"] {
		t.Fatalf("nonce differs from PHP: %q", actual)
	}
	body := `{"address":"0x` + strings.Repeat("a", 40) + `","signature":"0x` + strings.Repeat("b", 130) + `"}`
	w = request(h, "POST", "/web3/verify", body, "", nil)
	if w.Code != 419 {
		t.Fatal("missing CSRF accepted", w.Code, w.Body.String())
	}
	csrf := request(h, "GET", "/sanctum/csrf-cookie", "", "", nil)
	if csrf.Code != 204 {
		t.Fatal(csrf.Code)
	}
	var public string
	for _, c := range csrf.Result().Cookies() {
		if c.Name == "XSRF-TOKEN" {
			public = c.Value
		}
	}
	r = httptest.NewRequest("POST", "/web3/verify", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("X-XSRF-TOKEN", public)
	for _, c := range csrf.Result().Cookies() {
		r.AddCookie(c)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 || !strings.Contains(w.Body.String(), "Signature verification failed") {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, jsonRequest := range []bool{false, true} {
		r = httptest.NewRequest("GET", "/web3/link/nonce?address=0x"+strings.Repeat("a", 40), nil)
		if jsonRequest {
			r.Header.Set("Accept", "application/json")
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if jsonRequest {
			if w.Code != 401 || !strings.Contains(w.Body.String(), "Unauthenticated.") {
				t.Fatal(w.Code, w.Body.String())
			}
		} else if w.Code != 302 || w.Header().Get("Location") != "http://localhost:3000/login" {
			t.Fatal(w.Code, w.Header())
		}
	}
}

func TestLaravelRouteInventoryHasNoMissingPathsOrMethods(t *testing.T) {
	h := server(t, true)
	data, err := os.ReadFile("testdata/laravel-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Routes []struct {
			URI     string
			Methods []string
		}
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	peer := 0
	for _, route := range fixture.Routes {
		for _, method := range route.Methods {
			t.Run(method+" "+route.URI, func(t *testing.T) {
				path := "/" + route.URI
				for _, key := range []string{"{user}", "{id}", "{hash}", "{token}"} {
					path = strings.ReplaceAll(path, key, "42")
				}
				peer++
				r := httptest.NewRequest(method, path, strings.NewReader("{}"))
				r.RemoteAddr = fmt.Sprintf("198.51.100.%d:1234", peer)
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Accept", "application/json")
				r.Header.Set("Sec-Fetch-Site", "same-origin")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code == 405 || (w.Code == 404 && strings.Contains(w.Body.String(), "صفحه یا مسیر درخواست")) {
					t.Fatal("missing Laravel route", w.Code, w.Body.String())
				}
			})
		}
	}
}
