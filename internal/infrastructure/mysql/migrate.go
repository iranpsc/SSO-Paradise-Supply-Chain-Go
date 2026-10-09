package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

type dbExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func writeProfile(ctx context.Context, db dbExecutor, p domain.PersonalInfo) error {
	_, err := db.ExecContext(ctx, `UPDATE personal_infos SET first_name=?,last_name=?,is_company=?,mobile=?,telephone=?,national_code=?,address=?,company_name=?,company_address=?,company_registration_number=?,company_national_number=?,company_tax_number=?,company_executive_name=?,verification_messages=? WHERE user_id=?`, p.FirstName, p.LastName, p.IsCompany, p.Mobile, p.Telephone, p.NationalCode, p.Address, p.CompanyName, p.CompanyAddress, p.CompanyRegistrationNumber, p.CompanyNationalNumber, p.CompanyTaxNumber, p.CompanyExecutiveName, p.VerificationMessages, p.UserID)
	return err
}

// Migrate is serialized on one connection. MySQL DDL implicitly commits; each
// step checks metadata so an interrupted migration can be resumed. Run offline
// after a backup when upgrading an existing installation.
func Migrate(ctx context.Context, db *sql.DB) error {
	var legacy int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='users' AND column_name='password'`).Scan(&legacy); err != nil {
		return err
	}
	if legacy > 0 {
		return fmt.Errorf("Laravel database detected; use a separate Go database and the import command, never migrate Laravel tables in place")
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var locked int
	if err = conn.QueryRowContext(ctx, `SELECT GET_LOCK(CONCAT(DATABASE(), ':sso:migrate'), 30)`).Scan(&locked); err != nil {
		return err
	}
	if locked != 1 {
		return fmt.Errorf("migration lock unavailable")
	}
	defer conn.ExecContext(context.Background(), `SELECT RELEASE_LOCK(CONCAT(DATABASE(), ':sso:migrate'))`)
	if _, err = conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version INT PRIMARY KEY, applied_at DATETIME(6) NOT NULL) ENGINE=InnoDB`); err != nil {
		return err
	}
	var applied int
	if err = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=2`).Scan(&applied); err != nil {
		return err
	}
	if applied != 0 {
		return migrateOAuth(ctx, conn)
	}
	var present int
	if err = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='users'`).Scan(&present); err != nil {
		return err
	}
	if present == 0 {
		for _, script := range []string{canonicalSchema, canonicalFeatures} {
			for _, statement := range strings.Split(script, ";") {
				if strings.TrimSpace(statement) == "" {
					continue
				}
				if _, err = conn.ExecContext(ctx, statement); err != nil {
					return fmt.Errorf("fresh schema: %w", err)
				}
			}
		}
		if _, err = conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(2,?)`, stamp(time.Now())); err != nil {
			return err
		}
		return migrateOAuth(ctx, conn)
	}
	for _, script := range []string{schema, featuresSchema} {
		for _, statement := range strings.Split(script, ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err = conn.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("baseline: %w", err)
			}
		}
	}
	// Preflight every narrowing conversion before changing any existing data.
	lengths := map[string]map[string]int{
		"users":    {"username": 30, "password_hash": 255, "code": 32, "referral": 32},
		"sessions": {"token_hash": 64}, "actions": {"token_hash": 64, "kind": 6},
		"wallets": {"address": 42}, "challenges": {"key": 100},
		"media":              {"kind": 20, "content_type": 32},
		"session_attributes": {"token_hash": 64, "name": 32}, "registration_callbacks": {"url": 2048},
	}
	for table, cols := range lengths {
		for col, max := range cols {
			var count int
			query := fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE CHAR_LENGTH(`%s`)>?", table, col)
			if err = conn.QueryRowContext(ctx, query, max).Scan(&count); err != nil {
				return err
			}
			if count > 0 {
				return fmt.Errorf("migration blocked: %s.%s has %d values exceeding %d characters", table, col, count, max)
			}
		}
	}
	// Load and validate legacy JSON before DDL. Keep it as an audit archive;
	// all runtime reads/writes use atomic columns after this migration.
	rows, err := conn.QueryContext(ctx, `SELECT user_id,payload FROM profile_details`)
	if err != nil {
		return err
	}
	var profiles []domain.PersonalInfo
	for rows.Next() {
		var id int64
		var payload string
		var p domain.PersonalInfo
		if err = rows.Scan(&id, &payload); err != nil {
			rows.Close()
			return err
		}
		if err = json.Unmarshal([]byte(payload), &p); err != nil {
			rows.Close()
			return fmt.Errorf("profile %d: %w", id, err)
		}
		p.UserID = id
		if err = checkProfileLengths(p); err != nil {
			rows.Close()
			return err
		}
		for name, value := range map[string]string{"company_registration_number": p.CompanyRegistrationNumber, "company_national_number": p.CompanyNationalNumber, "company_tax_number": p.CompanyTaxNumber} {
			if len([]rune(value)) > 32 {
				rows.Close()
				return fmt.Errorf("profile %d: %s exceeds 32 characters", id, name)
			}
		}
		profiles = append(profiles, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if err = preflightDates(ctx, conn); err != nil {
		return err
	}
	for _, fk := range []struct{ table, column, parent, parentColumn string }{{"personal_infos", "user_id", "users", "id"}, {"profile_details", "user_id", "users", "id"}, {"sessions", "user_id", "users", "id"}, {"actions", "user_id", "users", "id"}, {"media", "user_id", "users", "id"}, {"wallets", "user_id", "users", "id"}, {"registration_callbacks", "user_id", "users", "id"}, {"session_attributes", "token_hash", "sessions", "token_hash"}} {
		var orphans int
		if err = conn.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM `%s` c LEFT JOIN `%s` p ON p.`%s`=c.`%s` WHERE p.`%s` IS NULL", fk.table, fk.parent, fk.parentColumn, fk.column, fk.parentColumn)).Scan(&orphans); err != nil {
			return err
		}
		if orphans > 0 {
			return fmt.Errorf("migration blocked: %s.%s contains %d orphan records", fk.table, fk.column, orphans)
		}
	}
	// Drop owned foreign keys before widening referenced IDs. MySQL 9 also
	// enforces the legacy inline REFERENCES syntax. Reinstall explicit CASCADE
	// constraints below; do not run concurrently with application traffic.
	fkRows, e := conn.QueryContext(ctx, `SELECT table_name,constraint_name FROM information_schema.table_constraints WHERE constraint_schema=DATABASE() AND constraint_type='FOREIGN KEY' AND table_name IN ('personal_infos','profile_details','sessions','actions','media','wallets','session_attributes','registration_callbacks')`)
	if e != nil {
		return e
	}
	type fk struct{ table, name string }
	var fks []fk
	for fkRows.Next() {
		var f fk
		if e = fkRows.Scan(&f.table, &f.name); e != nil {
			fkRows.Close()
			return e
		}
		fks = append(fks, f)
	}
	e = fkRows.Err()
	fkRows.Close()
	if e != nil {
		return e
	}
	for _, f := range fks {
		if _, e = conn.ExecContext(ctx, fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`", f.table, f.name)); e != nil {
			return e
		}
	}
	columns := []struct{ table, name, definition string }{
		{"users", "id", "BIGINT NOT NULL AUTO_INCREMENT"}, {"users", "username", "VARCHAR(30) NULL"},
		{"users", "email", "VARCHAR(255) NULL"}, {"users", "password_hash", "VARCHAR(255) NULL"},
		{"users", "code", "VARCHAR(32) NULL"}, {"users", "referral", "VARCHAR(32) NOT NULL DEFAULT ''"},
		{"personal_infos", "user_id", "BIGINT NOT NULL"}, {"profile_details", "user_id", "BIGINT NOT NULL"},
		{"sessions", "user_id", "BIGINT NOT NULL"}, {"sessions", "token_hash", "CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"},
		{"actions", "user_id", "BIGINT NOT NULL"}, {"actions", "token_hash", "CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"}, {"actions", "kind", "VARCHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"},
		{"media", "user_id", "BIGINT NOT NULL"}, {"media", "kind", "VARCHAR(20) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"}, {"media", "content_type", "VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"}, {"media", "data", "MEDIUMBLOB NOT NULL"},
		{"wallets", "user_id", "BIGINT NOT NULL"}, {"wallets", "address", "CHAR(42) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"},
		{"challenges", "key", "VARCHAR(100) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"},
		{"session_attributes", "token_hash", "CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"}, {"session_attributes", "name", "VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"},
		{"registration_callbacks", "user_id", "BIGINT NOT NULL"}, {"registration_callbacks", "url", "VARCHAR(2048) NOT NULL"},
		{"code_sequence", "value", "BIGINT NOT NULL"},
	}
	for _, c := range columns {
		if _, err = conn.ExecContext(ctx, fmt.Sprintf("ALTER TABLE `%s` MODIFY `%s` %s", c.table, c.name, c.definition)); err != nil {
			return fmt.Errorf("%s.%s: %w", c.table, c.name, err)
		}
	}
	for _, c := range []struct{ name, definition string }{
		{"is_company", "BOOLEAN NULL"}, {"mobile", "VARCHAR(11) CHARACTER SET ascii NOT NULL DEFAULT ''"},
		{"telephone", "VARCHAR(11) CHARACTER SET ascii NOT NULL DEFAULT ''"}, {"national_code", "VARCHAR(10) CHARACTER SET ascii NOT NULL DEFAULT ''"},
		{"address", "VARCHAR(255) NOT NULL DEFAULT ''"}, {"company_name", "VARCHAR(255) NOT NULL DEFAULT ''"},
		{"company_address", "VARCHAR(255) NOT NULL DEFAULT ''"}, {"company_registration_number", "VARCHAR(32) NOT NULL DEFAULT ''"},
		{"company_national_number", "VARCHAR(32) NOT NULL DEFAULT ''"}, {"company_tax_number", "VARCHAR(32) NOT NULL DEFAULT ''"},
		{"company_executive_name", "VARCHAR(255) NOT NULL DEFAULT ''"}, {"verification_messages", "TEXT NOT NULL DEFAULT ('')"},
	} {
		var exists int
		if err = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='personal_infos' AND column_name=?`, c.name).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			if _, err = conn.ExecContext(ctx, "ALTER TABLE personal_infos ADD `"+c.name+"` "+c.definition); err != nil {
				return err
			}
		}
	}
	for _, c := range []struct{ name, definition string }{{"mobile", "VARCHAR(11) CHARACTER SET ascii NULL"}, {"updated_at", "DATETIME(6) NULL"}} {
		var exists int
		if err = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='users' AND column_name=?`, c.name).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			if _, err = conn.ExecContext(ctx, "ALTER TABLE users ADD `"+c.name+"` "+c.definition); err != nil {
				return err
			}
		}
	}
	// Dates are normalized in UTC and stored as DATETIME(6), never compared as
	// variable-precision RFC3339 strings. The temporary update stays resumable.
	for table, cols := range map[string][]string{
		"users": {"created_at", "email_verified_at"}, "sessions": {"expires_at"}, "actions": {"expires_at"},
		"challenges": {"expires_at"}, "session_attributes": {"expires_at"}, "registration_callbacks": {"expires_at"},
	} {
		for _, col := range cols {
			if err = convertDate(ctx, conn, table, col); err != nil {
				return err
			}
		}
	}
	if _, err = conn.ExecContext(ctx, `UPDATE users SET updated_at=created_at WHERE updated_at IS NULL`); err != nil {
		return err
	}
	if _, err = conn.ExecContext(ctx, `ALTER TABLE users MODIFY updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)`); err != nil {
		return err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if err = writeProfile(ctx, tx, p); err != nil {
			tx.Rollback()
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE code_sequence SET value=GREATEST(value,COALESCE((SELECT MAX(CAST(SUBSTRING(code,4) AS UNSIGNED)) FROM users WHERE code REGEXP '^hm-[0-9]+$'),1999999)) WHERE id=1`); err != nil {
		tx.Rollback()
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	indexes := []struct{ table, name, cols string }{
		{"users", "users_cleanup", "email_verified_at,created_at"}, {"sessions", "sessions_user", "user_id"},
		{"sessions", "sessions_expiry", "expires_at"}, {"actions", "actions_expiry", "expires_at"},
		{"actions", "actions_reset_lookup", "email,kind,token_hash"}, {"challenges", "challenges_expiry", "expires_at"},
		{"session_attributes", "session_attributes_expiry", "expires_at"}, {"registration_callbacks", "registration_callbacks_expiry", "expires_at"},
	}
	for _, i := range indexes {
		var count int
		if err = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? AND index_name=?`, i.table, i.name).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			if _, err = conn.ExecContext(ctx, fmt.Sprintf("CREATE INDEX `%s` ON `%s`(%s)", i.name, i.table, i.cols)); err != nil {
				return err
			}
		}
	}
	// Explicit constraints work across MySQL versions that either ignore or enforce inline REFERENCES.
	for _, table := range []string{"personal_infos", "profile_details", "sessions", "actions", "media", "wallets", "registration_callbacks"} {
		if err = ensureFK(ctx, conn, table, "user_id", "users", "id"); err != nil {
			return err
		}
	}
	if err = ensureFK(ctx, conn, "session_attributes", "token_hash", "sessions", "token_hash"); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(2,?)`, stamp(time.Now()))
	if err != nil {
		return err
	}
	return migrateOAuth(ctx, conn)
}

func convertDate(ctx context.Context, conn *sql.Conn, table, col string) error {
	var kind string
	if err := conn.QueryRowContext(ctx, `SELECT data_type FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?`, table, col).Scan(&kind); err != nil {
		return err
	}
	if kind == "datetime" {
		return nil
	}
	rows, err := conn.QueryContext(ctx, fmt.Sprintf("SELECT DISTINCT `%s` FROM `%s` WHERE `%s` IS NOT NULL", col, table, col))
	if err != nil {
		return err
	}
	type date struct{ old, new string }
	var dates []date
	for rows.Next() {
		var old string
		if err = rows.Scan(&old); err != nil {
			rows.Close()
			return err
		}
		t, e := parseStamp(old)
		if e != nil {
			rows.Close()
			return fmt.Errorf("%s.%s: %w", table, col, e)
		}
		dates = append(dates, date{old, stamp(t)})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, d := range dates {
		if _, err = conn.ExecContext(ctx, fmt.Sprintf("UPDATE `%s` SET `%s`=? WHERE `%s`=?", table, col, col), d.new, d.old); err != nil {
			return err
		}
	}
	nullable := "NOT NULL"
	if col == "email_verified_at" {
		nullable = "NULL"
	}
	_, err = conn.ExecContext(ctx, fmt.Sprintf("ALTER TABLE `%s` MODIFY `%s` DATETIME(6) %s", table, col, nullable))
	return err
}

func ensureFK(ctx context.Context, conn *sql.Conn, table, col, parent, parentCol string) error {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.key_column_usage WHERE table_schema=DATABASE() AND table_name=? AND column_name=? AND referenced_table_name=?`, table, col, parent).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := conn.ExecContext(ctx, fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `fk_%s_%s` FOREIGN KEY (`%s`) REFERENCES `%s`(`%s`) ON DELETE CASCADE", table, table, col, col, parent, parentCol))
	if err != nil {
		return fmt.Errorf("foreign key %s.%s (check legacy orphans): %w", table, col, err)
	}
	return nil
}

func migrateOAuth(ctx context.Context, conn *sql.Conn) error {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=3`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return migrateOperations(ctx, conn)
	}
	for _, statement := range strings.Split(oauthSchema, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(3,?)`, stamp(time.Now()))
	if err != nil {
		return err
	}
	return migrateOperations(ctx, conn)
}

func preflightDates(ctx context.Context, conn *sql.Conn) error {
	for table, columns := range map[string][]string{"users": {"created_at", "email_verified_at"}, "sessions": {"expires_at"}, "actions": {"expires_at"}, "challenges": {"expires_at"}, "session_attributes": {"expires_at"}, "registration_callbacks": {"expires_at"}} {
		for _, column := range columns {
			rows, err := conn.QueryContext(ctx, fmt.Sprintf("SELECT DISTINCT `%s` FROM `%s` WHERE `%s` IS NOT NULL", column, table, column))
			if err != nil {
				return err
			}
			for rows.Next() {
				var value string
				if err = rows.Scan(&value); err != nil {
					rows.Close()
					return err
				}
				if _, err = parseStamp(value); err != nil {
					rows.Close()
					return fmt.Errorf("%s.%s: %w", table, column, err)
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
