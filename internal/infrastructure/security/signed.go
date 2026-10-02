package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func LoadAppKey(value, path string) ([]byte, error) {
	if value != "" {
		// Laravel UrlGenerator signs with the literal config app.key, including
		// its base64: prefix. Decoding it would break existing signed URLs.
		return []byte(value), nil
	}
	raw, err := os.ReadFile(path)
	if err == nil {
		return raw, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	_, err = file.Write(key)
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	return key, closeErr
}
func EmailFingerprint(email string) string {
	sum := sha1.Sum([]byte(email))
	return hex.EncodeToString(sum[:])
}
func SignedVerificationURL(origin string, id int64, email string, until time.Time, key []byte) string {
	raw := strings.TrimRight(origin, "/") + "/email/verify/" + strconv.FormatInt(id, 10) + "/" + EmailFingerprint(email) + "?expires=" + strconv.FormatInt(until.Unix(), 10)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(raw))
	return raw + "&signature=" + hex.EncodeToString(h.Sum(nil))
}
func ValidLaravelSignature(origin string, path, rawQuery string, now time.Time, key []byte) bool {
	if len(key) == 0 {
		return false
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil || len(values["signature"]) != 1 || len(values["expires"]) != 1 {
		return false
	}
	expiry, err := strconv.ParseInt(values.Get("expires"), 10, 64)
	if err != nil || expiry < now.Unix() {
		return false
	}
	signature, err := hex.DecodeString(values.Get("signature"))
	if err != nil {
		return false
	}
	var parts []string
	for _, part := range strings.Split(rawQuery, "&") {
		name := strings.SplitN(part, "=", 2)[0]
		decoded, e := url.QueryUnescape(name)
		if e != nil {
			return false
		}
		if decoded != "signature" {
			parts = append(parts, part)
		}
	}
	raw := strings.TrimRight(origin, "/") + path + "?" + strings.Join(parts, "&")
	h := hmac.New(sha256.New, key)
	h.Write([]byte(raw))
	return hmac.Equal(signature, h.Sum(nil))
}
