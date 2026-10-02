package cache

import (
	"context"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"time"
)

func (s *Store) CheckRate(ctx context.Context, peer string, now time.Time) (application.RateDecision, error) {
	return s.DB.CheckRate(ctx, peer, now)
}
func (s *Store) CheckBudget(ctx context.Context, peer string, limit int, now time.Time) (application.RateDecision, error) {
	return s.DB.CheckBudget(ctx, peer, limit, now)
}
