package mysql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
)

type ImportReport struct {
	Sessions           int  `json:"sessions"`
	SkippedSessions    int  `json:"skipped_sessions"`
	RememberTokens     int  `json:"remember_tokens"`
	Users              int  `json:"users"`
	Profiles           int  `json:"profiles"`
	Wallets            int  `json:"wallets"`
	Clients            int  `json:"clients"`
	Media              int  `json:"media"`
	Applied            bool `json:"applied"`
	AccessTokens       int  `json:"access_tokens"`
	RefreshTokens      int  `json:"refresh_tokens"`
	PasswordResets     int  `json:"password_resets"`
	AuthorizationCodes int  `json:"authorization_codes"`
}
type importedClient struct {
	id                          int64
	name, secret                string
	redirects                   []string
	personal, password, revoked bool
	created                     time.Time
}
type importedUser struct {
	remember string
	user     domain.User
	wallet   string
}

// ImportLaravel reads a consistent, read-only snapshot and applies it in one
// transaction to an empty Go target. The caller must stop writes during cutover.
// No source records, files, active tokens, or Laravel tables are modified.
func (s *Store) ImportLaravel(ctx context.Context, source *sql.DB, mediaRoot string, apply bool) (ImportReport, error) {
	return s.ImportLaravelWithOptions(ctx, source, mediaRoot, apply, ImportOptions{})
}
func (s *Store) ImportLaravelWithOptions(ctx context.Context, source *sql.DB, mediaRoot string, apply bool, options ImportOptions) (ImportReport, error) {
	var report ImportReport
	var sourceName, targetName string
	if err := source.QueryRowContext(ctx, `SELECT DATABASE()`).Scan(&sourceName); err != nil {
		return report, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT DATABASE()`).Scan(&targetName); err != nil {
		return report, err
	}
	if sourceName == targetName {
		return report, fmt.Errorf("source and target database names must differ")
	}
	read, err := source.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return report, err
	}
	defer read.Rollback()
	var rememberColumn int
	if err = read.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='users' AND column_name='remember_token'`).Scan(&rememberColumn); err != nil {
		return report, err
	}
	rememberExpr := "''"
	if rememberColumn > 0 {
		rememberExpr = "COALESCE(remember_token,'')"
	}
	var users []importedUser
	rows, err := read.QueryContext(ctx, `SELECT id,mobile,updated_at,name,COALESCE(email,''),COALESCE(password,''),code,COALESCE(referral,''),email_verified_at,created_at,COALESCE(wallet_address,''),`+rememberExpr+` FROM users ORDER BY id`)
	if err != nil {
		return report, err
	}
	maxCode := int64(2000000)
	for rows.Next() {
		var item importedUser
		var verified sql.NullTime
		u := &item.user
		if err = rows.Scan(&u.ID, &u.Mobile, &u.UpdatedAt, &u.Name, &u.Email, &u.PasswordHash, &u.Code, &u.Referral, &verified, &u.CreatedAt, &item.wallet, &item.remember); err != nil {
			rows.Close()
			return report, err
		}
		if verified.Valid {
			u.EmailVerifiedAt = &verified.Time
		}
		u.Email = strings.ToLower(u.Email)
		item.wallet = strings.ToLower(item.wallet)
		if utf8.RuneCountInString(u.Name) > 255 || len(u.Email) > 255 || len(u.Referral) > 32 || (u.Code != nil && len(*u.Code) > 32) {
			rows.Close()
			return report, fmt.Errorf("user %d exceeds target column limits", u.ID)
		}
		if u.PasswordHash != "" && !strings.HasPrefix(u.PasswordHash, "$2") {
			rows.Close()
			return report, fmt.Errorf("user %d: password hash is not bcrypt; configure a compatible verifier before import", u.ID)
		}
		if u.PasswordHash != "" {
			if _, e := bcrypt.Cost([]byte(u.PasswordHash)); e != nil {
				rows.Close()
				return report, fmt.Errorf("user %d has an invalid bcrypt hash", u.ID)
			}
		}
		if u.Mobile != nil && (len(*u.Mobile) > 11 || strings.IndexFunc(*u.Mobile, func(r rune) bool { return r < '0' || r > '9' }) >= 0) {
			rows.Close()
			return report, fmt.Errorf("user %d mobile exceeds target format/length", u.ID)
		}
		if item.wallet != "" {
			if !domain.ValidWalletAddress(item.wallet) {
				rows.Close()
				return report, fmt.Errorf("user %d: invalid wallet", u.ID)
			}
			report.Wallets++
		}
		if u.Code != nil {
			if n, e := strconv.ParseInt(strings.TrimPrefix(*u.Code, "hm-"), 10, 64); e == nil && n > maxCode {
				maxCode = n
			}
		}
		if len(item.remember) > 100 {
			return report, fmt.Errorf("unsupported Laravel remember token")
		}
		if item.remember != "" {
			report.RememberTokens++
		}
		users = append(users, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	report.Users = len(users)
	profiles := map[int64]domain.PersonalInfo{}
	profileOwners := map[int64]int64{}
	rows, err = read.QueryContext(ctx, `SELECT id,user_id,is_company,COALESCE(first_name,''),COALESCE(last_name,''),COALESCE(mobile,''),COALESCE(telephone,''),COALESCE(national_code,''),COALESCE(address,''),COALESCE(company_name,''),COALESCE(company_address,''),COALESCE(company_registration_number,''),COALESCE(company_national_number,''),COALESCE(company_tax_number,''),COALESCE(company_executive_name,''),is_verified,COALESCE(verification_messages,'') FROM personal_infos ORDER BY id`)
	if err != nil {
		return report, err
	}
	for rows.Next() {
		var id int64
		var p domain.PersonalInfo
		if err = rows.Scan(&id, &p.UserID, &p.IsCompany, &p.FirstName, &p.LastName, &p.Mobile, &p.Telephone, &p.NationalCode, &p.Address, &p.CompanyName, &p.CompanyAddress, &p.CompanyRegistrationNumber, &p.CompanyNationalNumber, &p.CompanyTaxNumber, &p.CompanyExecutiveName, &p.IsVerified, &p.VerificationMessages); err != nil {
			rows.Close()
			return report, err
		}
		if _, exists := profiles[p.UserID]; exists {
			rows.Close()
			return report, fmt.Errorf("multiple personal_infos for user %d", p.UserID)
		}
		if err = checkProfileLengths(p); err != nil {
			rows.Close()
			return report, err
		}
		profiles[p.UserID] = p
		profileOwners[id] = p.UserID
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	report.Profiles = len(profiles)
	var clients []importedClient
	rows, err = read.QueryContext(ctx, `SELECT id,name,COALESCE(secret,''),COALESCE(redirect,''),personal_access_client,password_client,revoked,created_at FROM oauth_clients ORDER BY id`)
	if err != nil {
		return report, err
	}
	for rows.Next() {
		var c importedClient
		var redirect string
		if err = rows.Scan(&c.id, &c.name, &c.secret, &redirect, &c.personal, &c.password, &c.revoked, &c.created); err != nil {
			rows.Close()
			return report, err
		}

		if c.secret != "" && !strings.HasPrefix(c.secret, "$2") {
			if strings.HasPrefix(c.secret, "$") {
				rows.Close()
				return report, fmt.Errorf("client %d has an unsupported secret hash", c.id)
			}
			c.secret, err = (security.Bcrypt{Cost: 10}).Hash(c.secret)
			if err != nil {
				rows.Close()
				return report, err
			}
		}
		for _, raw := range strings.Split(redirect, ",") {
			if uri := strings.TrimSpace(raw); uri != "" {
				if len(uri) > 2048 {
					rows.Close()
					return report, fmt.Errorf("client %d redirect exceeds 2048 bytes", c.id)
				}
				c.redirects = append(c.redirects, uri)
			}
		}
		clients = append(clients, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	report.Clients = len(clients)
	var media []domain.Media
	rows, err = read.QueryContext(ctx, `SELECT id,model_type,model_id,collection_name,file_name,disk FROM media ORDER BY id`)
	if err != nil {
		return report, err
	}
	seenMedia := map[string]bool{}
	for rows.Next() {
		var id, modelID int64
		var model, kind, filename, disk string
		if err = rows.Scan(&id, &model, &modelID, &kind, &filename, &disk); err != nil {
			rows.Close()
			return report, err
		}
		userID := modelID
		switch model {
		case `App\Models\User`:
			if kind != "avatars" {
				rows.Close()
				return report, fmt.Errorf("unsupported user collection %q", kind)
			}
		case `App\Models\PersonalInfo`:
			var ok bool
			userID, ok = profileOwners[modelID]
			if !ok {
				rows.Close()
				return report, fmt.Errorf("orphan media %d", id)
			}
			if kind != "melli_card_scan" && kind != "certificate_scan" && kind != "bank_card_scan" {
				rows.Close()
				return report, fmt.Errorf("unsupported document collection %q", kind)
			}
		default:
			rows.Close()
			return report, fmt.Errorf("unsupported media model %q", model)
		}
		key := fmt.Sprintf("%d:%s", userID, kind)
		if seenMedia[key] {
			rows.Close()
			return report, fmt.Errorf("multiple media rows for %s; choose the current file before import", key)
		}
		seenMedia[key] = true
		if filepath.Base(filename) != filename || strings.ContainsAny(filename, `/\`) {
			rows.Close()
			return report, fmt.Errorf("unsafe media filename for row %d", id)
		}
		root := mediaRoot
		switch disk {
		case "local":
		case "public":
			root = filepath.Join(root, "public")
		default:
			rows.Close()
			return report, fmt.Errorf("unsupported media disk %q", disk)
		}
		path := filepath.Join(root, strconv.FormatInt(id, 10), filename)
		resolved, e := filepath.EvalSymlinks(path)
		if e != nil {
			rows.Close()
			return report, fmt.Errorf("media %d file unavailable: %w", id, e)
		}
		absRoot, e := filepath.Abs(root)
		if e != nil {
			rows.Close()
			return report, e
		}
		absPath, e := filepath.Abs(resolved)
		if e != nil {
			rows.Close()
			return report, e
		}
		relative, e := filepath.Rel(absRoot, absPath)
		if e != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			rows.Close()
			return report, fmt.Errorf("media %d escapes configured root", id)
		}
		info, e := os.Stat(absPath)
		if e != nil {
			rows.Close()
			return report, e
		}
		if info.Size() > 1<<20 {
			rows.Close()
			return report, fmt.Errorf("media %d exceeds 1 MB", id)
		}
		data, e := os.ReadFile(absPath)
		if e != nil {
			rows.Close()
			return report, e
		}
		if reason := domain.ValidateImage(filename, data); reason != "" {
			rows.Close()
			return report, fmt.Errorf("media %d: %s", id, reason)
		}
		media = append(media, domain.Media{UserID: userID, Kind: kind, ContentType: http.DetectContentType(data), Data: data})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	report.Media = len(media)
	// Validate referential integrity and uniqueness even for dry-run.
	seenID := map[int64]bool{}
	emails, codes, wallets := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, item := range users {
		u := item.user
		seenID[u.ID] = true
		for _, pair := range []struct {
			value string
			seen  map[string]bool
		}{{u.Email, emails}, {item.wallet, wallets}} {
			if pair.value != "" && pair.seen[pair.value] {
				return report, fmt.Errorf("duplicate email or wallet for user %d", u.ID)
			}
			pair.seen[pair.value] = true
		}
		if u.Code != nil {
			if codes[strings.ToLower(*u.Code)] {
				return report, fmt.Errorf("duplicate member code")
			}
			codes[strings.ToLower(*u.Code)] = true
		}
	}
	for id := range profiles {
		if !seenID[id] {
			return report, fmt.Errorf("orphan personal_info user %d", id)
		}
	}
	for _, m := range media {
		if !seenID[m.UserID] {
			return report, fmt.Errorf("orphan media owner %d", m.UserID)
		}
	}
	grants, err := readPassportGrants(ctx, read, seenID, clients)
	if err != nil {
		return report, err
	}
	for _, g := range grants {
		if g.Code {
			report.AuthorizationCodes++
		} else {
			report.AccessTokens++
		}
		if g.Refresh != "" {
			report.RefreshTokens++
		}
	}
	resets, err := readLaravelResets(ctx, read, users)
	if err != nil {
		return report, err
	}
	report.PasswordResets = len(resets)
	sessions, skipped, err := readLaravelFileSessions(ctx, options, users)
	if err != nil {
		return report, err
	}
	report.Sessions = len(sessions)
	report.SkippedSessions = skipped
	var occupied int
	if err = s.db.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM users)+(SELECT COUNT(*) FROM oauth_clients)`).Scan(&occupied); err != nil {
		return report, err
	}
	if occupied > 0 {
		return report, fmt.Errorf("import requires an empty target")
	}
	if !apply {
		return report, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	for _, item := range users {
		u := item.user
		if _, err = tx.ExecContext(ctx, `INSERT INTO users(id,mobile,updated_at,name,email,password_hash,code,referral,email_verified_at,created_at) VALUES(?,?,?,?,NULLIF(?,''),NULLIF(?,''),?,?,?,?)`, u.ID, u.Mobile, u.UpdatedAt, u.Name, u.Email, u.PasswordHash, u.Code, u.Referral, u.EmailVerifiedAt, u.CreatedAt); err != nil {
			return report, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO personal_infos(user_id) VALUES(?)`, u.ID); err != nil {
			return report, err
		}
		if p, ok := profiles[u.ID]; ok {
			if err = writeProfile(ctx, tx, p); err != nil {
				return report, err
			}
			if _, err = tx.ExecContext(ctx, `UPDATE personal_infos SET is_verified=? WHERE user_id=?`, p.IsVerified, u.ID); err != nil {
				return report, err
			}
		}
		if item.wallet != "" {
			if _, err = tx.ExecContext(ctx, `INSERT INTO wallets(user_id,address) VALUES(?,?)`, u.ID, item.wallet); err != nil {
				return report, err
			}
		}
	}
	for _, item := range users {
		if item.remember != "" {
			if _, err = tx.ExecContext(ctx, `INSERT INTO legacy_remember_tokens(user_id,token_hash,expires_at) VALUES(?,?,?)`, item.user.ID, securityHash(item.remember), stamp(time.Now().Add(400*24*time.Hour))); err != nil {
				return report, err
			}
		}
	}
	for _, session := range sessions {
		if _, err = tx.ExecContext(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, session.Hash, session.User, stamp(session.Expiry)); err != nil {
			return report, err
		}
		if session.Wallet {
			if _, err = tx.ExecContext(ctx, `INSERT INTO session_attributes(token_hash,name,value,expires_at) VALUES(?,'wallet_login','true',?)`, session.Hash, stamp(session.Expiry)); err != nil {
				return report, err
			}
		}
		if session.Confirmed > 0 {
			expiry := time.Unix(session.Confirmed, 0).Add(3 * time.Hour)
			if expiry.After(session.Expiry) {
				expiry = session.Expiry
			}
			if expiry.After(time.Now()) {
				if _, err = tx.ExecContext(ctx, `INSERT INTO session_attributes(token_hash,name,value,expires_at) VALUES(?,'password_confirmed_at',?,?)`, session.Hash, strconv.FormatInt(session.Confirmed, 10), stamp(expiry)); err != nil {
					return report, err
				}
			}
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE code_sequence SET value=? WHERE id=1`, maxCode); err != nil {
		return report, err
	}
	var personalID int64
	for _, c := range clients {
		if c.personal && !c.revoked {
			personalID = c.id
		}
	}
	for _, c := range clients {
		var purpose any
		if c.id == personalID {
			purpose = "personal_access"
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO oauth_clients(id,name,secret_hash,purpose,revoked,created_at) VALUES(?,?,NULLIF(?,''),?,?,?)`, c.id, c.name, c.secret, purpose, c.revoked, c.created); err != nil {
			return report, err
		}
		for _, uri := range c.redirects {
			if _, err = tx.ExecContext(ctx, `INSERT INTO oauth_client_redirects(client_id,uri_hash,uri) VALUES(?,?,?)`, c.id, securityHash(uri), uri); err != nil {
				return report, err
			}
		}
	}
	for _, reset := range resets {
		if _, err = tx.ExecContext(ctx, `INSERT INTO legacy_password_resets(user_id,token_hash,expires_at) VALUES(?,?,?)`, reset.User, reset.Hash, stamp(reset.Expiry)); err != nil {
			return report, err
		}
	}
	if err = importPassportGrants(ctx, tx, grants); err != nil {
		return report, err
	}
	for _, m := range media {
		if err = saveMedia(ctx, tx, m); err != nil {
			return report, err
		}
	}
	if err = tx.Commit(); err != nil {
		return report, err
	}
	report.Applied = true
	return report, nil
}
func securityHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func checkProfileLengths(p domain.PersonalInfo) error {
	for _, field := range []struct {
		name, value string
		max         int
	}{
		{"first_name", p.FirstName, 255}, {"last_name", p.LastName, 255}, {"mobile", p.Mobile, 11}, {"telephone", p.Telephone, 11}, {"national_code", p.NationalCode, 10}, {"address", p.Address, 255}, {"company_name", p.CompanyName, 255}, {"company_address", p.CompanyAddress, 255}, {"company_registration_number", p.CompanyRegistrationNumber, 32}, {"company_national_number", p.CompanyNationalNumber, 32}, {"company_tax_number", p.CompanyTaxNumber, 32}, {"company_executive_name", p.CompanyExecutiveName, 255},
	} {
		if utf8.RuneCountInString(field.value) > field.max {
			return fmt.Errorf("profile %d: %s exceeds %d characters", p.UserID, field.name, field.max)
		}
	}
	return nil
}
