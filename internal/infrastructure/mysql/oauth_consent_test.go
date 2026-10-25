package mysql_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
)

func TestGrantedOAuthScopesRequireActiveUserClientAccessTokens(t *testing.T) {
	store := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	user, err := store.Create(ctx, domain.User{Name: "Consent", Email: "consent@example.test", CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("mysql", store.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	exec := func(query string, args ...any) sql.Result {
		t.Helper()
		result, err := raw.ExecContext(ctx, query, args...)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	for _, tc := range []struct {
		name                                                                        string
		requested                                                                   []string
		grants                                                                      [][]string
		expired, consumed, revoked, clientRevoked, codeOnly, otherUser, otherClient bool
		want                                                                        bool
	}{
		{name: "no authorization", want: false},
		{name: "approval without token", grants: [][]string{{}}, codeOnly: true},
		{name: "active unscoped token", grants: [][]string{{}}, want: true},
		{name: "expired access with valid refresh", grants: [][]string{{}}, expired: true},
		{name: "rotated token", grants: [][]string{{}}, consumed: true},
		{name: "revoked grant", grants: [][]string{{}}, revoked: true},
		{name: "revoked client", grants: [][]string{{}}, clientRevoked: true},
		{name: "another user", grants: [][]string{{}}, otherUser: true},
		{name: "another client", grants: [][]string{{}}, otherClient: true},
		{name: "scope covered", grants: [][]string{{"profile"}}, requested: []string{"profile"}, want: true},
		{name: "scope missing", grants: [][]string{{"profile"}}, requested: []string{"profile", "email"}},
		{name: "scope union", grants: [][]string{{"profile"}, {"email"}}, requested: []string{"profile", "email"}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clientID, err := store.CreateOAuthClient(ctx, tc.name, "", []string{"http://localhost/callback"})
			if err != nil {
				t.Fatal(err)
			}
			owner := user.ID
			if tc.otherUser {
				other, err := store.Create(ctx, domain.User{Name: "Other", Email: "other@example.test", CreatedAt: now})
				if err != nil {
					t.Fatal(err)
				}
				owner = other.ID
			}
			grantClient := clientID
			if tc.otherClient {
				grantClient, err = store.CreateOAuthClient(ctx, "Other client", "", []string{"http://localhost/other"})
				if err != nil {
					t.Fatal(err)
				}
			}
			if tc.clientRevoked {
				exec(`UPDATE oauth_clients SET revoked=1 WHERE id=?`, clientID)
			}
			for i, scopes := range tc.grants {
				result := exec(`INSERT INTO oauth_grants(user_id,client_id) VALUES(?,?)`, owner, grantClient)
				grantID, err := result.LastInsertId()
				if err != nil {
					t.Fatal(err)
				}
				if tc.revoked {
					exec(`UPDATE oauth_grants SET revoked_at=? WHERE id=?`, now, grantID)
				}
				for _, scope := range scopes {
					exec(`INSERT INTO oauth_grant_scopes(grant_id,scope) VALUES(?,?)`, grantID, scope)
				}
				if tc.codeOnly {
					continue
				}
				expires := now.Add(time.Hour)
				if tc.expired {
					expires = now
				}
				var consumed any
				if tc.consumed {
					consumed = now
				}
				exec(`INSERT INTO oauth_tokens(access_hash,refresh_hash,grant_id,access_expires_at,refresh_expires_at,consumed_at) VALUES(?,?,?,?,?,?)`,
					application.Digest(fmt.Sprintf("access-%d-%d", clientID, i)), application.Digest(fmt.Sprintf("refresh-%d-%d", clientID, i)), grantID, expires, now.Add(24*time.Hour), consumed)
			}
			got, err := store.HasGrantedOAuthScopes(ctx, user.ID, clientID, tc.requested, now)
			if err != nil || got != tc.want {
				t.Fatalf("got %t, %v; want %t", got, err, tc.want)
			}
		})
	}
}
