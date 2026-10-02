package cache

import (
	"context"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"time"
)

func (s *Store) Cleanup(ctx context.Context, now time.Time, apply bool) (mysql.CleanupResult, error) {
	result, err := s.DB.Cleanup(ctx, now, apply)
	if err != nil || !apply {
		return result, err
	}
	for _, u := range result.Users {
		s.dropSessions(ctx, u.ID)
		s.invalidateUser(ctx, u.ID)
		if u.Email != "" {
			s.del(ctx, emailKey(u.Email))
		}
		if u.Username != "" {
			s.del(ctx, usernameKey(u.Username))
		}
	}
	return result, nil
}
