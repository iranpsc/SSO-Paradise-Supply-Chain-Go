package main

import (
	"context"
	"database/sql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/envfile"
	"log"
	"net"
	"os"
	"regexp"
	"time"

	driver "github.com/go-sql-driver/mysql"
	store "github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
)

func main() {
	envfile.MustLoad()
	name := env("MYSQL_DATABASE", "paradise")
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(name) {
		log.Fatal("invalid MYSQL_DATABASE")
	}
	config := driver.NewConfig()
	config.User = env("MYSQL_USER", "root")
	config.Passwd = os.Getenv("MYSQL_PASSWORD")
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(env("MYSQL_HOST", "127.0.0.1"), env("MYSQL_PORT", "3306"))
	config.ParseTime = true
	config.Loc = time.UTC
	config.Timeout = 5 * time.Second
	config.Params = map[string]string{"charset": "utf8mb4"}
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	if _, err = db.Exec("CREATE DATABASE IF NOT EXISTS `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"); err != nil {
		log.Fatal(err)
	}
	db.Close()
	config.DBName = name
	db, err = sql.Open("mysql", config.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err = store.Migrate(ctx, db); err != nil {
		log.Fatal(err)
	}
	log.Print("Versioned migrations complete; existing data preserved")
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
