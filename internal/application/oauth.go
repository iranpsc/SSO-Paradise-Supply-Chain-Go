package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

type OAuthClient struct {
	FirstParty bool     `json:"-"`
	GrantTypes []string `json:"-"`
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	SecretHash string   `json:"-"`
	Redirects  []string `json:"redirect_uris"`
	Revoked    bool     `json:"-"`
}

func (c OAuthClient) SkipsAuthorization() bool {
	return c.FirstParty && c.SecretHash != ""
}

func (c OAuthClient) SupportsGrant(kind string) bool {
	if c.GrantTypes == nil {
		return kind == "authorization_code" || kind == "refresh_token"
	}
	for _, grant := range c.GrantTypes {
		if grant == kind {
			return true
		}
	}
	return false
}

type Authorization struct {
	Laravel      bool   `json:"-"`
	ClientID     int64  `json:"client_id"`
	RedirectURI  string `json:"redirect_uri"`
	ResponseType string `json:"response_type"`
	Scope        string `json:"scope"`
	State        string `json:"state"`
	Challenge    string `json:"code_challenge"`
	Method       string `json:"code_challenge_method"`
	Approve      bool   `json:"approve"`
}
type OAuthCode struct {
	Hash                   string
	UserID, ClientID       int64
	RedirectURI, Challenge string
	Scopes                 []string
	ExpiresAt              time.Time
}
type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}
type TokenPair struct {
	AccessHash, RefreshHash     string
	AccessExpiry, RefreshExpiry time.Time
}
type GrantIdentity struct {
	UserID int64
	Scopes []string
}
type AccessSigner interface {
	SignAccess(string, int64, int64, []string, time.Time, time.Time) (string, error)
	VerifyAccess(string, time.Time) (string, error)
}
type OAuthRepository interface {
	PersonalOAuthClient(context.Context) (int64, error)
	IssuePersonalOAuthToken(context.Context, int64, int64, TokenPair) (GrantIdentity, error)
	OAuthClient(context.Context, int64) (OAuthClient, error)
	SaveOAuthCode(context.Context, OAuthCode) error
	ExchangeOAuthCode(context.Context, string, int64, string, string, TokenPair, time.Time) (GrantIdentity, error)
	RotateOAuthRefresh(context.Context, string, int64, TokenPair, time.Time) (GrantIdentity, error)
	OAuthUser(context.Context, string, string, time.Time) (domain.User, error)
	RevokeOAuthToken(context.Context, int64, string) error
}
type OAuth struct {
	AccessTTL, RefreshTTL, PersonalTTL time.Duration
	LegacyCipher                       LegacyTokenCipher
	Signer                             AccessSigner
	Store                              OAuthRepository
	Passwords                          Passwords
	Now                                func() time.Time
}
type OAuthError string

func (e OAuthError) Error() string { return string(e) }

func (e OAuthError) PersianMessage() string {
	switch e {
	case "invalid_client":
		return "سامانهٔ درخواست‌کننده معتبر نیست."
	case "invalid_request":
		return "درخواست ارسال‌شده معتبر نیست."
	case "invalid_grant":
		return "مجوز ورود نامعتبر یا منقضی شده است."
	case "invalid_scope":
		return "سطح دسترسی درخواست‌شده معتبر نیست."
	case "access_denied":
		return "درخواست دسترسی تأیید نشد."
	case "unsupported_grant_type":
		return "روش دریافت مجوز پشتیبانی نمی‌شود."
	case "unsupported_response_type":
		return "نوع پاسخ درخواست‌شده پشتیبانی نمی‌شود."
	default:
		return "درخواست دریافت مجوز انجام نشد؛ دوباره تلاش کنید."
	}
}

const (
	OAuthInvalidClient  OAuthError = "invalid_client"
	OAuthInvalidRequest OAuthError = "invalid_request"
	OAuthInvalidGrant   OAuthError = "invalid_grant"
	OAuthInvalidScope   OAuthError = "invalid_scope"
)

