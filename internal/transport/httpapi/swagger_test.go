package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
)

func docsServer() http.Handler {
	a := &application.Auth{Now: time.Now, SigningKey: []byte(strings.Repeat("s", 32)), RateLimit: 60}
	return New(a, nil, nil, "https://frontend.example", true, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestSwaggerRootAssetsAndExistingRoutes(t *testing.T) {
	for _, tc := range []struct {
		method, path, contentType string
		status                    int
	}{
		{"GET", "/", "text/html", 200},
		{"HEAD", "/", "text/html", 200},
		{"GET", "/swagger/", "text/html", 200},
		{"GET", "/swagger/swagger-ui.css", "text/css", 200},
		{"GET", "/swagger/swagger-ui-bundle.js", "javascript", 200},
		{"GET", "/swagger/swagger.js", "javascript", 200},
		{"GET", "/openapi.json", "application/json", 200},
		{"GET", "/not-an-api", "application/json", 404},
		{"GET", "/api/account", "application/json", 401},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			docsServer().ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status || !strings.Contains(w.Header().Get("Content-Type"), tc.contentType) {
				t.Fatalf("status=%d content-type=%q", w.Code, w.Header().Get("Content-Type"))
			}
			if w.Header().Get("Location") != "" {
				t.Fatalf("unexpected redirect: %s", w.Header().Get("Location"))
			}
			if tc.path == "/" && tc.method == "GET" {
				for _, asset := range []string{"/swagger/swagger-ui.css", "/swagger/swagger-ui-bundle.js", "/swagger/swagger.js"} {
					if !strings.Contains(w.Body.String(), asset) {
						t.Fatalf("missing local asset %s", asset)
					}
				}
			}
		})
	}
	w := httptest.NewRecorder()
	docsServer().ServeHTTP(w, httptest.NewRequest("GET", "/login", nil))
	if w.Code != 302 || w.Header().Get("Location") != "https://frontend.example/login" {
		t.Fatal("frontend login redirect changed")
	}
}

func TestSwaggerCatalogMatchesRegistrationAndHasValidParameters(t *testing.T) {
	w := httptest.NewRecorder()
	docsServer().ServeHTTP(w, httptest.NewRequest("GET", "/openapi.json", nil))
	var spec struct {
		OpenAPI string `json:"openapi"`
		Servers []struct {
			URL string `json:"url"`
		} `json:"servers"`
		Paths map[string]map[string]struct {
			ID         string `json:"operationId"`
			Parameters []struct {
				Name, In string
				Required bool
			} `json:"parameters"`
			Responses map[string]any `json:"responses"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	if spec.OpenAPI != "3.0.3" || len(spec.Servers) != 1 || spec.Servers[0].URL != "/" {
		t.Fatal("unexpected version or backend origin")
	}
	ids := map[string]bool{}
	count := 0
	for path, methods := range spec.Paths {
		for method, op := range methods {
			count++
			if ids[op.ID] || op.ID == "" || len(op.Responses) == 0 {
				t.Fatalf("invalid operation %s %s", method, path)
			}
			ids[op.ID] = true
			for _, match := range regexp.MustCompile(`\{([^}]+)\}`).FindAllStringSubmatch(path, -1) {
				found := false
				for _, p := range op.Parameters {
					if p.Name == match[1] && p.In == "path" && p.Required {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing path parameter: %s %s", path, match[1])
				}
			}
		}
	}
	if count < 60 {
		t.Fatalf("API inventory incomplete: %d operations", count)
	}
	var metadata map[string]any
	if err := json.Unmarshal(operationMetadata, &metadata); err != nil {
		t.Fatal(err)
	}
	for pattern := range metadata {
		method, path, _ := strings.Cut(pattern, " ")
		if _, ok := spec.Paths[path][strings.ToLower(method)]; !ok {
			t.Errorf("metadata references unregistered API: %s", pattern)
		}
	}
	for _, path := range []string{"/api/account", "/api/web3/nonce", "/oauth/token", "/oauth/device/code", "/healthz", "/readyz", "/account", "/sanctum/csrf-cookie"} {
		if spec.Paths[path] == nil {
			t.Errorf("missing API %s", path)
		}
	}
	t.Logf("Documented %d registered API operations", count)

	// Newly registered endpoints appear without a second route list to maintain.
	m := newDocumentedMux()
	m.HandleFunc("GET /api/new/{id}", func(http.ResponseWriter, *http.Request) {})
	if m.paths["/api/new/{id}"]["get"] == nil {
		t.Fatal("new route not collected")
	}
}
