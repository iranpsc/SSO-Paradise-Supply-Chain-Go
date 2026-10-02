// seed-local prepares manually usable credentials on a development MySQL only.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/envfile"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/metarang"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"golang.org/x/crypto/sha3"
)

const demoPassword = "TestPass!2026"

type account struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Verified bool   `json:"verified"`
	ID       int64  `json:"id"`
	Code     string `json:"code"`
}
type wallet struct {
	Scenario   string `json:"scenario"`
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
}
type manifest struct {
	Accounts []account `json:"accounts"`
	Wallets  []wallet  `json:"wallets"`
}

func main() {
	envfile.MustLoad()
	check := flag.Bool("check", false, "check existing seed credentials through the local frontend without consuming signup/link wallets")
	flag.Parse()
	if *check {
		if err := checkFrontend(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	cfg := mysql.ConfigFromEnv()
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil || os.Getenv("APP_ENV") != "development" || !(host == "localhost" || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())) || cfg.DBName == "sso" {
		return errors.New("seed-local only supports a separate development database on localhost")
	}
	root := filepath.Join("var", "test-data")
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	file := filepath.Join(root, "credentials.json")
	var m manifest
	if data, err := os.ReadFile(file); err == nil {
		if err := json.Unmarshal(data, &m); err != nil {
			return errors.New("invalid existing seed manifest")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(m.Wallets) == 0 {
		for _, scenario := range []string{"wallet_login", "wallet_unverified", "wallet_signup", "wallet_link", "metarang_member"} {
			key, err := btcec.NewPrivateKey()
			if err != nil {
				return err
			}
			h := sha3.NewLegacyKeccak256()
			h.Write(key.PubKey().SerializeUncompressed()[1:])
			sum := h.Sum(nil)
			m.Wallets = append(m.Wallets, wallet{scenario, "0x" + hex.EncodeToString(sum[12:]), hex.EncodeToString(key.Serialize())})
		}
	}
	if len(m.Wallets) != 5 {
		return errors.New("unexpected seed wallet manifest")
	}
	// Save generated keys before database writes so retries retain the same wallets.
	if err := writeJSON(file, m); err != nil {
		return err
	}
	db, err := mysql.Open(cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	hash, err := (security.Bcrypt{Cost: 10}).Hash(demoPassword)
	if err != nil {
		return err
	}
	definitions := []account{{Username: "demo_verified", Email: "verified@example.test", Verified: true}, {Username: "demo_pending", Email: "pending@example.test"}, {Username: "demo_wallet", Email: "wallet@example.test", Verified: true}, {Username: "demo_wallet_pending", Email: "wallet.pending@example.test"}, {Username: "demo_member", Email: "member@example.test", Verified: true}}
	m.Accounts = nil
	for _, item := range definitions {
		u, err := db.ByLogin(ctx, item.Email)
		if errors.Is(err, domain.ErrNotFound) {
			u, err = db.Create(ctx, domain.User{Name: strings.ReplaceAll(item.Username, "_", " "), Username: item.Username, Email: item.Email, PasswordHash: hash, CreatedAt: time.Now()})
			if err == nil && item.Verified {
				err = db.VerifySignedEmail(ctx, u.ID, u.Email, time.Now())
			}
		} else if err == nil && u.Username != item.Username {
			return errors.New("seed email belongs to a different account")
		}
		if err != nil {
			return err
		}
		item.ID = u.ID
		item.Password = demoPassword
		if u.Code != nil {
			item.Code = *u.Code
		}
		m.Accounts = append(m.Accounts, item)
	}
	for _, binding := range []struct{ account, wallet int }{{2, 0}, {3, 1}} {
		outcome, err := db.AttachWallet(ctx, m.Accounts[binding.account].ID, m.Wallets[binding.wallet].Address)
		if err != nil {
			return err
		}
		if outcome == domain.WalletLinkAlreadyLinked {
			return errors.New("seed wallet belongs to another account")
		}
	}
	fixtures := map[string]metarang.Fixture{m.Wallets[4].Address: {Registered: true, Code: m.Accounts[4].Code}}
	if err := writeJSON(filepath.Join(root, "metarang.json"), fixtures); err != nil {
		return err
	}
	if err := writeJSON(file, m); err != nil {
		return err
	}
	var guide strings.Builder
	fmt.Fprintln(&guide, "# Local frontend test credentials\n\nURL: http://localhost:3000\n\nAll passwords: `"+demoPassword+"`\n\n| Username | Email | Verified |\n|---|---|---|")
	for _, a := range m.Accounts {
		fmt.Fprintf(&guide, "| %s | %s | %t |\n", a.Username, a.Email, a.Verified)
	}
	fmt.Fprintln(&guide, "\nFresh email registration: `new.member@example.test`, username `new_member` (use a new suffix after each successful registration).\n\nImport a private key into a separate MetaMask test account. Keys have no funds and are for local testing only. WalletConnect uses the same account imported into a compatible mobile wallet.\n\n| Scenario | Address | Private key |\n|---|---|---|")
	for _, w := range m.Wallets {
		fmt.Fprintf(&guide, "| %s | `%s` | `%s` |\n", w.Scenario, w.Address, w.PrivateKey)
	}
	fmt.Fprintln(&guide, "\nwallet_login: existing verified account. wallet_unverified: verification screen. wallet_signup: unused wallet; first signature creates an account, later signatures log in. wallet_link: unused wallet to connect from a password account. metarang_member: membership lookup finds demo_member by code and attaches its wallet.\n\nLogout between scenarios. Use demo_verified for wallet_link. Try wallet_login while logged into demo_verified to test ownership conflict. Test invalid password with WrongPass!2026; five failures lock that email/IP for one minute. Duplicate username/email registration should show validation errors. Verification/reset mail is written to var/mail. Local Metarang fixtures affect only membership lookup; signatures and nonce validation remain real. These fixtures are rejected in production.\n\nRe-running seed-local keeps wallet keys and existing accounts; it does not erase registrations, detach wallets or reset changed passwords.")
	if err := os.WriteFile(filepath.Join(root, "frontend-test-guide.md"), []byte(guide.String()), 0600); err != nil {
		return err
	}
	fmt.Printf("Prepared %d local accounts and %d wallet scenarios. Guide: %s\n", len(m.Accounts), len(m.Wallets), filepath.Join(root, "frontend-test-guide.md"))
	return nil
}
func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}
