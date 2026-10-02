// Package mysqltest creates a throwaway MySQL database per test.
//
// Unlike SQLite, a MySQL server holds a single shared namespace, so tests that
// all point at the same database collide on the fixed e-mails and usernames they
// register. Each test therefore gets its own database, created before the test
// and dropped during cleanup.
//
// Tests skip unless MYSQL_HOST is set, so the suite still runs on machines with
// no MySQL server.
package mysqltest

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
)

// Open creates an empty database, applies the schema, and returns a store bound
// to it. The database is dropped when the test finishes.
func Open(t *testing.T) *mysql.Store {
	t.Helper()
	dsn := Database(t)
	store, err := mysql.Open(dsn)
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// Database allocates an empty, isolated schema without applying migrations.
func Database(t *testing.T) string {
	t.Helper()
	host := os.Getenv("MYSQL_HOST")
	if host == "" {
		t.Skip("MYSQL_HOST is not set; skipping MySQL-backed test")
	}

	name := "paradise_test_" + randomSuffix(t)
	params := "multiStatements=true&charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=true"
	credentials := env("MYSQL_USER", "root") + ":" + env("MYSQL_PASSWORD", "") + "@tcp(" + host + ":" + env("MYSQL_PORT", "3306") + ")/"

	admin, err := sql.Open("mysql", credentials+"?charset=utf8mb4")
	if err != nil {
		t.Fatalf("connect to mysql: %v", err)
	}
	defer admin.Close()
	if _, err := admin.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"); err != nil {
		t.Fatalf("create test database %s: %v", name, err)
	}
	t.Cleanup(func() {
		drop, err := sql.Open("mysql", credentials+"?charset=utf8mb4")
		if err != nil {
			return
		}
		defer drop.Close()
		_, _ = drop.Exec("DROP DATABASE IF EXISTS " + name)
	})

	return credentials + name + "?" + params
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random suffix: %v", err)
	}
	return fmt.Sprintf("%s_%s", hex.EncodeToString(b), fmt.Sprint(os.Getpid()))
}
