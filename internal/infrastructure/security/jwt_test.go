package security

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestJWTRejectsTamperingWrongKeyAndExpiry(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	j := JWT{key}
	now := time.Now()
	id := strings.Repeat("a", 64)
	token, err := j.SignAccess(id, 2, 3, []string{"profile"}, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := j.VerifyAccess(token, now); err != nil || got != id {
		t.Fatal("valid token rejected")
	}
	parts := strings.Split(token, ".")
	parts[0] = base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"JWT","alg":"none"}`))
	if _, err = j.VerifyAccess(strings.Join(parts, "."), now); err == nil {
		t.Fatal("alg none accepted")
	}
	if _, err = j.VerifyAccess(token, now.Add(time.Hour)); err == nil {
		t.Fatal("expired token accepted")
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = (JWT{other}).VerifyAccess(token, now); err == nil {
		t.Fatal("wrong key accepted")
	}
}
func TestLaravelSignedURL(t *testing.T) {
	key := []byte("laravel-application-key")
	now := time.Now()
	link := SignedVerificationURL("https://accounts.example", 12, "a@example.com", now.Add(time.Hour), key)
	parts := strings.SplitN(strings.TrimPrefix(link, "https://accounts.example"), "?", 2)
	if !ValidLaravelSignature("https://accounts.example", parts[0], parts[1], now, key) {
		t.Fatal("valid signature rejected")
	}
	if ValidLaravelSignature("https://evil.example", parts[0], parts[1], now, key) || ValidLaravelSignature("https://accounts.example", parts[0], parts[1]+"&extra=1", now, key) || ValidLaravelSignature("https://accounts.example", parts[0], parts[1], now.Add(2*time.Hour), key) {
		t.Fatal("invalid signature accepted")
	}
}
