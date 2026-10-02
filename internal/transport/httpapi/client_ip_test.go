package httpapi

import (
	"net/netip"
	"testing"
)

func TestTrustedProxyCannotSpoofClientIP(t *testing.T) {
	s := &Server{trustedProxies: []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}}
	for _, c := range []struct{ peer, header, want string }{
		{"192.0.2.1", "198.51.100.2", "192.0.2.1"},
		{"127.0.0.1", "198.51.100.2, 192.0.2.1", "192.0.2.1"},
		{"127.0.0.1", "192.0.2.1", "192.0.2.1"},
		{"127.0.0.1", "garbage", "127.0.0.1"},
	} {
		if got := s.clientIP(c.peer, c.header); got != c.want {
			t.Errorf("got %s want %s", got, c.want)
		}
	}
}
