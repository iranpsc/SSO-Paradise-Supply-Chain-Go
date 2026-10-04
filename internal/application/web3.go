package application

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
)

// Web3 errors retain their sentinel identities and HTTP status semantics;
// messages displayed to users are Persian.
var (
	ErrWeb3Nonce       = errors.New("درخواست امضا منقضی شده یا یافت نشد؛ دوباره تلاش کنید.")
	ErrWeb3Signature   = errors.New("امضای کیف پول معتبر نیست.")
	ErrWeb3Upstream    = errors.New("ورود با کیف پول انجام نشد؛ دوباره تلاش کنید.")
	ErrWalletConnected = errors.New("این کیف پول قبلاً به همین حساب متصل شده است.")
	ErrWalletLinked    = errors.New("این کیف پول به حساب دیگری متصل است.")
)

// NonceTTL mirrors Web3AuthController::NONCE_TTL_MINUTES.
const NonceTTL = 5 * time.Minute

// Web3 ports Web3AuthController: login/link nonce issuance, signature
// verification, wallet resolution (existing wallet, Metarang-registered
// member, or brand-new wallet-only user) and wallet_login session flags.
type Web3 struct {
	SessionIdle bool
	Wallets     Wallets
	Challenges  Challenges
	Attributes  SessionAttributes
	Registry    WalletRegistry
	Sessions    Sessions
	Now         func() time.Time
	SessionTTL  time.Duration
	AppName     string
	PublicURL   string
}

// VerifyOutcome describes a POST /api/web3/verify result.
type VerifyOutcome struct {
	User   domain.User
	Linked bool   // an authenticated user linked the wallet instead of logging in
	Token  string // fresh session token for guest logins ("" when linked)
	// Redirect is the absolute post-login target: /home, or /email/verify
	// for members whose email is still unverified.
	Redirect string
}

func randomNonce() (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b), nil
}

// messageDomain mirrors applicationDomain(): the host of the application
// URL, falling back to localhost. Laravel reads config('app.url'); the Go
// port reads PUBLIC_URL, which plays the same role.
func (w *Web3) messageDomain() string {
	if u, err := url.Parse(w.PublicURL); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return "localhost"
}

func (w *Web3) loginMessage(address, nonce string) string {
	return "ورود به حساب کاربری.\n\nدامنهٔ سامانه: " + w.messageDomain() + "\nآدرس کیف پول: " + address + "\nکد یک‌بارمصرف: " + nonce
}

