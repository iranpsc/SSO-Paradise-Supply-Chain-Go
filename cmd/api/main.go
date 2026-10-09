package main

import (
	"context"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/envfile"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/logging"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/cache"
	mailadapter "github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mail"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/metarang"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func run() error {
	logger, err := logging.New()
	if err != nil {
		return err
	}
	if err := validateEnvironment(); err != nil {
		return err
	}
	publicURL := strings.TrimRight(env("PUBLIC_URL", "http://localhost:3000"), "/")
	u, err := url.Parse(publicURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" {
		return errors.New("PUBLIC_URL must be an http(s) origin")
	}
	dsn := buildDSN()
	db, err := mysql.Open(dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	rateLimit, e := strconv.Atoi(env("RATE_LIMIT_PER_MINUTE", "60"))
	if e != nil || rateLimit < 1 || rateLimit > 1000 {
		return errors.New("invalid RATE_LIMIT_PER_MINUTE")
	}
	db.ConfigureRateLimit(rateLimit)
	sessionTTL, e := durationEnv("SESSION_TTL", 2*time.Hour)
	if e != nil {
		return e
	}
	rememberTTL, e := durationEnv("REMEMBER_SESSION_TTL", 400*24*time.Hour)
	if e != nil {
		return e
	}
	store := &cache.Store{DB: db, UserTTL: userCacheTTL(), SessionTTL: sessionTTL, Log: logger}
	if client := openRedis(); client != nil {
		store.Redis = client
		defer client.Close()
		logger.Info("redis user cache enabled", "address", client.Options().Addr)
	} else {
		logger.Warn("redis unavailable, running database-only")
	}
	auth := &application.Auth{SessionIdle: true, Accounts: store, Sessions: store, Actions: store, Passwords: security.Bcrypt{Cost: 10}, Safety: security.BreachedPasswords{Client: &http.Client{Timeout: 5 * time.Second}, URL: "https://api.pwnedpasswords.com", FailOpen: true, OnError: func(err error) { logger.Warn("password breach lookup unavailable", "error", err) }}, Mailer: mailadapter.File{Directory: env("MAIL_DIRECTORY", "var/mail")}, PublicURL: publicURL, Now: time.Now, SessionTTL: sessionTTL, RememberTTL: rememberTTL, RateLimit: rateLimit}
	appKey, err := security.LoadAppKey(os.Getenv("APP_KEY"), "var/app.key")
	if err != nil {
		return err
	}
	auth.SigningKey = appKey
	signer, err := security.LoadJWT(env("OAUTH_PRIVATE_KEY_PATH", "var/oauth-private.key"), env("APP_ENV", "development") == "development")
	if err != nil {
		return err
	}
	auth.OAuth = &application.OAuth{Store: db, Passwords: auth.Passwords, Now: time.Now, Signer: signer}
	legacyCipher, e := security.NewPassportCipher(appKey)
	if e != nil {
		return e
	}
	auth.OAuth.LegacyCipher = legacyCipher
	auth.LegacyCookies = legacyCipher
	auth.LegacySessionCookie = os.Getenv("LARAVEL_SESSION_COOKIE")
	auth.LegacyRememberCookie = os.Getenv("LARAVEL_REMEMBER_COOKIE")
	if auth.LegacySessionCookie != "" && auth.LegacyRememberCookie == "" {
		auth.LegacyRememberCookie = "remember_web_" + security.SessionGuardHash()
	}

	auth.OAuth.AccessTTL, e = durationEnv("OAUTH_ACCESS_TTL", time.Hour)
	if e != nil {
		return e
	}
	auth.OAuth.RefreshTTL, e = durationEnv("OAUTH_REFRESH_TTL", 2*time.Hour)
	if e != nil {
		return e
	}
	auth.OAuth.PersonalTTL, e = durationEnv("OAUTH_PERSONAL_TTL", time.Hour)
	if e != nil {
		return e
	}

	if env("MAIL_MAILER", "file") == "smtp" {
		smtp, e := mailadapter.SMTPFromEnv()
		if e != nil {
			return e
		}
		auth.Mailer = smtp
		if env("MAIL_QUEUE", "async") != "sync" {
			auth.Mailer = mailadapter.Outbox{Queue: db, Now: time.Now}
		}
	}
	auth.AppName = env("APP_NAME", "Laravel")
	auth.Locale = "fa"
	auth.Mailer = mailadapter.Notifications{Transport: auth.Mailer}
	profiles := &application.Profile{Store: store}
	registry, err := metarang.LoadRegistry(env("Metarang_API", ""), os.Getenv("METARANG_LOCAL_FIXTURES"), env("APP_ENV", "development"))
	if err != nil {
		return err
	}
	if os.Getenv("METARANG_LOCAL_FIXTURES") != "" {
		logger.Warn("local wallet membership fixtures enabled; external Metarang lookup is simulated")
	}
	web3 := &application.Web3{SessionIdle: true, Wallets: store, Challenges: store, Attributes: store, Registry: registry, Sessions: store, Now: time.Now, SessionTTL: sessionTTL, AppName: env("APP_NAME", "Laravel"), PublicURL: publicURL}
	var proxies []netip.Prefix
	for _, value := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		if strings.TrimSpace(value) == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			return errors.New("invalid TRUSTED_PROXY_CIDRS")
		}
		proxies = append(proxies, prefix)
	}
	srv := &http.Server{Addr: env("HTTP_ADDR", "127.0.0.1:8080"), Handler: httpapi.New(auth, web3, profiles, publicURL, u.Scheme == "https", logger, proxies...), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { logger.Info("API listening", "address", srv.Addr); done <- srv.ListenAndServe() }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	}
}
func buildDSN() string { return mysql.ConfigFromEnv().FormatDSN() }

func userCacheTTL() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("REDIS_USER_TTL_MINUTES")); raw != "" {
		if minutes, err := strconv.Atoi(raw); err == nil && minutes > 0 {
			return time.Duration(minutes) * time.Minute
		}
	}
	return 10 * time.Minute
}

func openRedis() *redis.Client {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	dbNumber := 0
	if raw := strings.TrimSpace(os.Getenv("REDIS_DB")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			dbNumber = n
		}
	}
	password := os.Getenv("REDIS_PASSWORD")
	if password == "null" {
		password = ""
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: dbNumber})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil
	}
	return client
}
func main() {
	envfile.MustLoad()
	if err := run(); err != nil {
		slog.Error("API stopped", "error", err)
		os.Exit(1)
	}
}
