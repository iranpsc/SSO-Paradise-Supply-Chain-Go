package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

const cookieName = "paradise_session"

type Server struct {
	csrfKey        []byte
	auth           *application.Auth
	web3           *application.Web3
	profiles       *application.Profile
	origin         string
	secure         bool
	log            *slog.Logger
	limits         *limiter
	sharedLimits   application.RateLimiter
	trustedProxies []netip.Prefix
}

func New(auth *application.Auth, web3 *application.Web3, profiles *application.Profile, origin string, secure bool, logger *slog.Logger, trustedProxies ...netip.Prefix) http.Handler {
	s := &Server{auth: auth, web3: web3, profiles: profiles, origin: origin, secure: secure, log: logger, limits: &limiter{entries: make(map[string]rateState)}}
	s.csrfKey = append([]byte(nil), auth.SigningKey...)
	if len(s.csrfKey) == 0 {
		s.csrfKey = make([]byte, 32)
		if _, err := rand.Read(s.csrfKey); err != nil {
			panic(err)
		}
	}
	s.trustedProxies = trustedProxies
	if shared, ok := auth.Accounts.(application.RateLimiter); ok {
		s.sharedLimits = shared
	}
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { respond(w, 200, map[string]string{"status": "ok"}) })
	m.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if ready, ok := auth.Accounts.(interface{ Ready(context.Context) error }); ok {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := ready.Ready(ctx); err != nil {
				respond(w, 503, map[string]string{"status": "unavailable"})
				return
			}
		}
		respond(w, 200, map[string]string{"status": "ready"})
	})
	m.HandleFunc("POST /api/register", s.register)
	m.HandleFunc("POST /api/login", s.login)
	m.HandleFunc("POST /api/logout", s.protected(false, s.logout))
	m.HandleFunc("GET /api/account", s.protected(false, s.account))
	m.HandleFunc("GET /api/user", s.protected(false, func(w http.ResponseWriter, r *http.Request, u domain.User) { respond(w, 200, laravelUser(u)) }))
	m.HandleFunc("POST /api/me", s.protected(false, s.me))
	m.HandleFunc("GET /api/users/{user}/avatar", s.publicAvatar)
	m.HandleFunc("PUT /api/account", s.protected(true, s.updateAccount))
	m.HandleFunc("PUT /api/change-password", s.protected(true, s.changePassword))
	m.HandleFunc("POST /api/email/verification-notification", s.protected(false, s.resend))
	m.HandleFunc("POST /api/email/verify", s.protected(false, s.verify))
	m.HandleFunc("POST /api/password/email", s.forgot)
	m.HandleFunc("POST /api/password/reset", s.reset)
	m.HandleFunc("GET /api/personal-info", s.protected(true, s.personalInfo))
	m.HandleFunc("PUT /api/personal-info", s.protected(true, s.updatePersonalInfo))
	m.HandleFunc("GET /api/personal-info/documents/{kind}", s.protected(true, s.document))
	m.HandleFunc("GET /api/account/avatar", s.protected(false, s.ownAvatar))
	m.HandleFunc("PUT /api/account/avatar", s.protected(true, s.updateAvatar))
	m.HandleFunc("GET /api/web3/nonce", s.web3Nonce)
	m.HandleFunc("POST /api/web3/verify", s.web3Verify)
	m.HandleFunc("GET /api/web3/link/nonce", s.protected(true, s.web3LinkNonce))
	m.HandleFunc("POST /api/web3/link", s.protected(true, s.web3Link))
	m.HandleFunc("GET /email/verify/{id}/{hash}", func(w http.ResponseWriter, r *http.Request) {
		if !s.legacyGuard(w, r, false) {
			return
		}
		s.protected(false, s.verifySignedEmail)(w, r)
	})
	m.HandleFunc("GET /oauth/authorize", s.oauthConsent)
	m.HandleFunc("POST /oauth/authorize", s.passportWeb(s.oauthAuthorize))
	m.HandleFunc("DELETE /oauth/authorize", s.passportWeb(s.oauthAuthorize))
	m.HandleFunc("POST /oauth/token/refresh", s.passportWeb(s.refreshBrowserToken))
	m.HandleFunc("GET /api/oauth/authorize", s.oauthConsent)
	m.HandleFunc("GET /api/oauth/consent", s.protected(false, s.pendingConsent))
	m.HandleFunc("POST /api/oauth/consent", s.protected(false, s.decideConsent))
	m.HandleFunc("POST /api/oauth/authorize", s.protected(false, s.oauthAuthorize))
	m.HandleFunc("POST /oauth/token", s.oauthToken)
	m.HandleFunc("POST /oauth/revoke", s.oauthRevoke)
	m.HandleFunc("POST /api/password/confirm", s.protected(false, s.confirmPassword))
	s.registerLegacyRoutes(m)
	return s.middleware(localizedRouting(m))
}

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func validationMessages(fields domain.Validation) map[string][]string {
	result := make(map[string][]string, len(fields))
	for field, message := range fields {
		result[field] = []string{message}
	}
	return result
}
func (s *Server) fail(w http.ResponseWriter, err error) {
	var v domain.Validation
	switch {
	case errors.As(err, &v):
		respond(w, 422, map[string]any{"message": "اطلاعات واردشده معتبر نیست.", "errors": validationMessages(v)})
	case errors.Is(err, domain.ErrConflict):
		respond(w, 422, map[string]any{"message": "این ایمیل قبلاً ثبت شده است.", "errors": map[string][]string{"email": {"این ایمیل قبلاً ثبت شده است."}}})
	case errors.Is(err, domain.ErrUsernameConflict):
		respond(w, 422, map[string]any{"message": "این نام کاربری قبلاً انتخاب شده است.", "errors": map[string][]string{"username": {"این نام کاربری قبلاً انتخاب شده است."}}})
	case errors.Is(err, domain.ErrCredentials):
		respond(w, 401, map[string]string{"message": "اطلاعات ورود معتبر نیست."})
	case errors.Is(err, domain.ErrUnverified):
		respond(w, 403, map[string]string{"message": "ابتدا ایمیل خود را تأیید کنید.", "code": "email_unverified"})
	case errors.Is(err, domain.ErrToken):
		respond(w, 422, map[string]string{"message": "پیوند نامعتبر یا منقضی شده است."})
	case errors.Is(err, domain.ErrNotFound):
		respond(w, 404, map[string]string{"message": "یافت نشد."})
	default:
		s.log.Error("request failed", "error", err)
		respond(w, 500, map[string]string{"message": "خطای داخلی؛ دوباره تلاش کنید."})
	}
}
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		respond(w, 415, map[string]string{"message": "ساختار دادهٔ ارسال‌شده پشتیبانی نمی‌شود."})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		respond(w, 400, map[string]string{"message": "دادهٔ درخواست معتبر نیست."})
		return false
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		respond(w, 400, map[string]string{"message": "دادهٔ درخواست معتبر نیست."})
		return false
	}
	return true
}
func token(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if c, e := r.Cookie(cookieName); e == nil {
		return c.Value
	}
	return ""
}
func (s *Server) setSession(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: value, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: int(s.auth.SessionTTL.Seconds())})
}
func (s *Server) protected(verified bool, next func(http.ResponseWriter, *http.Request, domain.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := s.authenticate(r, w)
		if errors.Is(err, domain.ErrCredentials) && s.auth.OAuth != nil && r.Header.Get("Authorization") != "" && (r.URL.Path == "/api/me" || r.URL.Path == "/api/user" || (r.URL.Path == "/api/account" && r.Method == "GET") || r.URL.Path == "/api/logout") {
			u, err = s.auth.OAuth.AccessUser(r.Context(), token(r), "")
		}
		if err != nil {
			if errors.Is(err, domain.ErrCredentials) && legacyAPI(r.URL.Path) {
				respond(w, 401, map[string]string{"message": "Unauthenticated."})
			} else {
				s.fail(w, err)
			}
			return
		}
		if verified && u.EmailVerifiedAt == nil {
			s.fail(w, domain.ErrUnverified)
			return
		}
		next(w, r, u)
	}
}
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in application.Registration
	if !decode(w, r, &in) {
		return
	}
	u, err := s.auth.Register(r.Context(), in)
	if err != nil {
		s.fail(w, err)
		return
	}
	t, err := s.auth.StartSession(r.Context(), u.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.setSession(w, t)
	sent := s.auth.SendVerification(r.Context(), u) == nil
	if !sent {
		s.log.Error("verification delivery failed", "user_id", u.ID)
	}
	respond(w, 201, map[string]any{"data": u, "verification_sent": sent})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	identifier, password, remember, ok := s.loginInput(w, r)
	if !ok {
		return
	}
	peer, _, peerErr := net.SplitHostPort(r.RemoteAddr)
	if peerErr != nil {
		peer = r.RemoteAddr
	}
	peer = s.clientIP(peer, r.Header.Get("X-Forwarded-For"))
	failureKey := strings.ToLower(strings.TrimSpace(identifier)) + "\x00" + peer
	failures, hasFailureLimiter := s.auth.Accounts.(application.LoginAttemptLimiter)
	if hasFailureLimiter {
		retry, e := failures.LoginRetry(r.Context(), failureKey, time.Now())
		if e != nil {
			s.fail(w, e)
			return
		}
		if retry > 0 {
			seconds := int((retry + time.Second - 1) / time.Second)
			message := fmt.Sprintf("تعداد تلاش های ناموفق زیاد بود . لطفا بعد از %d ثانیه ی دیگر تلاش کنید .", seconds)
			respond(w, 429, map[string]any{"message": message, "errors": map[string][]string{"email": {message}}})
			return
		}
	}
	loggedIn, t, err := s.auth.LoginRemember(r.Context(), identifier, password, remember)
	if err != nil {
		if hasFailureLimiter && errors.Is(err, domain.ErrCredentials) {
			if e := failures.FailedLogin(r.Context(), failureKey, time.Now()); e != nil {
				s.fail(w, e)
				return
			}
		}
		if errors.Is(err, domain.ErrCredentials) {
			respond(w, http.StatusUnauthorized, map[string]string{"message": "Invalid credentials"})
		} else {
			s.fail(w, err)
		}
		return
	}
	if hasFailureLimiter {
		if e := failures.ClearFailedLogins(r.Context(), failureKey); e != nil {
			s.fail(w, e)
			return
		}
	}
	if remember {
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: t, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: int(s.auth.RememberSessionTTL().Seconds())})
	} else {
		s.setSession(w, t)
	}
	apiToken := t
	if s.auth.OAuth != nil {
		ttl := s.auth.OAuth.PersonalTTL
		apiToken, err = s.auth.OAuth.PersonalAccess(r.Context(), loggedIn.ID, ttl)
		if err != nil {
			s.fail(w, err)
			return
		}
	}
	respond(w, 200, map[string]any{"message": "Login successful", "token": apiToken})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request, u domain.User) {
	if err := s.auth.Logout(r.Context(), u.ID); err != nil {
		s.fail(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode})
	respond(w, 200, map[string]string{"message": "Logged out successfully"})
}
func (s *Server) account(w http.ResponseWriter, r *http.Request, u domain.User) {
	respond(w, 200, map[string]any{"data": u})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request, u domain.User) {
	if s.profiles != nil {
		if p, err := s.profiles.Store.PublicProfile(r.Context(), u.ID); err == nil {
			if p.AvatarAbsolute {
				p.Avatar = s.origin + p.Avatar
			}
			respond(w, 200, map[string]any{"data": p})
			return
		} else {
			s.fail(w, err)
			return
		}
	}
	respond(w, 200, map[string]any{"data": map[string]any{"id": u.ID, "name": u.Name, "code": u.Code, "avatar": ""}})
}
func (s *Server) publicUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("user"), 10, 64)
	if err != nil || id < 1 {
		s.modelNotFound(w, r.PathValue("user"))
		return
	}
	if s.profiles != nil {
		if p, err := s.profiles.Store.PublicProfile(r.Context(), id); err == nil {
			if p.AvatarAbsolute {
				p.Avatar = s.origin + p.Avatar
			}
			respond(w, 200, map[string]any{"data": p})
			return
		} else if errors.Is(err, domain.ErrNotFound) {
			s.modelNotFound(w, r.PathValue("user"))
			return
		} else {
			s.fail(w, err)
			return
		}
	}
	u, err := s.auth.PublicUser(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.me(w, r, u)
}
func (s *Server) updateAccount(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	updated, err := s.auth.UpdateAccount(r.Context(), u, in.Name, in.Email)
	if err != nil {
		s.fail(w, err)
		return
	}
	sent := true
	if updated.EmailVerifiedAt == nil {
		sent = s.auth.SendVerification(r.Context(), updated) == nil
	}
	respond(w, 200, map[string]any{"data": updated, "verification_sent": sent})
}
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in struct {
		Current      string `json:"current_password"`
		Password     string `json:"password"`
		Confirmation string `json:"password_confirmation"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.auth.ChangePassword(r.Context(), u, in.Current, in.Password, in.Confirmation, token(r)); err != nil {
		s.fail(w, err)
		return
	}
	respond(w, 200, map[string]string{"message": "رمز عبور به‌روزرسانی شد."})
}
func (s *Server) resend(w http.ResponseWriter, r *http.Request, u domain.User) {
	if err := s.auth.SendVerification(r.Context(), u); err != nil {
		s.fail(w, err)
		return
	}
	respond(w, 200, map[string]string{"message": "پیوند تأیید ارسال شد."})
}
func (s *Server) verify(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.auth.Verify(r.Context(), u.ID, in.Token); err != nil {
		s.fail(w, err)
		return
	}
	target, err := s.auth.VerificationRedirect(r.Context(), u.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	respond(w, 200, map[string]string{"message": "ایمیل تأیید شد.", "redirect": target})
}
func (s *Server) forgot(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.auth.ForgotPassword(r.Context(), in.Email); err != nil {
		s.log.Error("reset delivery failed", "error", err)
	}
	respond(w, 200, map[string]string{"message": "اگر حسابی با این ایمیل وجود داشته باشد، پیوند بازیابی ارسال می‌شود."})
}
func (s *Server) reset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email        string `json:"email"`
		Token        string `json:"token"`
		Password     string `json:"password"`
		Confirmation string `json:"password_confirmation"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.auth.Reset(r.Context(), in.Email, in.Token, in.Password, in.Confirmation); err != nil {
		s.fail(w, err)
		return
	}
	u, session, err := s.auth.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.setSession(w, session)
	respond(w, 200, map[string]any{"message": "رمز عبور با موفقیت بازیابی شد.", "data": u})
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		securityHeaders(w, r, s.secure)
		if s.cors(w, r) {
			return
		}
		if r.Method == "POST" && isLegacyWeb(r.URL.Path) {
			in, ok := laravelInput(w, r)
			if !ok {
				return
			}
			method := strings.ToUpper(inputString(in, "_method"))
			if method == "" {
				method = strings.ToUpper(r.Header.Get("X-HTTP-Method-Override"))
			}
			if method == "PUT" || method == "PATCH" || method == "DELETE" {
				r.Method = method
			}
			if r.MultipartForm == nil {
				replaceJSONBody(r, in)
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == "GET" && r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		ip = s.clientIP(ip, r.Header.Get("X-Forwarded-For"))
		budgetPeer := ip
		identity, identityErr := s.auth.Authenticate(r.Context(), token(r))
		if errors.Is(identityErr, domain.ErrCredentials) && s.auth.OAuth != nil && r.Header.Get("Authorization") != "" {
			identity, identityErr = s.auth.OAuth.AccessUser(r.Context(), token(r), "")
		}
		if identityErr == nil {
			budgetPeer = "user:" + strconv.FormatInt(identity.ID, 10)
		}
		decision := rateDecision{}
		if isLegacyWeb(r.URL.Path) || strings.HasPrefix(r.URL.Path, "/storage/") {
			decision = rateDecision{remaining: s.auth.RateLimit}
		} else if s.sharedLimits != nil {
			shared, err := s.sharedLimits.CheckRate(r.Context(), budgetPeer, time.Now())
			if err != nil {
				s.fail(w, err)
				return
			}
			decision = rateDecision{remaining: shared.Remaining, retry: shared.Retry}
		} else {
			decision = s.limits.check(budgetPeer, time.Now())
		}
		limit := s.auth.RateLimit
		if limit <= 0 {
			limit = 10
		}
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(decision.remaining))
		if decision.retry > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int((decision.retry+time.Second-1)/time.Second)))
			rateFailure(w, r)
			return
		}
		// Laravel gives Web3 its own 10/minute budget. API's global budget is
		// 60/minute in runtime, so normal account flows do not exhaust it.
		web3Path := strings.HasPrefix(r.URL.Path, "/api/web3/") || strings.HasPrefix(r.URL.Path, "/web3/")
		if web3Path || r.URL.Path == "/api/email/verification-notification" || r.URL.Path == "/email/verification-notification" || strings.HasPrefix(r.URL.Path, "/email/verify/") {
			if named, ok := s.auth.Accounts.(application.NamedRateLimiter); ok {
				peer := budgetPeer
				budget := 10
				namespace := "web3:"
				if !web3Path {
					budget = 6
					namespace = "verify:"
				}
				d, e := named.CheckBudget(r.Context(), namespace+peer, budget, time.Now())
				if e != nil {
					s.fail(w, e)
					return
				}
				if d.Retry > 0 {
					w.Header().Set("Retry-After", strconv.Itoa(int((d.Retry+time.Second-1)/time.Second)))
					rateFailure(w, r)
					return
				}
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(budget))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(d.Remaining))
			}
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			origin := r.Header.Get("Origin")
			crossOriginAPI := strings.HasPrefix(r.URL.Path, "/api/") && allowedCORSOrigin(origin) && r.Header.Get("Authorization") != "" && !s.hasBrowserCookie(r) && r.Method == http.MethodPost
			if !crossOriginAPI && ((origin != "" && origin != s.origin) || (r.Header.Get("Sec-Fetch-Site") == "cross-site")) {
				respond(w, 403, map[string]string{"message": "درخواست از مبدأ مجاز نیست."})
				return
			}
			// Browser cookie requests require an explicit same-origin check. CLI bearer requests do not.
			if !isLegacyWeb(r.URL.Path) && s.hasBrowserCookie(r) && r.Header.Get("Authorization") == "" && origin != s.origin {
				respond(w, 403, map[string]string{"message": "مبدأ درخواست مشخص نیست؛ صفحه را تازه‌سازی کنید و دوباره تلاش کنید."})
				return
			}
		}
		if s.auth.SessionIdle && identityErr == nil && r.Header.Get("Authorization") == "" {
			if cookie, err := r.Cookie(cookieName); err == nil {
				if deadlines, ok := s.auth.Sessions.(application.SessionDeadline); ok {
					now := s.auth.Now()
					_, until, err := deadlines.SessionLookup(r.Context(), application.Digest(cookie.Value), now)
					if err != nil {
						s.fail(w, err)
						return
					}
					// Refresh the browser deadline too; remember cookies retain their
					// fixed database expiry instead of gaining another full lifetime.
					http.SetCookie(w, &http.Cookie{Name: cookieName, Value: cookie.Value, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: int((until.Sub(now) + time.Second - 1) / time.Second)})
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
