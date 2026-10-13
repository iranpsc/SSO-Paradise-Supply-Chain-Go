package mysql_test

import (
	"context"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
)

func TestCredentialTableNamesUpgradeAndCollision(t *testing.T) {
	for _, collision := range []bool{false, true} {
		t.Run(map[bool]string{false: "upgrade", true: "collision"}[collision], func(t *testing.T) {
			store := mysqltest.Open(t)
			db := rawDB(t, store.DSN())
			for _, q := range []string{
				`INSERT INTO users(id,name,email,password_hash,created_at,updated_at) VALUES(90001,'Rename test','rename@example.com','hash',NOW(6),NOW(6))`,
				`INSERT INTO password_reset_tokens(user_id,token_hash,expires_at) VALUES(90001,'reset-hash','2030-01-01')`,
				`INSERT INTO remember_tokens(user_id,token_hash,expires_at) VALUES(90001,REPEAT('a',64),'2030-01-01')`,
				`RENAME TABLE password_reset_tokens TO legacy_password_resets, remember_tokens TO legacy_remember_tokens`,
				`DELETE FROM schema_migrations WHERE version=13`,
			} {
				if _, err := db.Exec(q); err != nil {
					t.Fatal(err)
				}
			}
			if collision {
				if _, err := db.Exec(`CREATE TABLE remember_tokens LIKE legacy_remember_tokens`); err != nil {
					t.Fatal(err)
				}
			}
			upgraded, err := mysql.Open(store.DSN())
			if collision {
				if err == nil {
					upgraded.Close()
					t.Fatal("expected collision")
				}
				var n int
				if err := db.QueryRow(`SELECT COUNT(*) FROM legacy_password_resets WHERE token_hash='reset-hash'`).Scan(&n); err != nil || n != 1 {
					t.Fatalf("partial rename or row loss: %d %v", n, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer upgraded.Close()
			var hash string
			if err := db.QueryRow(`SELECT token_hash FROM password_reset_tokens WHERE user_id=90001`).Scan(&hash); err != nil || hash != "reset-hash" {
				t.Fatalf("reset row changed: %v", err)
			}
			if _, err := upgraded.LegacyRememberUser(context.Background(), 90001, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Now()); err != nil {
				t.Fatal(err)
			}
			again, err := mysql.Open(store.DSN())
			if err != nil {
				t.Fatal(err)
			}
			again.Close()
		})
	}
}
