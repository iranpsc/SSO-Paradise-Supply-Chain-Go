package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Archive tables are retained for audit/resuming media import, never used as
// authentication tables. Rename the whole set atomically after collision checks.
func migrateArchiveNames(ctx context.Context, conn *sql.Conn) error {
	var applied int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=12`).Scan(&applied); err != nil {
		return err
	}
	if applied > 0 {
		return migrateCredentialTableNames(ctx, conn)
	}
	rows, err := conn.QueryContext(ctx, `SELECT table_name FROM information_schema.tables WHERE table_schema=DATABASE() AND table_type='BASE TABLE'`)
	if err != nil {
		return err
	}
	present := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		present[name] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	local := regexp.MustCompile(`^go_before_laravel_[0-9]{8}_(.+)$`)
	quote := func(name string) string { return "`" + strings.ReplaceAll(name, "`", "``") + "`" }
	claimed := map[string]bool{}
	var renames []string
	for old := range present {
		var name string
		if strings.HasPrefix(old, "laravel_archive_") {
			name = "archive_laravel_" + strings.TrimPrefix(old, "laravel_archive_")
		} else if matches := local.FindStringSubmatch(old); matches != nil {
			name = "archive_local_" + matches[1]
		} else {
			continue
		}
		if present[name] || claimed[name] {
			return fmt.Errorf("archive rename collision for %s; no tables renamed", name)
		}
		claimed[name] = true
		renames = append(renames, quote(old)+" TO "+quote(name))
	}
	if len(renames) > 0 {
		if _, err := conn.ExecContext(ctx, "RENAME TABLE "+strings.Join(renames, ", ")); err != nil {
			return err
		}
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(12,?)`, stamp(time.Now()))
	if err != nil {
		return err
	}
	return migrateCredentialTableNames(ctx, conn)
}
