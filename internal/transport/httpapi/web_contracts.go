package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

type contractResponse struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (c *contractResponse) Header() http.Header { return c.header }
func (c *contractResponse) WriteHeader(n int) {
	if c.status == 0 {
		c.status = n
	}
}
func (c *contractResponse) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = 200
	}
	return c.body.Write(b)
}
func (c *contractResponse) flush(w http.ResponseWriter) {
	for k, v := range c.header {
		w.Header()[k] = v
	}
	if c.status == 0 {
		c.status = 200
	}
	w.WriteHeader(c.status)
	_, _ = w.Write(c.body.Bytes())
}

func (s *Server) registerLegacyRoutes(m *http.ServeMux) {
	s.registerDeviceRoutes(m)
	m.HandleFunc("GET /sanctum/csrf-cookie", s.csrfCookie)
	m.HandleFunc("GET /api/legacy/flash", s.readFlash)
	m.HandleFunc("GET /storage/{media}/{filename}", s.legacyAvatar)
	m.HandleFunc("GET /web3/nonce", s.legacyWeb3(s.web3Nonce, false, false))
	m.HandleFunc("POST /web3/verify", s.legacyWeb3(s.web3Verify, false, false))
	m.HandleFunc("GET /web3/link/nonce", s.legacyWeb3(s.protected(true, s.web3LinkNonce), true, true))
	m.HandleFunc("POST /web3/link", s.legacyWeb3(s.protected(true, s.web3Link), true, true))
	for _, method := range []string{"PUT", "PATCH"} {
		m.HandleFunc(method+" /account", s.legacyAction(s.protected(true, s.updateAccount), "/account", true, true))
		m.HandleFunc(method+" /personal-info", s.legacyAction(s.protected(true, s.updatePersonalInfo), "/personal-info", true, true))
	}
	m.HandleFunc("PUT /change-password", s.legacyAction(s.protected(true, s.changePassword), "/change-password", true, true))
	m.HandleFunc("POST /login", s.legacyAction(s.login, "/home", false, false))
	m.HandleFunc("POST /register", s.legacyAction(s.legacyRegister, "/home", false, false))
	m.HandleFunc("POST /logout", s.legacyAction(s.legacyLogout, "/", false, false))
	m.HandleFunc("POST /password/email", s.legacyAction(s.legacyForgot, "/password/reset", false, false))
	m.HandleFunc("POST /password/reset", s.legacyAction(s.reset, "/home", false, false))
	m.HandleFunc("POST /password/confirm", s.legacyAction(s.protected(false, s.confirmPassword), "/home", true, false))
	m.HandleFunc("POST /email/verification-notification", s.legacyResend)
	// Page rendering belongs to the separate Next application. These redirects
	// are for callers accessing the API server directly.
	for _, page := range []string{"/home", "/account", "/account/edit", "/personal-info", "/personal-info/edit", "/change-password", "/password/confirm", "/email/verify", "/login", "/register", "/password/reset"} {
		path := page
		m.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
			protected := path != "/login" && path != "/register" && path != "/password/reset"
			verified := path == "/home" || strings.HasPrefix(path, "/account") || strings.HasPrefix(path, "/personal-info") || path == "/change-password"
			if protected && !s.legacyGuard(w, r, verified) {
				return
			}
			if !protected && path != "/password/reset" {
				if _, err := s.authenticate(r, w); err == nil {
					http.Redirect(w, r, s.origin+"/home", 302)
					return
				}
			}
			http.Redirect(w, r, s.origin+r.URL.RequestURI(), 302)
		})
	}
	m.HandleFunc("GET /password/reset/{token}", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, s.origin+r.URL.RequestURI(), 302) })
	m.HandleFunc("GET /up", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	m.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, s.origin+"/home", 302) })
}

func (s *Server) passportWeb(next func(http.ResponseWriter, *http.Request, domain.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in, ok := laravelInput(w, r)
		if !ok || !s.legacyCSRF(w, r, in) || !s.legacyGuard(w, r, false) {
			return
		}
		if strings.Contains(r.Header.Get("Content-Type"), "json") {
			replaceJSONBody(r, in)
		}
		s.protected(false, next)(w, r)
	}
}

