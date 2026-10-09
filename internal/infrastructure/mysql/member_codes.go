package mysql

import (
	"context"
	"database/sql"
	"time"
)

func migrateMemberCodeFloor(ctx context.Context, conn *sql.Conn) error {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=11`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	// Correct only the old unused starting value. Allocated codes are preserved.
	if _, err := conn.ExecContext(ctx, `UPDATE code_sequence SET value=1999999 WHERE id=1 AND value=2000000 AND NOT EXISTS(SELECT 1 FROM users WHERE code IS NOT NULL)`); err != nil {
		return err
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(11,?)`, stamp(time.Now()))
	return err
}
