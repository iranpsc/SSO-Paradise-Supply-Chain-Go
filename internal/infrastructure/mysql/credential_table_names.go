package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Migration 13 renames tables without changing rows, indexes or foreign keys.
// The preflight prevents a partial rename when a destination already exists.
func migrateCredentialTableNames(ctx context.Context, conn *sql.Conn) error {
	var applied int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=13`).Scan(&applied); err != nil {
		return err
	}
	if applied > 0 {
		return nil
	}
	var renames []string
	for _, pair := range [][2]string{{"legacy_password_resets", "password_reset_tokens"}, {"legacy_remember_tokens", "remember_tokens"}} {
		var old, current int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?`, pair[0]).Scan(&old); err != nil {
			return err
		}
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?`, pair[1]).Scan(&current); err != nil {
			return err
		}
		if old > 0 && current > 0 {
			return fmt.Errorf("credential table rename collision: %s", pair[1])
		}
		if old == 0 && current == 0 {
			return fmt.Errorf("missing credential table: %s", pair[1])
		}
		if old > 0 {
			renames = append(renames, "`"+pair[0]+"` TO `"+pair[1]+"`")
		}
	}
	if len(renames) > 0 {
		if _, err := conn.ExecContext(ctx, "RENAME TABLE "+strings.Join(renames, ", ")); err != nil {
			return err
		}
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(13,?)`, stamp(time.Now()))
	return err
}
