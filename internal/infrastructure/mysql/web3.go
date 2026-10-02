package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

// Wallet, challenge and session-attribute storage for the Web3 milestone.
// Wallet ownership lives in the separate wallets table (user_id PK, address
// UNIQUE) instead of a users.wallet_address column; every mutation runs in a
// transaction with row locks, mirroring the lockForUpdate calls in
// Web3AuthController.

// --- wallets ---

func (s *Store) UserByWallet(ctx context.Context, address string) (domain.User, error) {
	return scan(s.db.QueryRowContext(ctx, "SELECT "+columns+" FROM users WHERE id=(SELECT user_id FROM wallets WHERE address=?)", address))
}

func (s *Store) UserByCode(ctx context.Context, code string) (domain.User, error) {
	return scan(s.db.QueryRowContext(ctx, "SELECT "+columns+" FROM users WHERE code=?", code))
}

// WalletOf returns the address linked to a user, or "" when none is linked.
func (s *Store) WalletOf(ctx context.Context, userID int64) (string, error) {
	var address sql.NullString
	err := s.db.QueryRowContext(ctx, "SELECT address FROM wallets WHERE user_id=?", userID).Scan(&address)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return address.String, nil
}

func (s *Store) WalletTaken(ctx context.Context, address string) (bool, error) {
	var found bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM wallets WHERE address=?)", address).Scan(&found)
	return found, err
}

func duplicateAddress(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// AttachWallet links address to userID transactionally, returning one of the
// domain.WalletLink* outcomes. Row locks serialize concurrent links the way
// lockForUpdate does in Laravel.
func (s *Store) AttachWallet(ctx context.Context, userID int64, address string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var mine string
	err = tx.QueryRowContext(ctx, "SELECT address FROM wallets WHERE user_id=? FOR UPDATE", userID).Scan(&mine)
	if err == nil {
		return domain.WalletLinkAlreadyConnected, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	var other int64
	err = tx.QueryRowContext(ctx, "SELECT user_id FROM wallets WHERE address=? FOR UPDATE", address).Scan(&other)
	if err == nil {
		return domain.WalletLinkAlreadyLinked, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO wallets(user_id,address) VALUES(?,?)", userID, address); err != nil {
		if !duplicateAddress(err) {
			return "", err
		}
		// Lost a race with a concurrent link; report the resulting state.
		if err := tx.Rollback(); err != nil {
			return "", err
		}
		if mine, err := s.WalletOf(ctx, userID); err != nil {
			return "", err
		} else if mine != "" {
			return domain.WalletLinkAlreadyConnected, nil
		}
		return domain.WalletLinkAlreadyLinked, nil
	}
	return domain.WalletLinkSuccess, tx.Commit()
}

// CreateWalletUser creates a passwordless, email-less, verified user that
// owns address, plus its personal_infos and wallets rows. The member code
// comes from the same code_sequence VerifyEmail uses. On a concurrent
// insert for the same address the winning owner is returned, matching the
// re-check inside Laravel's resolveWalletUser transaction.
func (s *Store) CreateWalletUser(ctx context.Context, address string, now time.Time) (domain.User, error) {
	var u domain.User
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return u, err
	}
	defer tx.Rollback()
	var owner int64
	err = tx.QueryRowContext(ctx, "SELECT user_id FROM wallets WHERE address=? FOR UPDATE", address).Scan(&owner)
	if err == nil {
		if err := tx.Rollback(); err != nil {
			return u, err
		}
		return s.UserByWallet(ctx, address)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return u, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE code_sequence SET value=value+1 WHERE id=1"); err != nil {
		return u, err
	}
	var code string
	if err := tx.QueryRowContext(ctx, "SELECT CONCAT('hm-', value) FROM code_sequence WHERE id=1").Scan(&code); err != nil {
		return u, err
	}
	name := "User_" + address[2:8]
	r, err := tx.ExecContext(ctx, "INSERT INTO users(name,email,password_hash,code,referral,email_verified_at,created_at) VALUES(?,NULL,NULL,?,'',?,?)",
		name, code, stamp(now), stamp(now))
	if err != nil {
		return u, constraint(err)
	}
	id, err := r.LastInsertId()
	if err != nil {
		return u, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO personal_infos(user_id) VALUES(?)", id); err != nil {
		return u, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO wallets(user_id,address) VALUES(?,?)", id, address); err != nil {
		if !duplicateAddress(err) {
			return u, err
		}
		if err := tx.Rollback(); err != nil {
			return u, err
		}
		return s.UserByWallet(ctx, address)
	}
	if err := tx.Commit(); err != nil {
		return u, err
	}
	return s.UserByWallet(ctx, address)
}

// AttachWalletByCode links address to the user holding code (the Metarang
// already_registered flow). A missing code reports domain.ErrNotFound so the
// caller answers 502 exactly like Laravel's "Registered wallet user was not
// found." lookup exception. An address claimed concurrently resolves to its
// current owner.
func (s *Store) AttachWalletByCode(ctx context.Context, address, code string) (domain.User, error) {
	var u domain.User
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return u, err
	}
	defer tx.Rollback()
	var id int64
	if err := tx.QueryRowContext(ctx, "SELECT id FROM users WHERE code=? FOR UPDATE", code).Scan(&id); errors.Is(err, sql.ErrNoRows) {
		return u, domain.ErrNotFound
	} else if err != nil {
		return u, err
	}
	// Overwrite any previous link of this user, mirroring Laravel which sets
	// wallet_address unconditionally here.
	if _, err := tx.ExecContext(ctx, "INSERT INTO wallets(user_id,address) VALUES(?,?) ON DUPLICATE KEY UPDATE address=VALUES(address)", id, address); err != nil {
		if !duplicateAddress(err) {
			return u, err
		}
		if err := tx.Rollback(); err != nil {
			return u, err
		}
		return s.UserByWallet(ctx, address)
	}
	if err := tx.Commit(); err != nil {
		return u, err
	}
	return s.UserByWallet(ctx, address)
}

