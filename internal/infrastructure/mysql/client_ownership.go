package mysql

import (
	"context"
	"database/sql"
	"time"
)

func migrateClientOwnership(ctx context.Context, conn *sql.Conn) error {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=10`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return migrateMemberCodeFloor(ctx, conn)
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='oauth_clients' AND column_name='first_party'`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		// Old imports did not retain ownership. Require consent for them until
		// an operator classifies them; new administrator-created clients are internal.
		if _, err := conn.ExecContext(ctx, `ALTER TABLE oauth_clients ADD COLUMN first_party BOOLEAN NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if _, err := conn.ExecContext(ctx, `ALTER TABLE oauth_clients ALTER COLUMN first_party SET DEFAULT 1`); err != nil {
		return err
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(10,?)`, stamp(time.Now()))
	if err != nil {
		return err
	}
	return migrateMemberCodeFloor(ctx, conn)
}
