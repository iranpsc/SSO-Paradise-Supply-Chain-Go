package httpapi

import (
	"sync"
	"time"
)

var penalties = [...]time.Duration{
	time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute,
	time.Hour, 3 * time.Hour, 24 * time.Hour,
}

type rateState struct {
	start        time.Time
	blockedUntil time.Time
	count        int
	strikes      int
}

type rateDecision struct {
	remaining int
	retry     time.Duration
}

// One shared budget per peer IP, independent of route or HTTP method.
// History lives for the server process lifetime; active bans are never evicted.
type limiter struct {
	mu      sync.Mutex
	entries map[string]rateState
}

func (l *limiter) check(ip string, now time.Time) rateDecision {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Bound memory without discarding penalty history or active bans.
	if _, exists := l.entries[ip]; !exists && len(l.entries) >= 65536 {
		for key, state := range l.entries {
			if state.strikes == 0 && now.Sub(state.start) >= time.Minute {
				delete(l.entries, key)
			}
		}
		if len(l.entries) >= 65536 {
			return rateDecision{retry: time.Minute}
		}
	}
	s := l.entries[ip]
	if now.Before(s.blockedUntil) {
		return rateDecision{retry: s.blockedUntil.Sub(now)}
	}
	if s.start.IsZero() || now.Sub(s.start) >= time.Minute {
		s.start, s.count = now, 0
	}
	if s.count >= 10 {
		s.blockedUntil = now.Add(penalties[s.strikes])
		if s.strikes < len(penalties)-1 {
			s.strikes++
		}
		l.entries[ip] = s
		return rateDecision{retry: s.blockedUntil.Sub(now)}
	}
	s.count++
	l.entries[ip] = s
	return rateDecision{remaining: 10 - s.count}
}
