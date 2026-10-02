package mysql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"time"
)

func (s *Store) LoginRetry(ctx context.Context, key string, now time.Time) (time.Duration, error) {
	var start time.Time
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT window_start,request_count FROM rate_limits WHERE peer_hash=?`, application.Digest("login:"+key)).Scan(&start, &count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	until := start.Add(time.Minute)
	if count >= 5 && until.After(now) {
		return until.Sub(now), nil
	}
	return 0, nil
}
func (s *Store) FailedLogin(ctx context.Context, key string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO rate_limits(peer_hash,window_start,request_count) VALUES(?,?,1) ON DUPLICATE KEY UPDATE request_count=IF(window_start<=?,1,LEAST(request_count+1,65535)),window_start=IF(window_start<=?,VALUES(window_start),window_start)`, application.Digest("login:"+key), stamp(now), stamp(now.Add(-time.Minute)), stamp(now.Add(-time.Minute)))
	return err
}
func (s *Store) ClearFailedLogins(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM rate_limits WHERE peer_hash=?`, application.Digest("login:"+key))
	return err
}
