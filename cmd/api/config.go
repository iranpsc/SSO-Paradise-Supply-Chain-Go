package main

import (
	"encoding/base64"
	"errors"
	mailadapter "github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mail"
	"net/url"
	"os"
	"strings"
	"time"
)

func validateEnvironment() error {
	mode := env("APP_ENV", "development")
	if mode != "development" && mode != "production" && mode != "staging" {
		return errors.New("APP_ENV must be development, staging or production")
	}
	raw := strings.TrimRight(env("PUBLIC_URL", "http://localhost:3000"), "/")
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("PUBLIC_URL must be an http(s) origin")
	}
	if mode == "development" {
		return nil
	}
	if u.Scheme != "https" {
		return errors.New("production/staging PUBLIC_URL requires HTTPS")
	}
	key := os.Getenv("APP_KEY")
	if key == "" {
		return errors.New("production/staging requires APP_KEY")
	}
	decoded := []byte(key)
	if strings.HasPrefix(key, "base64:") {
		decoded, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(key, "base64:"))
		if err != nil {
			return errors.New("APP_KEY is not valid base64")
		}
	}
	if len(decoded) != 32 {
		return errors.New("APP_KEY must contain 32 key bytes")
	}
	if os.Getenv("OAUTH_PRIVATE_KEY_PATH") == "" {
		return errors.New("production/staging requires OAUTH_PRIVATE_KEY_PATH")
	}
	if env("MAIL_MAILER", "file") != "smtp" {
		return errors.New("production/staging requires MAIL_MAILER=smtp")
	}
	smtp, err := mailadapter.SMTPFromEnv()
	if err != nil {
		return err
	}
	if smtp.Encryption == "" {
		return errors.New("production/staging SMTP requires TLS")
	}
	if os.Getenv("MAIL_QUEUE") == "sync" {
		return errors.New("production/staging requires durable queued mail")
	}
	return nil
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid positive duration for " + key)
	}
	return value, nil
}
