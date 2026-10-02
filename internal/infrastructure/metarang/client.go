// Package metarang queries the Metarang platform to decide whether a wallet
// logging in for the first time belongs to an already registered member,
// porting App\Services\MetarangWalletClient.
package metarang

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client looks up wallet registrations on the Metarang platform.
type Client struct {
	// BaseURL is the Metarang host (the Metarang_API setting). Empty means
	// not configured, and every lookup fails like the Laravel client.
	BaseURL string
	HTTP    *http.Client
}

func (c Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// LookupRegistration returns whether address is already registered on the
// Metarang side and, when it is, the member code to attach the wallet to.
// Every failure mode (unconfigured, connection error, non-2xx status,
// non-JSON or mistyped payload, registered wallet without a user code)
// returns an error, and the caller answers 502 exactly like the Laravel
// MetarangWalletLookupException path.
func (c Client) LookupRegistration(ctx context.Context, address string) (bool, string, error) {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return false, "", errors.New("Metarang API is not configured.")
	}
	body, err := json.Marshal(map[string]string{"wallet_address": address})
	if err != nil {
		return false, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/wallets/registered", bytes.NewReader(body))
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return false, "", fmt.Errorf("metarang wallet lookup: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, "", fmt.Errorf("metarang wallet lookup: unexpected status %d", resp.StatusCode)
	}
	var payload struct {
		AlreadyRegistered *bool   `json:"already_registered"`
		UserCode          *string `json:"user_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, "", fmt.Errorf("metarang wallet lookup: %w", err)
	}
	if payload.AlreadyRegistered == nil {
		return false, "", errors.New("Invalid Metarang wallet registration response.")
	}
	code := ""
	if payload.UserCode != nil {
		code = *payload.UserCode
	}
	if *payload.AlreadyRegistered && code == "" {
		return false, "", errors.New("Metarang did not return a user code for the registered wallet.")
	}
	return *payload.AlreadyRegistered, code, nil
}
