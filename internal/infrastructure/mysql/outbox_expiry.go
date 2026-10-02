package mysql

import (
	"context"
	"database/sql"
	"time"
)

func migrateOutboxExpiry(ctx context.Context, conn *sql.Conn) error {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=6`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return migrateSlidingSessions(ctx, conn)
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='mail_outbox' AND column_name='expires_at'`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := conn.ExecContext(ctx, `ALTER TABLE mail_outbox ADD COLUMN expires_at DATETIME(6) NULL`); err != nil {
			return err
		}
	}
	if _, err := conn.ExecContext(ctx, `UPDATE mail_outbox SET expires_at=DATE_ADD(created_at,INTERVAL 1 HOUR) WHERE expires_at IS NULL`); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `ALTER TABLE mail_outbox MODIFY expires_at DATETIME(6) NOT NULL`); err != nil {
		return err
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='mail_outbox' AND index_name='outbox_expiry'`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := conn.ExecContext(ctx, `ALTER TABLE mail_outbox ADD KEY outbox_expiry(expires_at,state)`); err != nil {
			return err
		}
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(6,?)`, stamp(time.Now()))
	if err != nil {
		return err
	}
	return migrateSlidingSessions(ctx, conn)
}