func (s *Server) legacyAvatar(w http.ResponseWriter, r *http.Request) {
	if s.profiles == nil {
		s.fail(w, domain.ErrNotFound)
		return
	}
	reader, ok := s.profiles.Store.(interface {
		LegacyPublicMedia(context.Context, string) (domain.Media, error)
	})
	if !ok {
		s.fail(w, domain.ErrNotFound)
		return
	}
	media, err := reader.LegacyPublicMedia(r.Context(), r.URL.EscapedPath())
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", media.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(media.Data)
}

func (s *Server) legacyGuard(w http.ResponseWriter, r *http.Request, verified bool) bool {
	u, err := s.authenticate(r, w)
	if err != nil {
		if expectsJSON(r) {
			respond(w, 401, map[string]string{"message": "Unauthenticated."})
		} else {
			http.Redirect(w, r, s.origin+"/login", 302)
		}
		return false
	}
	if verified && u.EmailVerifiedAt == nil {
		if expectsJSON(r) {
			respond(w, 403, map[string]string{"message": "Your email address is not verified."})
		} else {
			http.Redirect(w, r, s.origin+"/email/verify", 302)
		}
		return false
	}
	return true
}

func (s *Server) csrfCookie(w http.ResponseWriter, r *http.Request) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		s.fail(w, err)
		return
	}
	token := hex.EncodeToString(random)
	mac := hmac.New(sha256.New, s.csrfKey)
	mac.Write([]byte(token))
	signed := token + "." + hex.EncodeToString(mac.Sum(nil))
	http.SetCookie(w, &http.Cookie{Name: "paradise_csrf", Value: signed, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: 7200})
	http.SetCookie(w, &http.Cookie{Name: "XSRF-TOKEN", Value: signed, Path: "/", Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: 7200})
	w.WriteHeader(204)
}

func (s *Server) legacyCSRF(w http.ResponseWriter, r *http.Request, in map[string]any) bool {
	if r.Method == "GET" || r.Method == "HEAD" {
		return true
	}
	if r.Header.Get("Sec-Fetch-Site") == "same-origin" && (r.Header.Get("Origin") == "" || r.Header.Get("Origin") == s.origin) {
		return true
	}
	supplied := r.Header.Get("X-CSRF-TOKEN")
	if supplied == "" {
		supplied = r.Header.Get("X-XSRF-TOKEN")
	}
	if supplied == "" {
		supplied = inputString(in, "_token")
	}
	supplied, _ = url.QueryUnescape(supplied)
	cookie, err := r.Cookie("paradise_csrf")
	parts := strings.Split(supplied, ".")
	if err == nil && hmac.Equal([]byte(cookie.Value), []byte(supplied)) && len(parts) == 2 {
		mac := hmac.New(sha256.New, s.csrfKey)
		mac.Write([]byte(parts[0]))
		sig, err := hex.DecodeString(parts[1])
		if err == nil && hmac.Equal(sig, mac.Sum(nil)) {
			return true
		}
	}
	// Preserve an imported Laravel session's CSRF token during the cookie bridge.
	if supplied != "" && s.auth.LegacyCookies != nil {
		if encrypted := r.Header.Get("X-XSRF-TOKEN"); encrypted != "" {
			if plain, err := s.auth.LegacyCookies.DecodeCookie("XSRF-TOKEN", encrypted); err == nil {
				supplied = plain
			}
		}
		if _, err := s.authenticate(r, w); err == nil {
			if attributes, ok := s.auth.Sessions.(application.SessionAttributes); ok {
				stored, err := attributes.SessionAttribute(r.Context(), application.Digest(token(r)), "csrf_token", s.auth.Now())
				if err == nil && hmac.Equal([]byte(supplied), []byte(stored)) {
					return true
				}
			}
		}
	}
	respond(w, 419, map[string]string{"message": "CSRF token mismatch."})
	return false
}

