package httpapi

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

// Swagger UI is served from the API binary, including in offline deployments.
//
//go:embed swaggerui/*
var swaggerFiles embed.FS

//go:embed openapi_operations.json
var operationMetadata []byte

type documentedMux struct {
	*http.ServeMux
	paths    map[string]map[string]any
	metadata map[string]map[string]any
}

func newDocumentedMux() *documentedMux {
	m := &documentedMux{ServeMux: http.NewServeMux(), paths: map[string]map[string]any{}}
	if err := json.Unmarshal(operationMetadata, &m.metadata); err != nil {
		panic(err)
	}
	return m
}

// Collect the actual registered methods, including compatibility routes and
// methods registered in loops. UI page redirects are not API operations.
func (m *documentedMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.ServeMux.HandleFunc(pattern, handler)
	method, path, ok := strings.Cut(pattern, " ")
	if !ok {
		return
	}
	op := m.metadata[pattern]
	if op == nil {
		op = map[string]any{"summary": method + " " + path}
	}
	op["operationId"] = strings.ToLower(method) + strings.NewReplacer("/", "_", "{", "", "}", "", "-", "_").Replace(path)
	tag := "Browser compatibility"
	switch {
	case path == "/healthz" || path == "/readyz" || path == "/up":
		tag = "Health"
	case strings.Contains(path, "web3"):
		tag = "Wallet"
	case strings.Contains(path, "oauth"):
		tag = "OAuth"
	case strings.Contains(path, "avatar") || strings.Contains(path, "documents") || strings.HasPrefix(path, "/storage/"):
		tag = "Media"
	case strings.Contains(path, "personal-info"):
		tag = "Personal information"
	case strings.HasPrefix(path, "/api/"):
		tag = "Accounts and authentication"
	}
	op["tags"] = []string{tag}
	if op["responses"] == nil {
		op["responses"] = map[string]any{"default": map[string]any{"description": "Handler response. Authentication, validation, redirects and throttling retain their existing API contracts."}}
	}
	parameters, _ := op["parameters"].([]any)
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			parameters = append(parameters, map[string]any{"name": strings.Trim(segment, "{}"), "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
		}
	}
	if len(parameters) > 0 {
		op["parameters"] = parameters
	}
	if strings.Contains(path, "/oauth/device") {
		op["description"] = "Laravel compatibility route only. Device-code grants are not enabled; the current backend returns the original unsupported/error/redirect responses."
	}
	if m.paths[path] == nil {
		m.paths[path] = map[string]any{}
	}
	m.paths[path][strings.ToLower(method)] = op
}

func (s *Server) registerSwaggerRoutes(m *documentedMux) {
	spec, err := json.Marshal(map[string]any{
		"openapi": "3.0.3",
		"info":    map[string]any{"title": "SSO Paradise Supply Chain API", "version": "1.0.0", "description": "Go backend API reference. JSON APIs, OAuth, wallet, media and browser-compatibility routes are listed from the registered handlers. Browser actions require the configured frontend origin and applicable session/CSRF checks; protected endpoints may require a verified email. This documentation viewer does not execute requests or persist credentials. Health endpoints are /healthz and /readyz."},
		"servers": []map[string]string{{"url": "/", "description": "This Go backend"}},
		"paths":   m.paths,
		"components": map[string]any{"securitySchemes": map[string]any{
			"bearerAuth":    map[string]any{"type": "http", "scheme": "bearer", "description": "Bearer access token where accepted by the endpoint."},
			"sessionCookie": map[string]any{"type": "apiKey", "in": "cookie", "name": cookieName, "description": "Browser session; mutation requests also require origin/CSRF validation."},
		}},
	})
	if err != nil {
		panic(err)
	}
	assets, err := fs.Sub(swaggerFiles, "swaggerui")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(assets))
	// Exact-root patterns avoid turning unknown paths into the Swagger page.
	m.ServeMux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		r = r.Clone(r.Context())
		r.URL.Path = "/"
		files.ServeHTTP(w, r)
	})
	m.ServeMux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusTemporaryRedirect)
	})
	m.ServeMux.Handle("GET /swagger/", http.StripPrefix("/swagger/", files))
	m.ServeMux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(spec)
	})
}
