package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, stmt := range []string{"PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=5000", "PRAGMA journal_mode=WAL", schema, featuresSchema} {
		if _, err := db.Exec(stmt); err != nil {
			db.Close()
			return nil, err
		}
	}
	if err := migrateUserIdentity(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db}, nil
}

// Remove the legacy username column and make name required without losing
// existing identities or references from related tables.
func migrateUserIdentity(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		return err
	}
	var hasUsername, nameRequired, hasName bool
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, dataType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return err
		}
		switch name {
		case "username":
			hasUsername = true
		case "name":
			hasName = true
			nameRequired = notNull != 0
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !hasName {
		return fmt.Errorf("migration blocked: users.name column is missing")
	}
	if !hasUsername && nameRequired {
		return nil
	}

	if _, err := db.Exec("PRAGMA foreign_keys=OFF"); err != nil {
		return err
	}
	migrationErr := func() error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		nameExpression := `COALESCE(NULLIF(TRIM(name),''),NULLIF(CASE WHEN INSTR(email,'@')>1 THEN SUBSTR(email,1,INSTR(email,'@')-1) END,''),'user_'||id)`
		if hasUsername {
			nameExpression = `COALESCE(NULLIF(TRIM(name),''),NULLIF(TRIM(username),''),NULLIF(CASE WHEN INSTR(email,'@')>1 THEN SUBSTR(email,1,INSTR(email,'@')-1) END,''),'user_'||id)`
		}
		if _, err := tx.Exec(`UPDATE users SET name=` + nameExpression + ` WHERE name IS NULL OR TRIM(name)=''`); err != nil {
			return err
		}
		if _, err := tx.Exec(`CREATE TABLE users_identity_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL COLLATE NOCASE UNIQUE,
			password_hash TEXT NOT NULL,
			code TEXT UNIQUE,
			referral TEXT NOT NULL DEFAULT '',
			email_verified_at TEXT,
			created_at TEXT NOT NULL
		)`); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO users_identity_new(id,name,email,password_hash,code,referral,email_verified_at,created_at)
			SELECT id,name,email,password_hash,code,referral,email_verified_at,created_at FROM users`); err != nil {
			return err
		}
		if _, err := tx.Exec("DROP TABLE users"); err != nil {
			return err
		}
		if _, err := tx.Exec("ALTER TABLE users_identity_new RENAME TO users"); err != nil {
			return err
		}
		return tx.Commit()
	}()
	_, foreignKeysErr := db.Exec("PRAGMA foreign_keys=ON")
	if migrationErr != nil {
		return migrationErr
	}
	return foreignKeysErr
}
func (s *Store) Close() error  { return s.db.Close() }
func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

const columns = "id,COALESCE(name,''),email,password_hash,code,referral,email_verified_at,created_at"

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.User, error) {
	var u domain.User
	var verified sql.NullString
	var created string
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Code, &u.Referral, &verified, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return u, domain.ErrNotFound
	}
	if err != nil {
		return u, err
	}
	u.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return u, err
	}
	if verified.Valid {
		t, e := time.Parse(time.RFC3339Nano, verified.String)
		if e != nil {
			return u, e
		}
		u.EmailVerifiedAt = &t
	}
	return u, nil
}
func constraint(err error) error {
	var coded interface{ Code() int }
	if errors.As(err, &coded) && (coded.Code() == 2067 || coded.Code() == 1555) {
		return domain.ErrConflict
	}
	return err
}
func (s *Store) Create(ctx context.Context, u domain.User) (domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return u, err
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, "INSERT INTO users(name,email,password_hash,referral,created_at) VALUES(?,?,?,?,?)", u.Name, u.Email, u.PasswordHash, u.Referral, stamp(u.CreatedAt))
	if err != nil {
		return u, constraint(err)
	}
	u.ID, err = r.LastInsertId()
	if err != nil {
		return u, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO personal_infos(user_id) VALUES(?)", u.ID); err != nil {
		return u, err
	}
	return u, tx.Commit()
}
func (s *Store) ByEmail(ctx context.Context, email string) (domain.User, error) {
	return scan(s.db.QueryRowContext(ctx, "SELECT "+columns+" FROM users WHERE email=?", email))
}
func (s *Store) ByLogin(ctx context.Context, identifier string) (domain.User, error) {
	return s.ByEmail(ctx, identifier)
}
func (s *Store) ByID(ctx context.Context, id int64) (domain.User, error) {
	return scan(s.db.QueryRowContext(ctx, "SELECT "+columns+" FROM users WHERE id=?", id))
}
func (s *Store) CodeExists(ctx context.Context, code string) (bool, error) {
	var found bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE code=?)", code).Scan(&found)
	return found, err
}
func (s *Store) CreateSession(ctx context.Context, id int64, hash string, until time.Time) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO sessions(user_id,token_hash,expires_at) VALUES(?,?,?)", id, hash, stamp(until))
	return err
}
func (s *Store) SessionUser(ctx context.Context, hash string, now time.Time) (domain.User, error) {
	u, err := scan(s.db.QueryRowContext(ctx, "SELECT "+columns+" FROM users WHERE id=(SELECT user_id FROM sessions WHERE token_hash=? AND julianday(expires_at)>julianday(?))", hash, stamp(now)))
	if errors.Is(err, domain.ErrNotFound) {
		err = domain.ErrCredentials
	}
	return u, err
}

// SessionLookup returns the owner and expiry of a session without loading
// the user row, so caches can store the token->user mapping with the right TTL.
func (s *Store) SessionLookup(ctx context.Context, hash string, now time.Time) (int64, time.Time, error) {
	var id int64
	var expires string
	err := s.db.QueryRowContext(ctx, "SELECT user_id,expires_at FROM sessions WHERE token_hash=? AND julianday(expires_at)>julianday(?)", hash, stamp(now)).Scan(&id, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, time.Time{}, domain.ErrCredentials
	}
	if err != nil {
		return 0, time.Time{}, err
	}
	exp, err := time.Parse(time.RFC3339Nano, expires)
	if err != nil {
		return 0, time.Time{}, err
	}
	return id, exp, nil
}
func (s *Store) RevokeSessions(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id=?", id)
	return err
}
func (s *Store) SaveAction(ctx context.Context, id int64, kind, hash, email string, until time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO actions(user_id,kind,token_hash,email,expires_at) VALUES(?,?,?,?,?)
	 ON CONFLICT(user_id,kind) DO UPDATE SET token_hash=excluded.token_hash,email=excluded.email,expires_at=excluded.expires_at`, id, kind, hash, email, stamp(until))
	return err
}
func (s *Store) VerifyEmail(ctx context.Context, id int64, hash string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var email string
	err = tx.QueryRowContext(ctx, `SELECT a.email FROM actions a JOIN users u ON u.id=a.user_id AND u.email=a.email
	 WHERE a.user_id=? AND a.kind='verify' AND a.token_hash=? AND julianday(a.expires_at)>julianday(?)`, id, hash, stamp(now)).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrToken
	}
	if err != nil {
		return err
	}
	var nextValue int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(CAST(substr(code,4) AS INTEGER)),1999999)+1 FROM users WHERE code LIKE 'hm-%'`).Scan(&nextValue); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO code_sequence(id,value) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET value=excluded.value`, nextValue); err != nil {
		return err
	}
	code := fmt.Sprintf("hm-%d", nextValue)
	if _, err = tx.ExecContext(ctx, "UPDATE users SET email_verified_at=?,code=COALESCE(code,?) WHERE id=?", stamp(now), code, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM actions WHERE user_id=? AND kind='verify'", id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) ResetPassword(ctx context.Context, email, tokenHash, passwordHash string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `SELECT a.user_id FROM actions a JOIN users u ON u.id=a.user_id AND u.email=a.email
	 WHERE a.email=? COLLATE NOCASE AND a.kind='reset' AND a.token_hash=? AND julianday(a.expires_at)>julianday(?)`, email, tokenHash, stamp(now)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrToken
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE users SET password_hash=? WHERE id=?", passwordHash, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id=?", id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM actions WHERE user_id=? AND kind='reset'", id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) UpdateIdentity(ctx context.Context, id int64, name, email string) (domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `UPDATE users SET name=?,email_verified_at=CASE WHEN email=? COLLATE NOCASE THEN email_verified_at ELSE NULL END,email=? WHERE id=?`, name, email, email, id)
	if err != nil {
		return domain.User{}, constraint(err)
	}
	u, err := scan(tx.QueryRowContext(ctx, "SELECT "+columns+" FROM users WHERE id=?", id))
	if err != nil {
		return u, err
	}
	return u, tx.Commit()
}
func (s *Store) ChangePassword(ctx context.Context, id int64, hash, keepSession string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "UPDATE users SET password_hash=? WHERE id=?", hash, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("change password: %w", domain.ErrNotFound)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id=? AND token_hash<>?", id, keepSession); err != nil {
		return err
	}
	return tx.Commit()
}
