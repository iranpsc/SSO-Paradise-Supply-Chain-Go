package application

import (
	"context"
	"crypto/subtle"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"strconv"
	"strings"
	"time"
)

type LegacyCookieDecoder interface {
	DecodeCookie(string, string) (string, error)
}
type LegacyRememberStore interface {
	LegacyRememberUser(context.Context, int64, string, time.Time) (domain.User, error)
}
type LegacySessionRestore struct {
	IdleTTL                                         time.Duration
	Owner                                           int64
	SourceHash, RememberHash, PasswordHash, NewHash string
	Now, Expires                                    time.Time
}
type LegacySessionRestorer interface {
	RestoreLegacySession(context.Context, LegacySessionRestore) (domain.User, error)
}

func (a *Auth) RestoreLegacyCookie(ctx context.Context, name, value string) (domain.User, string, error) {
	if a.LegacyCookies == nil {
		return domain.User{}, "", domain.ErrCredentials
	}
	plain, err := a.LegacyCookies.DecodeCookie(name, value)
	if err != nil {
		return domain.User{}, "", domain.ErrCredentials
	}
	var u domain.User
	remember := false
	oldSession := ""
	rememberHash := ""
	if name == a.LegacySessionCookie {
		if len(plain) != 40 || strings.IndexFunc(plain, func(r rune) bool {
			return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
		}) >= 0 {
			return u, "", domain.ErrCredentials
		}
		oldSession = Digest(plain)
		u, err = a.Sessions.SessionUser(ctx, oldSession, a.Now())
	} else if name == a.LegacyRememberCookie {
		parts := strings.Split(plain, "|")
		if len(parts) != 3 || len(parts[1]) > 100 {
			return u, "", domain.ErrCredentials
		}
		id, e := strconv.ParseInt(parts[0], 10, 64)
		if e != nil || id <= 0 {
			return u, "", domain.ErrCredentials
		}
		store, ok := a.Accounts.(LegacyRememberStore)
		if !ok {
			return u, "", domain.ErrCredentials
		}
		u, err = store.LegacyRememberUser(ctx, id, Digest(parts[1]), a.Now())
		rememberHash = Digest(parts[1])
		if err == nil && subtle.ConstantTimeCompare([]byte(parts[2]), []byte(security.LaravelPasswordMAC(u.PasswordHash, a.SigningKey))) != 1 && subtle.ConstantTimeCompare([]byte(parts[2]), []byte(u.PasswordHash)) != 1 {
			return domain.User{}, "", domain.ErrCredentials
		}
		remember = true
	} else {
		return u, "", domain.ErrCredentials
	}
	if err != nil {
		return u, "", err
	}
	ttl := a.SessionTTL
	if remember {
		ttl = a.RememberSessionTTL()
	}
	session, err := newToken()
	if err != nil {
		return u, "", err
	}
	restorer, ok := a.Accounts.(LegacySessionRestorer)
	if !ok {
		return u, "", domain.ErrCredentials
	}
	idleTTL := time.Duration(0)
	if a.SessionIdle && !remember {
		idleTTL = a.SessionTTL
	}
	u, err = restorer.RestoreLegacySession(ctx, LegacySessionRestore{IdleTTL: idleTTL, Owner: u.ID, SourceHash: oldSession, RememberHash: rememberHash, PasswordHash: u.PasswordHash, NewHash: Digest(session), Now: a.Now(), Expires: a.Now().Add(ttl)})
	if err != nil {
		return domain.User{}, "", err
	}
	return u, session, nil
}
