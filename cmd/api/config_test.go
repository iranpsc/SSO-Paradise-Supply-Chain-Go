package main

import (
	"testing"
)

func TestProductionRejectsIncompleteOrUnsafeConfiguration(t *testing.T) {
	for _, key := range []string{"APP_KEY", "MAIL_HOST", "MAIL_FROM_ADDRESS", "OAUTH_PRIVATE_KEY_PATH", "MAIL_QUEUE"} {
		t.Setenv(key, "")
	}
	t.Setenv("APP_ENV", "production")
	t.Setenv("ALLOW_INSECURE_HTTP", "false")
	t.Setenv("PUBLIC_URL", "http://accounts.example")
	if validateEnvironment() == nil {
		t.Fatal("HTTP production origin accepted")
	}
	t.Setenv("PUBLIC_URL", "https://accounts.example")
	t.Setenv("APP_KEY", "base64:YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU=")
	t.Setenv("OAUTH_PRIVATE_KEY_PATH", "synthetic.pem")
	t.Setenv("MAIL_MAILER", "smtp")
	t.Setenv("MAIL_HOST", "smtp.example")
	t.Setenv("MAIL_FROM_ADDRESS", "accounts@example.com")
	t.Setenv("MAIL_ENCRYPTION", "tls")
	if err := validateEnvironment(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PUBLIC_URL", "http://accounts.example")
	for _, mode := range []string{"production", "staging"} {
		t.Setenv("APP_ENV", mode)
		for _, flag := range []string{"", "false", "1", "true"} {
			t.Setenv("ALLOW_INSECURE_HTTP", flag)
			if err := validateEnvironment(); (err == nil) != (flag == "true") {
				t.Fatalf("mode=%s flag=%q: %v", mode, flag, err)
			}
		}
	}
	t.Setenv("APP_KEY", "")
	if validateEnvironment() == nil {
		t.Fatal("HTTP opt-in bypassed production key requirement")
	}
	t.Setenv("APP_KEY", "base64:YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU=")
	t.Setenv("MAIL_ENCRYPTION", "none")
	if validateEnvironment() == nil {
		t.Fatal("plaintext production SMTP accepted")
	}
}
