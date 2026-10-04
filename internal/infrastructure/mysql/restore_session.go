package mysql

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"time"
)

func (s *Store) RestoreLegacySession(ctx context.Context, in application.LegacySessionRestore) (domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()
	u, err := scan(tx.QueryRowContext(ctx, `SELECT `+columns+` FROM users WHERE id=? FOR UPDATE`, in.Owner))
	if errors.Is(err, domain.ErrNotFound) {
		return u, domain.ErrCredentials
	}
	if err != nil {
		return u, err
	}
	if subtle.ConstantTimeCompare([]byte(u.PasswordHash), []byte(in.PasswordHash)) != 1 {
		return u, domain.ErrCredentials
	}
	var found int
	if in.SourceHash != "" {
		err = tx.QueryRowContext(ctx, `SELECT 1 FROM sessions WHERE user_id=? AND token_hash=? AND expires_at>? FOR UPDATE`, in.Owner, in.SourceHash, stamp(in.Now)).Scan(&found)
	} else if in.RememberHash != "" {
		err = tx.QueryRowContext(ctx, `SELECT 1 FROM legacy_remember_tokens WHERE user_id=? AND token_hash=? AND expires_at>? FOR UPDATE`, in.Owner, in.RememberHash, stamp(in.Now)).Scan(&found)
	} else {
		return u, domain.ErrCredentials
	}
	if errors.Is(err, sql.ErrNoRows) {
		return u, domain.ErrCredentials
	}
	if err != nil {
		return u, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at,idle_seconds) VALUES(?,?,?,?)`, in.NewHash, in.Owner, stamp(in.Expires), int64(in.IdleTTL/time.Second)); err != nil {
		return u, err
	}
	if in.SourceHash != "" {
		rows, e := tx.QueryContext(ctx, `SELECT name,value,expires_at FROM session_attributes WHERE token_hash=? AND name IN ('wallet_login','password_confirmed_at','csrf_token') AND expires_at>? FOR UPDATE`, in.SourceHash, stamp(in.Now))
		if e != nil {
			return u, e
		}
		type attribute struct {
			name, value string
			expiry      time.Time
		}
		var attrs []attribute
		for rows.Next() {
			var a attribute
			if e = rows.Scan(&a.name, &a.value, &a.expiry); e != nil {
				rows.Close()
				return u, e
			}
			if a.expiry.After(in.Expires) {
				a.expiry = in.Expires
			}
			attrs = append(attrs, a)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return u, e
		}
		for _, a := range attrs {
			if _, err = tx.ExecContext(ctx, `INSERT INTO session_attributes(token_hash,name,value,expires_at) VALUES(?,?,?,?)`, in.NewHash, a.name, a.value, stamp(a.expiry)); err != nil {
				return u, err
			}
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM session_attributes WHERE token_hash=? AND name='wallet_login'`, in.SourceHash); err != nil {
			return u, err
		}
	}
	return u, tx.Commit()
}
