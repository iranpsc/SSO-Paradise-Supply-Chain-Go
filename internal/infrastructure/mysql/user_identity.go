package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Migration 14 removes the separate username identity. Existing display names
// are preserved; blank names are filled from the old username or email local
// part before name becomes required.
func migrateUserIdentity(ctx context.Context, conn *sql.Conn) error {
	var applied int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=14`).Scan(&applied); err != nil {
		return err
	}
	if applied > 0 {
		return nil
	}
	var nameExists, usernameExists int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='users' AND column_name='name'`).Scan(&nameExists); err != nil {
		return err
	}
	if nameExists == 0 {
		return fmt.Errorf("migration blocked: users.name column is missing")
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='users' AND column_name='username'`).Scan(&usernameExists); err != nil {
		return err
	}
	fallback := `COALESCE(NULLIF(SUBSTRING_INDEX(email,'@',1),''),CONCAT('user_',id))`
	query := `UPDATE users SET name=COALESCE(NULLIF(TRIM(name),''),` + fallback + `) WHERE name IS NULL OR TRIM(name)=''`
	if usernameExists > 0 {
		query = `UPDATE users SET name=COALESCE(NULLIF(TRIM(name),''),NULLIF(TRIM(username),''),` + fallback + `) WHERE name IS NULL OR TRIM(name)=''`
	}
	if _, err := conn.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("preserve names before username removal: %w", err)
	}
	if _, err := conn.ExecContext(ctx, `ALTER TABLE users MODIFY name VARCHAR(255) NOT NULL`); err != nil {
		return fmt.Errorf("make users.name required: %w", err)
	}
	if usernameExists > 0 {
		if _, err := conn.ExecContext(ctx, `ALTER TABLE users DROP COLUMN username`); err != nil {
			return fmt.Errorf("remove users.username: %w", err)
		}
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(14,?)`, stamp(time.Now()))
	return err
}
