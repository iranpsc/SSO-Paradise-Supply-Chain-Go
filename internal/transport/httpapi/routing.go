package httpapi

import "net/http"

type routingResult struct {
	headers http.Header
	status  int
}

func (r *routingResult) Header() http.Header    { return r.headers }
func (r *routingResult) WriteHeader(status int) { r.status = status }
func (r *routingResult) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = 200
	}
	return len(data), nil
}

// Preserve ServeMux route selection, path values and Allow headers while
// replacing its generated English 404/405 pages with Persian API messages.
func localizedRouting(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, pattern := mux.Handler(r)
		if pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		result := &routingResult{headers: make(http.Header)}
		handler.ServeHTTP(result, r)
		if allow := result.headers.Get("Allow"); allow != "" {
			w.Header().Set("Allow", allow)
		}
		switch result.status {
		case 404:
			respond(w, 404, map[string]string{"message": "صفحه یا مسیر درخواستی یافت نشد."})
		case 405:
			respond(w, 405, map[string]string{"message": "روش ارسال این درخواست مجاز نیست."})
		default:
			if location := result.headers.Get("Location"); location != "" {
				w.Header().Set("Location", location)
			}
			w.WriteHeader(result.status)
		}
	})
}
