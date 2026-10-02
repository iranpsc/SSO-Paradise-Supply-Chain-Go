package cache

import (
	"context"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

func (s *Store) CreateSlidingSession(ctx context.Context, id int64, hash string, until time.Time, idle time.Duration) error {
	if err := s.DB.CreateSlidingSession(ctx, id, hash, until, idle); err != nil {
		return err
	}
	s.trackSession(ctx, id, hash, time.Until(until))
	return nil
}

func (s *Store) SlidingSessionUser(ctx context.Context, hash string, now time.Time) (domain.User, error) {
	if _, err := s.DB.SlidingSessionUser(ctx, hash, now); err != nil {
		return domain.User{}, err
	}
	return s.SessionUser(ctx, hash, now)
}

func (s *Store) SessionLookup(ctx context.Context, hash string, now time.Time) (int64, time.Time, error) {
	return s.DB.SessionLookup(ctx, hash, now)
}
