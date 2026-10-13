package mysql_test

import (
	"context"
	"testing"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
)

func TestArchiveNamesPreserveRowsAndHandleCollisions(t *testing.T) {
	for _, collision := range []bool{false, true} {
		t.Run(map[bool]string{false: "rename", true: "collision"}[collision], func(t *testing.T) {
			store := mysqltest.Open(t)
			db := rawDB(t, store.DSN())
			for _, query := range []string{
				`DELETE FROM schema_migrations WHERE version=12`,
				`CREATE TABLE go_before_laravel_20261013_users(id BIGINT PRIMARY KEY)`,
				`INSERT INTO go_before_laravel_20261013_users VALUES(7)`,
				`CREATE TABLE laravel_archive_media(id BIGINT PRIMARY KEY)`,
				`INSERT INTO laravel_archive_media VALUES(30)`,
			} {
				if _, err := db.Exec(query); err != nil {
					t.Fatal(err)
				}
			}
			if collision {
				if _, err := db.Exec(`CREATE TABLE archive_local_users(id BIGINT PRIMARY KEY)`); err != nil {
					t.Fatal(err)
				}
			}
			err := mysql.Migrate(context.Background(), db)
			if collision {
				if err == nil {
					t.Fatal("archive collision accepted")
				}
				var id int
				if err := db.QueryRow(`SELECT id FROM laravel_archive_media`).Scan(&id); err != nil || id != 30 {
					t.Fatal("collision partially renamed tables", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for table, want := range map[string]int{"archive_local_users": 7, "archive_laravel_media": 30} {
				var id int
				if err := db.QueryRow("SELECT id FROM " + table).Scan(&id); err != nil || id != want {
					t.Fatalf("%s data changed: %d %v", table, id, err)
				}
			}
			if err := mysql.Migrate(context.Background(), db); err != nil {
				t.Fatal("repeat migration", err)
			}
		})
	}
}
