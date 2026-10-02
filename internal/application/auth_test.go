package application_test

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
)

const password = "SecurePass!2026"

func TestLoginAcceptsUsernameAndEmail(t *testing.T) {
	a, _, _ := fixture(t)
	u := register(t, a, "username@example.com")
	ctx := context.Background()
	for _, identifier := range []string{u.Email, strings.ToUpper(u.Username)} {
		got, token, err := a.Login(ctx, identifier, password)
		if err != nil || got.ID != u.ID || token == "" {
			t.Fatalf("login failed for %q: %v", identifier, err)
		}
		if _, _, err := a.Login(ctx, identifier, "wrong"); !errors.Is(err, domain.ErrCredentials) {
			t.Fatal("wrong password accepted")
		}
	}
	for _, username := range []string{"", "ab", "1member", "member name", "member@example.com"} {
		_, err := a.Register(ctx, application.Registration{Username: username, Name: "Name", Email: "another@example.com", Password: password, Confirmation: password})
		var fields domain.Validation
		if !errors.As(err, &fields) || fields["username"] == "" {
			t.Fatalf("accepted invalid username: %q", username)
		}
	}
}

type mailbox struct{ body string }

type safePasswords struct{}

type compromisedPasswords struct{}

func (compromisedPasswords) Compromised(context.Context, string) (bool, error) { return true, nil }

func (safePasswords) Compromised(context.Context, string) (bool, error) { return false, nil }

