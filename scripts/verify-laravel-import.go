// Run after import with MYSQL_* and LARAVEL_MYSQL_DSN set. Outputs counts only.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
)

func main() {
	ctx := context.Background()
	config, err := driver.ParseDSN(os.Getenv("LARAVEL_MYSQL_DSN"))
	if err != nil {
		log.Fatal("Invalid source DSN")
	}
	source, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		log.Fatal("Cannot open source")
	}
	defer source.Close()
	target, err := mysql.Open(mysql.ConfigFromEnv().FormatDSN())
	if err != nil {
		log.Fatal("Cannot open target")
	}
	defer target.Close()
	db, err := sql.Open("mysql", target.DSN())
	if err != nil {
		log.Fatal("Cannot open target verification connection")
	}
	defer db.Close()
	name := mysql.ConfigFromEnv().DBName
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(name) {
		log.Fatal("Invalid target name")
	}
	sourcePrefix := os.Getenv("LARAVEL_SOURCE_TABLE_PREFIX")
	if sourcePrefix != "" && !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(sourcePrefix) {
		log.Fatal("Invalid source table prefix")
	}
	runtimeOnly := os.Getenv("LARAVEL_VERIFY_RUNTIME_ONLY") == "1"
	checks := map[string]int{}
	queries := map[string]string{
		"users":                     "SELECT COUNT(*) FROM users",
		"profiles":                  "SELECT COUNT(*) FROM personal_infos",
		"wallets":                   "SELECT COUNT(*) FROM wallets",
		"clients":                   "SELECT COUNT(*) FROM oauth_clients",
		"access_tokens":             "SELECT COUNT(*) FROM oauth_tokens",
		"authorization_codes":       "SELECT COUNT(*) FROM oauth_codes",
		"deferred_media":            "SELECT COUNT(*) FROM archive_laravel_media",
		"user_value_mismatches":     `SELECT COUNT(*) FROM users t JOIN archive_laravel_users s ON s.id=t.id WHERE NOT(BINARY t.name <=> BINARY s.name) OR NOT(BINARY COALESCE(t.password_hash,'') <=> BINARY COALESCE(s.password,'')) OR NOT(BINARY LOWER(COALESCE(s.email,'')) <=> BINARY COALESCE(t.email,'')) OR NOT(BINARY t.code <=> BINARY s.code) OR NOT(BINARY t.mobile <=> BINARY s.mobile) OR NOT(BINARY t.referral <=> BINARY COALESCE(s.referral,''))`,
		"profile_status_mismatches": `SELECT COUNT(*) FROM personal_infos t JOIN archive_laravel_personal_infos s ON s.user_id=t.user_id WHERE t.is_verified<>s.is_verified`,
		"archived_access_only":      `SELECT COUNT(*) FROM archive_laravel_oauth_access_tokens s LEFT JOIN oauth_tokens t ON t.access_hash=SHA2(s.id,256) WHERE t.access_hash IS NULL`,
		"archived_refresh_only":     `SELECT COUNT(*) FROM archive_laravel_oauth_refresh_tokens s LEFT JOIN oauth_tokens t ON t.refresh_hash=SHA2(s.id,256) WHERE t.refresh_hash IS NULL`,
		"archived_codes_only":       `SELECT COUNT(*) FROM archive_laravel_oauth_auth_codes s LEFT JOIN oauth_codes t ON t.token_hash=SHA2(s.id,256) WHERE t.token_hash IS NULL`,
		"archived_resets_only":      `SELECT COUNT(*) FROM archive_laravel_password_reset_tokens s LEFT JOIN users u ON BINARY LOWER(TRIM(s.email))=BINARY u.email LEFT JOIN password_reset_tokens t ON t.user_id=u.id WHERE t.user_id IS NULL`,
	}
	if runtimeOnly {
		for key, query := range queries {
			if strings.Contains(query, "archive_laravel_") {
				delete(queries, key)
			}
		}
	}
	for label, query := range queries {
		var count int
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			log.Fatalf("Verification failed: %s", label)
		}
		checks[label] = count
	}
	if checks["user_value_mismatches"] != 0 || checks["profile_status_mismatches"] != 0 {
		log.Fatal("Imported values differ from source")
	}
	if !runtimeOnly {
		rows, err := source.QueryContext(ctx, "SELECT id,COALESCE(secret,'') FROM "+sourcePrefix+"oauth_clients")
		if err != nil {
			log.Fatal("Cannot read original clients")
		}
		for rows.Next() {
			var id int64
			var secret string
			if err := rows.Scan(&id, &secret); err != nil {
				log.Fatal("Cannot read client")
			}
			client, err := target.OAuthClient(ctx, id)
			if err != nil {
				log.Fatal("Imported client is missing")
			}
			valid := secret == "" && client.SecretHash == "" || strings.HasPrefix(secret, "$2") && secret == client.SecretHash || !strings.HasPrefix(secret, "$2") && secret != "" && (security.Bcrypt{}).Matches(client.SecretHash, secret)
			if !valid {
				log.Fatalf("Client secret mismatch for id %d", id)
			}
			checks["verified_client_secrets"]++
		}
		if err := rows.Err(); err != nil {
			log.Fatal("Client verification interrupted")
		}
		rows.Close()
	}
	// Exercise the production handler without a live server or real login.
	auth := &application.Auth{Accounts: target, Sessions: target, Actions: target, Now: time.Now, PublicURL: "https://dev-accounts.irpsc.com", SigningKey: []byte(strings.Repeat("k", 32))}
	auth.OAuth = &application.OAuth{Store: target, Passwords: security.Bcrypt{}, Now: time.Now}
	h := httpapi.New(auth, &application.Web3{Attributes: target}, &application.Profile{Store: target}, auth.PublicURL, true, slog.New(slog.NewTextHandler(io.Discard, nil)))
	query := url.Values{"client_id": {"38"}, "redirect_uri": {"https://dev-api.metarang.com/api/auth/callback"}, "response_type": {"code"}, "scope": {""}, "state": {"import-verification"}}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/authorize?"+query.Encode(), nil))
	expectedLogin := auth.PublicURL + "/login?" + url.Values{"return_to": {"/oauth/authorize?" + query.Encode()}}.Encode()
	if w.Code != 302 || w.Header().Get("Location") != expectedLogin {
		log.Fatalf("Client 38 guest authorization failed with HTTP %d", w.Code)
	}
	checks["client_38_guest_authorization_http"] = w.Code
	fmt.Println("Imported client 38 routes to login successfully; no invalid_client error")
	json.NewEncoder(os.Stdout).Encode(checks)
}
