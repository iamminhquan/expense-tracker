package api

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestIPLimiterRefillsAndForgetsIdleClients(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	l := newIPLimiter(rate.Every(time.Minute), 2)
	l.now = func() time.Time { return now }

	for i := range 2 {
		if ok, _ := l.take("a"); !ok {
			t.Fatalf("take %d inside the burst was refused", i+1)
		}
	}
	ok, retry := l.take("a")
	if ok || retry != time.Minute {
		t.Fatalf("third take = (%v, %v), want refused with a minute to wait", ok, retry)
	}
	// A refusal must not spend a token, or hammering would extend the wait.
	if _, retry := l.take("a"); retry != time.Minute {
		t.Fatalf("a second refusal reports %v, want still a minute", retry)
	}

	now = now.Add(time.Minute)
	if ok, _ := l.take("a"); !ok {
		t.Fatal("a token should have refilled after a minute")
	}

	now = now.Add(l.idleAfter() + time.Minute)
	l.take("b")
	if _, kept := l.buckets["a"]; kept {
		t.Error("an idle client's bucket was never swept")
	}
}