// --- challenges (single-use nonces) ---

func (s *Store) SaveChallenge(ctx context.Context, key, message string, expiresAt time.Time) error {
	if _, err := s.db.ExecContext(ctx, "INSERT INTO challenges(`key`,message,expires_at) VALUES(?,?,?) ON DUPLICATE KEY UPDATE message=VALUES(message),expires_at=VALUES(expires_at)", key, message, stamp(expiresAt)); err != nil {
		return err
	}
	// Opportunistic cleanup; the table only ever holds short-lived nonces.
	_, err := s.db.ExecContext(ctx, "DELETE FROM challenges WHERE expires_at<=?", stamp(time.Now()))
	return err
}

// PullChallenge consumes a nonce exactly once, mirroring Cache::pull: the row
// is deleted whether or not it is still valid, and a missing or expired row
// reports domain.ErrNotFound.
func (s *Store) PullChallenge(ctx context.Context, key string, now time.Time) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var message, expires string
	err = tx.QueryRowContext(ctx, "SELECT message,expires_at FROM challenges WHERE `key`=? FOR UPDATE", key).Scan(&message, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM challenges WHERE `key`=?", key); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	exp, err := parseStamp(expires)
	if err != nil || !exp.After(now) {
		return "", domain.ErrNotFound
	}
	return message, nil
}

// --- session attributes (wallet_login flag and future OAuth callbacks) ---

func (s *Store) SetSessionAttribute(ctx context.Context, tokenHash, name, value string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO session_attributes(token_hash,name,value,expires_at) VALUES(?,?,?,?)
		ON DUPLICATE KEY UPDATE value=VALUES(value),expires_at=VALUES(expires_at)`, tokenHash, name, value, stamp(expiresAt))
	return err
}

func (s *Store) SessionAttribute(ctx context.Context, tokenHash, name string, now time.Time) (string, error) {
	var value, expires string
	err := s.db.QueryRowContext(ctx, "SELECT value,expires_at FROM session_attributes WHERE token_hash=? AND name=?", tokenHash, name).Scan(&value, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	exp, err := parseStamp(expires)
	if err != nil || !exp.After(now) {
		return "", domain.ErrNotFound
	}
	return value, nil
}

func (s *Store) PullSessionAttribute(ctx context.Context, hash, name string, now time.Time) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var value string
	var expiry time.Time
	err = tx.QueryRowContext(ctx, `SELECT value,expires_at FROM session_attributes WHERE token_hash=? AND name=? FOR UPDATE`, hash, name).Scan(&value, &expiry)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM session_attributes WHERE token_hash=? AND name=?`, hash, name); err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	if !expiry.After(now) {
		return "", domain.ErrNotFound
	}
	return value, nil
}
