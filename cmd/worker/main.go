package main

import (
	"context"
	"errors"
	"flag"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/cache"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/envfile"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/logging"
	mailadapter "github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mail"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func run() error {
	logger, err := logging.New()
	if err != nil {
		return err
	}
	slog.SetDefault(logger)
	once := flag.Bool("once", false, "deliver due mail and run cleanup once, then exit")
	cleanupEvery := flag.Duration("cleanup-every", 0, "override midnight cleanup with a positive interval")
	flag.Parse()
	if *cleanupEvery < 0 {
		return errors.New("cleanup interval must be positive")
	}
	timezone := os.Getenv("APP_TIMEZONE")
	if timezone == "" {
		timezone = "Asia/Tehran"
	}
	zone, err := time.LoadLocation(timezone)
	if err != nil {
		return errors.New("invalid APP_TIMEZONE")
	}
	var sender application.Mailer
	if os.Getenv("MAIL_MAILER") == "smtp" {
		smtp, err := mailadapter.SMTPFromEnv()
		if err != nil {
			return err
		}
		sender = smtp
	} else {
		if os.Getenv("APP_ENV") == "production" || os.Getenv("APP_ENV") == "staging" {
			return errors.New("production worker requires SMTP")
		}
		directory := os.Getenv("MAIL_DIRECTORY")
		if directory == "" {
			directory = "var/mail"
		}
		sender = mailadapter.File{Directory: directory}
	}
	db, err := mysql.Open(mysql.ConfigFromEnv().FormatDSN())
	if err != nil {
		return err
	}
	defer db.Close()
	store := &cache.Store{DB: db}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	number, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	password := os.Getenv("REDIS_PASSWORD")
	if password == "null" {
		password = ""
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: number})
	defer client.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ping, cancel := context.WithTimeout(ctx, 2*time.Second)
	if client.Ping(ping).Err() == nil {
		store.Redis = client
	}
	cancel()
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	nextCleanup := time.Time{}
	for {
		var batchErrors []error
		for i := 0; i < 100; i++ {
			sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			found, err := mailadapter.DeliverOne(sendCtx, db, sender, time.Now())
			cancel()
			if err != nil {
				slog.Error("mail attempt failed; retry scheduled", "error", err)
				batchErrors = append(batchErrors, err)
			}
			if !found || ctx.Err() != nil {
				break
			}
		}
		if time.Now().After(nextCleanup) {
			cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			report, err := store.Cleanup(cleanupCtx, time.Now(), true)
			cancel()
			if err != nil {
				slog.Error("scheduled cleanup failed", "error", err)
				batchErrors = append(batchErrors, err)
				nextCleanup = time.Now().Add(time.Minute)
			} else {
				slog.Info("scheduled cleanup completed", "users", report.DeletedUsers, "expired", report.ExpiredRecords)
				if *cleanupEvery > 0 {
					nextCleanup = time.Now().Add(*cleanupEvery)
				} else {
					nextCleanup = nextMidnight(time.Now(), zone)
				}
			}
		}
		if *once {
			return errors.Join(batchErrors...)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
	}
}
func main() {
	envfile.MustLoad()
	if err := run(); err != nil {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