func (w *Web3) linkMessage(userID int64, address, nonce string) string {
	return "اتصال کیف پول به حساب کاربری.\n\nدامنهٔ سامانه: " + w.messageDomain() + "\nشناسهٔ حساب: " + itoa(userID) + "\nآدرس کیف پول: " + address + "\nکد یک‌بارمصرف: " + nonce
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

func loginNonceKey(address string) string { return "web3_nonce_login_" + address }

func linkNonceKey(userID int64, address string) string {
	return "web3_nonce_link_" + itoa(userID) + "_" + address
}

func validateAddress(address string) (string, error) {
	address = strings.ToLower(strings.TrimSpace(address))
	if address == "" {
		return "", domain.Validation{"address": "آدرس کیف پول الزامی است."}
	}
	if !domain.ValidWalletAddress(address) {
		return "", domain.Validation{"address": "قالب آدرس کیف پول معتبر نیست."}
	}
	return address, nil
}

func validateSignature(signature string) error {
	if strings.TrimSpace(signature) == "" {
		return domain.Validation{"signature": "امضای کیف پول الزامی است."}
	}
	if !domain.ValidWalletSignature(strings.TrimSpace(signature)) {
		return domain.Validation{"signature": "قالب امضای کیف پول معتبر نیست."}
	}
	return nil
}

// LoginNonce issues a login challenge for address, overwriting any previous
// one. No user row is created, mirroring getLoginNonce.
func (w *Web3) LoginNonce(ctx context.Context, address string) (string, error) {
	return w.loginNonce(ctx, address, false)
}
func (w *Web3) LaravelLoginNonce(ctx context.Context, address string) (string, error) {
	return w.loginNonce(ctx, address, true)
}
func (w *Web3) loginNonce(ctx context.Context, address string, legacy bool) (string, error) {
	address, err := validateAddress(address)
	if err != nil {
		return "", err
	}
	nonce, err := randomNonce()
	if err != nil {
		return "", err
	}
	message := w.loginMessage(address, nonce)
	if legacy {
		message = "Sign in to " + w.AppName + " at " + w.messageDomain() + ".\n\nWallet: " + address + "\nNonce: " + nonce
	}
	if err := w.Challenges.SaveChallenge(ctx, loginNonceKey(address), message, w.Now().Add(NonceTTL)); err != nil {
		return "", err
	}
	return message, nil
}

// LinkNonce issues a link challenge for an authenticated, verified member.
// It mirrors getLinkNonce's pre-checks; AttachWallet re-checks them inside
// its transaction.
func (w *Web3) LinkNonce(ctx context.Context, user domain.User, address string) (string, error) {
	return w.linkNonce(ctx, user, address, false)
}
func (w *Web3) LaravelLinkNonce(ctx context.Context, user domain.User, address string) (string, error) {
	return w.linkNonce(ctx, user, address, true)
}
func (w *Web3) linkNonce(ctx context.Context, user domain.User, address string, legacy bool) (string, error) {
	address, err := validateAddress(address)
	if err != nil {
		return "", err
	}
	mine, err := w.Wallets.WalletOf(ctx, user.ID)
	if err != nil {
		return "", err
	}
	if mine != "" {
		return "", ErrWalletConnected
	}
	taken, err := w.Wallets.WalletTaken(ctx, address)
	if err != nil {
		return "", err
	}
	if taken {
		return "", ErrWalletLinked
	}
	nonce, err := randomNonce()
	if err != nil {
		return "", err
	}
	message := w.linkMessage(user.ID, address, nonce)
	if legacy {
		message = "Link wallet to your " + w.AppName + " account at " + w.messageDomain() + ".\n\nAccount ID: " + itoa(user.ID) + "\nWallet: " + address + "\nNonce: " + nonce
	}
	if err := w.Challenges.SaveChallenge(ctx, linkNonceKey(user.ID, address), message, w.Now().Add(NonceTTL)); err != nil {
		return "", err
	}
	return message, nil
}

// Link verifies a link challenge signature and attaches the wallet,
// returning the domain.WalletLink* outcome.
func (w *Web3) Link(ctx context.Context, user domain.User, address, signature string) (string, error) {
	address, err := validateAddress(address)
	if err != nil {
		return "", err
	}
	if err := validateSignature(signature); err != nil {
		return "", err
	}
	message, err := w.Challenges.PullChallenge(ctx, linkNonceKey(user.ID, address), w.Now())
	if errors.Is(err, domain.ErrNotFound) {
		return "", ErrWeb3Nonce
	}
	if err != nil {
		return "", err
	}
	if !security.VerifyWalletSignature(address, strings.TrimSpace(signature), message) {
		return "", ErrWeb3Signature
	}
	return w.Wallets.AttachWallet(ctx, user.ID, address)
}

// Verify checks a login-challenge signature. An authenticated caller links
// the wallet to their own account (Laravel verifySignature behaviour);
// a guest resolves the wallet user (existing wallet, Metarang-registered
// member, or new wallet-only user) and starts a wallet session.
func (w *Web3) Verify(ctx context.Context, address, signature string, current *domain.User) (VerifyOutcome, error) {
	var out VerifyOutcome
	address, err := validateAddress(address)
	if err != nil {
		return out, err
	}
	if err := validateSignature(signature); err != nil {
		return out, err
	}
	message, err := w.Challenges.PullChallenge(ctx, loginNonceKey(address), w.Now())
	if errors.Is(err, domain.ErrNotFound) {
		return out, ErrWeb3Nonce
	}
	if err != nil {
		return out, err
	}
	if !security.VerifyWalletSignature(address, strings.TrimSpace(signature), message) {
		return out, ErrWeb3Signature
	}
	if current != nil {
		result, err := w.Wallets.AttachWallet(ctx, current.ID, address)
		if err != nil {
			return out, err
		}
		switch result {
		case domain.WalletLinkAlreadyConnected:
			return out, ErrWalletConnected
		case domain.WalletLinkAlreadyLinked:
			return out, ErrWalletLinked
		}
		u, err := w.Wallets.UserByWallet(ctx, address)
		if err != nil {
			return out, err
		}
		out.User, out.Linked, out.Redirect = u, true, w.PublicURL+"/home"
		return out, nil
	}
	user, err := w.resolveWalletUser(ctx, address)
	if err != nil {
		return out, err
	}
	token, err := w.startWalletSession(ctx, user.ID)
	if err != nil {
		return out, err
	}
	out.User, out.Token = user, token
	if user.EmailVerifiedAt == nil {
		out.Redirect = w.PublicURL + "/email/verify"
	} else {
		out.Redirect = w.PublicURL + "/home"
	}
	return out, nil
}

// resolveWalletUser mirrors resolveWalletUser without holding a database
// transaction across the Metarang HTTP call: the wallet is re-checked after
// the lookup (so a user created mid-lookup wins, as the Laravel test
// requires), and the wallets.address unique key plus owner fallback close
// the remaining race window.
func (w *Web3) resolveWalletUser(ctx context.Context, address string) (domain.User, error) {
	if u, err := w.Wallets.UserByWallet(ctx, address); err == nil {
		return u, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, err
	}
	registered, code, err := w.Registry.LookupRegistration(ctx, address)
	if err != nil {
		return domain.User{}, ErrWeb3Upstream
	}
	if u, err := w.Wallets.UserByWallet(ctx, address); err == nil {
		return u, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, err
	}
	if registered {
		if code == "" {
			return domain.User{}, ErrWeb3Upstream
		}
		u, err := w.Wallets.AttachWalletByCode(ctx, address, code)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, ErrWeb3Upstream
		}
		return u, err
	}
	return w.Wallets.CreateWalletUser(ctx, address, w.Now())
}

// startWalletSession mirrors Auth::login plus the wallet_login session flag
// (session_attributes), so later OAuth callbacks can tell wallet logins
// apart. Password logins mint fresh tokens with no attributes, which is why
// Laravel's explicit flag clearing has no counterpart here.
func (w *Web3) startWalletSession(ctx context.Context, userID int64) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	until := w.Now().Add(w.SessionTTL)
	if sliding, ok := w.Sessions.(SlidingSessions); ok && w.SessionIdle {
		err = sliding.CreateSlidingSession(ctx, userID, Digest(token), until, w.SessionTTL)
	} else {
		err = w.Sessions.CreateSession(ctx, userID, Digest(token), until)
	}
	if err != nil {
		return "", err
	}
	if err := w.Attributes.SetSessionAttribute(ctx, Digest(token), "wallet_login", "true", until); err != nil {
		return "", err
	}
	return token, nil
}
