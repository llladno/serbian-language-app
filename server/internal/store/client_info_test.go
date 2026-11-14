package store

import "testing"

func TestNormalizeClientInfo(t *testing.T) {
	cases := []struct {
		version, platform string
		wantV, wantP      string
		ok                bool
	}{
		{"10.1", "ios", "10.1", "ios", true},
		{" 9.6 ", " Android ", "9.6", "android", true},
		{"10", "tdesktop", "10", "tdesktop", true},
		{"10.4.1", "", "10.4.1", "", true},
		{"", "ios", "", "", false},
		{"v10.1", "ios", "", "", false},
		{"10.1; DROP TABLE users", "ios", "", "", false},
		{"10.1", "iOS <b>", "10.1", "", true}, // a junk platform is dropped, the version is kept
	}
	for _, c := range cases {
		v, p, ok := NormalizeClientInfo(c.version, c.platform)
		if v != c.wantV || p != c.wantP || ok != c.ok {
			t.Errorf("NormalizeClientInfo(%q, %q) = %q, %q, %v; want %q, %q, %v", c.version, c.platform, v, p, ok, c.wantV, c.wantP, c.ok)
		}
	}
}

func TestNoteClientInfo(t *testing.T) {
	s := newStore(t)
	uid, err := s.CreateUser("gri_92")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.ClientInfoOf(uid); err != nil || got != (ClientInfo{}) {
		t.Fatalf("before any report: %+v, %v; want empty", got, err)
	}
	if err := s.NoteClientInfo(uid, "10.1", "ios"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.ClientInfoOf(uid)
	if got.Version != "10.1" || got.Platform != "ios" || got.SeenAt == "" {
		t.Fatalf("after first report: %+v", got)
	}
	// Junk is ignored and never erases what is there.
	if err := s.NoteClientInfo(uid, "evil", "x"); err != nil {
		t.Fatal(err)
	}
	if after, _ := s.ClientInfoOf(uid); after != got {
		t.Fatalf("junk changed the row: %+v -> %+v", got, after)
	}
	// The same report again within a day writes nothing (the stamp stays).
	if err := s.NoteClientInfo(uid, "10.1", "ios"); err != nil {
		t.Fatal(err)
	}
	if after, _ := s.ClientInfoOf(uid); after != got {
		t.Fatalf("a repeat report rewrote the row: %+v -> %+v", got, after)
	}
	// A newer app wins.
	if err := s.NoteClientInfo(uid, "10.4", "android"); err != nil {
		t.Fatal(err)
	}
	if after, _ := s.ClientInfoOf(uid); after.Version != "10.4" || after.Platform != "android" {
		t.Fatalf("after an upgrade: %+v", after)
	}
	// A stale stamp (older than a day) is refreshed even if nothing else changed.
	if _, err := s.db.Exec(`UPDATE users SET tg_client_seen_at = '2020-01-01T00:00:00Z' WHERE id = ?`, uid); err != nil {
		t.Fatal(err)
	}
	if err := s.NoteClientInfo(uid, "10.4", "android"); err != nil {
		t.Fatal(err)
	}
	if after, _ := s.ClientInfoOf(uid); after.SeenAt == "2020-01-01T00:00:00Z" {
		t.Fatalf("a day-old stamp was not refreshed: %+v", after)
	}
}
