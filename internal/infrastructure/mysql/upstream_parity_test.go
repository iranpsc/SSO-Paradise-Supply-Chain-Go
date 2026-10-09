package mysql_test

import (
	"context"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
)

func TestUpstreamMigrationPreservesCodesAndRequiresConsentForUnknownOwnership(t *testing.T) {
	store := mysqltest.Open(t)
	ctx := context.Background()
	id, err := store.CreateOAuthClient(ctx, "Existing", "secret-hash", []string{"https://client.example/callback"})
	if err != nil {
		t.Fatal(err)
	}
	raw := rawDB(t, store.DSN())
	for _, query := range []string{
		`DELETE FROM schema_migrations WHERE version>=10`,
		`ALTER TABLE oauth_clients DROP COLUMN first_party`,
		`UPDATE code_sequence SET value=2000000 WHERE id=1`,
	} {
		if _, err := raw.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	if err := mysql.Migrate(ctx, raw); err != nil {
		t.Fatal(err)
	}
	client, err := store.OAuthClient(ctx, id)
	if err != nil || client.FirstParty || client.SkipsAuthorization() {
		t.Fatal("old unknown ownership skipped consent", client, err)
	}
	id, err = store.CreateOAuthClient(ctx, "New internal", "secret-hash", []string{"https://client.example/callback"})
	if err != nil {
		t.Fatal(err)
	}
	client, err = store.OAuthClient(ctx, id)
	if err != nil || !client.SkipsAuthorization() {
		t.Fatal("new internal client classification lost", client, err)
	}
	var sequence int64
	if err := raw.QueryRow(`SELECT value FROM code_sequence WHERE id=1`).Scan(&sequence); err != nil || sequence != 1999999 {
		t.Fatal("first member code floor", sequence, err)
	}
	code := "hm-2000000"
	u, err := store.Create(ctx, domain.User{Name: "Existing member", Email: "member@example.com", Code: &code, CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`UPDATE users SET code=? WHERE id=?`, code, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`UPDATE code_sequence SET value=2000000 WHERE id=1; DELETE FROM schema_migrations WHERE version=11`); err != nil {
		t.Fatal(err)
	}
	if err := mysql.Migrate(ctx, raw); err != nil {
		t.Fatal(err)
	}
	fresh, err := store.ByID(ctx, u.ID)
	if err != nil || fresh.Code == nil || *fresh.Code != code {
		t.Fatal("existing code changed", fresh, err)
	}
	if err := raw.QueryRow(`SELECT value FROM code_sequence WHERE id=1`).Scan(&sequence); err != nil || sequence != 2000000 {
		t.Fatal("allocated sequence reset", sequence, err)
	}
}
