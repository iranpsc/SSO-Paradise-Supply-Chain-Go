package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"strings"
	"time"
)

//go:embed legacy_sessions.sql
var legacySessionsSchema string

func migrateLegacySessions(ctx context.Context, conn *sql.Conn) error {
	var n int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=5`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return migrateOutboxExpiry(ctx, conn)
	}
	for _, statement := range strings.Split(legacySessionsSchema, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(5,?)`, stamp(time.Now()))
	if err != nil {
		return err
	}
	return migrateOutboxExpiry(ctx, conn)
}
func (s *Store) LegacyRememberUser(ctx context.Context, id int64, hash string, now time.Time) (domain.User, error) {
	var owner int64
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM legacy_remember_tokens WHERE user_id=? AND token_hash=? AND expires_at>?`, id, hash, stamp(now)).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrCredentials
	}
	if err != nil {
		return domain.User{}, err
	}
	return s.ByID(ctx, owner)
}
