package metarang_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/metarang"
)

// stub records the incoming lookup request and replays a canned response,
// asserting the Laravel client's contract: POST /api/wallets/registered
// with a JSON wallet_address body.
type stub struct {
	t      *testing.T
	status int
	body   string
}

func (s stub) handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/api/wallets/registered" {
		s.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		s.t.Errorf("unexpected content type %q", ct)
	}
	raw, _ := io.ReadAll(r.Body)
	var payload map[string]string
	if err := json.Unmarshal(raw, &payload); err != nil || payload["wallet_address"] != "0xabc" {
		s.t.Errorf("unexpected body %s", raw)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s.status)
	_, _ = w.Write([]byte(s.body))
}

func client(t *testing.T, status int, body string) metarang.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(stub{t, status, body}.handler))
	t.Cleanup(srv.Close)
	return metarang.Client{BaseURL: srv.URL}
}

func TestLookupRegistration(t *testing.T) {
	c := client(t, 200, `{"already_registered":true,"user_code":"hm-2000001"}`)
	registered, code, err := c.LookupRegistration(context.Background(), "0xabc")
	if err != nil || !registered || code != "hm-2000001" {
		t.Fatalf("registered=%v code=%q err=%v", registered, code, err)
	}
}

func TestLookupUnregistered(t *testing.T) {
	c := client(t, 200, `{"already_registered":false,"user_code":null}`)
	registered, code, err := c.LookupRegistration(context.Background(), "0xabc")
	if err != nil || registered || code != "" {
		t.Fatalf("registered=%v code=%q err=%v", registered, code, err)
	}
}

func TestLookupFailures(t *testing.T) {
	cases := map[string]metarang.Client{
		"unconfigured":        {},
		"blank base":          {BaseURL: "  "},
		"http 500":            client(t, 500, `{"message":"upstream error"}`),
		"connection refused":  {BaseURL: "http://127.0.0.1:1"},
		"invalid json":        client(t, 200, `not json`),
		"missing flag":        client(t, 200, `{"unexpected":true}`),
		"flag not boolean":    client(t, 200, `{"already_registered":"yes"}`),
		"registered w/o code": client(t, 200, `{"already_registered":true,"user_code":""}`),
		"registered no code":  client(t, 200, `{"already_registered":true}`),
	}
	for name, c := range cases {
		if _, _, err := c.LookupRegistration(context.Background(), "0xabc"); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestLookupTrimsTrailingSlash(t *testing.T) {
	c := client(t, 200, `{"already_registered":false}`)
	c.BaseURL += "/"
	if _, _, err := c.LookupRegistration(context.Background(), "0xabc"); err != nil {
		t.Fatal(err)
	}
}
