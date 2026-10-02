package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	driver "github.com/go-sql-driver/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/envfile"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"log"
	"os"
	"time"
	_ "time/tzdata"
)

func main() {
	envfile.MustLoad()
	apply := flag.Bool("apply", false, "apply snapshot to empty Go target; default is dry-run")
	root := flag.String("media-root", "../SSO-Paradise-Supply-Chain/storage/app", "Laravel storage/app root")
	sessionsRoot := flag.String("sessions-root", "", "optional Laravel file-session directory; requires matching APP_KEY")
	lifetime := flag.Duration("session-lifetime", 2*time.Hour, "source SESSION_LIFETIME as a duration")
	sourceTimezone := flag.String("source-timezone", "Asia/Tehran", "timezone used by Laravel database timestamps")
	flag.Parse()
	zone, err := time.LoadLocation(*sourceTimezone)
	if err != nil {
		log.Fatal("invalid source timezone")
	}
	if *sessionsRoot != "" && os.Getenv("APP_KEY") == "" {
		log.Fatal("session import requires the exact Laravel APP_KEY")
	}
	if *lifetime <= 0 {
		log.Fatal("source session lifetime must be positive")
	}
	raw := os.Getenv("LARAVEL_MYSQL_DSN")
	if raw == "" {
		log.Fatal("LARAVEL_MYSQL_DSN is required; it must point to the read-only source")
	}
	config, err := driver.ParseDSN(raw)
	if err != nil {
		log.Fatal("invalid LARAVEL_MYSQL_DSN")
	}
	config.ParseTime = true
	config.Loc = zone
	config.MultiStatements = false
	source, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		log.Fatal("source connection failed")
	}
	defer source.Close()
	target, err := mysql.Open(mysql.ConfigFromEnv().FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	defer target.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	report, err := target.ImportLaravelWithOptions(ctx, source, *root, *apply, mysql.ImportOptions{SessionsRoot: *sessionsRoot, SessionLifetime: *lifetime, AppKey: []byte(os.Getenv("APP_KEY"))})
	if err != nil {
		log.Fatal(err)
	}
	json.NewEncoder(os.Stdout).Encode(report)
}
