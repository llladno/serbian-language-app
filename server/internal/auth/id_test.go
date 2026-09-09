package auth

import (
	"regexp"
	"testing"
)

func TestNewID(t *testing.T) {
	re := regexp.MustCompile(`^usr_[0-9A-Za-z]{22}$`)
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewUserID()
		if !re.MatchString(id) {
			t.Fatalf("bad id: %q", id)
		}
		if seen[id] {
			t.Fatalf("collision: %q", id)
		}
		seen[id] = true
	}
	if got := NewIdentityID()[:4]; got != "idn_" {
		t.Fatalf("prefix = %q", got)
	}
}
