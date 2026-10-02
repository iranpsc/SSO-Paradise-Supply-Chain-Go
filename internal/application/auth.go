package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
)

type Auth struct {
	SessionIdle                               bool
	RateLimit                                 int
	RememberTTL                               time.Duration
	LegacyCookies                             LegacyCookieDecoder
	LegacySessionCookie, LegacyRememberCookie string
	AppName, Locale                           string
	SigningKey                                []byte
	OAuth                                     *OAuth
	Accounts                                  Accounts
	Sessions                                  Sessions
	Actions                                   Actions
	Passwords                                 Passwords
	Safety                                    PasswordSafety
	Mailer                                    Mailer
	PublicURL                                 string
	Now                                       func() time.Time
	SessionTTL                                time.Duration
}

func Digest(token string) string { h := sha256.Sum256([]byte(token)); return hex.EncodeToString(h[:]) }
func newToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func (a *Auth) Register(ctx context.Context, in Registration) (domain.User, error) {
	in.Name, in.Email = strings.TrimSpace(in.Name), strings.ToLower(strings.TrimSpace(in.Email))
	v := domain.ValidateIdentity(in.Name, in.Email, 50)
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	if !domain.ValidUsername(in.Username) {
		v["username"] = "نام کاربری باید ۳ تا ۳۰ نویسه، با حرف انگلیسی آغاز و فقط شامل حروف انگلیسی، عدد و زیرخط باشد."
	}
	for k, msg := range domain.ValidatePassword(in.Password, in.Confirmation) {
		v[k] = msg
	}
	if in.Referral != "" {
		ok, err := a.Accounts.CodeExists(ctx, in.Referral)
		if err != nil {
			return domain.User{}, err
		}
		if !ok {
			v["referral"] = "کد معرف معتبر نیست."
		}
	}
	if in.ClientID != nil || in.RedirectURI != "" {
		if a.OAuth == nil || in.ClientID == nil {
			v["client_id"] = "کلاینت معتبر الزامی است."
		} else {
			client, err := a.OAuth.Store.OAuthClient(ctx, *in.ClientID)
			if err != nil && !errors.Is(err, domain.ErrNotFound) {
				return domain.User{}, err
			}
			if errors.Is(err, domain.ErrNotFound) || client.Revoked {
				v["client_id"] = "کلاینت معتبر نیست."
			}
			if in.RedirectURI != "" {
				allowed := false
				for _, uri := range client.Redirects {
					allowed = allowed || uri == in.RedirectURI
				}
				if !allowed {
					v["redirect_uri"] = "آدرس بازگشت متعلق به کلاینت نیست."
				}
			}
		}
	}
	if in.BackURL != "" {
		u, e := url.Parse(in.BackURL)
		if e != nil || len(in.BackURL) > 2048 || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
			v["back_url"] = "آدرس بازگشت معتبر نیست."
		}
	}
	if len(v) > 0 {
		return domain.User{}, v
	}
	hash, err := a.Passwords.Hash(in.Password)
	if err != nil {
		return domain.User{}, err
	}
	user := domain.User{Name: in.Name, Username: in.Username, Email: in.Email, PasswordHash: hash, Referral: in.Referral, CreatedAt: a.Now()}
	if in.BackURL != "" {
		registrations, ok := a.Accounts.(RegistrationCallbacks)
		if !ok {
			return domain.User{}, errors.New("registration callbacks not configured")
		}
		return registrations.CreateRegistration(ctx, user, in.BackURL, a.Now().Add(time.Hour))
	}
	return a.Accounts.Create(ctx, user)
}

func (a *Auth) Login(ctx context.Context, email, password string) (domain.User, string, error) {
	return a.LoginRemember(ctx, email, password, false)
}
func (a *Auth) LoginRemember(ctx context.Context, email, password string, remember bool) (domain.User, string, error) {
	identifier := strings.ToLower(strings.TrimSpace(email))
	var u domain.User
	var err error
	if credentials, ok := a.Accounts.(interface {
		CredentialsByLogin(context.Context, string) (domain.User, error)
	}); ok {
		u, err = credentials.CredentialsByLogin(ctx, identifier)
	} else {
		u, err = a.Accounts.ByLogin(ctx, identifier)
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, "", err
	}
	// Run bcrypt even for unknown accounts to reduce account enumeration by timing.
	hash := u.PasswordHash
	if errors.Is(err, domain.ErrNotFound) {
		hash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	}
	valid := a.Passwords.Matches(hash, password)
	if err != nil || !valid {
		return domain.User{}, "", domain.ErrCredentials
	}
	ttl := a.SessionTTL
	if remember {
		ttl = a.RememberSessionTTL()
	}
	token, err := a.StartSessionFor(ctx, u.ID, ttl)
	return u, token, err
}

func (a *Auth) StartSession(ctx context.Context, id int64) (string, error) {
	return a.StartSessionFor(ctx, id, a.SessionTTL)
}
func (a *Auth) StartSessionFor(ctx context.Context, id int64, ttl time.Duration) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	if sliding, ok := a.Sessions.(SlidingSessions); ok && a.SessionIdle && ttl == a.SessionTTL {
		return token, sliding.CreateSlidingSession(ctx, id, Digest(token), a.Now().Add(ttl), ttl)
	}
	return token, a.Sessions.CreateSession(ctx, id, Digest(token), a.Now().Add(ttl))
}
func (a *Auth) Authenticate(ctx context.Context, token string) (domain.User, error) {
	if len(token) != 64 {
		return domain.User{}, domain.ErrCredentials
	}
	if sliding, ok := a.Sessions.(SlidingSessions); ok && a.SessionIdle {
		return sliding.SlidingSessionUser(ctx, Digest(token), a.Now())
	}
	return a.Sessions.SessionUser(ctx, Digest(token), a.Now())
}
func (a *Auth) Logout(ctx context.Context, id int64) error { return a.Sessions.RevokeSessions(ctx, id) }

