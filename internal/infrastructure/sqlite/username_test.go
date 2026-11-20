package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

func TestUserIdentityMigrationRemovesUsernameAndRequiresName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = old.Exec(`PRAGMA foreign_keys=ON;
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			username TEXT UNIQUE,
			email TEXT NOT NULL COLLATE NOCASE UNIQUE,
			password_hash TEXT NOT NULL,
			code TEXT UNIQUE,
			referral TEXT NOT NULL DEFAULT '',
			email_verified_at TEXT,
			created_at TEXT NOT NULL
		);
		CREATE TABLE personal_infos (
			user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			is_verified INTEGER NOT NULL DEFAULT 0,
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO users(name,username,email,password_hash,created_at)
		VALUES('', 'legacy-name', 'old@example.com', 'hash', '2026-01-01T00:00:00Z');
		INSERT INTO personal_infos(user_id,first_name) VALUES(1,'Legacy profile');`); err != nil {
		t.Fatal(err)
	}
	if err = old.Close(); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		store, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		user, err := store.ByLogin(context.Background(), "old@example.com")
		if err != nil || user.Name != "legacy-name" {
			t.Fatalf("legacy identity changed: %+v %v", user, err)
		}
		var usernameColumns, nameRequired int
		if err = store.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='username'").Scan(&usernameColumns); err != nil {
			t.Fatal(err)
		}
		if err = store.db.QueryRow("SELECT \"notnull\" FROM pragma_table_info('users') WHERE name='name'").Scan(&nameRequired); err != nil {
			t.Fatal(err)
		}
		if usernameColumns != 0 || nameRequired != 1 {
			t.Fatalf("users identity schema not migrated: username columns=%d name required=%d", usernameColumns, nameRequired)
		}
		profile, err := store.PersonalInfo(context.Background(), user.ID)
		if err != nil || profile.FirstName != "Legacy profile" {
			t.Fatalf("related profile lost: %+v %v", profile, err)
		}
		_, nullNameErr := store.db.Exec(`INSERT INTO users(name,email,password_hash,created_at) VALUES(NULL,'blank@example.com','hash',?)`, stamp(time.Now()))
		if err = store.Close(); err != nil {
			t.Fatal(err)
		}
		if nullNameErr == nil {
			t.Fatal("database accepted a user without a name")
		}
	}
}

func TestByLoginUsesEmailOnly(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "users.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	user, err := store.Create(ctx, domain.User{
		Name: "Shared name", Email: "member@example.com", PasswordHash: "hash", CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := store.ByLogin(ctx, "MEMBER@example.com"); err != nil || got.ID != user.ID {
		t.Fatalf("email login failed: %v", err)
	}
	if _, err := store.ByLogin(ctx, "Shared name"); err != domain.ErrNotFound {
		t.Fatalf("display name used as login: %v", err)
	}
}
