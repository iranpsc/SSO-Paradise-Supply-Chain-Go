package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"strings"
	"time"
)

type importedGrant struct {
	ID             string
	User, Client   int64
	Scopes         []string
	Revoked        bool
	Expiry         time.Time
	Refresh        string
	RefreshExpiry  time.Time
	RefreshRevoked bool
	Code           bool
}

func readPassportGrants(ctx context.Context, source *sql.Tx, users map[int64]bool, clients []importedClient) ([]importedGrant, error) {
	var entries []importedGrant
	available := map[string]bool{}
	for _, table := range []string{"oauth_access_tokens", "oauth_refresh_tokens", "oauth_auth_codes"} {
		var n int
		if err := source.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?`, table).Scan(&n); err != nil {
			return nil, err
		}
		available[table] = n > 0
	}
	clientSet := map[int64]bool{}
	for _, c := range clients {
		clientSet[c.id] = true
	}
	access := map[string]int{}
	for _, table := range []string{"oauth_access_tokens", "oauth_auth_codes"} {
		if !available[table] {
			continue
		}
		rows, err := source.QueryContext(ctx, `SELECT id,user_id,client_id,COALESCE(scopes,'[]'),revoked,expires_at FROM `+table)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var g importedGrant
			var raw string
			var owner sql.NullInt64
			var expiry sql.NullTime
			if err = rows.Scan(&g.ID, &owner, &g.Client, &raw, &g.Revoked, &expiry); err != nil {
				rows.Close()
				return nil, err
			}
			if !owner.Valid {
				rows.Close()
				return nil, fmt.Errorf("unsupported client-credentials token in source; no user owner")
			}
			g.User = owner.Int64
			if !users[g.User] || !clientSet[g.Client] {
				rows.Close()
				return nil, fmt.Errorf("orphan Passport grant")
			}
			if (len(g.ID) != 80 && len(g.ID) != 64) || strings.Trim(g.ID, "0123456789abcdef") != "" {
				rows.Close()
				return nil, fmt.Errorf("unsupported Passport identifier")
			}
			if !expiry.Valid {
				rows.Close()
				return nil, fmt.Errorf("Passport grant without expiry")
			}
			g.Expiry = expiry.Time
			if err = json.Unmarshal([]byte(raw), &g.Scopes); err != nil {
				rows.Close()
				return nil, fmt.Errorf("malformed Passport scopes")
			}
			seen := map[string]bool{}
			for _, scope := range g.Scopes {
				if len(scope) == 0 || len(scope) > 32 || seen[scope] || strings.IndexFunc(scope, func(r rune) bool { return r > 127 }) >= 0 {
					rows.Close()
					return nil, fmt.Errorf("unsupported Passport scopes")
				}
				seen[scope] = true
			}
			g.Code = table == "oauth_auth_codes"
			if !g.Code {
				access[g.ID] = len(entries)
			}
			entries = append(entries, g)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	if available["oauth_refresh_tokens"] {
		rows, err := source.QueryContext(ctx, `SELECT id,access_token_id,revoked,expires_at FROM oauth_refresh_tokens`)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id, owner string
			var revoked bool
			var expiry sql.NullTime
			if err = rows.Scan(&id, &owner, &revoked, &expiry); err != nil {
				rows.Close()
				return nil, err
			}
			index, ok := access[owner]
			if !ok {
				rows.Close()
				return nil, fmt.Errorf("orphan Passport refresh token")
			}
			if entries[index].Refresh != "" {
				rows.Close()
				return nil, fmt.Errorf("multiple refresh tokens for one Passport access token")
			}
			if len(id) != 80 || strings.Trim(id, "0123456789abcdef") != "" || !expiry.Valid {
				rows.Close()
				return nil, fmt.Errorf("unsupported Passport refresh token")
			}
			entries[index].Refresh = id
			entries[index].RefreshExpiry = expiry.Time
			entries[index].RefreshRevoked = revoked
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return entries, nil
}
func importPassportGrants(ctx context.Context, tx *sql.Tx, entries []importedGrant) error {
	for _, g := range entries {
		var revoked any
		if g.Revoked {
			revoked = stamp(time.Now())
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO oauth_grants(user_id,client_id,revoked_at) VALUES(?,?,?)`, g.User, g.Client, revoked)
		if err != nil {
			return err
		}
		grant, err := result.LastInsertId()
		if err != nil {
			return err
		}
		for _, scope := range g.Scopes {
			if _, err = tx.ExecContext(ctx, `INSERT INTO oauth_grant_scopes(grant_id,scope) VALUES(?,?)`, grant, scope); err != nil {
				return err
			}
		}
		if g.Code {
			_, err = tx.ExecContext(ctx, `INSERT INTO oauth_codes(token_hash,grant_id,redirect_uri,challenge,expires_at,legacy) VALUES(?,?,'','',?,1)`, application.Digest(g.ID), grant, stamp(g.Expiry))
		} else {
			refresh := g.Refresh
			expiry := g.RefreshExpiry
			var consumed any
			if refresh == "" {
				refresh = "no-refresh:" + g.ID
				expiry = g.Expiry
			}
			if g.RefreshRevoked {
				consumed = stamp(time.Now())
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO oauth_tokens(access_hash,refresh_hash,grant_id,access_expires_at,refresh_expires_at,consumed_at) VALUES(?,?,?,?,?,?)`, application.Digest(g.ID), application.Digest(refresh), grant, stamp(g.Expiry), stamp(expiry), consumed)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
