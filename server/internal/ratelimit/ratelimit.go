// Package ratelimit provides a keyed token-bucket limiter and a soft account
// lock (fail counter) used by the auth HTTP handlers to throttle abusive
// callers without a shared datastore. Both types are safe for concurrent use.
package ratelimit

import (
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// minIdleTTL is the floor on how long a key may go untouched before Limiter
// drops its bucket. Eviction is lazy: a full sweep runs on each Allow call.
// The effective TTL is per-Limiter — see NewLimiter.
const minIdleTTL = 10 * time.Minute

// maxIdleTTL caps the derived TTL so a very slow (or misconfigured, zero-rate)
// limiter cannot pin buckets in memory forever.
const maxIdleTTL = 24 * time.Hour

// Limiter is a keyed set of token buckets: each key gets an independent
// golang.org/x/time/rate.Limiter with the same rate and burst. Buckets for
// keys untouched for longer than idleTTL are evicted lazily so the map cannot
// grow without bound.
type Limiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	idleTTL time.Duration
	now     func() time.Time
	seen    map[string]*bucket
}

// bucket pairs a per-key rate limiter with the last time it was used.
type bucket struct {
	lim  *rate.Limiter
	last time.Time
}

// NewLimiter returns a Limiter that admits perSecond events per key on average
// with room for short bursts of burst events.
//
// The idle-eviction TTL is derived from the rate, never fixed. Eviction hands a
// returning key a FRESH, FULL bucket, so an eviction TTL shorter than the time
// a legitimate bucket needs to refill from empty to burst silently multiplies
// the limit: an attacker paces requests just over the TTL apart and collects a
// whole new burst each time. A "3 per hour" limiter evicted after 10 minutes
// admits ~18 per hour. So the TTL is at least the natural refill time
// (burst / perSecond), floored at minIdleTTL so fast limiters still drop cold
// keys promptly.
func NewLimiter(perSecond float64, burst int) *Limiter {
	ttl := refillTime(perSecond, burst)
	if ttl < minIdleTTL {
		ttl = minIdleTTL
	}
	if ttl > maxIdleTTL {
		ttl = maxIdleTTL
	}
	return &Limiter{
		limit:   rate.Limit(perSecond),
		burst:   burst,
		idleTTL: ttl,
		now:     time.Now,
		seen:    make(map[string]*bucket),
	}
}

// refillTime is how long a bucket needs to go from empty back to a full burst
// at perSecond. It rounds UP to a whole second so binary rounding in a rate
// written as a fraction (3.0/3600) cannot land a nanosecond short of the real
// hour, and saturates at maxIdleTTL for a non-positive rate — a bucket that
// never refills, which would otherwise divide to +Inf and overflow the
// conversion to time.Duration.
func refillTime(perSecond float64, burst int) time.Duration {
	if perSecond <= 0 || burst <= 0 {
		return maxIdleTTL
	}
	secs := math.Ceil(float64(burst) / perSecond)
	if secs >= maxIdleTTL.Seconds() {
		return maxIdleTTL
	}
	return time.Duration(secs) * time.Second
}

// Allow reports whether an event for key may proceed now, consuming one token
// from key's bucket. It returns false when the bucket is empty.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.evictLocked(now)

	b := l.seen[key]
	if b == nil {
		b = &bucket{lim: rate.NewLimiter(l.limit, l.burst)}
		l.seen[key] = b
	}
	b.last = now
	return b.lim.AllowN(now, 1)
}

// SetNow overrides the clock used for token refill and eviction. It is a test
// seam; production code leaves the default of time.Now.
func (l *Limiter) SetNow(fn func() time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.now = fn
}

// evictLocked drops every bucket untouched for longer than l.idleTTL. The
// caller must hold l.mu.
func (l *Limiter) evictLocked(now time.Time) {
	for k, b := range l.seen {
		if now.Sub(b.last) > l.idleTTL {
			delete(l.seen, k)
		}
	}
}

// size reports the number of tracked keys. Test seam.
func (l *Limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.seen)
}

// FailCounter is the soft account lock: it records recent failed attempts per
// key and reports a key as locked once at least threshold failures fall within
// a trailing window. Old failures age out, so a key unlocks on its own once
// the window passes; a success clears the count via Reset.
type FailCounter struct {
	mu        sync.Mutex
	threshold int
	window    time.Duration
	now       func() time.Time
	recs      map[string][]time.Time
}

// NewFailCounter returns a FailCounter that locks a key after threshold
// failures within window.
func NewFailCounter(threshold int, window time.Duration) *FailCounter {
	return &FailCounter{
		threshold: threshold,
		window:    window,
		now:       time.Now,
		recs:      make(map[string][]time.Time),
	}
}

// Locked reports whether key currently has at least threshold failures within
// the trailing window. Failures older than the window are pruned as a side
// effect, and a key with none left is forgotten.
func (f *FailCounter) Locked(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	ts, ok := f.recs[key]
	if !ok {
		return false
	}
	ts = pruneBefore(ts, f.now().Add(-f.window))
	if len(ts) == 0 {
		delete(f.recs, key)
		return false
	}
	f.recs[key] = ts
	return len(ts) >= f.threshold
}

// Fail records one failed attempt for key at the current time.
func (f *FailCounter) Fail(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := f.now()
	ts := pruneBefore(f.recs[key], now.Add(-f.window))
	f.recs[key] = append(ts, now)
}

// Reset clears every recorded failure for key. Call it after a successful
// attempt.
func (f *FailCounter) Reset(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.recs, key)
}

// SetNow overrides the clock used for windowing. It is a test seam; production
// code leaves the default of time.Now.
func (f *FailCounter) SetNow(fn func() time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = fn
}

// pruneBefore returns the suffix of ts (assumed ascending, as appended) whose
// entries are strictly after cutoff.
func pruneBefore(ts []time.Time, cutoff time.Time) []time.Time {
	i := 0
	for i < len(ts) && !ts[i].After(cutoff) {
		i++
	}
	return ts[i:]
}