var pkceVerifier = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)
var pkceChallenge = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func PKCE(value string) string {
	h := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
func (o *OAuth) ValidateAuthorization(ctx context.Context, in Authorization) (OAuthClient, []string, error) {
	c, err := o.Store.OAuthClient(ctx, in.ClientID)
	if errors.Is(err, domain.ErrNotFound) || c.Revoked {
		return c, nil, OAuthInvalidClient
	}
	if err != nil {
		return c, nil, err
	}
	if !c.SupportsGrant("authorization_code") {
		return c, nil, OAuthError("unauthorized_client")
	}
	if in.RedirectURI == "" && len(c.Redirects) == 1 {
		in.RedirectURI = c.Redirects[0]
	}
	allowed := false
	for _, redirect := range c.Redirects {
		if in.RedirectURI == redirect {
			allowed = true
		}
	}
	validChallenge := in.Method == "S256" && pkceChallenge.MatchString(in.Challenge) || in.Method == "plain" && pkceVerifier.MatchString(in.Challenge)
	if !allowed || in.ResponseType != "code" || len(in.State) > 1024 || len(in.RedirectURI) > 2048 || (in.Challenge != "" && !validChallenge) || (in.Challenge == "" && (in.Method != "" || c.SecretHash == "")) {
		return c, nil, OAuthInvalidRequest
	}
	scopes := strings.Fields(in.Scope)
	if len(scopes) == 0 && !in.Laravel {
		scopes = []string{"profile"}
	}
	seen := map[string]bool{}
	result := []string{}
	for _, scope := range scopes {
		if in.Laravel || scope != "profile" {
			return c, nil, OAuthInvalidScope
		}
		if !seen[scope] {
			result = append(result, scope)
			seen[scope] = true
		}
	}
	return c, result, nil
}
func (o *OAuth) Authorize(ctx context.Context, u domain.User, in Authorization, walletLogin bool) (string, error) {
	client, scopes, err := o.ValidateAuthorization(ctx, in)
	if err != nil {
		return "", err
	}
	if in.RedirectURI == "" && len(client.Redirects) == 1 {
		in.RedirectURI = client.Redirects[0]
	}
	target, err := url.Parse(in.RedirectURI)
	if err != nil {
		return "", OAuthInvalidRequest
	}
	query := target.Query()
	if !in.Approve {
		query.Set("error", "access_denied")
		query.Set("error_description", OAuthError("access_denied").PersianMessage())
	} else {
		code, e := newToken()
		if e != nil {
			return "", e
		}
		challenge := in.Challenge
		if in.Method == "plain" {
			challenge = PKCE(challenge)
		}
		if err = o.Store.SaveOAuthCode(ctx, OAuthCode{Hash: Digest(code), UserID: u.ID, ClientID: in.ClientID, RedirectURI: in.RedirectURI, Challenge: challenge, Scopes: scopes, ExpiresAt: o.Now().Add(5 * time.Minute)}); err != nil {
			return "", err
		}
		query.Set("code", code)
	}
	if in.State != "" {
		query.Set("state", in.State)
	}
	if walletLogin {
		query.Set("wallet_login", "true")
	}
	target.RawQuery = query.Encode()
	return target.String(), nil
}
func (o *OAuth) AuthenticateClient(ctx context.Context, id int64, secret string) (OAuthClient, error) {
	c, err := o.Store.OAuthClient(ctx, id)
	if errors.Is(err, domain.ErrNotFound) || c.Revoked {
		return c, OAuthInvalidClient
	}
	if err != nil {
		return c, err
	}
	if c.SecretHash != "" && !o.Passwords.Matches(c.SecretHash, secret) {
		return c, OAuthInvalidClient
	}
	if c.SecretHash == "" && secret != "" {
		return c, OAuthInvalidClient
	}
	return c, nil
}
func (o *OAuth) Token(ctx context.Context, id int64, secret, grant, code, redirect, verifier, refresh string) (OAuthTokens, error) {
	var result OAuthTokens
	if _, err := o.AuthenticateClient(ctx, id, secret); err != nil {
		return result, err
	}
	access, err := newOAuthID()
	if err != nil {
		return result, err
	}
	next, err := newToken()
	if err != nil {
		return result, err
	}
	now := o.Now()
	pair := TokenPair{Digest(access), Digest(next), oauthExpiry(now, o.AccessTTL), oauthExpiry(now, o.RefreshTTL)}
	var identity GrantIdentity
	switch grant {
	case "authorization_code":
		if verifier != "" && !pkceVerifier.MatchString(verifier) {
			return result, OAuthInvalidGrant
		}
		challenge := ""
		if verifier != "" {
			challenge = PKCE(verifier)
		}
		if len(code) > 80 {
			legacy, e := o.decodeLegacy(code, id)
			if e != nil || legacy.CodeID == "" || legacy.Redirect != redirect {
				return result, OAuthInvalidGrant
			}
			expected := ""
			if legacy.Challenge != "" {
				if verifier == "" {
					return result, OAuthInvalidGrant
				}
				switch legacy.Method {
				case "S256":
					if !EqualChallenge(legacy.Challenge, challenge) {
						return result, OAuthInvalidGrant
					}
				case "plain":
					if !EqualChallenge(legacy.Challenge, verifier) {
						return result, OAuthInvalidGrant
					}
				default:
					return result, OAuthInvalidGrant
				}
				expected = challenge
			} else if challenge != "" {
				return result, OAuthInvalidGrant
			}
			binder, ok := o.Store.(LegacyCodeBinder)
			if !ok {
				return result, OAuthInvalidGrant
			}
			if e = binder.BindLegacyCode(ctx, Digest(legacy.CodeID), id, redirect, expected); e != nil {
				return result, e
			}
			code = legacy.CodeID
		}
		identity, err = o.Store.ExchangeOAuthCode(ctx, Digest(code), id, redirect, challenge, pair, now)
	case "refresh_token":
		if len(refresh) > 80 {
			legacy, e := o.decodeLegacy(refresh, id)
			if e != nil || legacy.RefreshID == "" {
				return result, OAuthInvalidGrant
			}
			refresh = legacy.RefreshID
		}
		identity, err = o.Store.RotateOAuthRefresh(ctx, Digest(refresh), id, pair, now)
	default:
		return result, OAuthError("unsupported_grant_type")
	}
	if err != nil {
		return result, err
	}
	if o.Signer != nil {
		access, err = o.Signer.SignAccess(access, id, identity.UserID, identity.Scopes, now, pair.AccessExpiry)
		if err != nil {
			return result, err
		}
	}
	return OAuthTokens{access, next, "Bearer", int(pair.AccessExpiry.Sub(now).Seconds()), strings.Join(identity.Scopes, " ")}, nil
}

// Constant-time comparison is used by repositories for stored PKCE challenges.
func EqualChallenge(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

func (o *OAuth) AccessUser(ctx context.Context, token, scope string) (domain.User, error) {
	id := token
	if o.Signer != nil {
		var err error
		id, err = o.Signer.VerifyAccess(token, o.Now())
		if err != nil {
			return domain.User{}, domain.ErrCredentials
		}
	}
	return o.Store.OAuthUser(ctx, Digest(id), scope, o.Now())
}
func (o *OAuth) PersonalAccess(ctx context.Context, userID int64, ttl time.Duration) (string, error) {
	client, err := o.Store.PersonalOAuthClient(ctx)
	if err != nil {
		return "", err
	}
	access, err := newOAuthID()
	if err != nil {
		return "", err
	}
	refresh, err := newToken()
	if err != nil {
		return "", err
	}
	now := o.Now()
	until := oauthExpiry(now, ttl)
	_, err = o.Store.IssuePersonalOAuthToken(ctx, userID, client, TokenPair{Digest(access), Digest(refresh), until, until})
	if err != nil {
		return "", err
	}
	if o.Signer != nil {
		return o.Signer.SignAccess(access, client, userID, []string{}, now, until)
	}
	return access, nil
}
func (o *OAuth) Revoke(ctx context.Context, clientID int64, token string) error {
	id := token
	if o.Signer != nil && strings.Contains(token, ".") {
		parsed, err := o.Signer.VerifyAccess(token, o.Now())
		if err != nil {
			return nil
		}
		id = parsed
	}
	return o.Store.RevokeOAuthToken(ctx, clientID, Digest(id))
}

// League OAuth2 Server / Passport uses 40 random bytes for JWT identifiers.
func newOAuthID() (string, error) {
	value := make([]byte, 40)
	_, err := rand.Read(value)
	return hex.EncodeToString(value), err
}

// Passport defaults to calendar-year lifetimes (P1Y), not 365 fixed days.
func oauthExpiry(now time.Time, ttl time.Duration) time.Time {
	if ttl <= 0 {
		return now.AddDate(1, 0, 0)
	}
	return now.Add(ttl)
}
