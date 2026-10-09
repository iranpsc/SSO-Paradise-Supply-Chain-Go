package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

func migrateSlidingSessions(ctx context.Context, conn *sql.Conn) error {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=7`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return migrateMediaPaths(ctx, conn)
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='sessions' AND column_name='idle_seconds'`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := conn.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN idle_seconds INT UNSIGNED NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(7,?)`, stamp(time.Now()))
	if err != nil {
		return err
	}
	return migrateMediaPaths(ctx, conn)
}

func (s *Store) CreateSlidingSession(ctx context.Context, id int64, hash string, until time.Time, idle time.Duration) error {
	seconds := int64(idle / time.Second)
	if seconds <= 0 || seconds > 4294967295 {
		return errors.New("invalid session idle duration")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions(user_id,token_hash,expires_at,idle_seconds) VALUES(?,?,?,?)`, id, hash, stamp(until), seconds)
	return err
}

func (s *Store) SlidingSessionUser(ctx context.Context, hash string, now time.Time) (domain.User, error) {
	// UPDATE cannot insert a deleted session or extend an expired one. Reading
	// afterward keeps revocation authoritative even if it races this update.
	if _, err := s.db.ExecContext(ctx, `UPDATE sessions SET expires_at=GREATEST(expires_at,DATE_ADD(?,INTERVAL idle_seconds SECOND)) WHERE token_hash=? AND expires_at>? AND idle_seconds>0`, stamp(now), hash, stamp(now)); err != nil {
		return domain.User{}, err
	}
	return s.SessionUser(ctx, hash, now)
}
