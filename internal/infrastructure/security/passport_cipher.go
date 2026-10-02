package security

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
	"io"
	"strings"
)

// Passport's default token encryption uses Defuse V2 encryptWithPassword with
// the decoded Laravel encryption key. URL signatures use the literal APP_KEY.
type PassportCipher struct{ Key []byte }

func NewPassportCipher(appKey []byte) (PassportCipher, error) {
	key := appKey
	var err error
	if strings.HasPrefix(string(key), "base64:") {
		key, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(string(key), "base64:"))
	}
	if err != nil || len(key) != 32 {
		return PassportCipher{}, errors.New("Passport encryption requires a 32-byte APP_KEY")
	}
	return PassportCipher{Key: key}, nil
}
func (c PassportCipher) Decode(token string) ([]byte, error) {
	invalid := errors.New("invalid Passport encrypted token")
	if len(c.Key) != 32 || len(token) < 168 || len(token) > 16384 {
		return nil, invalid
	}
	raw, err := hex.DecodeString(token)
	if err != nil || len(raw) < 84 || !bytes.Equal(raw[:4], []byte{0xde, 0xf5, 2, 0}) {
		return nil, invalid
	}
	salt, iv := raw[4:36], raw[36:52]
	prehash := sha256.Sum256(c.Key)
	prekey := pbkdf2.Key(prehash[:], salt, 100000, 32, sha256.New)
	derive := func(info string) []byte {
		k := make([]byte, 32)
		_, _ = io.ReadFull(hkdf.New(sha256.New, prekey, salt, []byte(info)), k)
		return k
	}
	mac := hmac.New(sha256.New, derive("DefusePHP|V2|KeyForAuthentication"))
	mac.Write(raw[:len(raw)-32])
	if !hmac.Equal(mac.Sum(nil), raw[len(raw)-32:]) {
		return nil, invalid
	}
	block, err := aes.NewCipher(derive("DefusePHP|V2|KeyForEncryption"))
	if err != nil {
		return nil, invalid
	}
	plain := make([]byte, len(raw)-84)
	cipher.NewCTR(block, iv).XORKeyStream(plain, raw[52:len(raw)-32])
	return plain, nil
}
