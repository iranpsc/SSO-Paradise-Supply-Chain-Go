package mysql_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
)

func TestLaravelImportDryRunAtomicApplyAndMedia(t *testing.T) {
	source := rawDB(t, mysqltest.Database(t))
	target := mysqltest.Open(t)
	ctx := context.Background()
	schema := `CREATE TABLE users(id BIGINT PRIMARY KEY,mobile VARCHAR(255),updated_at DATETIME(6),name VARCHAR(255),email VARCHAR(255),password VARCHAR(255),code VARCHAR(255),referral VARCHAR(255),email_verified_at DATETIME(6),created_at DATETIME(6),wallet_address VARCHAR(255),remember_token VARCHAR(100));
 CREATE TABLE personal_infos(id BIGINT PRIMARY KEY,user_id BIGINT,is_company BOOLEAN,first_name VARCHAR(255),last_name VARCHAR(255),mobile VARCHAR(255),telephone VARCHAR(255),national_code VARCHAR(255),address VARCHAR(255),company_name VARCHAR(255),company_address VARCHAR(255),company_registration_number VARCHAR(255),company_national_number VARCHAR(255),company_tax_number VARCHAR(255),company_executive_name VARCHAR(255),is_verified BOOLEAN,verification_messages JSON);
 CREATE TABLE oauth_clients(id BIGINT PRIMARY KEY,name VARCHAR(255),secret VARCHAR(100),redirect TEXT,personal_access_client BOOLEAN,password_client BOOLEAN,revoked BOOLEAN,created_at DATETIME(6));
 CREATE TABLE oauth_access_tokens(id VARCHAR(100) PRIMARY KEY,user_id BIGINT,client_id BIGINT,scopes TEXT,revoked BOOLEAN,expires_at DATETIME(6));
 CREATE TABLE oauth_refresh_tokens(id VARCHAR(100) PRIMARY KEY,access_token_id VARCHAR(100),revoked BOOLEAN,expires_at DATETIME(6));
 CREATE TABLE oauth_auth_codes(id VARCHAR(100) PRIMARY KEY,user_id BIGINT,client_id BIGINT,scopes TEXT,revoked BOOLEAN,expires_at DATETIME(6));
 CREATE TABLE password_reset_tokens(email VARCHAR(255) PRIMARY KEY,token VARCHAR(255),created_at DATETIME(6));
 CREATE TABLE media(id BIGINT PRIMARY KEY,model_type VARCHAR(255),model_id BIGINT,collection_name VARCHAR(255),file_name VARCHAR(255),disk VARCHAR(255));`
	if _, err := source.Exec(schema); err != nil {
		t.Fatal(err)
	}
	hash, err := (security.Bcrypt{Cost: 4}).Hash("OldPassword!2026")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	if _, err = source.Exec(`INSERT INTO users(id,mobile,updated_at,name,email,password,code,created_at) VALUES(7,'09123456789',?,'Legacy','LEGACY@example.com',?,'hm-2000099',?)`, now, hash, now); err != nil {
		t.Fatal(err)
	}
	if _, err = source.Exec(`INSERT INTO personal_infos(id,user_id,is_company,first_name,last_name,mobile,national_code,company_registration_number,is_verified) VALUES(11,7,0,'Old','Member','09123456789','0012345678','000123',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err = source.Exec(`INSERT INTO oauth_clients(id,name,secret,redirect,personal_access_client,password_client,revoked,created_at) VALUES(42,'Old client','client-secret','https://client.example/callback',0,0,0,?)`, now); err != nil {
		t.Fatal(err)
	}
	if _, err = source.Exec(`INSERT INTO media(id,model_type,model_id,collection_name,file_name,disk) VALUES(2,?,7,'avatars','avatar.png','local')`, `App\Models\User`); err != nil {
		t.Fatal(err)
	}
	oldAccess, oldRefresh, oldCode := strings.Repeat("a", 80), strings.Repeat("b", 80), strings.Repeat("c", 80)
	for _, insert := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO oauth_access_tokens VALUES(?,7,42,'[]',0,?)`, []any{oldAccess, now.Add(time.Hour)}},
		{`INSERT INTO oauth_refresh_tokens VALUES(?,?,0,?)`, []any{oldRefresh, oldAccess, now.Add(24 * time.Hour)}},
		{`INSERT INTO oauth_auth_codes VALUES(?,7,42,'[]',0,?)`, []any{oldCode, now.Add(5 * time.Minute)}},
	} {
		if _, err = source.Exec(insert.query, insert.args...); err != nil {
			t.Fatal(err)
		}
	}
	resetToken := strings.Repeat("r", 64)
	resetHash, e := (security.Bcrypt{Cost: 4}).Hash(resetToken)
	if e != nil {
		t.Fatal(e)
	}
	if _, err = source.Exec(`INSERT INTO password_reset_tokens VALUES('legacy@example.com',?,?)`, resetHash, now); err != nil {
		t.Fatal(err)
	}
	sourceKey := []byte("base64:YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU=")
	rememberToken := strings.Repeat("t", 60)
	if _, err = source.Exec(`UPDATE users SET remember_token=? WHERE id=7`, rememberToken); err != nil {
		t.Fatal(err)
	}
	sessionsRoot := t.TempDir()
	sessionID := strings.Repeat("s", 40)
	guardKey := "login_web_" + security.SessionGuardHash()
	sessionMAC := security.LaravelPasswordMAC(hash, sourceKey)
	serialized := fmt.Sprintf(`a:3:{s:%d:"%s";i:7;s:17:"password_hash_web";s:%d:"%s";s:12:"wallet_login";b:1;}`, len(guardKey), guardKey, len(sessionMAC), sessionMAC)
	if err = os.WriteFile(filepath.Join(sessionsRoot, sessionID), []byte(serialized), 0600); err != nil {
		t.Fatal(err)
	}
	options := mysql.ImportOptions{SessionsRoot: sessionsRoot, AppKey: sourceKey, SessionLifetime: 2 * time.Hour}
	root := t.TempDir()
	if err = os.Mkdir(filepath.Join(root, "2"), 0700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(root, "2", "avatar.png"))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(file, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	report, err := target.ImportLaravelWithOptions(ctx, source, root, false, options)
	if err != nil || report.Applied || report.Users != 1 || report.Media != 1 {
		t.Fatalf("dry-run: %+v %v", report, err)
	}
	if _, err = target.ByID(ctx, 7); err != domain.ErrNotFound {
		t.Fatal("dry-run wrote account")
	}
	// An incompatible field prevents apply before any target data is written.
	if _, err = source.Exec(`UPDATE personal_infos SET company_registration_number=?`, strings.Repeat("x", 33)); err != nil {
		t.Fatal(err)
	}
	if _, err = target.ImportLaravelWithOptions(ctx, source, root, true, options); err == nil {
		t.Fatal("incompatible profile imported")
	}
	if _, err = target.ByID(ctx, 7); err != domain.ErrNotFound {
		t.Fatal("partial import survived")
	}
	if _, err = source.Exec(`UPDATE personal_infos SET company_registration_number='000123'`); err != nil {
		t.Fatal(err)
	}
	report, err = target.ImportLaravelWithOptions(ctx, source, root, true, options)
	if err != nil || !report.Applied {
		t.Fatalf("apply: %+v %v", report, err)
	}
	user, err := target.ByEmail(ctx, "legacy@example.com")
	if err != nil || user.ID != 7 || user.Mobile == nil || *user.Mobile != "09123456789" || !user.UpdatedAt.Equal(now) || !(security.Bcrypt{}).Matches(user.PasswordHash, "OldPassword!2026") {
		t.Fatalf("user not preserved: %+v %v", user, err)
	}
	profile, err := target.PersonalInfo(ctx, 7)
	if err != nil || profile.NationalCode != "0012345678" || profile.CompanyRegistrationNumber != "000123" {
		t.Fatalf("profile: %+v %v", profile, err)
	}
	client, err := target.OAuthClient(ctx, 42)
	if err != nil || !(security.Bcrypt{}).Matches(client.SecretHash, "client-secret") {
		t.Fatal("client secret not preserved")
	}
	if _, err = target.Media(ctx, 7, "avatars"); err != nil {
		t.Fatal(err)
	}
	if _, err = target.ImportLaravelWithOptions(ctx, source, root, true, options); err == nil {
		t.Fatal("non-empty target accepted")
	}
	if report.Sessions != 1 || report.RememberTokens != 1 {
		t.Fatal("browser credentials not preserved", report)
	}
	a := application.Auth{Accounts: target, Sessions: target, Actions: target, Now: time.Now, SessionTTL: time.Hour, SigningKey: sourceKey, LegacyCookies: legacyCookieStub{}, LegacySessionCookie: "laravel_session", LegacyRememberCookie: "remember_web_test"}
	restored, newSession, e := a.RestoreLegacyCookie(ctx, "laravel_session", sessionID)
	if e != nil || restored.ID != 7 {
		t.Fatal("old session not restored", e)
	}
	if flag, e := target.SessionAttribute(ctx, application.Digest(newSession), "wallet_login", time.Now()); e != nil || flag != "true" {
		t.Fatal("wallet callback flag lost")
	}
	if _, e := target.SessionAttribute(ctx, application.Digest(sessionID), "wallet_login", time.Now()); e != domain.ErrNotFound {
		t.Fatal("old wallet flag replayable")
	}
	recaller := "7|" + rememberToken + "|" + sessionMAC
	if restored, _, e = a.RestoreLegacyCookie(ctx, "remember_web_test", recaller); e != nil || restored.ID != 7 {
		t.Fatal("remember cookie not restored", e)
	}
	if report.AccessTokens != 1 || report.RefreshTokens != 1 || report.AuthorizationCodes != 1 {
		t.Fatal("Passport records were not imported", report)
	}
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	signer := security.JWT{Key: key}
	oauth := application.OAuth{Store: target, Passwords: security.Bcrypt{}, Now: time.Now, Signer: signer}
	oldJWT, e := signer.SignAccess(oldAccess, 42, 7, nil, now, now.Add(time.Hour))
	if e != nil {
		t.Fatal(e)
	}
	if u, e := oauth.AccessUser(ctx, oldJWT, ""); e != nil || u.ID != 7 {
		t.Fatal("imported JWT rejected", e)
	}
	payload, _ := json.Marshal(map[string]any{"client_id": "42", "refresh_token_id": oldRefresh, "expire_time": now.Add(time.Hour).Unix()})
	oauth.LegacyCipher = legacyCipherStub{payload: payload}
	encrypted := strings.Repeat("d", 200)
	pair, e := oauth.Token(ctx, 42, "client-secret", "refresh_token", "", "", "", encrypted)
	if e != nil {
		t.Fatal("imported encrypted refresh failed", e)
	}
	if u, e := oauth.AccessUser(ctx, pair.AccessToken, ""); e != nil || u.ID != 7 {
		t.Fatal("refreshed JWT rejected", e)
	}
	if _, e = oauth.Token(ctx, 42, "client-secret", "refresh_token", "", "", "", encrypted); e == nil {
		t.Fatal("imported refresh replay accepted")
	}
	if _, e = oauth.AccessUser(ctx, pair.AccessToken, ""); e == nil {
		t.Fatal("refresh replay did not revoke grant")
	}
	verifier := strings.Repeat("v", 43)
	payload, _ = json.Marshal(map[string]any{"client_id": "42", "auth_code_id": oldCode, "expire_time": now.Add(time.Hour).Unix(), "redirect_uri": "https://client.example/callback", "code_challenge_method": "S256", "code_challenge": application.PKCE(verifier)})
	oauth.LegacyCipher = legacyCipherStub{payload: payload}
	if _, e = oauth.Token(ctx, 42, "client-secret", "authorization_code", encrypted, "https://client.example/callback", strings.Repeat("x", 43), ""); e == nil {
		t.Fatal("wrong legacy verifier accepted")
	}
	if _, e = oauth.Token(ctx, 42, "client-secret", "authorization_code", encrypted, "https://client.example/callback", verifier, ""); e != nil {
		t.Fatal("imported encrypted authorization code failed", e)
	}
	if report.PasswordResets != 1 {
		t.Fatal("reset link not imported")
	}
	newHash, e := (security.Bcrypt{Cost: 4}).Hash("FreshPassword!2026")
	if e != nil {
		t.Fatal(e)
	}
	if e = target.ResetLegacyPassword(ctx, "legacy@example.com", strings.Repeat("x", 64), newHash, time.Now()); e == nil {
		t.Fatal("wrong imported reset token accepted")
	}
	if e = target.ResetLegacyPassword(ctx, "legacy@example.com", resetToken, newHash, time.Now()); e != nil {
		t.Fatal("imported reset link failed", e)
	}
	if e = target.ResetLegacyPassword(ctx, "legacy@example.com", resetToken, newHash, time.Now()); e == nil {
		t.Fatal("imported reset link replay accepted")
	}
	if _, _, e = a.RestoreLegacyCookie(ctx, "remember_web_test", recaller); e == nil {
		t.Fatal("old remember credential survived password reset")
	}
	var original string
	if err = source.QueryRow(`SELECT email FROM users WHERE id=7`).Scan(&original); err != nil || original != "LEGACY@example.com" {
		t.Fatal("source was modified")
	}
}

type legacyCipherStub struct{ payload []byte }

func (s legacyCipherStub) Decode(string) ([]byte, error) { return s.payload, nil }

type legacyCookieStub struct{}

func (legacyCookieStub) DecodeCookie(_, value string) (string, error) { return value, nil }
