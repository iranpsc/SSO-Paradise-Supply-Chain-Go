package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

// Laravel's model serialization is a public contract, independent of the
// richer domain object used by the new first-party frontend.
func laravelUser(u domain.User) map[string]any {
	nullable := func(v string) any {
		if v == "" {
			return nil
		}
		return v
	}
	stamp := func(t time.Time) any {
		if t.IsZero() {
			return nil
		}
		return t.UTC().Format("2006-01-02T15:04:05.000000Z")
	}
	var verified any
	if u.EmailVerifiedAt != nil {
		verified = stamp(*u.EmailVerifiedAt)
	}
	return map[string]any{"id": u.ID, "name": nullable(u.Name), "email": nullable(u.Email), "mobile": u.Mobile, "email_verified_at": verified, "referral": nullable(u.Referral), "code": u.Code, "created_at": stamp(u.CreatedAt), "updated_at": stamp(u.UpdatedAt), "wallet_address": u.Wallet}
}

func expectsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "json") || (r.Header.Get("X-Requested-With") == "XMLHttpRequest" && !strings.Contains(r.Header.Get("Accept"), "text/html"))
}

func laravelValidation(w http.ResponseWriter, fields map[string][]string, order ...string) {
	if len(order) == 0 {
		for key := range fields {
			order = append(order, key)
		}
		sort.Strings(order)
	}
	var messages []string
	for _, key := range order {
		messages = append(messages, fields[key]...)
	}
	message := "The given data was invalid."
	if len(messages) > 0 {
		message = messages[0]
	}
	if len(messages) == 2 {
		message += " (and 1 more error)"
	} else if len(messages) > 2 {
		message += fmt.Sprintf(" (and %d more errors)", len(messages)-1)
	}
	respond(w, 422, map[string]any{"message": message, "errors": fields})
}

// Like Laravel, unknown input keys are ignored by validation. Form/query
// inputs and JSON share the same path; password whitespace is preserved.
func laravelInput(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	in := map[string]any{}
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			respond(w, 413, map[string]string{"message": "The payload is too large."})
			return nil, false
		}
		// Laravel's InputBag treats malformed JSON as empty input.
		_ = json.Unmarshal(data, &in)
		if in == nil {
			in = map[string]any{}
		}
	} else {
		var err error
		if strings.HasPrefix(ct, "multipart/form-data") {
			err = r.ParseMultipartForm(6 << 20)
		} else {
			err = r.ParseForm()
		}
		if err != nil {
			respond(w, 400, map[string]string{"message": "Bad Request"})
			return nil, false
		}
		for key, values := range r.PostForm {
			if len(values) > 0 {
				in[key] = values[0]
			}
		}
	}
	for key, values := range r.URL.Query() {
		if _, ok := in[key]; !ok && len(values) > 0 {
			in[key] = values[0]
		}
	}
	for key, value := range in {
		if v, ok := value.(string); ok && key != "password" && key != "password_confirmation" && key != "current_password" {
			in[key] = strings.TrimSpace(v)
		}
	}
	return in, true
}

func inputString(in map[string]any, key string) string { v, _ := in[key].(string); return v }

func (s *Server) loginInput(w http.ResponseWriter, r *http.Request) (string, string, bool, bool) {
	if _, err := s.authenticate(r, w); err == nil {
		http.Redirect(w, r, s.origin+"/home", 302)
		return "", "", false, false
	}
	in, ok := laravelInput(w, r)
	if !ok {
		return "", "", false, false
	}
	login := inputString(in, "login")
	email := inputString(in, "email")
	password := inputString(in, "password")
	fields := map[string][]string{}
	if login == "" {
		if email == "" {
			fields["email"] = []string{"فیلد پست الکترونیکی الزامی است"}
		} else if address, err := mail.ParseAddress(email); err != nil || address.Address != email || !strings.Contains(email, "@") {
			fields["email"] = []string{"فرمت پست الکترونیکی معتبر نیست."}
		}
	}
	if value, exists := in["password"]; !exists || value == nil {
		fields["password"] = []string{"فیلد رمز عبور الزامی است"}
	} else if v, ok := value.(string); !ok {
		fields["password"] = []string{"رمز عبور باید رشته باشد."}
	} else if v == "" {
		fields["password"] = []string{"فیلد رمز عبور الزامی است"}
	}
	if len(fields) > 0 {
		laravelValidation(w, fields, "email", "password")
		return "", "", false, false
	}
	if login != "" && email != "" {
		laravelValidation(w, map[string][]string{"login": {"فقط یک نام کاربری یا ایمیل وارد کنید."}}, "login")
		return "", "", false, false
	}
	if login == "" {
		login = email
	}
	remember := false
	switch value := in["remember"].(type) {
	case bool:
		remember = value
	case string:
		remember = value == "1" || value == "true"
	case float64:
		remember = value == 1
	}
	return login, password, remember, true
}

func replaceJSONBody(r *http.Request, in map[string]any) {
	data, _ := json.Marshal(in)
	r.Body = io.NopCloser(bytes.NewReader(data))
	r.ContentLength = int64(len(data))
	r.Header.Set("Content-Type", "application/json")
}

func legacyAPI(path string) bool {
	return path == "/api/me" || path == "/api/user" || path == "/api/logout" || strings.HasPrefix(path, "/api/users/")
}

func (s *Server) modelNotFound(w http.ResponseWriter, id string) {
	respond(w, 404, map[string]string{"message": "No query results for model [App\\Models\\User] " + id})
}

func isLegacyWeb(path string) bool {
	if path == "/oauth/authorize" || path == "/oauth/token/refresh" || path == "/oauth/device" || path == "/oauth/device/authorize" {
		return true
	}
	if strings.HasPrefix(path, "/web3/") || strings.HasPrefix(path, "/email/verify/") {
		return true
	}
	switch path {
	case "/", "/home", "/account", "/account/edit", "/personal-info", "/personal-info/edit", "/change-password", "/login", "/logout", "/register", "/password/email", "/password/reset", "/password/confirm", "/email/verify", "/email/verification-notification", "/sanctum/csrf-cookie":
		return true
	}
	return false
}

func rateFailure(w http.ResponseWriter, r *http.Request) {
	if legacyAPI(r.URL.Path) || isLegacyWeb(r.URL.Path) || r.URL.Path == "/api/login" || strings.HasPrefix(r.URL.Path, "/oauth/") {
		respond(w, 429, map[string]string{"message": "Too Many Attempts."})
	} else {
		respond(w, 429, map[string]string{"message": "تعداد تلاش‌ها بیش از حد مجاز است؛ کمی صبر کنید و دوباره تلاش کنید."})
	}
}
