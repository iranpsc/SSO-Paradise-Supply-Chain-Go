package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"golang.org/x/crypto/sha3"
)

func checkFrontend() error {
	if os.Getenv("APP_ENV") != "development" {
		return errors.New("seed check only supports development")
	}
	data, err := os.ReadFile(filepath.Join("var", "test-data", "credentials.json"))
	if err != nil {
		return errors.New("run seed-local first")
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	call := func(method, path string, payload any, cookie *http.Cookie) (int, map[string]any, []*http.Cookie, error) {
		body, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, nil, err
		}
		req, err := http.NewRequest(method, "http://localhost:3000"+path, bytes.NewReader(body))
		if err != nil {
			return 0, nil, nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:3000")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		response, err := client.Do(req)
		if err != nil {
			return 0, nil, nil, err
		}
		defer response.Body.Close()
		var result map[string]any
		err = json.NewDecoder(response.Body).Decode(&result)
		return response.StatusCode, result, response.Cookies(), err
	}
	for _, a := range m.Accounts {
		for _, identifier := range []string{a.Username, a.Email} {
			status, _, cookies, err := call("POST", "/api/login", map[string]string{"login": identifier, "password": a.Password}, nil)
			if err != nil || status != 200 || len(cookies) == 0 {
				return fmt.Errorf("frontend credential check failed for %s: HTTP %d", identifier, status)
			}
			status, body, _, err := call("GET", "/api/account", nil, cookies[0])
			if err != nil || status != 200 {
				return fmt.Errorf("account check failed for %s", identifier)
			}
			user, ok := body["data"].(map[string]any)
			if !ok || user["email"] != a.Email {
				return fmt.Errorf("wrong frontend identity for %s", identifier)
			}
		}
		fmt.Println("Username/email login OK:", a.Username)
	}
	for i, expected := range []string{"/home", "/email/verify"} {
		w := m.Wallets[i]
		status, body, _, err := call("GET", "/api/web3/nonce?address="+w.Address, nil, nil)
		if err != nil || status != 200 {
			return fmt.Errorf("wallet nonce failed: HTTP %d", status)
		}
		message, ok := body["nonce"].(string)
		if !ok {
			return errors.New("wallet nonce missing")
		}
		raw, err := hex.DecodeString(w.PrivateKey)
		if err != nil {
			return err
		}
		key, _ := btcec.PrivKeyFromBytes(raw)
		h := sha3.NewLegacyKeccak256()
		h.Write([]byte("\x19Ethereum Signed Message:\n" + strconv.Itoa(len(message)) + message))
		compact := ecdsa.SignCompact(key, h.Sum(nil), false)
		signature := "0x" + hex.EncodeToString(append(append([]byte{}, compact[1:]...), compact[0]))
		status, body, _, err = call("POST", "/api/web3/verify", map[string]string{"address": w.Address, "signature": signature}, nil)
		redirect, _ := body["redirect"].(string)
		if err != nil || status != 200 || !strings.HasSuffix(redirect, expected) {
			return fmt.Errorf("real wallet signature check failed for %s: HTTP %d", w.Scenario, status)
		}
		fmt.Println("Real wallet signature/login OK:", w.Scenario)
	}
	fmt.Println("Unused registration/link/member wallets preserved for manual frontend testing.")
	return nil
}