func (s *Server) legacyAction(next http.HandlerFunc, target string, authenticated, verified bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destination := target
		in, ok := laravelInput(w, r)
		if !ok {
			return
		}
		if !s.legacyCSRF(w, r, in) {
			return
		}
		if authenticated && !s.legacyGuard(w, r, verified) {
			return
		}
		var avatar []byte
		var avatarName string
		var avatarUser domain.User
		if r.URL.Path == "/account" && r.MultipartForm != nil && s.profiles != nil {
			file, header, err := r.FormFile("avatar")
			if err == nil {
				avatar, err = io.ReadAll(io.LimitReader(file, (1<<20)+1))
				_ = file.Close()
				avatarName = header.Filename
				if err != nil || len(avatar) == 0 || len(avatar) > 1<<20 {
					laravelValidation(w, map[string][]string{"avatar": {"حجم فایل نباید بیشتر از ۱ مگابایت باشد."}}, "avatar")
					return
				}
				if message := domain.ValidateImage(avatarName, avatar); message != "" {
					laravelValidation(w, map[string][]string{"avatar": {message}}, "avatar")
					return
				}
				avatarUser, err = s.authenticate(r, w)
				if err != nil {
					s.fail(w, err)
					return
				}
			}
		}
		// Keep multipart documents for the profile handler; ordinary form inputs
		// are converted to its existing JSON transport without unknown keys.
		if r.MultipartForm == nil || r.URL.Path != "/personal-info" {
			replaceJSONBody(r, filterLegacyInput(r.URL.Path, in))
		}
		captured := &contractResponse{header: make(http.Header)}
		if r.URL.Path != "/personal-info" || legacyPersonalRequired(captured, r, in) {
			next(captured, r)
		}
		if captured.status >= 200 && captured.status < 300 {
			if len(avatar) > 0 {
				if err := s.profiles.UpdateAvatar(r.Context(), avatarUser, avatarName, avatar, http.DetectContentType(avatar)); err != nil {
					s.fail(w, err)
					return
				}
			}
			for _, v := range captured.header.Values("Set-Cookie") {
				w.Header().Add("Set-Cookie", v)
			}
			if r.URL.Path == "/account" {
				var body struct{ Data domain.User }
				_ = json.Unmarshal(captured.body.Bytes(), &body)
				if body.Data.EmailVerifiedAt == nil {
					destination = "/email/verify"
				}
			}
			if r.URL.Path == "/login" && expectsJSON(r) {
				w.WriteHeader(204)
				return
			}
			if expectsJSON(r) && (r.URL.Path == "/logout" || r.URL.Path == "/password/confirm") {
				w.WriteHeader(204)
				return
			}
			if r.URL.Path == "/password/email" {
				if expectsJSON(r) {
					captured.flush(w)
					return
				}
				s.flashErrors(w, captured.body.Bytes())
				http.Redirect(w, r, s.legacyBack(r), 302)
				return
			}
			if r.URL.Path == "/password/reset" {
				message := map[string]string{"message": "رمز عبور شما با موفقیت تغییر یافت!"}
				if expectsJSON(r) {
					respond(w, 200, message)
					return
				}
				s.flashErrors(w, mustJSON(message))
			}
			http.Redirect(w, r, s.origin+destination, 302)
			return
		}
		if !expectsJSON(r) && captured.status >= 400 && captured.status < 500 {
			// Preserve Laravel's redirect-on-form-error semantics. Errors are carried
			// in a signed, short-lived flash cookie and consumed by the frontend.
			s.flashErrors(w, captured.body.Bytes())
			http.Redirect(w, r, s.legacyBack(r), 302)
			return
		}
		captured.flush(w)
	}
}

// The original form requires new scans on every submission. The separate
// /api/personal-info extension may retain already uploaded documents.
func legacyPersonalRequired(w http.ResponseWriter, r *http.Request, in map[string]any) bool {
	order := strings.Fields("is_company first_name last_name mobile telephone national_code address company_name company_address company_registration_number company_national_number company_tax_number company_executive_name melli_card_scan certificate_scan bank_card_scan")
	labels := strings.Split("شرکت است؟|نام|نام خانوادگی|تلفن همراه|تلفن|کد ملی|نشانی|نام شرکت|آدرس شرکت|شماره ثبت شرکت|شماره ملی شرکت|شماره مالیات شرکت|نام مدیر عامل شرکت|اسکن کارت ملی|اسکن شناسنامه|اسکن کارت بانکی", "|")
	company := in["is_company"] == true || in["is_company"] == "1" || in["is_company"] == "true" || in["is_company"] == float64(1)
	fields := map[string][]string{}
	for i, key := range order {
		if strings.HasPrefix(key, "company_") && !company {
			continue
		}
		missing := in[key] == nil || in[key] == ""
		if documentKinds[key] {
			missing = r.MultipartForm == nil || len(r.MultipartForm.File[key]) == 0
		}
		if missing {
			fields[key] = []string{"فیلد " + labels[i] + " الزامی است"}
		}
	}
	if len(fields) > 0 {
		laravelValidation(w, fields, order...)
		return false
	}
	return true
}

