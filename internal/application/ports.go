package application

import (
	"context"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"time"
)

// Repositories own transaction boundaries; transports never query storage directly.
type Accounts interface {
	Create(context.Context, domain.User) (domain.User, error)
	ByEmail(context.Context, string) (domain.User, error)
	ByLogin(context.Context, string) (domain.User, error)
	ByID(context.Context, int64) (domain.User, error)
	CodeExists(context.Context, string) (bool, error)
	UpdateIdentity(context.Context, int64, string, string) (domain.User, error)
	ChangePassword(context.Context, int64, string, string) error
}

type Sessions interface {
	CreateSession(context.Context, int64, string, time.Time) error
	SessionUser(context.Context, string, time.Time) (domain.User, error)
	RevokeSessions(context.Context, int64) error
}

type Actions interface {
	SaveAction(context.Context, int64, string, string, string, time.Time) error
	VerifyEmail(context.Context, int64, string, time.Time) error
	ResetPassword(context.Context, string, string, string, time.Time) error
}

type Passwords interface {
	Hash(string) (string, error)
	Matches(string, string) bool
}

type Mailer interface {
	Send(context.Context, string, string, string) error
}

type PasswordSafety interface {
	Compromised(context.Context, string) (bool, error)
}

// Wallets owns wallet-address storage (separate wallets table) and the
// transactional link/create flows of the Web3 milestone. Repositories own
// transaction boundaries; transports never query storage directly.
type Wallets interface {
	UserByWallet(context.Context, string) (domain.User, error)
	UserByCode(context.Context, string) (domain.User, error)
	// WalletOf returns the address linked to a user, or "" when none is.
	WalletOf(context.Context, int64) (string, error)
	WalletTaken(context.Context, string) (bool, error)
	// AttachWallet links address to userID, returning one of the
	// domain.WalletLink* outcomes.
	AttachWallet(ctx context.Context, userID int64, address string) (string, error)
	// CreateWalletUser creates a verified, passwordless user owning address.
	CreateWalletUser(ctx context.Context, address string, now time.Time) (domain.User, error)
	// AttachWalletByCode links address to the holder of code, reporting
	// domain.ErrNotFound when no such user exists.
	AttachWalletByCode(ctx context.Context, address, code string) (domain.User, error)
}

// Challenges stores single-use Web3 nonces (the challenges table backs what
// Laravel keeps in Cache::put/pull with a 5-minute TTL).
type Challenges interface {
	SaveChallenge(ctx context.Context, key, message string, expiresAt time.Time) error
	// PullChallenge consumes a nonce exactly once, reporting
	// domain.ErrNotFound when it is missing or expired.
	PullChallenge(ctx context.Context, key string, now time.Time) (string, error)
}

// SessionAttributes stores per-session flags such as wallet_login (the
// session_attributes table), consumed by OAuth callbacks.
type SessionAttributes interface {
	SetSessionAttribute(ctx context.Context, tokenHash, name, value string, expiresAt time.Time) error
	SessionAttribute(ctx context.Context, tokenHash, name string, now time.Time) (string, error)
}

// WalletRegistry answers whether a wallet belongs to an already registered
// Metarang member (App\Services\MetarangWalletClient).
type WalletRegistry interface {
	LookupRegistration(ctx context.Context, address string) (registered bool, userCode string, err error)
}

type RegistrationCallbacks interface {
	CreateRegistration(context.Context, domain.User, string, time.Time) (domain.User, error)
	PullRegistrationCallback(context.Context, int64, time.Time) (string, error)
}
type Registration struct {
	ClientID     *int64 `json:"client_id"`
	RedirectURI  string `json:"redirect_uri"`
	BackURL      string `json:"back_url"`
	Username     string `json:"username"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Confirmation string `json:"password_confirmation"`
	Referral     string `json:"referral"`
}

type SignedEmailActions interface {
	VerifySignedEmail(context.Context, int64, string, time.Time) error
}

type SessionAttributeConsumer interface {
	PullSessionAttribute(context.Context, string, string, time.Time) (string, error)
}
