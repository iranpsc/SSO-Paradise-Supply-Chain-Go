package mysql

import (
	"context"
	"crypto/subtle"
	"fmt"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ImportOptions struct {
	SessionsRoot    string
	SessionLifetime time.Duration
	AppKey          []byte
}
type importedSession struct {
	CSRF      string
	Hash      string
	User      int64
	Expiry    time.Time
	Wallet    bool
	Confirmed int64
}

func readLaravelFileSessions(ctx context.Context, options ImportOptions, users []importedUser) ([]importedSession, int, error) {
	if options.SessionsRoot == "" {
		return nil, 0, nil
	}
	if options.SessionLifetime <= 0 {
		options.SessionLifetime = 2 * time.Hour
	}
	entries, err := os.ReadDir(options.SessionsRoot)
	if err != nil {
		return nil, 0, err
	}
	byID := map[int64]importedUser{}
	for _, u := range users {
		byID[u.user.ID] = u
	}
	var result []importedSession
	skipped := 0
	now := time.Now()
	for _, entry := range entries {
		if err = ctx.Err(); err != nil {
			return nil, skipped, err
		}
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if len(entry.Name()) != 40 || strings.IndexFunc(entry.Name(), func(r rune) bool {
			return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
		}) >= 0 {
			continue
		}
		info, e := entry.Info()
		if e != nil {
			return nil, skipped, e
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > 1<<20 {
			return nil, skipped, fmt.Errorf("unsupported session file")
		}
		expiry := info.ModTime().Add(options.SessionLifetime)
		if !expiry.After(now) {
			skipped++
			continue
		}
		raw, e := os.ReadFile(filepath.Join(options.SessionsRoot, entry.Name()))
		if e != nil {
			return nil, skipped, e
		}
		data, e := security.ParseLaravelFileSession(raw, options.AppKey)
		if e != nil {
			return nil, skipped, fmt.Errorf("unsupported source session serialization; no session was imported: %w", e)
		}
		var owner int64
		for key, value := range data {
			if strings.HasPrefix(key, "login_web_") {
				switch value := value.(type) {
				case int64:
					owner = value
				case string:
					owner, _ = strconv.ParseInt(value, 10, 64)
				}
			}
		}
		if owner == 0 {
			skipped++
			continue
		}
		u, ok := byID[owner]
		if !ok {
			return nil, skipped, fmt.Errorf("orphan Laravel session")
		}
		if stored, ok := data["password_hash_web"].(string); ok {
			mac := security.LaravelPasswordMAC(u.user.PasswordHash, options.AppKey)
			if subtle.ConstantTimeCompare([]byte(stored), []byte(u.user.PasswordHash)) != 1 && subtle.ConstantTimeCompare([]byte(stored), []byte(mac)) != 1 {
				skipped++
				continue
			}
		}
		session := importedSession{Hash: securityHash(entry.Name()), User: owner, Expiry: expiry}
		session.Wallet, _ = data["wallet_login"].(bool)
		session.Confirmed, _ = data["auth.password_confirmed_at"].(int64)
		session.CSRF, _ = data["_token"].(string)
		if len(session.CSRF) > 128 {
			return nil, skipped, fmt.Errorf("invalid source CSRF token length")
		}
		result = append(result, session)
	}
	return result, skipped, nil
}
