package cache_test

import (
	"context"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/cache"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
)

func fixture(t *testing.T) (*cache.Store, *mysql.Store, *miniredis.Miniredis) {
	t.Helper()
	db := mysqltest.Open(t)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	s := &cache.Store{DB: db, Redis: client, UserTTL: time.Minute, SessionTTL: time.Hour}
	return s, db, mr
}

func createUser(t *testing.T, s *cache.Store, username, email string) domain.User {
	t.Helper()
	u, err := s.Create(context.Background(), domain.User{
		Name: "Test User", Username: username, Email: email,
		PasswordHash: "hash", CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestUserReadThroughAndIdentityInvalidation(t *testing.T) {
	s, db, _ := fixture(t)
	ctx := context.Background()
	u := createUser(t, s, "cacheuser", "cache@example.com")

	// Cached read: change the database behind the cache, the store must
	// still return the cached (old) name.
	if _, err := db.UpdateIdentity(ctx, u.ID, "Changed Behind", u.Email); err != nil {
		t.Fatal(err)
	}
	got, err := s.ByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test User" {
		t.Fatalf("expected cached name, got %q", got.Name)
	}
	// ByEmail and ByLogin must also hit the cached user.
	if got, err := s.ByEmail(ctx, "CACHE@example.com"); err != nil || got.Name != "Test User" {
		t.Fatalf("ByEmail not cached: %v %q", err, got.Name)
	}
	if got, err := s.ByLogin(ctx, "CACHEUSER"); err != nil || got.ID != u.ID {
		t.Fatalf("ByLogin not cached: %v", err)
	}

	// A profile update through the store invalidates only this user.
	updated, err := s.UpdateIdentity(ctx, u.ID, "New Name", "new@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("update failed: %+v", updated)
	}
	if got, _ := s.ByID(ctx, u.ID); got.Name != "New Name" || got.Email != "new@example.com" {
		t.Fatalf("stale after update: %+v", got)
	}
	// Old email mapping is gone, new one works.
	if _, err := db.ByEmail(ctx, "cache@example.com"); err == nil {
		// Row still exists in DB under old email? No: UpdateIdentity changed it.
		t.Fatal("old email should not exist in DB")
	}
	if got, err := s.ByEmail(ctx, "new@example.com"); err != nil || got.ID != u.ID {
		t.Fatalf("new email lookup failed: %v", err)
	}
}

func TestSessionCacheAndRevocation(t *testing.T) {
	s, _, _ := fixture(t)
	ctx := context.Background()
	u := createUser(t, s, "sessuser", "sess@example.com")

	digest := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := s.CreateSession(ctx, u.ID, digest, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionUser(ctx, digest, time.Now()); err != nil {
		t.Fatalf("session miss: %v", err)
	}
	// Second read is served from Redis even with a clock far enough that only
	// the cached token path is exercised (DB row still valid here too).
	if _, err := s.SessionUser(ctx, digest, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeSessions(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionUser(ctx, digest, time.Now()); err == nil {
		t.Fatal("revoked session still accepted")
	}
}

func TestChangePasswordDropsSessions(t *testing.T) {
	s, _, _ := fixture(t)
	ctx := context.Background()
	u := createUser(t, s, "pwuser", "pw@example.com")
	hashes := []string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	for _, h := range hashes {
		if err := s.CreateSession(ctx, u.ID, h, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	bcrypt := security.Bcrypt{Cost: 4}
	newHash, err := bcrypt.Hash("NewSecure!2026")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ChangePassword(ctx, u.ID, newHash, "nonexistent-keep"); err != nil {
		t.Fatal(err)
	}
	for _, h := range hashes {
		if _, err := s.SessionUser(ctx, h, time.Now()); err == nil {
			t.Fatalf("session survived password change: %s", h[:8])
		}
	}
}

func TestProfileCacheInvalidatesSingleUser(t *testing.T) {
	s, db, _ := fixture(t)
	ctx := context.Background()
	a := createUser(t, s, "profilea", "a@example.com")
	b := createUser(t, s, "profileb", "b@example.com")

	mk := func(first string) domain.PersonalInfo {
		no := false
		return domain.PersonalInfo{UserID: 0, IsCompany: &no, FirstName: first, LastName: "X", Mobile: "09123456789", Telephone: "02112345678", NationalCode: "1111111111", Address: "Tehran"}
	}
	pa := mk("Ali")
	pa.UserID = a.ID
	if err := s.SavePersonalInfo(ctx, pa, nil); err != nil {
		t.Fatal(err)
	}
	pb := mk("Sara")
	pb.UserID = b.ID
	if err := s.SavePersonalInfo(ctx, pb, nil); err != nil {
		t.Fatal(err)
	}
	// Prime both caches with a read, then change B behind the cache.
	if got, _ := s.PersonalInfo(ctx, a.ID); got.FirstName != "Ali" {
		t.Fatalf("A prime failed: %q", got.FirstName)
	}
	if got, _ := s.PersonalInfo(ctx, b.ID); got.FirstName != "Sara" {
		t.Fatalf("B prime failed: %q", got.FirstName)
	}
	// Change B behind the cache; B must stay cached while A updates.
	pb2 := mk("Changed")
	pb2.UserID = b.ID
	if err := db.SavePersonalInfo(ctx, pb2, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.PersonalInfo(ctx, b.ID); got.FirstName != "Sara" {
		t.Fatalf("B not cached: %q", got.FirstName)
	}
	pa2 := mk("Reza")
	pa2.UserID = a.ID
	if err := s.SavePersonalInfo(ctx, pa2, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.PersonalInfo(ctx, a.ID); got.FirstName != "Reza" {
		t.Fatalf("A stale after save: %q", got.FirstName)
	}
	if got, _ := s.PersonalInfo(ctx, b.ID); got.FirstName != "Sara" {
		t.Fatalf("B evicted by A's update: %q", got.FirstName)
	}
}

// Regression: domain.User hides PasswordHash from JSON. The cache must store
// the hash explicitly, otherwise login served from Redis rejects every
// password with "invalid credentials".
func TestCachedUserKeepsPasswordHash(t *testing.T) {
	s, _, _ := fixture(t)
	ctx := context.Background()
	bcrypt := security.Bcrypt{Cost: 4}
	hash, err := bcrypt.Hash("TestUser!2026")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, domain.User{Name: "Hash User", Username: "hashuser", Email: "hash@example.com", PasswordHash: hash, CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	// Force the read to come from Redis (prime already happened in Create).
	got, err := s.ByLogin(ctx, "hashuser")
	if err != nil {
		t.Fatal(err)
	}
	if got.PasswordHash == "" {
		t.Fatal("password hash lost in cache round-trip")
	}
	if !bcrypt.Matches(got.PasswordHash, "TestUser!2026") {
		t.Fatal("cached hash does not verify")
	}
	if bcrypt.Matches(got.PasswordHash, "Wrong!2026") {
		t.Fatal("wrong password accepted")
	}
}

func TestNilRedisIsDatabaseOnly(t *testing.T) {
	db := mysqltest.Open(t)
	s := &cache.Store{DB: db}
	ctx := context.Background()
	u, err := s.Create(ctx, domain.User{Name: "No Redis", Username: "noredis", Email: "nr@example.com", PasswordHash: "h", CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.ByID(ctx, u.ID); err != nil || got.Email != "nr@example.com" {
		t.Fatalf("passthrough failed: %v", err)
	}
	if _, err := s.SessionUser(ctx, "missing", time.Now()); err == nil {
		t.Fatal("missing session accepted")
	}
}

func TestCachedSessionsCannotExtendExpiryOrResurrectCredentials(t *testing.T) {
	s, db, _ := fixture(t)
	ctx := context.Background()
	bcrypt := security.Bcrypt{Cost: 4}
	hash, err := bcrypt.Hash("OldPassword!2026")
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.Create(ctx, domain.User{Name: "Cached", Username: "cached_security", Email: "security@example.com", PasswordHash: hash, CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	a := &application.Auth{Accounts: s, Sessions: s, Passwords: bcrypt, Now: time.Now, SessionTTL: time.Hour}
	_, token, err := a.Login(ctx, u.Email, "OldPassword!2026")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SessionUser(ctx, application.Digest(token), time.Now().Add(2*time.Hour)); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("Redis extended expired session")
	}
	// Simulate a write whose cache invalidation was missed.
	next, err := bcrypt.Hash("NewPassword!2026")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.ChangePassword(ctx, u.ID, next, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err = a.Login(ctx, u.Email, "OldPassword!2026"); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("stale cached password accepted")
	}
	if _, _, err = a.Login(ctx, u.Email, "NewPassword!2026"); err != nil {
		t.Fatal(err)
	}
	if _, err = a.Authenticate(ctx, token); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("stale cached token accepted")
	}
}