func (a *Auth) SendVerification(ctx context.Context, u domain.User) error {
	if u.EmailVerifiedAt != nil {
		return nil
	}
	if len(a.SigningKey) > 0 {
		link := security.SignedVerificationURL(a.PublicURL, u.ID, u.Email, a.Now().Add(time.Hour), a.SigningKey)
		return a.deliverNotification(ctx, u, "verify", "تأیید نشانی ایمیل و تکمیل ثبت‌نام", link, "این پیوند تا یک ساعت معتبر است.\n\n"+link)
	}
	return a.sendAction(ctx, u, "verify", "تأیید ایمیل تونل زمان", "/email/verify")
}
func (a *Auth) ForgotPassword(ctx context.Context, email string) error {
	u, err := a.Accounts.ByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return a.sendAction(ctx, u, "reset", "بازنشانی رمز عبور حساب کاربری", "/password/reset")
}
func (a *Auth) sendAction(ctx context.Context, u domain.User, kind, subject, path string) error {
	token, err := newToken()
	if err != nil {
		return err
	}
	if err := a.Actions.SaveAction(ctx, u.ID, kind, Digest(token), u.Email, a.Now().Add(time.Hour)); err != nil {
		return err
	}
	link := a.PublicURL + path + "?" + url.Values{"token": {token}, "email": {u.Email}}.Encode()
	return a.deliverNotification(ctx, u, kind, subject, link, "این پیوند تا یک ساعت معتبر است.\n\n"+link)
}
func (a *Auth) Verify(ctx context.Context, id int64, token string) error {
	return a.Actions.VerifyEmail(ctx, id, Digest(token), a.Now())
}
func (a *Auth) Reset(ctx context.Context, email, token, password, confirmation string) error {
	if v := domain.ValidateResetPassword(password, confirmation); len(v) > 0 {
		return v
	}
	hash, err := a.Passwords.Hash(password)
	if err != nil {
		return err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	err = a.Actions.ResetPassword(ctx, email, Digest(token), hash, a.Now())
	if errors.Is(err, domain.ErrToken) {
		if legacy, ok := a.Actions.(LegacyPasswordResetActions); ok {
			return legacy.ResetLegacyPassword(ctx, email, token, hash, a.Now())
		}
	}
	return err
}
func (a *Auth) UpdateAccount(ctx context.Context, u domain.User, name, email string) (domain.User, error) {
	if u.EmailVerifiedAt == nil {
		return domain.User{}, domain.ErrUnverified
	}
	name, email = strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(email))
	if v := domain.ValidateIdentity(name, email, 255); len(v) > 0 {
		return domain.User{}, v
	}
	return a.Accounts.UpdateIdentity(ctx, u.ID, name, email)
}
func (a *Auth) ChangePassword(ctx context.Context, u domain.User, current, password, confirmation, sessionToken string) error {
	if u.EmailVerifiedAt == nil {
		return domain.ErrUnverified
	}
	if u.PasswordHash != "" && !a.Passwords.Matches(u.PasswordHash, current) {
		return domain.Validation{"current_password": "رمز فعلی صحیح نیست."}
	}
	if v := domain.ValidateChangedPassword(password, confirmation); len(v) > 0 {
		return v
	}
	if a.Safety == nil {
		return errors.New("password safety checker is not configured")
	}
	compromised, err := a.Safety.Compromised(ctx, password)
	if err != nil {
		return err
	}
	if compromised {
		return domain.Validation{"password": "این رمز در نشت اطلاعات دیده شده است؛ رمز دیگری انتخاب کنید."}
	}
	hash, err := a.Passwords.Hash(password)
	if err != nil {
		return err
	}
	return a.Accounts.ChangePassword(ctx, u.ID, hash, Digest(sessionToken))
}

func (a *Auth) VerificationRedirect(ctx context.Context, id int64) (string, error) {
	callbacks, ok := a.Accounts.(RegistrationCallbacks)
	if !ok {
		return a.PublicURL + "/home", nil
	}
	raw, err := callbacks.PullRegistrationCallback(ctx, id, a.Now())
	if errors.Is(err, domain.ErrNotFound) {
		return a.PublicURL + "/home", nil
	}
	if err != nil {
		return "", err
	}
	target, err := url.Parse(raw)
	if err != nil || target.Scheme != "https" || target.Host != "metarang.com" || target.User != nil {
		return a.PublicURL + "/home", nil
	}
	query := target.Query()
	query.Set("verified", "1")
	target.RawQuery = query.Encode()
	return target.String(), nil
}

func (a *Auth) deliverNotification(ctx context.Context, u domain.User, kind, subject, link, text string) error {
	if rich, ok := a.Mailer.(NotificationMailer); ok {
		return rich.SendNotification(ctx, u.Email, subject, Notification{Name: u.Name, Link: link, Kind: kind, AppName: a.AppName, Locale: a.Locale, Minutes: 60, Year: a.Now().Year()})
	}
	return a.Mailer.Send(ctx, u.Email, subject, text)
}

func (a *Auth) RememberSessionTTL() time.Duration {
	if a.RememberTTL > 0 {
		return a.RememberTTL
	}
	return 30 * 24 * time.Hour
}
