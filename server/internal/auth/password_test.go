package auth

import "testing"

func TestHashVerifyRoundTrip(t *testing.T) {
	h, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(h, "correct horse battery staple") {
		t.Fatal("valid password rejected")
	}
	if VerifyPassword(h, "wrong") {
		t.Fatal("wrong password accepted")
	}
}

func TestHashLongPasswordNotTruncated(t *testing.T) {
	// >72 bytes: bcrypt would truncate without the sha256 pre-hash.
	a := "A_" + string(make([]byte, 100)) + "_tail_1"
	b := "A_" + string(make([]byte, 100)) + "_tail_2"
	h, _ := HashPassword(a)
	if VerifyPassword(h, b) {
		t.Fatal("distinct >72-byte passwords collide")
	}
	if !VerifyPassword(h, a) {
		t.Fatal("exact >72-byte password rejected")
	}
}

func TestHashIsSalted(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Fatal("hash not salted")
	}
}
