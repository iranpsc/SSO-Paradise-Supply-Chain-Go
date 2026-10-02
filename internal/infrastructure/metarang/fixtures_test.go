package metarang

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalMembershipFixturesAndProductionIsolation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "fixtures.json")
	address := "0x" + strings.Repeat("a", 40)
	if err := os.WriteFile(file, []byte(`{"`+address+`":{"already_registered":true,"user_code":"hm-2000001"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, env := range []string{"production", "staging"} {
		if _, err := LoadRegistry("https://example.invalid", file, env); err == nil {
			t.Fatal("test membership could be enabled outside development")
		}
	}
	registry, err := LoadRegistry("https://example.invalid", file, "development")
	if err != nil {
		t.Fatal(err)
	}
	registered, code, err := registry.LookupRegistration(context.Background(), strings.ToUpper(address))
	if err != nil || !registered || code != "hm-2000001" {
		t.Fatal("member lookup did not use fixture", registered, code, err)
	}
	registered, code, err = registry.LookupRegistration(context.Background(), "0x"+strings.Repeat("b", 40))
	if err != nil || registered || code != "" {
		t.Fatal("fresh wallet was not unregistered", registered, code, err)
	}
	registry, err = LoadRegistry("https://example.invalid", "", "production")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.(Client); !ok {
		t.Fatal("normal production registry was replaced")
	}
}
