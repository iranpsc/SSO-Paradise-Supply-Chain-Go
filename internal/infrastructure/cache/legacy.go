package cache

import (
	"context"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"time"
)

func (s *Store) LegacyRememberUser(ctx context.Context, id int64, hash string, now time.Time) (domain.User, error) {
	return s.DB.LegacyRememberUser(ctx, id, hash, now)
}
func (s *Store) RestoreLegacySession(ctx context.Context, in application.LegacySessionRestore) (domain.User, error) {
	return s.DB.RestoreLegacySession(ctx, in)
}
