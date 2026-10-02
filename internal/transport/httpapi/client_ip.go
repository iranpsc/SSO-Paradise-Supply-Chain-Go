package httpapi

import (
	"net/netip"
	"strings"
)

func (s *Server) clientIP(peer, forwarded string) string {
	ip, err := netip.ParseAddr(peer)
	if err != nil {
		return peer
	}
	ip = ip.Unmap()
	chain := strings.Split(forwarded, ",")
	for i := len(chain) - 1; i >= 0; i-- {
		trusted := false
		for _, prefix := range s.trustedProxies {
			if prefix.Contains(ip) {
				trusted = true
				break
			}
		}
		if !trusted {
			break
		}
		next, err := netip.ParseAddr(strings.TrimSpace(chain[i]))
		if err != nil {
			return peer
		}
		ip = next.Unmap()
	}
	return ip.String()
}
