package security

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBreachedLookupSendsOnlyPrefix(t *testing.T) {
	const password = "Password1!"
	hash := fmt.Sprintf("%X", sha1.Sum([]byte(password)))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/range/"+hash[:5] || r.Header.Get("Add-Padding") != "true" {
			t.Error("incorrect range request")
		}
		fmt.Fprintf(w, "%s:8\r\n", hash[5:])
	}))
	defer srv.Close()
	b := BreachedPasswords{Client: srv.Client(), URL: srv.URL}
	found, err := b.Compromised(context.Background(), password)
	if err != nil || !found {
		t.Fatal(found, err)
	}
}
func TestBreachedLookupRejectsServiceErrorsAndIgnoresPadding(t *testing.T) {
	for _, status := range []int{200, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprintf(w, "%s:0\r\n", strings.Repeat("0", 35))
			}))
			defer srv.Close()
			found, err := (BreachedPasswords{Client: srv.Client(), URL: srv.URL}).Compromised(context.Background(), "Secure!2026")
			if status == 200 && (found || err != nil) {
				t.Fatal(found, err)
			}
			if status != 200 && err == nil {
				t.Fatal("service error ignored")
			}
		})
	}
}
