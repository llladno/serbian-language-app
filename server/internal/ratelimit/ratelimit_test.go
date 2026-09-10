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