func (m *mailbox) Send(_ context.Context, _, _, body string) error { m.body = body; return nil }
func (m *mailbox) token(t *testing.T) string {
	t.Helper()
	parts := strings.Split(m.body, "\n\n")
	u, err := url.Parse(parts[len(parts)-1])
	if err != nil {
		t.Fatal(err)
	}
	return u.Query().Get("token")
}
func fixture(t *testing.T) (*application.Auth, *mysql.Store, *mailbox) {
	t.Helper()
	db := mysqltest.Open(t)
	t.Cleanup(func() { db.Close() })
	m := &mailbox{}
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, Passwords: security.Bcrypt{Cost: 4}, Safety: safePasswords{}, Mailer: m, PublicURL: "http://localhost:3000", Now: time.Now, SessionTTL: time.Hour}
	return a, db, m
}
func register(t *testing.T, a *application.Auth, email string) domain.User {
	t.Helper()
	u, err := a.Register(context.Background(), application.Registration{Username: "user_" + strings.Split(email, "@")[0], Name: "کاربر تست", Email: email, Password: password, Confirmation: password})
	if err != nil {
		t.Fatal(err)
	}
	return u
}
func verify(t *testing.T, a *application.Auth, m *mailbox, u domain.User) domain.User {
	t.Helper()
	ctx := context.Background()
	if err := a.SendVerification(ctx, u); err != nil {
		t.Fatal(err)
	}
	if err := a.Verify(ctx, u.ID, m.token(t)); err != nil {
		t.Fatal(err)
	}
	u, err := a.Accounts.ByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestRegistrationPreservesLaravelRules(t *testing.T) {
	a, _, _ := fixture(t)
	ctx := context.Background()
	u := register(t, a, "USER@example.com")
	if u.Email != "user@example.com" || u.EmailVerifiedAt != nil || u.Code != nil || u.PasswordHash == password {
		t.Fatalf("invalid registration: %+v", u)
	}
	if !a.Passwords.Matches(u.PasswordHash, password) {
		t.Fatal("password not hashed correctly")
	}
	_, err := a.Register(ctx, application.Registration{Username: "tester", Name: "Other", Email: "user@example.com", Password: password, Confirmation: password})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate email: %v", err)
	}
	for _, name := range []string{"hm-admin", "HM-admin", "hM-admin", "Hm-admin", strings.Repeat("ن", 51)} {
		_, err := a.Register(ctx, application.Registration{Username: "tester", Name: name, Email: "other@example.com", Password: password, Confirmation: password})
		var v domain.Validation
		if !errors.As(err, &v) || v["name"] == "" {
			t.Errorf("accepted name %q", name)
		}
	}
	_, err = a.Register(ctx, application.Registration{Username: "tester", Name: "Other", Email: "other@example.com", Password: password, Confirmation: password, Referral: "hm-9999"})
	var v domain.Validation
	if !errors.As(err, &v) || v["referral"] == "" {
		t.Fatal("invalid referral accepted")
	}
}
func TestLoginAndLogoutRevokeEverySession(t *testing.T) {
	a, _, _ := fixture(t)
	ctx := context.Background()
	u := register(t, a, "a@example.com")
	_, first, err := a.Login(ctx, u.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := a.Login(ctx, u.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	for _, email := range []string{u.Email, "unknown@example.com"} {
		_, _, err := a.Login(ctx, email, "wrong")
		if !errors.Is(err, domain.ErrCredentials) {
			t.Fatal("wrong login accepted")
		}
	}
	if err := a.Logout(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{first, second} {
		if _, err := a.Authenticate(ctx, token); !errors.Is(err, domain.ErrCredentials) {
			t.Fatal("session survived logout")
		}
	}
}
func TestVerificationIsBoundToUserExpiresAndCannotReplay(t *testing.T) {
	a, _, m := fixture(t)
	ctx := context.Background()
	u := register(t, a, "a@example.com")
	other := register(t, a, "b@example.com")
	if err := a.SendVerification(ctx, u); err != nil {
		t.Fatal(err)
	}
	token := m.token(t)
	if err := a.Verify(ctx, other.ID, token); !errors.Is(err, domain.ErrToken) {
		t.Fatal("cross-account token accepted")
	}
	if err := a.Verify(ctx, u.ID, token); err != nil {
		t.Fatal(err)
	}
	if err := a.Verify(ctx, u.ID, token); !errors.Is(err, domain.ErrToken) {
		t.Fatal("verification replay accepted")
	}
	updated, err := a.Accounts.ByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Code == nil || *updated.Code != "hm-2000001" || updated.EmailVerifiedAt == nil {
		t.Fatal("verification did not assign code")
	}
	if err := a.SendVerification(ctx, other); err != nil {
		t.Fatal(err)
	}
	token = m.token(t)
	a.Now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if err := a.Verify(ctx, other.ID, token); !errors.Is(err, domain.ErrToken) {
		t.Fatal("expired token accepted")
	}
}
func TestEmailChangeInvalidatesVerificationAndOldLinks(t *testing.T) {
	a, _, m := fixture(t)
	ctx := context.Background()
	u := verify(t, a, m, register(t, a, "a@example.com"))
	if err := a.ForgotPassword(ctx, u.Email); err != nil {
		t.Fatal(err)
	}
	oldReset := m.token(t)
	u, err := a.UpdateAccount(ctx, u, "Updated", "new@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if u.EmailVerifiedAt != nil {
		t.Fatal("email remained verified")
	}
	if err := a.Reset(ctx, "a@example.com", oldReset, password, password); !errors.Is(err, domain.ErrToken) {
		t.Fatal("old-email reset accepted")
	}
	if _, err := a.UpdateAccount(ctx, u, "Again", u.Email); !errors.Is(err, domain.ErrUnverified) {
		t.Fatal("unverified account updated")
	}
}
func TestResetIsSingleUseAndRevokesSessions(t *testing.T) {
	a, _, m := fixture(t)
	ctx := context.Background()
	u := register(t, a, "a@example.com")
	_, session, err := a.Login(ctx, u.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.ForgotPassword(ctx, u.Email); err != nil {
		t.Fatal(err)
	}
	token := m.token(t)
	next := "NewSecure!2026"
	if err := a.Reset(ctx, u.Email, token, next, next); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Authenticate(ctx, session); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("session survived reset")
	}
	if err := a.Reset(ctx, u.Email, token, next, next); !errors.Is(err, domain.ErrToken) {
		t.Fatal("reset replay accepted")
	}
	if _, _, err := a.Login(ctx, u.Email, next); err != nil {
		t.Fatal(err)
	}
}
func TestPasswordChangeChecksCurrentAndKeepsOnlyCurrentSession(t *testing.T) {
	a, _, m := fixture(t)
	ctx := context.Background()
	u := verify(t, a, m, register(t, a, "a@example.com"))
	_, first, err := a.Login(ctx, u.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	_, other, err := a.Login(ctx, u.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.ChangePassword(ctx, u, "incorrect", "NewSecure!2026", "NewSecure!2026", first); err == nil {
		t.Fatal("incorrect current password accepted")
	}
	if err := a.ChangePassword(ctx, u, password, "NewSecure!2026", "NewSecure!2026", first); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Authenticate(ctx, first); err != nil {
		t.Fatal("current session was revoked")
	}
	if _, err := a.Authenticate(ctx, other); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("other session survived")
	}
}
func TestSessionsExpireAndDatabasePersists(t *testing.T) {
	db := mysqltest.Open(t)
	u, err := db.Create(context.Background(), domain.User{Name: "A", Email: "a@example.com", PasswordHash: "hash", CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	dsn := db.DSN()
	db, err = mysql.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ByID(context.Background(), u.ID); err != nil {
		t.Fatal("data lost on reopen", err)
	}
	if err := db.CreateSession(context.Background(), u.ID, "hash", time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SessionUser(context.Background(), "hash", time.Now()); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("expired session accepted")
	}
}

func TestCompromisedPasswordDoesNotChangeAccount(t *testing.T) {
	a, _, m := fixture(t)
	ctx := context.Background()
	u := verify(t, a, m, register(t, a, "a@example.com"))
	a.Safety = compromisedPasswords{}
	err := a.ChangePassword(ctx, u, password, "Password1!", "Password1!", "")
	var fields domain.Validation
	if !errors.As(err, &fields) || fields["password"] == "" {
		t.Fatal("compromised password accepted")
	}
	if _, _, err := a.Login(ctx, u.Email, password); err != nil {
		t.Fatal("original password was changed", err)
	}
}
