package auth

import (
	"testing"
)

func TestNewTokenShape(t *testing.T) {
	raw, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 40 {
		t.Fatalf("raw too short: %q", raw)
	} // 32 bytes -> 43 base64url chars
	if len(hash) != 64 {
		t.Fatalf("hash not hex-sha256: %q", hash)
	}
	if HashToken(raw) != hash {
		t.Fatal("HashToken(raw) != returned hash")
	}
	raw2, _, _ := NewToken()
	if raw2 == raw {
		t.Fatal("collision")
	}
}

func TestHashTokenStable(t *testing.T) {
	if HashToken("abc") != HashToken("abc") {
		t.Fatal("not deterministic")
	}
	if HashToken("abc") == HashToken("abd") {
		t.Fatal("no diffusion")
	}
}
