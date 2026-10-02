package mysql

import (
	"context"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"time"
)

type CleanupResult struct {
	Users          []domain.User `json:"users"`
	DeletedUsers   int64         `json:"deleted_users"`
	ExpiredRecords int64         `json:"expired_records"`
}

// Cleanup's delete predicate is rechecked under row locks so verification
// racing with cleanup cannot delete a newly verified account.
func (s *Store) Cleanup(ctx context.Context, now time.Time, apply bool) (CleanupResult, error) {
	result := CleanupResult{Users: []domain.User{}}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT `+columns+` FROM users WHERE email_verified_at IS NULL AND created_at<? ORDER BY id FOR UPDATE`, stamp(now.Add(-24*time.Hour)))
	if err != nil {
		return result, err
	}
	for rows.Next() {
		u, e := scan(rows)
		if e != nil {
			rows.Close()
			return result, e
		}
		result.Users = append(result.Users, u)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if !apply {
		return result, nil
	}
	for _, u := range result.Users {
		r, e := tx.ExecContext(ctx, `DELETE FROM users WHERE id=? AND email_verified_at IS NULL AND created_at<?`, u.ID, stamp(now.Add(-24*time.Hour)))
		if e != nil {
			return result, e
		}
		n, e := r.RowsAffected()
		if e != nil {
			return result, e
		}
		result.DeletedUsers += n
	}
	for _, table := range []string{"sessions", "actions", "challenges", "session_attributes", "registration_callbacks", "legacy_password_resets", "legacy_remember_tokens"} {
		r, e := tx.ExecContext(ctx, "DELETE FROM `"+table+"` WHERE expires_at<=?", stamp(now))
		if e != nil {
			return result, e
		}
		n, e := r.RowsAffected()
		if e != nil {
			return result, e
		}
		result.ExpiredRecords += n
	}
	// Preserve code tombstones for refresh replay detection. Token generations
	// remain until their refresh lifetime ends.
	for _, query := range []string{`DELETE FROM oauth_tokens WHERE refresh_expires_at<=?`, `DELETE FROM oauth_codes WHERE expires_at<=? AND consumed_at IS NULL`} {
		r, e := tx.ExecContext(ctx, query, stamp(now))
		if e != nil {
			return result, e
		}
		n, e := r.RowsAffected()
		if e != nil {
			return result, e
		}
		result.ExpiredRecords += n
	}
	return result, tx.Commit()
}
