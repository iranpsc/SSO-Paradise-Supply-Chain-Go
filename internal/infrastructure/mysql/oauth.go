package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

//go:embed oauth.sql
var oauthSchema string

func (s *Store) OAuthClient(ctx context.Context, id int64) (application.OAuthClient, error) {
	c := application.OAuthClient{ID: id}
	var grants []byte
	var purpose sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT name,COALESCE(secret_hash,''),revoked,grant_types,purpose,first_party FROM oauth_clients WHERE id=?`, id).Scan(&c.Name, &c.SecretHash, &c.Revoked, &grants, &purpose, &c.FirstParty)
	if errors.Is(err, sql.ErrNoRows) {
		return c, domain.ErrNotFound
	}
	if err != nil {
		return c, err
	}
	if grants != nil {
		if err = json.Unmarshal(grants, &c.GrantTypes); err != nil {
			return c, err
		}
	} else if purpose.String == "personal_access" {
		c.GrantTypes = []string{"personal_access"}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT uri FROM oauth_client_redirects WHERE client_id=? ORDER BY uri`, id)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		if err = rows.Scan(&uri); err != nil {
			return c, err
		}
		c.Redirects = append(c.Redirects, uri)
	}
	return c, rows.Err()
}
func (s *Store) CreateOAuthClient(ctx context.Context, name, hash string, redirects []string) (int64, error) {
	if len(name) == 0 || len([]rune(name)) > 255 || len(redirects) == 0 {
		return 0, application.OAuthInvalidRequest
	}
	for _, raw := range redirects {
		u, e := url.Parse(raw)
		if e != nil || len(raw) > 2048 || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && (u.Scheme != "http" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1"))) {
			return 0, fmt.Errorf("invalid OAuth redirect URI")
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO oauth_clients(name,secret_hash,created_at) VALUES(?,NULLIF(?,''),?)`, name, hash, stamp(time.Now()))
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, uri := range redirects {
		if _, err = tx.ExecContext(ctx, `INSERT INTO oauth_client_redirects(client_id,uri_hash,uri) VALUES(?,?,?)`, id, application.Digest(uri), uri); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}
func (s *Store) SaveOAuthCode(ctx context.Context, c application.OAuthCode) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO oauth_grants(user_id,client_id) VALUES(?,?)`, c.UserID, c.ClientID)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	for _, scope := range c.Scopes {
		if _, err = tx.ExecContext(ctx, `INSERT INTO oauth_grant_scopes(grant_id,scope) VALUES(?,?)`, id, scope); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO oauth_codes(token_hash,grant_id,redirect_uri,challenge,expires_at) VALUES(?,?,?,?,?)`, c.Hash, id, c.RedirectURI, c.Challenge, stamp(c.ExpiresAt)); err != nil {
		return err
	}
	return tx.Commit()
}
func insertOAuthTokens(ctx context.Context, tx *sql.Tx, grantID int64, p application.TokenPair) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO oauth_tokens(access_hash,refresh_hash,grant_id,access_expires_at,refresh_expires_at) VALUES(?,?,?,?,?)`, p.AccessHash, p.RefreshHash, grantID, stamp(p.AccessExpiry), stamp(p.RefreshExpiry))
	return err
}
func grantScopes(ctx context.Context, tx *sql.Tx, id int64) (application.GrantIdentity, error) {
	rows, err := tx.QueryContext(ctx, `SELECT scope FROM oauth_grant_scopes WHERE grant_id=? ORDER BY scope`, id)
	if err != nil {
		return application.GrantIdentity{}, err
	}
	defer rows.Close()
	scopes := []string{}
	for rows.Next() {
		var scope string
		if err = rows.Scan(&scope); err != nil {
			return application.GrantIdentity{}, err
		}
		scopes = append(scopes, scope)
	}
	if err = rows.Close(); err != nil {
		return application.GrantIdentity{}, err
	}
	var userID int64
	if err = tx.QueryRowContext(ctx, `SELECT user_id FROM oauth_grants WHERE id=?`, id).Scan(&userID); err != nil {
		return application.GrantIdentity{}, err
	}
	return application.GrantIdentity{UserID: userID, Scopes: scopes}, rows.Err()
}
func revokeReplay(ctx context.Context, tx *sql.Tx, id int64, now time.Time) (application.GrantIdentity, error) {
	if _, err := tx.ExecContext(ctx, `UPDATE oauth_grants SET revoked_at=? WHERE id=?`, stamp(now), id); err != nil {
		return application.GrantIdentity{}, err
	}
	if err := tx.Commit(); err != nil {
		return application.GrantIdentity{}, err
	}
	return application.GrantIdentity{}, application.OAuthInvalidGrant
}
func (s *Store) ExchangeOAuthCode(ctx context.Context, hash string, clientID int64, redirect, challenge string, p application.TokenPair, now time.Time) (application.GrantIdentity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return application.GrantIdentity{}, err
	}
	defer tx.Rollback()
	var id, owner int64
	var uri, expected string
	var expiry time.Time
	var consumed, revoked sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT c.grant_id,g.client_id,c.redirect_uri,c.challenge,c.expires_at,c.consumed_at,g.revoked_at FROM oauth_codes c JOIN oauth_grants g ON g.id=c.grant_id WHERE c.token_hash=? FOR UPDATE`, hash).Scan(&id, &owner, &uri, &expected, &expiry, &consumed, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return application.GrantIdentity{}, application.OAuthInvalidGrant
	}
	if err != nil {
		return application.GrantIdentity{}, err
	}
	if owner != clientID || uri != redirect || !application.EqualChallenge(expected, challenge) {
		return application.GrantIdentity{}, application.OAuthInvalidGrant
	}
	if consumed.Valid {
		return revokeReplay(ctx, tx, id, now)
	}
	if revoked.Valid || !expiry.After(now) {
		return application.GrantIdentity{}, application.OAuthInvalidGrant
	}
	if _, err = tx.ExecContext(ctx, `UPDATE oauth_codes SET consumed_at=? WHERE token_hash=?`, stamp(now), hash); err != nil {
		return application.GrantIdentity{}, err
	}
	if err = insertOAuthTokens(ctx, tx, id, p); err != nil {
		return application.GrantIdentity{}, err
	}
	scopes, err := grantScopes(ctx, tx, id)
	if err != nil {
		return application.GrantIdentity{}, err
	}
	return scopes, tx.Commit()
}
func (s *Store) RotateOAuthRefresh(ctx context.Context, hash string, clientID int64, p application.TokenPair, now time.Time) (application.GrantIdentity, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return application.GrantIdentity{}, err
	}
	defer tx.Rollback()
	var id, owner int64
	var expiry time.Time
	var consumed, revoked sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT t.grant_id,g.client_id,t.refresh_expires_at,t.consumed_at,g.revoked_at FROM oauth_tokens t JOIN oauth_grants g ON g.id=t.grant_id WHERE t.refresh_hash=? FOR UPDATE`, hash).Scan(&id, &owner, &expiry, &consumed, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return application.GrantIdentity{}, application.OAuthInvalidGrant
	}
	if err != nil {
		return application.GrantIdentity{}, err
	}
	if owner != clientID {
		return application.GrantIdentity{}, application.OAuthInvalidGrant
	}
	if consumed.Valid {
		return revokeReplay(ctx, tx, id, now)
	}
	if revoked.Valid || !expiry.After(now) {
		return application.GrantIdentity{}, application.OAuthInvalidGrant
	}
	if _, err = tx.ExecContext(ctx, `UPDATE oauth_tokens SET consumed_at=? WHERE refresh_hash=?`, stamp(now), hash); err != nil {
		return application.GrantIdentity{}, err
	}
	if err = insertOAuthTokens(ctx, tx, id, p); err != nil {
		return application.GrantIdentity{}, err
	}
	scopes, err := grantScopes(ctx, tx, id)
	if err != nil {
		return application.GrantIdentity{}, err
	}
	return scopes, tx.Commit()
}
func (s *Store) OAuthUser(ctx context.Context, hash, scope string, now time.Time) (domain.User, error) {
	u, err := scan(s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM users WHERE id=(SELECT g.user_id FROM oauth_tokens t JOIN oauth_grants g ON g.id=t.grant_id JOIN oauth_clients c ON c.id=g.client_id WHERE t.access_hash=? AND t.access_expires_at>? AND t.consumed_at IS NULL AND g.revoked_at IS NULL AND c.revoked=0 AND (?='' OR EXISTS(SELECT 1 FROM oauth_grant_scopes WHERE grant_id=g.id AND scope=?)))`, hash, stamp(now), scope, scope))
	if errors.Is(err, domain.ErrNotFound) {
		err = domain.ErrCredentials
	}
	return u, err
}
func (s *Store) RevokeOAuthToken(ctx context.Context, clientID int64, hash string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE oauth_grants SET revoked_at=? WHERE client_id=? AND id IN (SELECT grant_id FROM oauth_tokens WHERE access_hash=? OR refresh_hash=?)`, stamp(time.Now()), clientID, hash, hash)
	return err
}

func (s *Store) PersonalOAuthClient(ctx context.Context) (int64, error) {
	_, err := s.db.ExecContext(ctx, `INSERT INTO oauth_clients(name,purpose,created_at) VALUES('SSO personal access','personal_access',?) ON DUPLICATE KEY UPDATE id=id`, stamp(time.Now()))
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM oauth_clients WHERE purpose='personal_access' AND revoked=0`).Scan(&id)
	return id, err
}
func (s *Store) IssuePersonalOAuthToken(ctx context.Context, userID, clientID int64, p application.TokenPair) (application.GrantIdentity, error) {
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

func (s *Store) BindLegacyCode(ctx context.Context, hash string, clientID int64, redirect, challenge string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE oauth_codes SET redirect_uri=?,challenge=?,legacy=0 WHERE token_hash=? AND legacy=1 AND grant_id IN (SELECT id FROM oauth_grants WHERE client_id=?)`, redirect, challenge, hash, clientID)
	return err
}
