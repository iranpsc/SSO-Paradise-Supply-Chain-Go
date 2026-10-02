package mysql_test

import (
	"context"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOutboxLeaseRetryAndCompletion(t *testing.T) {
	db := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	if err := db.EnqueueMail(ctx, application.MailMessage{To: "test@example.com", Subject: "verify", Text: "text", HTML: "<p>html</p>"}, now); err != nil {
		t.Fatal(err)
	}
	var winners atomic.Int32
	var claimed application.MailMessage
	var mutex sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, err := db.ClaimMail(ctx, now)
			if err == nil {
				winners.Add(1)
				mutex.Lock()
				claimed = m
				mutex.Unlock()
			} else if !errors.Is(err, domain.ErrNotFound) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatal("message leased more than once")
	}
	if err := db.FinishMail(ctx, claimed, now, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ClaimMail(ctx, now); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("retry lacks backoff")
	}
	m, err := db.ClaimMail(ctx, now.Add(2*time.Minute))
	if err != nil || m.Attempts != 2 || m.HTML != "<p>html</p>" {
		t.Fatal("retry payload/attempt lost", err)
	}
	// An old worker cannot acknowledge a newer lease.
	if err = db.FinishMail(ctx, claimed, now, true); err != nil {
		t.Fatal(err)
	}
	if err = db.FinishMail(ctx, m, now, true); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ClaimMail(ctx, now.Add(time.Hour)); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("sent message claimed again")
	}
}
func TestRateLimitsSharedAcrossConnectionsAndRestarts(t *testing.T) {
	db := mysqltest.Open(t)
	second, err := mysql.Open(db.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	var admitted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store := db
			if i%2 == 0 {
				store = second
			}
			d, err := store.CheckRate(ctx, "192.0.2.1", now)
			if err != nil {
				t.Error(err)
			} else if d.Retry == 0 {
				admitted.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if admitted.Load() != 10 {
		t.Fatal("shared quota not atomic", admitted.Load())
	}
	d, err := second.CheckRate(ctx, "192.0.2.1", now.Add(30*time.Second))
	if err != nil || d.Retry <= 0 {
		t.Fatal("ban was lost", err)
	}
	for i := 0; i < 10; i++ {
		if _, err = db.CheckRate(ctx, "192.0.2.1", now.Add(2*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	d, err = second.CheckRate(ctx, "192.0.2.1", now.Add(2*time.Minute))
	if err != nil || d.Retry != 5*time.Minute {
		t.Fatal("penalty history was lost", d, err)
	}
}
func TestExistingOAuthVersionUpgradesOperationalTables(t *testing.T) {
	store := mysqltest.Open(t)
	raw := rawDB(t, store.DSN())
	if _, err := raw.Exec(`DELETE FROM schema_migrations WHERE version>=4;DROP TABLE legacy_remember_tokens;DROP TABLE legacy_password_resets;DROP TABLE mail_outbox;DROP TABLE rate_limits;ALTER TABLE oauth_codes DROP COLUMN legacy`); err != nil {
		t.Fatal(err)
	}
	if err := mysql.Migrate(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	if err := store.EnqueueMail(context.Background(), application.MailMessage{To: "test@example.com", Subject: "notice", Text: "hello"}, time.Now()); err != nil {
		t.Fatal("old version did not receive outbox", err)
	}
	if err := mysql.Migrate(context.Background(), raw); err != nil {
		t.Fatal("operational migration not idempotent", err)
	}
}
