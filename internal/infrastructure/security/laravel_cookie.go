package security

import (
	"bytes"
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
	plain, err := c.decryptLaravelEnvelope(value, 16384)
	if err != nil {
		return "", invalid
	}
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

// decryptLaravelEnvelope verifies the MAC before AES-CBC decryption.
func (c PassportCipher) decryptLaravelEnvelope(value string, limit int) ([]byte, error) {
	invalid := errors.New("invalid Laravel encrypted payload")
	if len(c.Key) != 32 || len(value) > limit {
		return nil, invalid
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, invalid
	}
	var envelope struct {
		IV    string `json:"iv"`
		Value string `json:"value"`
		MAC   string `json:"mac"`
		Tag   string `json:"tag"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Tag != "" {
		return nil, invalid
	}
	mac := hmac.New(sha256.New, c.Key)
	mac.Write([]byte(envelope.IV + envelope.Value))
	expected, err := hex.DecodeString(envelope.MAC)
	if err != nil || !hmac.Equal(mac.Sum(nil), expected) {
		return nil, invalid
	}
	iv, err := base64.StdEncoding.DecodeString(envelope.IV)
	if err != nil || len(iv) != aes.BlockSize {
		return nil, invalid
	}
	encrypted, err := base64.StdEncoding.DecodeString(envelope.Value)
	if err != nil || len(encrypted) == 0 || len(encrypted)%aes.BlockSize != 0 {
		return nil, invalid
	}
	block, err := aes.NewCipher(c.Key)
	if err != nil {
		return nil, invalid
	}
	plain := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, encrypted)
	padding := int(plain[len(plain)-1])
	if padding < 1 || padding > aes.BlockSize || padding > len(plain) {
		return nil, invalid
	}
	for _, b := range plain[len(plain)-padding:] {
		if int(b) != padding {
			return nil, invalid
		}
	}
	plain = plain[:len(plain)-padding]
	return plain, nil
}

// ParseLaravelFileSession accepts unencrypted PHP arrays and EncryptedStore
// payloads, without invoking PHP unserialize or constructing PHP objects.
func ParseLaravelFileSession(raw, appKey []byte) (map[string]any, error) {
	if len(raw) > 1<<20 {
		return nil, errors.New("session exceeds limit")
	}
	if bytes.HasPrefix(raw, []byte("a:")) {
		return ParsePHPSession(raw)
	}
	decoder, err := NewPassportCipher(appKey)
	if err != nil {
		return nil, err
	}
	plain, err := decoder.decryptLaravelEnvelope(string(raw), 1<<20)
	if err != nil {
		return nil, err
	}
	parser := phpParser{raw: plain}
	value, err := parser.value(0)
	serialized, ok := value.(string)
	if err != nil || !ok || parser.pos != len(plain) {
		return nil, errors.New("invalid encrypted session serialization")
	}
	return ParsePHPSession([]byte(serialized))
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
