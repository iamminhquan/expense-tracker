package api

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipLimiter is a token bucket per client IP, kept in memory. That is right for
// the single instance this runs as; a second instance would double every
// budget, and the fix then is a shared store, not a bigger number here.
//
// It exists because the account lockout (internal/auth/lockout.go) is per
// account and cuts two ways: it does nothing against one client trying a
// password on many accounts, and it lets anyone lock a victim out by getting
// their email wrong five times. Limiting the address closes both.
type ipLimiter struct {
	limit rate.Limit
	burst int
	now   func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPLimiter(limit rate.Limit, burst int) *ipLimiter {
	return &ipLimiter{limit: limit, burst: burst, now: time.Now, buckets: map[string]*bucket{}}
}

// idleAfter is how long a bucket sits unused before it is dropped: by then it
// has refilled completely, so a fresh one behaves identically.
func (l *ipLimiter) idleAfter() time.Duration {
	return time.Duration(float64(l.burst)/float64(l.limit)*float64(time.Second)) + time.Minute
}

// take spends one token for ip. When there is none it returns false and how
// long until there is.
func (l *ipLimiter) take(ip string) (ok bool, retryAfter time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastSweep) > time.Minute {
		for k, b := range l.buckets {
			if now.Sub(b.lastSeen) > l.idleAfter() {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}

	b, found := l.buckets[ip]
	if !found {
		b = &bucket{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.buckets[ip] = b
	}
	b.lastSeen = now

	res := b.limiter.ReserveN(now, 1)
	if delay := res.DelayFrom(now); delay > 0 {
		res.CancelAt(now)
		return false, delay
	}
	return true, 0
}

// rateLimit rejects a request over l's budget for its client IP with a 429
// and a Retry-After. The IP is c.ClientIP(), which is only as honest as the
// router's trusted-proxy list (Deps.TrustedProxies).
func rateLimit(l *ipLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, retryAfter := l.take(c.ClientIP())
		if ok {
			c.Next()
			return
		}
		c.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
		respondError(c, http.StatusTooManyRequests, "too many requests, try again later")
	}
}
