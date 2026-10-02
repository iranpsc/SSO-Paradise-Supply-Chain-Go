package mysql_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
)

func TestSlidingSessionActivityExpiryRevocationAndFixedCredentials(t *testing.T) {
	store := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	u, err := store.Create(ctx, domain.User{Name: "member", Email: "idle@example.com", CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSlidingSession(ctx, u.ID, "sliding", now.Add(2*time.Hour), 2*time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSession(ctx, u.ID, "fixed", now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SlidingSessionUser(ctx, "sliding", now.Add(90*time.Minute)); err != nil {
		t.Fatal(err)
	}
	// Activity keeps the ordinary browser session alive beyond its initial end.
	if _, err := store.SlidingSessionUser(ctx, "sliding", now.Add(150*time.Minute)); err != nil {
		t.Fatal("active session expired", err)
	}
	if _, err := store.SlidingSessionUser(ctx, "fixed", now.Add(150*time.Minute)); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("fixed credential was extended", err)
	}
	if _, err := store.SlidingSessionUser(ctx, "sliding", now.Add(271*time.Minute)); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("expired session was revived", err)
	}
	if err := store.CreateSlidingSession(ctx, u.ID, "revoked", now.Add(time.Hour), time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeSessions(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SlidingSessionUser(ctx, "revoked", now.Add(time.Minute)); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("revoked session was recreated", err)
	}
}

func TestVersionSixSessionUpgradePreservesExistingFixedExpiry(t *testing.T) {
	store := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	u, err := store.Create(ctx, domain.User{Name: "member", Email: "upgrade@example.com", CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSession(ctx, u.ID, "existing", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	raw := rawDB(t, store.DSN())
	if _, err := raw.Exec(`DELETE FROM schema_migrations WHERE version=7;ALTER TABLE sessions DROP COLUMN idle_seconds`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := mysql.Migrate(ctx, raw); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.SlidingSessionUser(ctx, "existing", now.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SlidingSessionUser(ctx, "existing", now.Add(time.Hour)); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("upgrade changed existing deadline", err)
	}
}
