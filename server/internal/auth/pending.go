package auth

import (
	"sync"
	"time"
)

// pendingTTL is how long a /start login/link token stays valid. Short on
// purpose: the whole round trip (open Telegram, tap Start, come back) takes
// seconds in practice, and a stale token left open longer is unnecessary
// attack surface for no real benefit to a legitimate caller.
const pendingTTL = 5 * time.Minute

// PendingStatus is where a /start token is in its lifecycle.
type PendingStatus string

const (
	PendingWaiting PendingStatus = "pending"
	PendingDone    PendingStatus = "done"
	PendingError   PendingStatus = "error"
)

// pendingEntry is one outstanding /start token.
type pendingEntry struct {
	// userID is "" for a login token (no account yet) or the caller's own id
	// for a link token (attach Telegram to an already-authenticated account).
	userID string
	status PendingStatus
	// resolvedUserID is set once status is PendingDone: the account the
	// token resolved to (a fresh login, an existing identity match, or the
	// linking caller's own userID echoed back).
	resolvedUserID string
	// errMsg is set only when status is PendingError (e.g. "telegram_taken").
	errMsg  string
	expires time.Time
}

// PendingStore tracks /start tokens between "generate a link" and "the
// Telegram webhook resolved it" — in-memory only. The app runs as a single
// instance and a token's whole life is minutes, so losing pending state on a
// restart is a non-event (the caller just presses the button again); this
// avoids a database table and a migration for state nobody needs to survive
// a restart. Safe for concurrent use.
type PendingStore struct {
	mu   sync.Mutex
	now  func() time.Time
	seen map[string]*pendingEntry // key: sha256 hash of the raw token
}

// NewPendingStore returns an empty PendingStore.
func NewPendingStore() *PendingStore {
	return &PendingStore{
		now:  time.Now,
		seen: make(map[string]*pendingEntry),
	}
}

// Create mints a fresh token for a login (userID == "") or a link (userID ==
// the caller's own id) and returns the raw token to embed in the t.me
// deep link. Only the hash is retained.
func (s *PendingStore) Create(userID string) (raw string, err error) {
	raw, hash, err := NewToken()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictLocked()
	s.seen[hash] = &pendingEntry{
		userID:  userID,
		status:  PendingWaiting,
		expires: s.now().Add(pendingTTL),
	}
	return raw, nil
}

// Lookup returns the entry for a raw token's hash, and whether it exists and
// has not expired. Used by the webhook handler to find the link/login target
// before resolving it.
func (s *PendingStore) Lookup(raw string) (userID string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictLocked()
	e, ok := s.seen[HashToken(raw)]
	if !ok {
		return "", false
	}
	return e.userID, true
}

// Resolve marks a pending token done with the account it resolved to. A
// token that does not exist (expired or unknown) is silently ignored — the
// webhook has no user-facing response to give beyond its own chat message.
func (s *PendingStore) Resolve(raw, resolvedUserID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.seen[HashToken(raw)]
	if !ok {
		return
	}
	e.status = PendingDone
	e.resolvedUserID = resolvedUserID
}

// Fail marks a pending token as failed with a caller-facing error code (e.g.
// "telegram_taken"), so poll can surface why instead of hanging until expiry.
func (s *PendingStore) Fail(raw, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.seen[HashToken(raw)]
	if !ok {
		return
	}
	e.status = PendingError
	e.errMsg = errMsg
}

// PollResult is what Poll reports back to the browser tab that's waiting on
// a token.
type PollResult struct {
	Status PendingStatus
	// CallerUserID is the token's original purpose: "" for a login token, the
	// caller's own id for a link token. The handler uses this to decide
	// whether a PendingDone result should mint a fresh session (login) or
	// not (link — the caller already has one).
	CallerUserID   string
	ResolvedUserID string // set when Status == PendingDone
	Error          string // set when Status == PendingError
}

// Poll reports a token's current state. A PendingDone or PendingError result
// is one-shot: the entry is deleted immediately after being read, so the
// browser cannot accidentally re-process the same completion twice (e.g. a
// retried request racing a slow network) and a completed token cannot be
// replayed. An unknown or expired token reports PendingError with no message
// distinguishable from "never existed" — nothing sensitive to leak either way.
func (s *PendingStore) Poll(raw string) PollResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictLocked()
	e, ok := s.seen[HashToken(raw)]
	if !ok {
		return PollResult{Status: PendingError, Error: "expired"}
	}
	if e.status == PendingWaiting {
		return PollResult{Status: PendingWaiting, CallerUserID: e.userID}
	}
	delete(s.seen, HashToken(raw))
	if e.status == PendingError {
		return PollResult{Status: PendingError, Error: e.errMsg, CallerUserID: e.userID}
	}
	return PollResult{Status: PendingDone, CallerUserID: e.userID, ResolvedUserID: e.resolvedUserID}
}

// SetNow overrides the clock used for expiry. Test seam; production leaves
// the default of time.Now.
func (s *PendingStore) SetNow(fn func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = fn
}

// evictLocked drops every entry past its expiry. Caller must hold s.mu.
func (s *PendingStore) evictLocked() {
	now := s.now()
	for k, e := range s.seen {
		if now.After(e.expires) {
			delete(s.seen, k)
		}
	}
}
