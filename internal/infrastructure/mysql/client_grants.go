package mysql

import (
	"context"
	"database/sql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"time"
)

func migrateClientGrants(ctx context.Context, conn *sql.Conn) error {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=9`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='oauth_clients' AND column_name='grant_types'`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := conn.ExecContext(ctx, `ALTER TABLE oauth_clients ADD COLUMN grant_types JSON NULL`); err != nil {
			return err
		}
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(9,?)`, stamp(time.Now()))
	return err
}

func (s *Store) IssuePasswordOAuthToken(ctx context.Context, userID, clientID int64, p application.TokenPair) (application.GrantIdentity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return application.GrantIdentity{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO oauth_grants(user_id,client_id) VALUES(?,?)`, userID, clientID)
	if err != nil {
		return application.GrantIdentity{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return application.GrantIdentity{}, err
	}
	if err = insertOAuthTokens(ctx, tx, id, p); err != nil {
		return application.GrantIdentity{}, err
	}
	return application.GrantIdentity{UserID: userID, Scopes: []string{}}, tx.Commit()
}