func (s *Server) legacyForgot(w http.ResponseWriter, r *http.Request) {
	in, ok := laravelInput(w, r)
	if !ok {
		return
	}
	email := inputString(in, "email")
	if email == "" {
		laravelValidation(w, map[string][]string{"email": {"فیلد پست الکترونیکی الزامی است"}}, "email")
		return
	}
	if address, err := mail.ParseAddress(email); err != nil || address.Address != email {
		laravelValidation(w, map[string][]string{"email": {"فرمت پست الکترونیکی معتبر نیست."}}, "email")
		return
	}
	if _, err := s.auth.Accounts.ByEmail(r.Context(), email); errors.Is(err, domain.ErrNotFound) {
		laravelValidation(w, map[string][]string{"email": {"کاربری با این ایمیل یافت نشد."}}, "email")
		return
	} else if err != nil {
		s.fail(w, err)
		return
	}
	if named, ok := s.auth.Accounts.(application.NamedRateLimiter); ok {
		decision, err := named.CheckBudget(r.Context(), "password-reset:"+strings.ToLower(email), 1, s.auth.Now())
		if err != nil {
			s.fail(w, err)
			return
		}
		if decision.Retry > 0 {
			laravelValidation(w, map[string][]string{"email": {"لطفا قبل تلاش مجدد منتظر بمانید."}}, "email")
			return
		}
	}
	if err := s.auth.ForgotPassword(r.Context(), email); err != nil {
		s.fail(w, err)
		return
	}
	respond(w, 200, map[string]string{"message": "لینک تغییر رمز عبور برای شما فرستاده شد!"})
}

func filterLegacyInput(path string, in map[string]any) map[string]any {
	keys := map[string]string{"/login": "login email password remember", "/account": "name email", "/register": "name email username password password_confirmation referral client_id redirect_uri back_url", "/password/email": "email", "/password/reset": "email token password password_confirmation", "/password/confirm": "password", "/change-password": "password password_confirmation current_password", "/personal-info": "is_company first_name last_name mobile telephone national_code address company_name company_address company_registration_number company_national_number company_tax_number company_executive_name"}
	allowed := strings.Fields(keys[path])
	out := map[string]any{}
	for _, key := range allowed {
		if v, ok := in[key]; ok {
			if key == "client_id" {
				if text, ok := v.(string); ok {
					if text == "" {
						v = nil
					} else if number, err := strconv.ParseInt(text, 10, 64); err == nil {
						v = number
					}
				}
			}
			if key == "is_company" {
				if text, ok := v.(string); ok {
					if boolean, valid := parseCompanyBool(text); valid {
						v = boolean
					}
				}
			}
			out[key] = v
		}
	}
	return out
}

func (s *Server) legacyBack(r *http.Request) string {
	if u, err := url.Parse(r.Referer()); err == nil && u.Scheme+"://"+u.Host == s.origin {
		return u.String()
	}
	return s.origin
}

