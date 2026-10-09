package httpapi_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"golang.org/x/crypto/sha3"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/transport/httpapi"
)

// stubRegistry is a controllable WalletRegistry (the Http::fake equivalent).
// hook runs inside the lookup to simulate a user created mid-lookup.
type stubRegistry struct {
	registered bool
	code       string
	err        error
	hook       func(address string)
	calls      int
	last       string
}

func (s *stubRegistry) LookupRegistration(_ context.Context, address string) (bool, string, error) {
	s.calls++
	s.last = address
	if s.hook != nil {
		s.hook(address)
	}
	return s.registered, s.code, s.err
}

type web3Fixture struct {
	handler http.Handler
	db      *mysql.Store
	auth    *application.Auth
	reg     *stubRegistry
	mail    *mailboxCapture
}

func web3Server(t *testing.T) *web3Fixture {
	t.Helper()
	db := mysqltest.Open(t)
	m := &mailboxCapture{}
	a := &application.Auth{Accounts: db, Sessions: db, Actions: db, Passwords: security.Bcrypt{Cost: 4}, Mailer: m, Now: time.Now, PublicURL: "http://localhost:3000", SessionTTL: time.Hour}
	reg := &stubRegistry{}
	w := &application.Web3{Wallets: db, Challenges: db, Attributes: db, Registry: reg, Sessions: db, Now: time.Now, SessionTTL: time.Hour, AppName: "Laravel", PublicURL: "http://localhost:3000"}
	p := &application.Profile{Store: db}
	h := httpapi.New(a, w, p, a.PublicURL, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return &web3Fixture{handler: h, db: db, auth: a, reg: reg, mail: m}
}

var web3CurveN = btcec.S256().N
var web3HalfN = new(big.Int).Rsh(new(big.Int).Set(web3CurveN), 1)
var errBoom = errors.New("metarang boom")

func web3Key(t *testing.T) (*btcec.PrivateKey, string) {
	t.Helper()
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		t.Fatal(err)
	}
	priv, _ := btcec.PrivKeyFromBytes(seed)
	pub := priv.PubKey().SerializeUncompressed()
	h := sha3.NewLegacyKeccak256()
	h.Write(pub[1:])
	return priv, "0x" + hex.EncodeToString(h.Sum(nil)[12:])
}

// web3Sign produces an Ethereum personal_sign signature with low-S
// normalization, the same construction as Web3AuthTest::signMessage.
func web3Sign(t *testing.T, priv *btcec.PrivateKey, message string) string {
	t.Helper()
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte("\x19Ethereum Signed Message:\n" + strconv.Itoa(len(message)) + message))
	compact := ecdsa.SignCompact(priv, h.Sum(nil), false)
	r, sBytes, v := compact[1:33], compact[33:65], compact[0]
	if new(big.Int).SetBytes(sBytes).Cmp(web3HalfN) > 0 {
		neg := new(big.Int).Sub(web3CurveN, new(big.Int).SetBytes(sBytes)).Bytes()
		padded := make([]byte, 32)
		copy(padded[32-len(neg):], neg)
		sBytes = padded
		v = 27 + ((v - 27) ^ 1)
	}
	return "0x" + hex.EncodeToString(append(append(r, sBytes...), v))
}

