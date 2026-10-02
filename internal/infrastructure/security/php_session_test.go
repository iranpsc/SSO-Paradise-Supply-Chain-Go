package security

import (
	"testing"
)

func TestPrimitivePHPSessionParserRejectsObjectsAndMalformedData(t *testing.T) {
	if SessionGuardHash() != "59ba36addc2b2f9401580f014c7f58ea4e30989d" {
		t.Fatal("session guard cookie name differs from Laravel")
	}
	data, err := ParsePHPSession([]byte(`a:3:{s:4:"user";i:7;s:4:"name";s:4:"test";s:5:"flags";a:1:{s:6:"wallet";b:1;}}`))
	if err != nil || data["user"] != int64(7) {
		t.Fatal(err)
	}
	for _, raw := range []string{`O:8:"stdClass":0:{}`, `a:1:{s:1:"x";R:1;}`, `a:100000000:{}`, `a:1:{s:100:"x";i:1;}`, `a:1:{s:1:"x";i:1;}` + "trailing"} {
		if _, err := ParsePHPSession([]byte(raw)); err == nil {
			t.Fatal("unsafe/malformed session accepted")
		}
	}
}
