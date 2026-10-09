package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

func sourceContracts(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile("testdata/laravel-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]json.RawMessage
	if err = json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func assertJSON(t *testing.T, expected json.RawMessage, value any) {
	t.Helper()
	var want, got any
	if err := json.Unmarshal(expected, &want); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("Laravel contract differs\nwant: %s\n got: %s", expected, data)
	}
}

func TestLaravelModelSerializationFromSourceFixture(t *testing.T) {
	source := sourceContracts(t)
	stamp, _ := time.Parse(time.RFC3339, "2026-01-01T23:34:05Z")
	code := "hm-2000042"
	u := domain.User{ID: 42, Name: "Test", Email: "test@example.com", Username: "extension_only", PasswordHash: "secret", Code: &code, CreatedAt: stamp, UpdatedAt: stamp, EmailVerifiedAt: &stamp}
	assertJSON(t, source["user"], laravelUser(u))
	nullable := laravelUser(domain.User{ID: 43})
	for _, key := range []string{"name", "email", "referral", "mobile", "wallet_address", "code", "email_verified_at", "created_at", "updated_at"} {
		if nullable[key] != nil && key != "mobile" && key != "wallet_address" && key != "code" {
			t.Fatalf("%s did not retain null", key)
		}
	}
}

func TestPassportErrorsFromLeagueSourceFixtures(t *testing.T) {
	var cases map[string]struct {
		Status int
		Body   json.RawMessage
	}
	if err := json.Unmarshal(sourceContracts(t)["oauth"], &cases); err != nil {
		t.Fatal(err)
	}
	for kind, fixture := range cases {
		t.Run(kind, func(t *testing.T) {
			parameter := ""
			if kind == "invalid_scope" {
				parameter = "bad"
			}
			if kind == "invalid_request" {
				parameter = "grant_type"
			}
			status, body := passportPayload(application.OAuthError(kind), parameter)
			if status != fixture.Status {
				t.Fatal(status, fixture.Status)
			}
			assertJSON(t, fixture.Body, body)
		})
	}
}

func TestLegacyPersonalRequiredMatchesLaravel(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/personal-info", nil)
	if legacyPersonalRequired(w, r, map[string]any{}) || w.Code != 422 {
		t.Fatal("missing fields accepted")
	}
	var actual any
	if err := json.Unmarshal(w.Body.Bytes(), &actual); err != nil {
		t.Fatal(err)
	}
	assertJSON(t, sourceContracts(t)["personal_empty"], actual)
}

func TestFixturesStillMatchInstalledLaravel(t *testing.T) {
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("PHP not installed; committed source fixtures still tested")
	}
	script, err := filepath.Abs("../../../scripts/laravel-contracts.php")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../../../SSO-Paradise-Supply-Chain")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "vendor", "autoload.php")); err != nil {
		t.Skip("Laravel vendor not installed; committed source fixtures still tested")
	}
	cmd := exec.Command(php, script, root)
	data, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var live map[string]json.RawMessage
	if err = json.Unmarshal(data, &live); err != nil {
		t.Fatalf("Laravel probe did not return JSON: %s", data)
	}
	for key, expected := range sourceContracts(t) {
		assertJSON(t, expected, live[key])
	}
}
