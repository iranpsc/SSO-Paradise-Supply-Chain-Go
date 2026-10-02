package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

//go:embed migrations/001_schema.sql
var schema string

//go:embed migrations/001_features.sql
var featuresSchema string

//go:embed schema.sql
var canonicalSchema string

//go:embed features.sql
var canonicalFeatures string

type Store struct {
	peerLimit int
	db        *sql.DB
	dsn       string
}

// DSN returns the connection string the store was opened with, so callers such
// as persistence tests can reopen the same database.
func (s *Store) DSN() string { return s.dsn }

func Open(dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := Migrate(context.Background(), db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, dsn: dsn}, nil
}

func (s *Store) Close() error  { return s.db.Close() }
func stamp(t time.Time) string { return t.UTC().Format("2006-01-02 15:04:05.000000") }

func parseStamp(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999"} {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid database timestamp %q", value)
}

const columns = "id,mobile,updated_at,name,COALESCE(username,''),COALESCE(email,''),COALESCE(password_hash,''),code,referral,email_verified_at,created_at,(SELECT address FROM wallets WHERE user_id=users.id)"

type scanner interface{ Scan(...any) error }

func scan(row scanner) (domain.User, error) {
	var u domain.User
	var verified sql.NullString
	var created string
	err := row.Scan(&u.ID, &u.Mobile, &u.UpdatedAt, &u.Name, &u.Username, &u.Email, &u.PasswordHash, &u.Code, &u.Referral, &verified, &created, &u.Wallet)
	if errors.Is(err, sql.ErrNoRows) {
		return u, domain.ErrNotFound
	}
	if err != nil {
		return u, err
	}
	u.CreatedAt, err = parseStamp(created)
	if err != nil {
		return u, err
	}
	if verified.Valid {
		t, e := parseStamp(verified.String)
		if e != nil {
			return u, e
		}
		u.EmailVerifiedAt = &t
	}
	return u, nil
}

func constraint(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		if strings.Contains(err.Error(), "username") {
			return domain.ErrUsernameConflict
		}
		return domain.ErrConflict
	}
	return err
}

