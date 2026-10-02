package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// DecodeCookie accepts Laravel's default authenticated AES-256-CBC cookie
// envelope. It checks both the envelope MAC and the cookie-name binding.
func (c PassportCipher) DecodeCookie(name, value string) (string, error) {
	invalid := errors.New("invalid Laravel cookie")
	if len(c.Key) != 32 || len(value) > 16384 {
		return "", invalid
	}
	// Symfony sends non-raw cookies URL-encoded; net/http retains that encoding.
	if strings.Contains(value, "%") {
		decoded, err := url.PathUnescape(value)
		if err != nil {
			return "", invalid
		}
		value = decoded
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", invalid
	}
	var envelope struct {
		IV    string `json:"iv"`
		Value string `json:"value"`
		MAC   string `json:"mac"`
		Tag   string `json:"tag"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Tag != "" {
		return "", invalid
	}
	mac := hmac.New(sha256.New, c.Key)
	mac.Write([]byte(envelope.IV + envelope.Value))
	expected, err := hex.DecodeString(envelope.MAC)
	if err != nil || !hmac.Equal(mac.Sum(nil), expected) {
		return "", invalid
	}
	iv, err := base64.StdEncoding.DecodeString(envelope.IV)
	if err != nil || len(iv) != aes.BlockSize {
		return "", invalid
	}
	encrypted, err := base64.StdEncoding.DecodeString(envelope.Value)
	if err != nil || len(encrypted) == 0 || len(encrypted)%aes.BlockSize != 0 {
		return "", invalid
	}
	block, err := aes.NewCipher(c.Key)
	if err != nil {
		return "", invalid
	}
	plain := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, encrypted)
	padding := int(plain[len(plain)-1])
	if padding < 1 || padding > aes.BlockSize || padding > len(plain) {
		return "", invalid
	}
	for _, b := range plain[len(plain)-padding:] {
		if int(b) != padding {
			return "", invalid
		}
	}
	plain = plain[:len(plain)-padding]
	prefix := hmac.New(sha1.New, c.Key)
	prefix.Write([]byte(name + "v2"))
	bound := hex.EncodeToString(prefix.Sum(nil)) + "|"
	if len(plain) < len(bound) || !hmac.Equal(plain[:len(bound)], []byte(bound)) {
		return "", invalid
	}
	result := string(plain[len(bound):])
	if strings.ContainsRune(result, 0) {
		return "", invalid
	}
	return result, nil
}
func LaravelPasswordMAC(hash string, literalAppKey []byte) string {
	mac := hmac.New(sha256.New, literalAppKey)
	mac.Write([]byte(hash))
	return hex.EncodeToString(mac.Sum(nil))
}

func SessionGuardHash() string {
	sum := sha1.Sum([]byte("Illuminate\\Auth\\SessionGuard"))
	return hex.EncodeToString(sum[:])
}
