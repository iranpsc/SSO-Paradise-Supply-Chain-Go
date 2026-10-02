package mysql_test

import (
	"context"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
)

func TestFailedLoginWindowIsolationAndSuccessfulClear(t *testing.T) {
	store := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	key := "member@example.com\x00192.0.2.1"
	for i := 0; i < 5; i++ {
		if retry, err := store.LoginRetry(ctx, key, now); err != nil || retry != 0 {
			t.Fatalf("attempt %d blocked early: %v %v", i, retry, err)
		}
		if err := store.FailedLogin(ctx, key, now); err != nil {
			t.Fatal(err)
		}
	}
	if retry, err := store.LoginRetry(ctx, key, now.Add(10*time.Second)); err != nil || retry != 50*time.Second {
		t.Fatalf("fifth failure did not lock remaining window: %v %v", retry, err)
	}
	for _, other := range []string{"other@example.com\x00192.0.2.1", "member@example.com\x00192.0.2.2"} {
		if retry, err := store.LoginRetry(ctx, other, now); err != nil || retry != 0 {
			t.Fatalf("unrelated login blocked: %v %v", retry, err)
		}
	}
	if retry, err := store.LoginRetry(ctx, key, now.Add(time.Minute)); err != nil || retry != 0 {
		t.Fatalf("window did not expire: %v %v", retry, err)
	}
	if err := store.FailedLogin(ctx, key, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if retry, err := store.LoginRetry(ctx, key, now.Add(time.Minute)); err != nil || retry != 0 {
		t.Fatalf("expired failures carried forward: %v %v", retry, err)
	}
	if err := store.ClearFailedLogins(ctx, key); err != nil {
		t.Fatal(err)
	}
	if retry, err := store.LoginRetry(ctx, key, now); err != nil || retry != 0 {
		t.Fatalf("successful login did not clear failures: %v %v", retry, err)
	}
}