func (s *Server) legacyLogout(w http.ResponseWriter, r *http.Request) {
	u, err := s.authenticate(r, w)
	if errors.Is(err, domain.ErrCredentials) {
		w.WriteHeader(204)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	s.logout(w, r, u)
}

func (s *Server) legacyRegister(w http.ResponseWriter, r *http.Request) {
	in, ok := laravelInput(w, r)
	if !ok {
		return
	}
	if inputString(in, "username") == "" {
		nonce := make([]byte, 8)
		if _, err := rand.Read(nonce); err != nil {
			s.fail(w, err)
			return
		}
		in["username"] = "member_" + hex.EncodeToString(nonce)
	}
	replaceJSONBody(r, in)
	s.register(w, r)
}

func (s *Server) legacyResend(w http.ResponseWriter, r *http.Request) {
	in, ok := laravelInput(w, r)
	if !ok || !s.legacyCSRF(w, r, in) || !s.legacyGuard(w, r, false) {
		return
	}
	u, err := s.authenticate(r, w)
	if err != nil {
		s.fail(w, err)
		return
	}
	if u.EmailVerifiedAt != nil {
		if expectsJSON(r) {
			w.WriteHeader(204)
		} else {
			http.Redirect(w, r, s.origin+"/home", 302)
		}
		return
	}
	if err = s.auth.SendVerification(r.Context(), u); err != nil {
		s.fail(w, err)
		return
	}
	if expectsJSON(r) {
		respond(w, 202, []any{})
	} else {
		http.Redirect(w, r, s.legacyBack(r), 302)
	}
}

func (s *Server) legacyWeb3(next http.HandlerFunc, authenticated, verified bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in, ok := laravelInput(w, r)
		if !ok || !s.legacyCSRF(w, r, in) {
			return
		}
		if authenticated && !s.legacyGuard(w, r, verified) {
			return
		}
		fields := map[string][]string{}
		for _, key := range []string{"address", "signature"} {
			if key == "signature" && r.Method == "GET" {
				continue
			}
			value := inputString(in, key)
			length := 42
			if key == "signature" {
				length = 132
			}
			if value == "" {
				fields[key] = []string{"فیلد " + key + " الزامی است"}
			} else if !validHex(value, length) {
				fields[key] = []string{"فرمت " + key + " معتبر نیست."}
			}
		}
		if len(fields) > 0 {
			laravelValidation(w, fields, "address", "signature")
			return
		}
		if r.Method == "GET" {
			query := r.URL.Query()
			query.Set("address", inputString(in, "address"))
			r.URL.RawQuery = query.Encode()
		} else {
			replaceJSONBody(r, map[string]any{"address": in["address"], "signature": in["signature"]})
		}
		captured := &contractResponse{header: make(http.Header)}
		next(captured, r)
		var body map[string]any
		if json.Unmarshal(captured.body.Bytes(), &body) == nil {
			text, _ := body["message"].(string)
			translations := map[string]string{application.ErrWeb3Nonce.Error(): "Nonce expired or not found. Please try again.", application.ErrWeb3Signature.Error(): "Signature verification failed", application.ErrWeb3Upstream.Error(): "Unable to complete wallet login. Please try again.", application.ErrWalletConnected.Error(): "Wallet already connected to this account.", application.ErrWalletLinked.Error(): "This wallet is already linked to another account.", "ورود با کیف پول با موفقیت انجام شد.": "Authenticated successfully", "کیف پول با موفقیت متصل شد.": "Wallet connected successfully"}
			if replacement, ok := translations[text]; ok {
				body["message"] = replacement
			}
			if !expectsJSON(r) && r.URL.Path != "/web3/link/nonce" && r.URL.Path != "/web3/nonce" {
				for _, v := range captured.header.Values("Set-Cookie") {
					w.Header().Add("Set-Cookie", v)
				}
				if captured.status >= 400 {
					message, _ := body["message"].(string)
					s.flashErrors(w, mustJSON(map[string]any{"errors": map[string][]string{"wallet": {message}}}))
					http.Redirect(w, r, s.legacyBack(r), 302)
					return
				}
				if target, ok := body["redirect"].(string); ok {
					http.Redirect(w, r, target, 302)
					return
				}
			}
			captured.body.Reset()
			captured.body.Write(mustJSON(body))
		}
		captured.flush(w)
	}
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func validHex(v string, length int) bool {
	if len(v) != length || !strings.HasPrefix(v, "0x") {
		return false
	}
	_, err := hex.DecodeString(v[2:])
	return err == nil
}

func (s *Server) flashErrors(w http.ResponseWriter, data []byte) {
	// A bounded flash payload carries form validation across the 302 response.
	if len(data) > 1400 {
		return
	}
	value := hex.EncodeToString(data)
	mac := hmac.New(sha256.New, s.csrfKey)
	mac.Write([]byte(value))
	value += "." + hex.EncodeToString(mac.Sum(nil))
	http.SetCookie(w, &http.Cookie{Name: "paradise_flash", Value: value, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: 120})
}

func (s *Server) readFlash(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("paradise_flash")
	if err != nil {
		respond(w, 200, map[string]any{"errors": map[string]any{}})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "paradise_flash", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode})
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		respond(w, 200, map[string]any{"errors": map[string]any{}})
		return
	}
	mac := hmac.New(sha256.New, s.csrfKey)
	mac.Write([]byte(parts[0]))
	sig, err := hex.DecodeString(parts[1])
	if err != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		respond(w, 200, map[string]any{"errors": map[string]any{}})
		return
	}
	data, err := hex.DecodeString(parts[0])
	if err != nil {
		respond(w, 200, map[string]any{"errors": map[string]any{}})
		return
	}
	var payload map[string]any
	if json.Unmarshal(data, &payload) != nil {
		payload = map[string]any{"errors": map[string]any{}}
	}
	respond(w, 200, payload)
}
