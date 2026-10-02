package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

func TestBrowserAuthenticationExtendsIdleWindowAndLogoutRemainsFinal(t *testing.T) {
	a, _, _ := fixture(t)
	a.SessionIdle = true
	now := time.Now().UTC().Truncate(time.Microsecond)
	a.Now = func() time.Time { return now }
	u := register(t, a, "sliding@example.com")
	ctx := context.Background()
	_, token, err := a.Login(ctx, u.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(45 * time.Minute)
	if _, err := a.Authenticate(ctx, token); err != nil {
		t.Fatal(err)
	}
	now = now.Add(45 * time.Minute)
	if _, err := a.Authenticate(ctx, token); err != nil {
		t.Fatal("active browser lost its session", err)
	}
	if err := a.Logout(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Authenticate(ctx, token); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("logout was undone by activity", err)
	}
}