func (s *Store) Create(ctx context.Context, u domain.User) (domain.User, error) {
	return s.CreateRegistration(ctx, u, "", time.Time{})
}
func (s *Store) CreateRegistration(ctx context.Context, u domain.User, backURL string, until time.Time) (domain.User, error) {
	u.CreatedAt = u.CreatedAt.UTC().Truncate(time.Microsecond)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return u, err
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, "INSERT INTO users(name,username,email,password_hash,referral,created_at,updated_at,mobile) VALUES(?,NULLIF(?,''),?,?,?,?,?,?)", u.Name, u.Username, u.Email, u.PasswordHash, u.Referral, stamp(u.CreatedAt), stamp(u.CreatedAt), u.Mobile)
	if err != nil {
		return u, constraint(err)
	}
	u.UpdatedAt = u.CreatedAt
	u.ID, err = r.LastInsertId()
	if err != nil {
		return u, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO personal_infos(user_id) VALUES(?)", u.ID); err != nil {
		return u, err
	}
	if backURL != "" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO registration_callbacks(user_id,url,expires_at) VALUES(?,?,?)`, u.ID, backURL, stamp(until)); err != nil {
			return u, err
		}
	}
	return u, tx.Commit()
}

func (s *Store) ByEmail(ctx context.Context, email string) (domain.User, error) {
	return scan(s.db.QueryRowContext(ctx, "SELECT "+columns+" FROM users WHERE email=?", email))
}

func (s *Store) ByLogin(ctx context.Context, identifier string) (domain.User, error) {
	if strings.Contains(identifier, "@") {
		return s.ByEmail(ctx, identifier)
	}
	return scan(s.db.QueryRowContext(ctx, "SELECT "+columns+" FROM users WHERE username=?", identifier))
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
	u, err := scan(s.db.QueryRowContext(ctx, "SELECT "+columns+" FROM users WHERE id=(SELECT user_id FROM sessions WHERE token_hash=? AND expires_at>?)", hash, stamp(now)))
	if errors.Is(err, domain.ErrNotFound) {
		err = domain.ErrCredentials
	}
	return u, err
}

func (s *Store) SessionLookup(ctx context.Context, hash string, now time.Time) (int64, time.Time, error) {
	var id int64
	var expires string
	err := s.db.QueryRowContext(ctx, "SELECT user_id,expires_at FROM sessions WHERE token_hash=? AND expires_at>?", hash, stamp(now)).Scan(&id, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, time.Time{}, domain.ErrCredentials
	}
	if err != nil {
		return 0, time.Time{}, err
	}
	exp, err := parseStamp(expires)
	if err != nil {
		return 0, time.Time{}, err
	}
	return id, exp, nil
}

func (s *Store) RevokeSessions(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var owner int64
	if err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id=? FOR UPDATE`, id).Scan(&owner); errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	} else if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id=?", id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE oauth_grants SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL", stamp(time.Now()), id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM legacy_remember_tokens WHERE user_id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SaveAction(ctx context.Context, id int64, kind, hash, email string, until time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO actions(user_id,kind,token_hash,email,expires_at) VALUES(?,?,?,?,?) ON DUPLICATE KEY UPDATE token_hash=VALUES(token_hash),email=VALUES(email),expires_at=VALUES(expires_at)`, id, kind, hash, email, stamp(until)); err != nil {
		return err
	}
	if kind == "reset" {
		if _, err = tx.ExecContext(ctx, `DELETE FROM legacy_password_resets WHERE user_id=?`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) VerifyEmail(ctx context.Context, id int64, hash string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var email string
	err = tx.QueryRowContext(ctx, `SELECT a.email FROM actions a JOIN users u ON u.id=a.user_id AND u.email=a.email
		WHERE a.user_id=? AND a.kind='verify' AND a.token_hash=? AND a.expires_at>? FOR UPDATE`, id, hash, stamp(now)).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrToken
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE code_sequence SET value=value+1 WHERE id=1"); err != nil {
		return err
	}
	var code string
	if err := tx.QueryRowContext(ctx, "SELECT CONCAT('hm-', value) FROM code_sequence WHERE id=1").Scan(&code); err != nil {
		return err
	}
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
		WHERE a.email=? AND a.kind='reset' AND a.token_hash=? AND a.expires_at>? FOR UPDATE`, email, tokenHash, stamp(now)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrToken
	}
	if err != nil {
		return err
	}
	if err = resetAccountPassword(ctx, tx, id, passwordHash, now); err != nil {
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
	_, err = tx.ExecContext(ctx, `UPDATE users SET name=?,email_verified_at=CASE WHEN email=? THEN email_verified_at ELSE NULL END,email=? WHERE id=?`, name, email, email, id)
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
	if _, err = tx.ExecContext(ctx, "UPDATE oauth_grants SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL", stamp(time.Now()), id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id=? AND token_hash<>?", id, keepSession); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM legacy_remember_tokens WHERE user_id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) PersonalInfo(ctx context.Context, id int64) (domain.PersonalInfo, error) {
	var p domain.PersonalInfo
	err := s.db.QueryRowContext(ctx, `SELECT user_id,first_name,last_name,is_verified,is_company,mobile,telephone,national_code,address,company_name,company_address,company_registration_number,company_national_number,company_tax_number,company_executive_name,verification_messages FROM personal_infos WHERE user_id=?`, id).Scan(&p.UserID, &p.FirstName, &p.LastName, &p.IsVerified, &p.IsCompany, &p.Mobile, &p.Telephone, &p.NationalCode, &p.Address, &p.CompanyName, &p.CompanyAddress, &p.CompanyRegistrationNumber, &p.CompanyNationalNumber, &p.CompanyTaxNumber, &p.CompanyExecutiveName, &p.VerificationMessages)
	if errors.Is(err, sql.ErrNoRows) {
		err = domain.ErrNotFound
	}
	return p, err
}

func saveMedia(ctx context.Context, tx *sql.Tx, m domain.Media) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO media(user_id,kind,content_type,data) VALUES(?,?,?,?) ON DUPLICATE KEY UPDATE content_type=VALUES(content_type),data=VALUES(data)`, m.UserID, m.Kind, m.ContentType, m.Data)
	return err
}

func (s *Store) SavePersonalInfo(ctx context.Context, p domain.PersonalInfo, media []domain.Media) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var verified bool
	if err = tx.QueryRowContext(ctx, "SELECT is_verified FROM personal_infos WHERE user_id=? FOR UPDATE", p.UserID).Scan(&verified); err != nil {
		return err
	}
	p.IsVerified = verified
	p.VerificationMessages = ""
	if err = writeProfile(ctx, tx, p); err != nil {
		return err
	}
	for _, m := range media {
		if err = saveMedia(ctx, tx, m); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) SaveAvatar(ctx context.Context, m domain.Media) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	m.Kind = "avatars"
	if err = saveMedia(ctx, tx, m); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Media(ctx context.Context, id int64, kind string) (domain.Media, error) {
	m := domain.Media{UserID: id, Kind: kind}
	err := s.db.QueryRowContext(ctx, "SELECT content_type,data FROM media WHERE user_id=? AND kind=?", id, kind).Scan(&m.ContentType, &m.Data)
	if errors.Is(err, sql.ErrNoRows) {
		err = domain.ErrNotFound
	}
	return m, err
}

func (s *Store) PublicProfile(ctx context.Context, id int64) (domain.PublicProfile, error) {
	var p domain.PublicProfile
	var avatar bool
	err := s.db.QueryRowContext(ctx, `SELECT u.id,CASE WHEN p.is_verified THEN CONCAT(p.first_name,' ',p.last_name) ELSE u.name END,u.code,EXISTS(SELECT 1 FROM media WHERE user_id=u.id AND kind='avatars') FROM users u JOIN personal_infos p ON p.user_id=u.id WHERE u.id=?`, id).Scan(&p.ID, &p.Name, &p.Code, &avatar)
	if errors.Is(err, sql.ErrNoRows) {
		err = domain.ErrNotFound
	}
	if avatar {
		p.Avatar = fmt.Sprintf("/api/users/%d/avatar", id)
	}
	return p, err
}

func (s *Store) PullRegistrationCallback(ctx context.Context, id int64, now time.Time) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var raw string
	var expiry time.Time
	err = tx.QueryRowContext(ctx, `SELECT url,expires_at FROM registration_callbacks WHERE user_id=? FOR UPDATE`, id).Scan(&raw, &expiry)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM registration_callbacks WHERE user_id=?`, id); err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	if !expiry.After(now) {
		return "", domain.ErrNotFound
	}
	return raw, nil
}

func (s *Store) VerifySignedEmail(ctx context.Context, id int64, email string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var verified sql.NullTime
	var stored string
	if err = tx.QueryRowContext(ctx, `SELECT email,email_verified_at FROM users WHERE id=? FOR UPDATE`, id).Scan(&stored, &verified); err != nil {
		return err
	}
	if stored != email {
		return domain.ErrToken
	}
	if verified.Valid {
		return nil
	}
	if _, err = tx.ExecContext(ctx, `UPDATE code_sequence SET value=value+1 WHERE id=1`); err != nil {
		return err
	}
	var code string
	if err = tx.QueryRowContext(ctx, `SELECT CONCAT('hm-',value) FROM code_sequence WHERE id=1`).Scan(&code); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE users SET email_verified_at=?,code=COALESCE(code,?) WHERE id=?`, stamp(now), code, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM actions WHERE user_id=? AND kind='verify'`, id); err != nil {
		return err
	}
	return tx.Commit()
}
