package httpapi

import (
	"net/http"
	"regexp"
	"strings"
)

func securityHeaders(w http.ResponseWriter, r *http.Request, secure bool) {
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
	w.Header().Set("Content-Security-Policy", "base-uri 'self'; object-src 'none'; frame-ancestors 'none'")
	if secure || r.TLS != nil {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}
}

var metarangOrigin = regexp.MustCompile(`^https://([a-z0-9-]+\.)*metarang\.com$`)

func allowedCORSOrigin(origin string) bool {
	return origin == "https://accounts.irpsc.com" || metarangOrigin.MatchString(origin)
}

// Token callers use the legacy API allowlist. Browser session mutations still
// require the configured same origin; PUT remains available to the Go frontend.
func (s *Server) cors(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/sanctum/csrf-cookie" {
		return false
	}
	w.Header().Add("Vary", "Origin")
	origin := r.Header.Get("Origin")
	if origin == "" || (!allowedCORSOrigin(origin) && origin != s.origin) {
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	if r.Method != http.MethodOptions {
		return false
	}
	w.Header().Add("Vary", "Access-Control-Request-Method")
	w.Header().Add("Vary", "Access-Control-Request-Headers")
	method := r.Header.Get("Access-Control-Request-Method")
	if method != "GET" && method != "POST" && !(origin == s.origin && (method == "PUT" || method == "PATCH" || method == "DELETE")) {
		w.WriteHeader(http.StatusForbidden)
		return true
	}
	allowed := map[string]bool{"accept": true, "authorization": true, "content-type": true, "x-requested-with": true, "x-csrf-token": true, "x-xsrf-token": true}
	for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
		if header = strings.ToLower(strings.TrimSpace(header)); header != "" && !allowed[header] {
			w.WriteHeader(http.StatusForbidden)
			return true
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", method)
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Requested-With, X-CSRF-TOKEN, X-XSRF-TOKEN")
	w.WriteHeader(http.StatusNoContent)
	return true
}
