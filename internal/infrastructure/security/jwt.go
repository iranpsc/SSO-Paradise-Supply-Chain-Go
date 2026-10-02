package security

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type JWT struct{ Key *rsa.PrivateKey }
type accessClaims struct {
	Audience  jwtAudience `json:"aud"`
	ID        string      `json:"jti"`
	IssuedAt  float64     `json:"iat"`
	NotBefore float64     `json:"nbf"`
	ExpiresAt float64     `json:"exp"`
	Subject   string      `json:"sub"`
	Scopes    []string    `json:"scopes"`
}

func LoadJWT(path string, create bool) (JWT, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && create {
		key, e := rsa.GenerateKey(rand.Reader, 2048)
		if e != nil {
			return JWT{}, e
		}
		if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			return JWT{}, e
		}
		out, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if errors.Is(e, os.ErrExist) {
			return LoadJWT(path, false)
		}
		if e != nil {
			return JWT{}, e
		}
		encoded := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
		_, e = out.Write(encoded)
		closeErr := out.Close()
		if e != nil {
			return JWT{}, e
		}
		if closeErr != nil {
			return JWT{}, closeErr
		}
		return JWT{key}, nil
	}
	if err != nil {
		return JWT{}, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return JWT{}, errors.New("invalid OAuth private-key PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsed, e := x509.ParsePKCS8PrivateKey(block.Bytes)
		if e != nil {
			return JWT{}, e
		}
		var ok bool
		key, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return JWT{}, errors.New("OAuth key must be RSA")
		}
	}
	if key.N.BitLen() < 2048 {
		return JWT{}, errors.New("OAuth RSA key must be at least 2048 bits")
	}
	if err = key.Validate(); err != nil {
		return JWT{}, err
	}
	return JWT{key}, nil
}
func (j JWT) SignAccess(id string, clientID, userID int64, scopes []string, now, until time.Time) (string, error) {
	if j.Key == nil {
		return "", errors.New("OAuth signing key missing")
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"JWT","alg":"RS256"}`))
	raw, err := json.Marshal(accessClaims{[]string{strconv.FormatInt(clientID, 10)}, id, numericDate(now), numericDate(now), numericDate(until), strconv.FormatInt(userID, 10), scopes})
	if err != nil {
		return "", err
	}
	signing := header + "." + base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(signing))
	signature, err := rsa.SignPKCS1v15(rand.Reader, j.Key, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}
func (j JWT) VerifyAccess(token string, now time.Time) (string, error) {
	invalid := errors.New("invalid OAuth access token")
	if j.Key == nil || len(token) > 8192 {
		return "", invalid
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", invalid
	}
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", invalid
	}
	var h struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	if json.Unmarshal(header, &h) != nil || h.Algorithm != "RS256" || h.Type != "JWT" {
		return "", invalid
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", invalid
	}
	hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(&j.Key.PublicKey, crypto.SHA256, hash[:], signature) != nil {
		return "", invalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", invalid
	}
	var c accessClaims
	if json.Unmarshal(payload, &c) != nil || (len(c.ID) != 64 && len(c.ID) != 80) || c.NotBefore > numericDate(now) || c.IssuedAt > numericDate(now) || c.ExpiresAt <= numericDate(now) || len(c.Audience) != 1 || c.Subject == "" {
		return "", invalid
	}
	return c.ID, nil
}

type jwtAudience []string

func (a jwtAudience) MarshalJSON() ([]byte, error) {
	if len(a) == 1 {
		return json.Marshal(a[0])
	}
	return json.Marshal([]string(a))
}
func (a *jwtAudience) UnmarshalJSON(raw []byte) error {
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		*a = jwtAudience{single}
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return err
	}
	*a = values
	return nil
}
func numericDate(t time.Time) float64 { return float64(t.UnixMicro()) / 1e6 }
