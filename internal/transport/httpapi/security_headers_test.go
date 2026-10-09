package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersAndHTTPS(t *testing.T) {
	for _, secure := range []bool{false, true} {
		w := httptest.NewRecorder()
		securityHeaders(w, httptest.NewRequest("GET", "/", nil), secure)
		for key, value := range map[string]string{
			"X-Frame-Options": "DENY", "X-Content-Type-Options": "nosniff",
			"Referrer-Policy":                   "strict-origin-when-cross-origin",
			"X-Permitted-Cross-Domain-Policies": "none",
			"Content-Security-Policy":           "base-uri 'self'; object-src 'none'; frame-ancestors 'none'",
		} {
			if w.Header().Get(key) != value {
				t.Fatalf("%s: %q", key, w.Header().Get(key))
			}
		}
		if (w.Header().Get("Strict-Transport-Security") != "") != secure {
			t.Fatal("incorrect HSTS policy")
		}
	}
}

func TestCORSAllowlistAndPreflight(t *testing.T) {
	s := &Server{origin: "http://localhost:3000"}
	for _, tc := range []struct {
		origin, method, headers string
		status                  int
		allowed                 bool
	}{
		{"https://accounts.irpsc.com", "POST", "authorization,content-type", 204, true},
		{"https://metarang.com", "GET", "", 204, true},
		{"https://app.metarang.com", "GET", "", 204, true},
		{"https://metarang.com.evil.test", "GET", "", 200, false},
		{"http://metarang.com", "GET", "", 200, false},
		{"https://metarang.com", "PUT", "", 403, true},
		{"https://metarang.com", "POST", "x-unsafe-header", 403, true},
		{"http://localhost:3000", "PUT", "content-type", 204, true},
	} {
		r := httptest.NewRequest(http.MethodOptions, "/api/account", nil)
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Access-Control-Request-Method", tc.method)
		r.Header.Set("Access-Control-Request-Headers", tc.headers)
		w := httptest.NewRecorder()
		s.cors(w, r)
		if w.Code != tc.status || (w.Header().Get("Access-Control-Allow-Origin") != "") != tc.allowed {
			t.Fatalf("%+v: %d %v", tc, w.Code, w.Header())
		}
		if w.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Fatal("cross-origin credentials enabled")
		}
	}
}
