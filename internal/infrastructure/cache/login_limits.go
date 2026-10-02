package cache

import (
	"context"
	"time"
)

func (s *Store) LoginRetry(ctx context.Context, key string, now time.Time) (time.Duration, error) {
	return s.DB.LoginRetry(ctx, key, now)
}
func (s *Store) FailedLogin(ctx context.Context, key string, now time.Time) error {
	return s.DB.FailedLogin(ctx, key, now)
}
func (s *Store) ClearFailedLogins(ctx context.Context, key string) error {
	return s.DB.ClearFailedLogins(ctx, key)
}
