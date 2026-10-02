package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"path/filepath"
	"testing"
	"time"
)

func TestUsernameMigrationPreservesOldAccountsAndIsRepeatable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = old.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err = old.Exec("INSERT INTO users(name,email,password_hash,created_at) VALUES(?,?,?,?)", "Legacy", "old@example.com", "hash", stamp(time.Now())); err != nil {
		t.Fatal(err)
	}
	old.Close()
	for range 2 {
		store, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		user, err := store.ByLogin(context.Background(), "old@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if user.Name != "Legacy" || user.Username != "" {
			t.Fatal("legacy account changed")
		}
		store.Close()
	}
}

func TestUsernameUniquenessIsCaseInsensitive(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "users.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	_, err = s.Create(ctx, domain.User{Name: "Shared name", Username: "member", Email: "one@example.com", PasswordHash: "hash", CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Create(ctx, domain.User{Name: "Shared name", Username: "MEMBER", Email: "two@example.com", PasswordHash: "hash", CreatedAt: time.Now()})
	if !errors.Is(err, domain.ErrUsernameConflict) {
		t.Fatalf("expected username conflict: %v", err)
	}
	if _, err = s.ByLogin(ctx, "MEMBER"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ByLogin(ctx, "Shared name"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("display name used as login")
	}
}
