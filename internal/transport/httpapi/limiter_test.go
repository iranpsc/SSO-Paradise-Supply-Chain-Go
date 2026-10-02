package httpapi

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEscalatingSharedLimit(t *testing.T) {
	l := &limiter{entries: map[string]rateState{}}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, penalty := range append(penalties[:], 24*time.Hour) {
		for i := 0; i < 10; i++ {
			if d := l.check("client", now); d.retry != 0 || d.remaining != 9-i {
				t.Fatalf("request %d: %+v", i, d)
			}
		}
		if d := l.check("client", now); d.retry != penalty {
			t.Fatalf("ban: got %v want %v", d.retry, penalty)
		}
		if d := l.check("other", now); d.retry != 0 {
			t.Fatal("another IP was blocked")
		}
		if d := l.check("client", now.Add(penalty-time.Second)); d.retry != time.Second {
			t.Fatal("blocked request extended or escalated ban", d)
		}
		now = now.Add(penalty)
	}
}

func TestWindowBoundaryAndConcurrentRequests(t *testing.T) {
	l := &limiter{entries: map[string]rateState{}}
	now := time.Now()
	for i := 0; i < 10; i++ {
		l.check("client", now)
	}
	if d := l.check("client", now.Add(time.Minute)); d.retry != 0 || d.remaining != 9 {
		t.Fatal("window did not reset", d)
	}
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.check("concurrent", now).retry == 0 {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 10 {
		t.Fatalf("accepted %d concurrent requests", accepted.Load())
	}
}
