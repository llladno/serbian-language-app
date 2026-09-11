package ratelimit

import (
	"sync"
	"testing"
	"time"
)

// clock is a deterministic time source for the SetNow test seam.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func newClock() *clock { return &clock{t: time.Unix(1_700_000_000, 0)} }

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestLimiterAllowsBurstThenBlocks(t *testing.T) {
	clk := newClock()
	l := NewLimiter(1.0/60, 5) // 5 per minute
	l.SetNow(clk.now)

	for i := 0; i < 5; i++ {
		if !l.Allow("ip") {
			t.Fatalf("Allow #%d for %q: got false, want true (burst)", i+1, "ip")
		}
	}
	if l.Allow("ip") {
		t.Fatalf("Allow #6 for %q: got true, want false (bucket empty)", "ip")
	}

	// A different key has its own bucket.
	if !l.Allow("ip2") {
		t.Fatalf("Allow for %q: got false, want true (per-key isolation)", "ip2")
	}
}

func TestLimiterRefills(t *testing.T) {
	clk := newClock()
	l := NewLimiter(1.0/60, 5)
	l.SetNow(clk.now)

	for i := 0; i < 5; i++ {
		l.Allow("ip")
	}
	if l.Allow("ip") {
		t.Fatal("bucket should be empty before refill")
	}

	clk.advance(60 * time.Second) // one token per minute
	if !l.Allow("ip") {
		t.Fatal("Allow after 60s: got false, want true (refilled)")
	}
	if l.Allow("ip") {
		t.Fatal("only one token should have refilled after 60s")
	}
}

func TestLimiterEvictsIdle(t *testing.T) {
	clk := newClock()
	l := NewLimiter(1.0/60, 5)
	l.SetNow(clk.now)

	l.Allow("ip")
	l.Allow("ip")
	if got := l.size(); got != 1 {
		t.Fatalf("size after first use: got %d, want 1", got)
	}

	// Idle past the TTL, then touch another key to trigger the sweep.
	clk.advance(11 * time.Minute)
	l.Allow("other")

	if got := l.size(); got != 1 {
		t.Fatalf("size after eviction: got %d, want 1 (only %q left)", got, "other")
	}

	// The evicted key comes back with a full bucket.
	for i := 0; i < 5; i++ {
		if !l.Allow("ip") {
			t.Fatalf("Allow #%d after eviction: got false, want true (fresh bucket)", i+1)
		}
	}
}

// TestLimiterTTLCoversRefillTime is the regression for the eviction hole: when
// the TTL is shorter than the time a bucket needs to refill from empty, an
// attacker pacing requests just over the TTL apart collects a whole fresh burst
// each cycle. The "3 per hour" register/resend/forgot limiter used to admit
// ~18/hour that way (evicted at 10 minutes, refilled at 60).
func TestLimiterTTLCoversRefillTime(t *testing.T) {
	clk := newClock()
	l := NewLimiter(3.0/3600, 3) // matches main.go's `slow` limiter
	l.SetNow(clk.now)

	if want := time.Hour; l.idleTTL != want {
		t.Fatalf("idleTTL = %v, want %v (burst/rate)", l.idleTTL, want)
	}

	for i := 0; i < 3; i++ {
		if !l.Allow("email:victim") {
			t.Fatalf("Allow #%d: got false, want true (burst)", i+1)
		}
	}
	if l.Allow("email:victim") {
		t.Fatal("Allow #4: got true, want false (burst spent)")
	}

	// Idle past the OLD 10-minute constant, but short of the 20 minutes one
	// token legitimately takes to trickle back. The bucket must survive the
	// sweep: under the old fixed TTL it was evicted here and the key came back
	// with a full burst of 3 — the whole bug.
	clk.advance(11 * time.Minute)
	l.Allow("other") // another key, to trigger the lazy sweep
	if l.Allow("email:victim") {
		t.Fatal("Allow after 11m idle: got true, want false (bucket was evicted -> fresh burst)")
	}
	if l.size() != 2 {
		t.Fatalf("tracked keys = %d, want 2 (neither bucket evicted yet)", l.size())
	}

	// 22 minutes in, exactly one token has trickled back — one, not a burst.
	clk.advance(11 * time.Minute)
	if !l.Allow("email:victim") {
		t.Fatal("Allow at 22m: got false, want true (one token refills every 20m)")
	}
	if l.Allow("email:victim") {
		t.Fatal("second Allow at 22m: got true, want false (only one token had refilled)")
	}

	// Past the full refill time the bucket is legitimately back to its burst of
	// 3 — and no more.
	clk.advance(time.Hour + time.Minute)
	for i := 0; i < 3; i++ {
		if !l.Allow("email:victim") {
			t.Fatalf("Allow #%d after a full refill: got false, want true", i+1)
		}
	}
	if l.Allow("email:victim") {
		t.Fatal("Allow #4 after a full refill: got true, want false (burst is 3)")
	}
}

// TestLimiterTTLFloor keeps the fast per-IP login limiter on the 10-minute
// floor: its natural refill time is only a minute, and cold keys should still
// be dropped promptly.
func TestLimiterTTLFloor(t *testing.T) {
	l := NewLimiter(5.0/60, 5) // matches main.go's per-IP `login` limiter
	if l.idleTTL != minIdleTTL {
		t.Fatalf("idleTTL = %v, want the %v floor", l.idleTTL, minIdleTTL)
	}
	// A degenerate zero rate must not overflow the duration conversion.
	if got := NewLimiter(0, 3).idleTTL; got != maxIdleTTL {
		t.Fatalf("idleTTL for a zero-rate limiter = %v, want %v", got, maxIdleTTL)
	}
}

func TestFailCounterLocksAfterThreshold(t *testing.T) {
	clk := newClock()
	f := NewFailCounter(3, 15*time.Minute)
	f.SetNow(clk.now)

	if f.Locked("acct") {
		t.Fatal("Locked before any failure: got true, want false")
	}
	f.Fail("acct")
	f.Fail("acct")
	if f.Locked("acct") {
		t.Fatal("Locked after 2 failures (threshold 3): got true, want false")
	}
	f.Fail("acct")
	if !f.Locked("acct") {
		t.Fatal("Locked after 3 failures: got false, want true")
	}

	// Another key is unaffected.
	if f.Locked("other") {
		t.Fatal("Locked for untouched key: got true, want false")
	}

	// Window passes -> unlocked.
	clk.advance(16 * time.Minute)
	if f.Locked("acct") {
		t.Fatal("Locked after window elapsed: got true, want false")
	}
}

func TestFailCounterResetClears(t *testing.T) {
	clk := newClock()
	f := NewFailCounter(3, 15*time.Minute)
	f.SetNow(clk.now)

	f.Fail("acct")
	f.Fail("acct")
	f.Fail("acct")
	if !f.Locked("acct") {
		t.Fatal("Locked after 3 failures: got false, want true")
	}

	f.Reset("acct")
	if f.Locked("acct") {
		t.Fatal("Locked after Reset: got true, want false")
	}
}

// TestConcurrentAccess exercises the mutexes under -race.
func TestConcurrentAccess(t *testing.T) {
	l := NewLimiter(1000, 100)
	f := NewFailCounter(5, time.Minute)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%4))
			for j := 0; j < 200; j++ {
				l.Allow(key)
				f.Fail(key)
				f.Locked(key)
				if j%10 == 0 {
					f.Reset(key)
				}
			}
		}(i)
	}
	wg.Wait()
}
