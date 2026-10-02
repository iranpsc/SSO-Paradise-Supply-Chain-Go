package security

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// BreachedPasswords implements Laravel's uncompromised password rule using the
// same range API. Only the first five SHA-1 characters leave this process.
// SHA-1 is a lookup identifier here, never a password storage algorithm.
type BreachedPasswords struct {
	Client   *http.Client
	URL      string
	FailOpen bool
	OnError  func(error)
}

func (b BreachedPasswords) Compromised(ctx context.Context, password string) (compromised bool, failure error) {
	defer func() {
		if failure != nil && b.FailOpen && ctx.Err() == nil {
			if b.OnError != nil {
				b.OnError(failure)
			}
			compromised, failure = false, nil
		}
	}()
	h := sha1.Sum([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(h[:]))
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, b.URL+"/range/"+hash[:5], nil)
	if err != nil {
		return false, err
	}
	r.Header.Set("Add-Padding", "true")
	response, err := b.Client.Do(r)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, errors.New("password safety service unavailable")
	}
	scanner := bufio.NewScanner(io.LimitReader(response.Body, 2<<20))
	valid := false
	for scanner.Scan() {
		suffix, count, ok := strings.Cut(strings.TrimSpace(scanner.Text()), ":")
		if !ok || len(suffix) != 35 {
			continue
		}
		n, err := strconv.Atoi(count)
		if err != nil || n < 0 {
			continue
		}
		valid = true
		if suffix == hash[5:] && n > 0 {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	if !valid {
		return false, errors.New("invalid password safety response")
	}
	return false, nil
}