func web3Nonce(t *testing.T, f *web3Fixture, address string, cookie *http.Cookie) string {
	t.Helper()
	w := request(f.handler, "GET", "/api/web3/nonce?address="+address, "", "", cookie)
	if w.Code != 200 {
		t.Fatalf("nonce %d %s", w.Code, w.Body.String())
	}
	var body struct {
		Nonce string `json:"nonce"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Nonce == "" {
		t.Fatalf("bad nonce body %s", w.Body.String())
	}
	return body.Nonce
}

func web3Verify(t *testing.T, f *web3Fixture, address, signature string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	origin := ""
	if cookie != nil {
		origin = "http://localhost:3000"
	}
	payload := `{"address":"` + address + `","signature":"` + signature + `"}`
	return request(f.handler, "POST", "/api/web3/verify", payload, origin, cookie)
}

func sessionCookie(t *testing.T, w interface{ Result() *http.Response }) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == "paradise_session" && c.Value != "" {
			return c
		}
	}
	t.Fatal("no session cookie")
	return nil
}

func web3User(t *testing.T, f *web3Fixture, address string) domain.User {
	t.Helper()
	u, err := f.db.UserByWallet(context.Background(), address)
	if err != nil {
		t.Fatalf("wallet user missing: %v", err)
	}
	return u
}

func TestWeb3NonceIssuesMessageWithoutUser(t *testing.T) {
	f := web3Server(t)
	address := "0x90f8bfac9c63c35718a7a77e94b002d274950e89"
	nonce := web3Nonce(t, f, address, nil)
	for _, part := range []string{"ورود به حساب کاربری", "آدرس کیف پول: " + address, "کد یک‌بارمصرف: "} {
		if !strings.Contains(nonce, part) {
			t.Fatalf("nonce missing %q: %s", part, nonce)
		}
	}
	if _, err := f.db.UserByWallet(context.Background(), address); err == nil {
		t.Fatal("nonce created a user")
	}
}

func TestWeb3NonceRejectsInvalidAddress(t *testing.T) {
	f := web3Server(t)
	for _, address := range []string{"invalid-eth-address", "", "0x123", "90f8bfac9c63c35718a7a77e94b002d274950e89"} {
		w := request(f.handler, "GET", "/api/web3/nonce?address="+address, "", "", nil)
		if w.Code != 422 {
			t.Fatalf("%q: got %d %s", address, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"address"`) {
			t.Fatalf("%q: missing address field error: %s", address, w.Body.String())
		}
	}
}

