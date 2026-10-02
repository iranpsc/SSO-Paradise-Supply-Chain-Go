package mysql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"time"
)

func (s *Store) ResetLegacyPassword(ctx context.Context, email, token, password string, now time.Time) error {
	if len(token) != 64 {
		return domain.ErrToken
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	var hash string
	err = tx.QueryRowContext(ctx, `SELECT r.user_id,r.token_hash FROM legacy_password_resets r JOIN users u ON u.id=r.user_id WHERE u.email=? AND r.expires_at>? FOR UPDATE`, email, stamp(now)).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrToken
	}
	if err != nil {
		return err
	}
	if !(security.Bcrypt{}).Matches(hash, token) {
		return domain.ErrToken
	}
	if err = resetAccountPassword(ctx, tx, id, password, now); err != nil {
		return err
	}
	return tx.Commit()
}
func resetAccountPassword(ctx context.Context, tx *sql.Tx, id int64, password string, now time.Time) error {
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{`UPDATE users SET password_hash=? WHERE id=?`, []any{password, id}},
		{`DELETE FROM sessions WHERE user_id=?`, []any{id}},
		{`UPDATE oauth_grants SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL`, []any{stamp(now), id}},
		{`DELETE FROM actions WHERE user_id=? AND kind='reset'`, []any{id}},
		{`DELETE FROM legacy_password_resets WHERE user_id=?`, []any{id}},
		{`DELETE FROM legacy_remember_tokens WHERE user_id=?`, []any{id}},
	} {
		if _, err := tx.ExecContext(ctx, statement.sql, statement.args...); err != nil {
			return err
		}
	}
	return nil
}
