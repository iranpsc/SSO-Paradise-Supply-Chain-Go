package mysql_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
)

func rawDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
func legacyDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn := mysqltest.Database(t)
	db := rawDB(t, dsn)
	for _, name := range []string{"migrations/001_schema.sql", "migrations/001_features.sql"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(data)); err != nil {
			t.Fatal(err)
		}
	}
	return db, dsn
}
func TestMigrationPreservesLegacyProfilesDatesAndForeignKeys(t *testing.T) {
	db, dsn := legacyDB(t)
	ctx := context.Background()
	created := "2026-01-02T03:04:05.123456+03:30"
	if _, err := db.Exec(`INSERT INTO users(id,name,email,password_hash,created_at) VALUES(7,'Legacy','legacy@example.com','hash',?)`, created); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO personal_infos(user_id,first_name,last_name) VALUES(7,'Old','User')`); err != nil {
		t.Fatal(err)
	}
	company := true
	p := domain.PersonalInfo{UserID: 7, IsCompany: &company, FirstName: "Old", LastName: "User", Mobile: "09123456789", NationalCode: "0012345678", CompanyRegistrationNumber: "000123", CompanyName: "Company"}
	payload, _ := json.Marshal(p)
	if _, err := db.Exec(`INSERT INTO profile_details(user_id,payload) VALUES(7,?)`, string(payload)); err != nil {
		t.Fatal(err)
	}
	if err := mysql.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store, err := mysql.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	u, err := store.ByID(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := time.Parse(time.RFC3339Nano, created)
	if !u.CreatedAt.Equal(expected) {
		t.Fatalf("timestamp changed: %v", u.CreatedAt)
	}
	got, err := store.PersonalInfo(ctx, 7)
	if err != nil || got.Mobile != p.Mobile || got.CompanyRegistrationNumber != "000123" {
		t.Fatalf("profile lost: %+v %v", got, err)
	}
	blob := bytes.Repeat([]byte{1}, 100000)
	if err = store.SaveAvatar(ctx, domain.Media{UserID: 7, ContentType: "image/png", Data: blob}); err != nil {
		t.Fatal("large image failed:", err)
	}
	if err = mysql.Migrate(ctx, db); err != nil {
		t.Fatal("migration not idempotent:", err)
	}
	if _, err = db.Exec(`INSERT INTO media(user_id,kind,content_type,data) VALUES(999,'avatars','image/png','x')`); err == nil {
		t.Fatal("foreign key accepted orphan")
	}
	if _, err = db.Exec(`DELETE FROM users WHERE id=7`); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"personal_infos", "profile_details", "media"} {
		var count int
		if err = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("cascade failed for %s", table)
		}
	}
}
func TestMigrationRejectsTruncationAndLaravelSource(t *testing.T) {
	db, _ := legacyDB(t)
	if _, err := db.Exec(`INSERT INTO users(name,username,email,password_hash,created_at) VALUES('Name',?,'a@example.com','hash','2026-01-01T00:00:00Z')`, strings.Repeat("x", 31)); err != nil {
		t.Fatal(err)
	}
	if err := mysql.Migrate(context.Background(), db); err == nil || !strings.Contains(err.Error(), "users.username") {
		t.Fatalf("did not reject narrowing: %v", err)
	}
	var username string
	if err := db.QueryRow(`SELECT username FROM users`).Scan(&username); err != nil || len(username) != 31 {
		t.Fatal("data was truncated")
	}
	source := rawDB(t, mysqltest.Database(t))
	if _, err := source.Exec(`CREATE TABLE users(id BIGINT PRIMARY KEY,password VARCHAR(255))`); err != nil {
		t.Fatal(err)
	}
	if err := mysql.Migrate(context.Background(), source); err == nil || !strings.Contains(err.Error(), "Laravel database detected") {
		t.Fatal("Laravel schema accepted in place")
	}
}
func TestChallengesPreserveOtherNoncesAndConsumeOnce(t *testing.T) {
	store := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now()
	if err := store.SaveChallenge(ctx, "first", "one", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveChallenge(ctx, "second", "two", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got, err := store.PullChallenge(ctx, "first", now); err != nil || got != "one" {
		t.Fatal("issuing a nonce deleted another valid nonce")
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := store.PullChallenge(ctx, "second", now); results <- err }()
	}
	wg.Wait()
	close(results)
	success, missing := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, domain.ErrNotFound):
			missing++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || missing != 1 {
		t.Fatalf("nonce replay: success=%d missing=%d", success, missing)
	}
}
func TestCleanupDryRunAndCascade(t *testing.T) {
	store := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now()
	old, err := store.Create(ctx, domain.User{Name: "old", Email: "old@example.com", CreatedAt: now.Add(-25 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := store.Create(ctx, domain.User{Name: "fresh", Email: "fresh@example.com", CreatedAt: now.Add(-23 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.CreateSession(ctx, old.ID, strings.Repeat("a", 64), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err = store.CreateSession(ctx, fresh.ID, strings.Repeat("b", 64), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	result, err := store.Cleanup(ctx, now, false)
	if err != nil || len(result.Users) != 1 || result.DeletedUsers != 0 {
		t.Fatalf("dry-run failed: %+v %v", result, err)
	}
	if _, err = store.ByID(ctx, old.ID); err != nil {
		t.Fatal("dry-run deleted user")
	}
	result, err = store.Cleanup(ctx, now, true)
	if err != nil || result.DeletedUsers != 1 || result.ExpiredRecords != 1 {
		t.Fatalf("cleanup failed: %+v %v", result, err)
	}
	if _, err = store.ByID(ctx, old.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("old user survived")
	}
	if _, err = store.ByID(ctx, fresh.ID); err != nil {
		t.Fatal("fresh user deleted")
	}
}
