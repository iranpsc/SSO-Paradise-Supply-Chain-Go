package application

import (
	"context"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

// SlidingSessions refreshes a live browser session without recreating revoked
// credentials. Fixed-lifetime credentials retain their original deadline.
type SlidingSessions interface {
	CreateSlidingSession(context.Context, int64, string, time.Time, time.Duration) error
	SlidingSessionUser(context.Context, string, time.Time) (domain.User, error)
}

type SessionDeadline interface {
	SessionLookup(context.Context, string, time.Time) (int64, time.Time, error)
}