func TestWeb3VerifyLogsInNewWalletUser(t *testing.T) {
	f := web3Server(t)
	priv, address := web3Key(t)
	w := web3Verify(t, f, address, web3Sign(t, priv, web3Nonce(t, f, address, nil)), nil)
	if w.Code != 200 {
		t.Fatalf("verify %d %s", w.Code, w.Body.String())
	}
	var body struct {
		Message  string `json:"message"`
		Redirect string `json:"redirect"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Message != "ورود با کیف پول با موفقیت انجام شد." || body.Redirect != "http://localhost:3000/home" {
		t.Fatalf("unexpected body %+v", body)
	}
	cookie := sessionCookie(t, w)
	u := web3User(t, f, address)
	if u.Name != "User_"+address[2:8] {
		t.Fatalf("unexpected wallet name %q", u.Name)
	}
	if u.EmailVerifiedAt == nil {
		t.Fatal("wallet user is not verified")
	}
	if u.Code == nil || *u.Code != "hm-2000000" {
		t.Fatalf("unexpected wallet code %+v", u.Code)
	}
	if u.Email != "" || u.Wallet == nil || *u.Wallet != address {
		t.Fatalf("unexpected wallet user %+v", u)
	}
	if _, err := f.db.PersonalInfo(context.Background(), u.ID); err != nil {
		t.Fatalf("personal_infos missing: %v", err)
	}
	if f.reg.calls != 1 || f.reg.last != address {
		t.Fatalf("metarang not consulted once: %+v", f.reg)
	}
	attr, err := f.db.SessionAttribute(context.Background(), application.Digest(cookie.Value), "wallet_login", time.Now())
	if err != nil || attr != "true" {
		t.Fatalf("wallet_login flag missing: %q %v", attr, err)
	}
	acc := request(f.handler, "GET", "/api/account", "", "", cookie)
	if acc.Code != 200 || !strings.Contains(acc.Body.String(), `"wallet_address":"`+address+`"`) {
		t.Fatalf("account missing wallet: %d %s", acc.Code, acc.Body.String())
	}
}

func TestWeb3VerifyRejectsReplay(t *testing.T) {
	f := web3Server(t)
	priv, address := web3Key(t)
	sig := web3Sign(t, priv, web3Nonce(t, f, address, nil))
	if w := web3Verify(t, f, address, sig, nil); w.Code != 200 {
		t.Fatalf("first verify %d %s", w.Code, w.Body.String())
	}
	w := web3Verify(t, f, address, sig, nil)
	if w.Code != 422 {
		t.Fatalf("replay got %d %s", w.Code, w.Body.String())
	}
	var body struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Message != "درخواست امضا منقضی شده یا یافت نشد؛ دوباره تلاش کنید." {
		t.Fatalf("unexpected replay body %s", w.Body.String())
	}
}

func TestWeb3VerifyRejectsBadSignature(t *testing.T) {
	f := web3Server(t)
	priv, address := web3Key(t)
	web3Nonce(t, f, address, nil)
	w := web3Verify(t, f, address, web3Sign(t, priv, "something else entirely"), nil)
	if w.Code != 401 {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "امضای کیف پول معتبر نیست.") {
		t.Fatalf("unexpected body %s", w.Body.String())
	}
	if _, err := f.db.UserByWallet(context.Background(), address); err == nil {
		t.Fatal("failed login created a user")
	}
}

func TestWeb3LinkFlow(t *testing.T) {
	f := web3Server(t)
	cookie := registerVerified(t, f.handler, f.auth, f.mail, "linker", "linker@example.com")
	priv, address := web3Key(t)
	w := request(f.handler, "GET", "/api/web3/link/nonce?address="+address, "", "", cookie)
	if w.Code != 200 {
		t.Fatalf("link nonce %d %s", w.Code, w.Body.String())
	}
	var nonce struct {
		Nonce string `json:"nonce"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &nonce)
	if !strings.Contains(nonce.Nonce, "اتصال کیف پول به حساب کاربری") {
		t.Fatalf("unexpected link message %s", nonce.Nonce)
	}
	payload := `{"address":"` + address + `","signature":"` + web3Sign(t, priv, nonce.Nonce) + `"}`
	w = request(f.handler, "POST", "/api/web3/link", payload, "http://localhost:3000", cookie)
	if w.Code != 200 {
		t.Fatalf("link %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "کیف پول با موفقیت متصل شد.") {
		t.Fatalf("unexpected link body %s", w.Body.String())
	}
	u, err := f.auth.Authenticate(context.Background(), cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	if mine, err := f.db.WalletOf(context.Background(), u.ID); err != nil || mine != address {
		t.Fatalf("wallet not attached: %q %v", mine, err)
	}
	pub := request(f.handler, "GET", "/api/users/"+strconv.FormatInt(u.ID, 10), "", "", nil)
	if pub.Code != http.StatusNotFound {
		t.Fatalf("public user %d", pub.Code)
	}
	for _, leak := range []string{"wallet_address", address} {
		if strings.Contains(pub.Body.String(), leak) {
			t.Fatalf("public endpoint leaks wallet: %s", pub.Body.String())
		}
	}
}

func TestWeb3LinkRejectsTakenWallet(t *testing.T) {
	f := web3Server(t)
	cookieA := registerVerified(t, f.handler, f.auth, f.mail, "holder", "holder@example.com")
	priv, address := web3Key(t)
	web3Link(t, f, cookieA, priv, address)
	cookieB := registerVerified(t, f.handler, f.auth, f.mail, "intruder", "intruder@example.com")
	w := request(f.handler, "GET", "/api/web3/link/nonce?address="+address, "", "", cookieB)
	if w.Code != 422 || !strings.Contains(w.Body.String(), "این کیف پول به حساب دیگری متصل است.") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestWeb3LinkRejectsWhenAlreadyConnected(t *testing.T) {
	f := web3Server(t)
	cookie := registerVerified(t, f.handler, f.auth, f.mail, "linked", "linked@example.com")
	priv, address := web3Key(t)
	web3Link(t, f, cookie, priv, address)
	_, other := web3Key(t)
	w := request(f.handler, "GET", "/api/web3/link/nonce?address="+other, "", "", cookie)
	if w.Code != 422 || !strings.Contains(w.Body.String(), "این کیف پول قبلاً به همین حساب متصل شده است.") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

// web3Link runs the whole link flow for an authenticated cookie.
func web3Link(t *testing.T, f *web3Fixture, cookie *http.Cookie, priv *btcec.PrivateKey, address string) {
	t.Helper()
	w := request(f.handler, "GET", "/api/web3/link/nonce?address="+address, "", "", cookie)
	if w.Code != 200 {
		t.Fatalf("link nonce %d %s", w.Code, w.Body.String())
	}
	var nonce struct {
		Nonce string `json:"nonce"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &nonce)
	payload := `{"address":"` + address + `","signature":"` + web3Sign(t, priv, nonce.Nonce) + `"}`
	w = request(f.handler, "POST", "/api/web3/link", payload, "http://localhost:3000", cookie)
	if w.Code != 200 {
		t.Fatalf("link %d %s", w.Code, w.Body.String())
	}
}

func TestWeb3LoginSignatureCannotLink(t *testing.T) {
	f := web3Server(t)
	cookie := registerVerified(t, f.handler, f.auth, f.mail, "member", "member@example.com")
	priv, address := web3Key(t)
	nonce := web3Nonce(t, f, address, nil)
	payload := `{"address":"` + address + `","signature":"` + web3Sign(t, priv, nonce) + `"}`
	w := request(f.handler, "POST", "/api/web3/link", payload, "http://localhost:3000", cookie)
	if w.Code != 422 || !strings.Contains(w.Body.String(), "درخواست امضا منقضی شده یا یافت نشد") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestWeb3LinkSignatureCannotLogin(t *testing.T) {
	f := web3Server(t)
	cookie := registerVerified(t, f.handler, f.auth, f.mail, "member", "member@example.com")
	priv, address := web3Key(t)
	w := request(f.handler, "GET", "/api/web3/link/nonce?address="+address, "", "", cookie)
	if w.Code != 200 {
		t.Fatalf("link nonce %d %s", w.Code, w.Body.String())
	}
	var nonce struct {
		Nonce string `json:"nonce"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &nonce)
	guest := web3Verify(t, f, address, web3Sign(t, priv, nonce.Nonce), nil)
	if guest.Code != 422 || !strings.Contains(guest.Body.String(), "درخواست امضا منقضی شده یا یافت نشد") {
		t.Fatalf("got %d %s", guest.Code, guest.Body.String())
	}
}

func TestWeb3VerifyLinksForAuthenticatedUser(t *testing.T) {
	f := web3Server(t)
	cookie := registerVerified(t, f.handler, f.auth, f.mail, "owner", "owner@example.com")
	me, err := f.auth.Authenticate(context.Background(), cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	priv, address := web3Key(t)
	nonce := web3Nonce(t, f, address, nil)
	w := web3Verify(t, f, address, web3Sign(t, priv, nonce), cookie)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "کیف پول با موفقیت متصل شد.") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
	if f.reg.calls != 0 {
		t.Fatal("metarang consulted for an authenticated link")
	}
	owner := web3User(t, f, address)
	if owner.ID != me.ID {
		t.Fatalf("wallet linked to %d, want %d", owner.ID, me.ID)
	}
}

func TestWeb3NonceGuestOnly(t *testing.T) {
	f := web3Server(t)
	cookie := registerVerified(t, f.handler, f.auth, f.mail, "member", "member@example.com")
	w := request(f.handler, "GET", "/api/web3/nonce?address=0x90f8bfac9c63c35718a7a77e94b002d274950e89", "", "", cookie)
	if w.Code != http.StatusFound {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "http://localhost:3000/home" {
		t.Fatalf("unexpected location %q", loc)
	}
}

func TestWeb3UnverifiedWalletUserRedirect(t *testing.T) {
	f := web3Server(t)
	w := request(f.handler, "POST", "/api/register", `{"username":"unverified","name":"Unverified","email":"unverified@example.com","password":"Secure!2026","password_confirmation":"Secure!2026"}`, "", nil)
	if w.Code != 201 {
		t.Fatalf("register %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Data domain.User `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	priv, address := web3Key(t)
	if _, err := f.db.AttachWallet(context.Background(), created.Data.ID, address); err != nil {
		t.Fatal(err)
	}
	sig := web3Sign(t, priv, web3Nonce(t, f, address, nil))
	out := web3Verify(t, f, address, sig, nil)
	if out.Code != 200 {
		t.Fatalf("verify %d %s", out.Code, out.Body.String())
	}
	if !strings.Contains(out.Body.String(), `"redirect":"http://localhost:3000/email/verify"`) {
		t.Fatalf("unexpected body %s", out.Body.String())
	}
}

func TestWeb3MetarangRegisteredAttachesByCode(t *testing.T) {
	f := web3Server(t)
	cookie := registerVerified(t, f.handler, f.auth, f.mail, "existing", "existing@example.com")
	me, err := f.auth.Authenticate(context.Background(), cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	if me.Code == nil {
		t.Fatal("verified user has no code")
	}
	f.reg.registered, f.reg.code = true, *me.Code
	priv, address := web3Key(t)
	out := web3Verify(t, f, address, web3Sign(t, priv, web3Nonce(t, f, address, nil)), nil)
	if out.Code != 200 || !strings.Contains(out.Body.String(), "ورود با کیف پول با موفقیت انجام شد.") {
		t.Fatalf("got %d %s", out.Code, out.Body.String())
	}
	owner := web3User(t, f, address)
	if owner.ID != me.ID || owner.Name != me.Name || owner.Email != me.Email {
		t.Fatalf("wrong owner %+v", owner)
	}
}

func TestWeb3MetarangFailures(t *testing.T) {
	for name, setup := range map[string]func(*stubRegistry){
		"lookup error":        func(r *stubRegistry) { r.err = errBoom },
		"registered w/o code": func(r *stubRegistry) { r.registered = true },
		"code not found":      func(r *stubRegistry) { r.registered, r.code = true, "hm-9999999" },
	} {
		f := web3Server(t)
		setup(f.reg)
		priv, address := web3Key(t)
		out := web3Verify(t, f, address, web3Sign(t, priv, web3Nonce(t, f, address, nil)), nil)
		if out.Code != 502 || !strings.Contains(out.Body.String(), "ورود با کیف پول انجام نشد") {
			t.Fatalf("%s: got %d %s", name, out.Code, out.Body.String())
		}
		if _, err := f.db.UserByWallet(context.Background(), address); err == nil {
			t.Fatalf("%s: failure created a user", name)
		}
	}
}

func TestWeb3ExistingWalletSkipsMetarang(t *testing.T) {
	f := web3Server(t)
	cookie := registerVerified(t, f.handler, f.auth, f.mail, "holder", "holder@example.com")
	me, err := f.auth.Authenticate(context.Background(), cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	priv, address := web3Key(t)
	web3Link(t, f, cookie, priv, address)
	out := web3Verify(t, f, address, web3Sign(t, priv, web3Nonce(t, f, address, nil)), nil)
	if out.Code != 200 {
		t.Fatalf("verify %d %s", out.Code, out.Body.String())
	}
	if f.reg.calls != 0 {
		t.Fatal("metarang consulted for a known wallet")
	}
	if owner := web3User(t, f, address); owner.ID != me.ID {
		t.Fatalf("wrong owner %d", owner.ID)
	}
}

func TestWeb3RaceDuringLookupUsesExistingUser(t *testing.T) {
	f := web3Server(t)
	priv, address := web3Key(t)
	var winner domain.User
	f.reg.hook = func(addr string) {
		var err error
		winner, err = f.db.CreateWalletUser(context.Background(), addr, time.Now())
		if err != nil {
			t.Errorf("hook create: %v", err)
		}
	}
	out := web3Verify(t, f, address, web3Sign(t, priv, web3Nonce(t, f, address, nil)), nil)
	if out.Code != 200 {
		t.Fatalf("verify %d %s", out.Code, out.Body.String())
	}
	if owner := web3User(t, f, address); owner.ID != winner.ID {
		t.Fatalf("race winner %d ignored, owner %d", winner.ID, owner.ID)
	}
}

func TestWeb3CodesIncrement(t *testing.T) {
	f := web3Server(t)
	var codes []string
	for range 2 {
		priv, address := web3Key(t)
		out := web3Verify(t, f, address, web3Sign(t, priv, web3Nonce(t, f, address, nil)), nil)
		if out.Code != 200 {
			t.Fatalf("verify %d %s", out.Code, out.Body.String())
		}
		u := web3User(t, f, address)
		if u.Code == nil {
			t.Fatal("wallet user has no code")
		}
		codes = append(codes, *u.Code)
	}
	if codes[0] != "hm-2000000" || codes[1] != "hm-2000001" {
		t.Fatalf("codes did not increment from the sequence: %v", codes)
	}
}

func TestWeb3PasswordLoginSetsNoFlag(t *testing.T) {
	f := web3Server(t)
	request(f.handler, "POST", "/api/register", `{"username":"plain","name":"Plain","email":"plain@example.com","password":"Secure!2026","password_confirmation":"Secure!2026"}`, "", nil)
	w := request(f.handler, "POST", "/api/login", `{"email":"plain@example.com","password":"Secure!2026"}`, "", nil)
	if w.Code != 200 {
		t.Fatalf("login %d %s", w.Code, w.Body.String())
	}
	cookie := sessionCookie(t, w)
	_, err := f.db.SessionAttribute(context.Background(), application.Digest(cookie.Value), "wallet_login", time.Now())
	if err == nil {
		t.Fatal("password login set the wallet flag")
	}
}
