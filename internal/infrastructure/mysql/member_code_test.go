package mysql_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
)

func TestCreateWalletUserUsesLatestUserCode(t *testing.T) {
	store := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now().UTC()

	raw, err := sql.Open("mysql", store.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	if _, err = raw.Exec(`INSERT INTO users(name,email,code,created_at,updated_at) VALUES
		('User 1','u1@example.com','hm-2000785',?,?),
		('User 2','u2@example.com','hm-2000786',?,?)`,
		now.Format("2006-01-02 15:04:05.000000"), now.Format("2006-01-02 15:04:05.000000"),
		now.Format("2006-01-02 15:04:05.000000"), now.Format("2006-01-02 15:04:05.000000")); err != nil {
		t.Fatal(err)
	}

	if _, err = raw.Exec(`UPDATE code_sequence SET value=1999999 WHERE id=1`); err != nil {
		t.Fatal(err)
	}

	u, err := store.CreateWalletUser(ctx, "0x1111111111111111111111111111111111111111", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if u.Code == nil || *u.Code != "hm-2000787" {
		t.Fatalf("unexpected next code: got %#v want %q", u.Code, "hm-2000787")
	}
}
