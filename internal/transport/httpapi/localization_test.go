package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestPersianGeneratedRoutingErrorsAndRequestValidation(t *testing.T) {
	h := server(t, true)
	for _, tc := range []struct {
		method, path, body, contentType string
		status                          int
	}{
		{"GET", "/api/missing", "", "application/json", 404},
		{"DELETE", "/api/login", "", "application/json", 405},
		{"POST", "/api/login", "{}", "text/plain", 422},
		{"POST", "/api/login", "{", "application/json", 422},
		{"POST", "/oauth/token", "{}", "application/json", 400},
		{"POST", "/oauth/token", "client_id=1&client_id=2", "application/x-www-form-urlencoded", 400},
		{"POST", "/api/login", `{"login":"unknown_member","password":"wrong"}`, "application/json", 401},
		{"GET", "/api/web3/nonce?address=invalid", "", "application/json", 422},
	} {
		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		r.Header.Set("Content-Type", tc.contentType)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Body.String())
		}
		var result struct {
			Message     string              `json:"message"`
			Errors      map[string][]string `json:"errors"`
			Error       string              `json:"error"`
			Description string              `json:"error_description"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal("error response is not JSON", err)
		}
		if strings.HasPrefix(tc.path, "/oauth/") {
			if result.Error != "invalid_request" || result.Description == "" || result.Message != "" {
				t.Fatal("Passport error shape differs", w.Body.String())
			}
			continue
		}
		if tc.path == "/api/login" && tc.status == 422 {
			continue
		} // exact Laravel validation summaries are compared to PHP fixtures
		if tc.path == "/api/login" && tc.status == http.StatusUnauthorized {
			if result.Message != "Invalid credentials" {
				t.Fatal("login error differs from Laravel", result.Message)
			}
			continue
		}
		messages := []string{result.Message}
		for _, values := range result.Errors {
			messages = append(messages, values...)
		}
		for _, message := range messages {
			if !regexp.MustCompile(`[\x{0600}-\x{06ff}]`).MatchString(message) || regexp.MustCompile(`[a-zA-Z]`).MatchString(message) {
				t.Fatal("non-Persian public message", message)
			}
		}
		if tc.status == http.StatusMethodNotAllowed && !strings.Contains(w.Header().Get("Allow"), "POST") {
			t.Fatal("localized routing lost Allow header")
		}
	}
}
