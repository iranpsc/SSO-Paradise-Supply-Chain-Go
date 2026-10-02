package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
)

var (
	_ application.Accounts          = (*Store)(nil)
	_ application.Sessions          = (*Store)(nil)
	_ application.Actions           = (*Store)(nil)
	_ application.Profiles          = (*Store)(nil)
	_ application.Wallets           = (*Store)(nil)
	_ application.Challenges        = (*Store)(nil)
	_ application.SessionAttributes = (*Store)(nil)
)

// cachedUser mirrors domain.User but includes the password hash, which the
// domain type hides from JSON. Login and password change compare against the
// hash, so a cached user without it would reject every credential.
type cachedUser struct {
	Mobile          *string    `json:"mobile"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"password_hash"`
	Code            *string    `json:"code"`
	Wallet          *string    `json:"wallet_address"`
	Referral        string     `json:"referral,omitempty"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

func cachedUserFrom(u domain.User) cachedUser {
	return cachedUser{
		Mobile: u.Mobile, UpdatedAt: u.UpdatedAt,
		ID: u.ID, Name: u.Name, Username: u.Username, Email: u.Email,
		PasswordHash: u.PasswordHash, Code: u.Code, Wallet: u.Wallet, Referral: u.Referral,
		EmailVerifiedAt: u.EmailVerifiedAt, CreatedAt: u.CreatedAt,
	}
}

func (c cachedUser) user() domain.User {
	return domain.User{
		Mobile: c.Mobile, UpdatedAt: c.UpdatedAt,
		ID: c.ID, Name: c.Name, Username: c.Username, Email: c.Email,
		PasswordHash: c.PasswordHash, Code: c.Code, Wallet: c.Wallet, Referral: c.Referral,
		EmailVerifiedAt: c.EmailVerifiedAt, CreatedAt: c.CreatedAt,
	}
}

// Store decorates mysql with a Redis read-through cache for user and profile
// data. Redis is best-effort: any Redis error falls back to the database, and
// a nil Redis client disables caching entirely (database-only mode).
//
// Design: the database stays the source of truth. The first login, register or
// verify primes per-user keys; later reads are served from Redis. Every write
// invalidates only keys belonging to that user, never the whole cache.
type Store struct {
	DB         *mysql.Store
	Redis      *redis.Client
	UserTTL    time.Duration
	SessionTTL time.Duration
	Log        *slog.Logger
}

func (s *Store) enabled() bool { return s != nil && s.Redis != nil }

func (s *Store) userTTL() time.Duration {
	if s.UserTTL > 0 {
		return s.UserTTL
	}
	return 10 * time.Minute
}

func (s *Store) sessionTTL() time.Duration {
	if s.SessionTTL > 0 {
		return s.SessionTTL
	}
	return 24 * time.Hour
}

func (s *Store) warn(ctx context.Context, op string, err error) {
	if s.Log != nil && err != nil && !isMiss(err) {
		s.Log.Warn("redis cache degraded, using database", "op", op, "error", err)
	}
	_ = ctx
}

func isMiss(err error) bool { return err == redis.Nil }

func userKey(id int64) string { return "sso:user:" + strconv.FormatInt(id, 10) }
func emailKey(email string) string {
	return "sso:uid:email:" + strings.ToLower(strings.TrimSpace(email))
}
func usernameKey(name string) string {
	return "sso:uid:username:" + strings.ToLower(strings.TrimSpace(name))
}
func sessionKey(hash string) string { return "sso:session:" + hash }
func tokensKey(id int64) string     { return "sso:tokens:" + strconv.FormatInt(id, 10) }
func profileKey(id int64) string    { return "sso:profile:" + strconv.FormatInt(id, 10) }
func publicKey(id int64) string     { return "sso:public:" + strconv.FormatInt(id, 10) }

func (s *Store) getJSON(ctx context.Context, key string, dst any) bool {
	if !s.enabled() {
		return false
	}
	raw, err := s.Redis.Get(ctx, key).Bytes()
	if err != nil {
		s.warn(ctx, "get "+key, err)
		return false
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		s.warn(ctx, "decode "+key, err)
		return false
	}
	return true
}

func (s *Store) setJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	if !s.enabled() {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	if err := s.Redis.Set(ctx, key, raw, ttl).Err(); err != nil {
		s.warn(ctx, "set "+key, err)
	}
}

func (s *Store) del(ctx context.Context, keys ...string) {
	if !s.enabled() || len(keys) == 0 {
		return
	}
	if err := s.Redis.Del(ctx, keys...).Err(); err != nil {
		s.warn(ctx, "del", err)
	}
}

// primeUser stores the user object plus stable id mappings. Email/username map
// to the numeric id so a later email change only needs key deletes, and the
// user object itself is invalidated separately from the mappings.
//
// Note: domain.User hides PasswordHash from JSON, but login and password
// change verify against it, so the cache stores an explicit copy including
// the hash. Redis here is trusted local infrastructure with a short TTL,
// same as the database file.
func (s *Store) primeUser(ctx context.Context, u domain.User) {
	if !s.enabled() || u.ID == 0 {
		return
	}
	ttl := s.userTTL()
	s.setJSON(ctx, userKey(u.ID), cachedUserFrom(u), ttl)
	if u.Email != "" {
		if err := s.Redis.Set(ctx, emailKey(u.Email), u.ID, ttl).Err(); err != nil {
			s.warn(ctx, "set email map", err)
		}
	}
	if u.Username != "" {
		if err := s.Redis.Set(ctx, usernameKey(u.Username), u.ID, ttl).Err(); err != nil {
			s.warn(ctx, "set username map", err)
		}
	}
}

// invalidateUser drops the user object, profile and public resource of one
// user only. Id mappings are kept unless the identifier itself changed.
func (s *Store) invalidateUser(ctx context.Context, id int64) {
	s.del(ctx, userKey(id), profileKey(id), publicKey(id))
}

func (s *Store) cachedID(ctx context.Context, key string) (int64, bool) {
	if !s.enabled() {
		return 0, false
	}
	id, err := s.Redis.Get(ctx, key).Int64()
	if err != nil {
		s.warn(ctx, "get "+key, err)
		return 0, false
	}
	return id, true
}

// dropSessions removes every cached token of one user (logout, password
// change/reset). The database rows are already handled by the primary call.
func (s *Store) dropSessions(ctx context.Context, id int64) {
	if !s.enabled() {
		return
	}
	key := tokensKey(id)
	hashes, err := s.Redis.SMembers(ctx, key).Result()
	if err != nil {
		s.warn(ctx, "smembers sessions", err)
		return
	}
	keys := make([]string, 0, len(hashes)+1)
	for _, h := range hashes {
		keys = append(keys, sessionKey(h))
	}
	keys = append(keys, key)
	s.del(ctx, keys...)
}

func (s *Store) trackSession(ctx context.Context, id int64, hash string, ttl time.Duration) {
	if !s.enabled() {
		return
	}
	if ttl <= 0 {
		ttl = s.sessionTTL()
	}
	pipe := s.Redis.Pipeline()
	pipe.Set(ctx, sessionKey(hash), id, ttl)
	pipe.SAdd(ctx, tokensKey(id), hash)
	// Keep the owner set as long as the longest remember-me session.
	pipe.Expire(ctx, tokensKey(id), 30*24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		s.warn(ctx, "track session", err)
	}
}
