package campaign

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func ids(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("usr_%06d", i)
	}
	return out
}

func TestHoldoutIsAboutFifteenPercentAndStable(t *testing.T) {
	held := 0
	for _, id := range ids(20000) {
		if Holdout(id) {
			held++
		}
		if Holdout(id) != Holdout(id) {
			t.Fatalf("Holdout(%s) is not stable", id)
		}
	}
	if share := float64(held) / 20000; share < 0.14 || share > 0.16 {
		t.Errorf("holdout share = %.3f, want about 0.15", share)
	}
}

func TestStyleForFollowsTheShares(t *testing.T) {
	for _, tc := range []struct {
		wave, leader int
		wantLeader   float64
	}{{1, StyleDirect, 0.5}, {2, StyleGlance, 0.55}, {3, StyleGlance, 0.6}, {3, StyleDirect, 0.6}} {
		shares := Shares(tc.wave, tc.leader)
		if shares[0]+shares[1] != 1000 {
			t.Fatalf("shares %v do not add up to 1000", shares)
		}
		count := [NumStyles]int{}
		for _, id := range ids(20000) {
			s := StyleFor(id, tc.wave, shares)
			if s != StyleFor(id, tc.wave, shares) {
				t.Fatalf("StyleFor(%s) is not stable", id)
			}
			count[s]++
		}
		got := float64(count[tc.leader]) / 20000
		if got < tc.wantLeader-0.015 || got > tc.wantLeader+0.015 {
			t.Errorf("wave %d leader %d: leader share = %.3f, want about %.2f", tc.wave, tc.leader, got, tc.wantLeader)
		}
	}
}

func TestStyleForDiffersBetweenWaves(t *testing.T) {
	shares := Shares(1, StyleDirect)
	same := 0
	for _, id := range ids(2000) {
		if StyleFor(id, 1, shares) == StyleFor(id, 2, shares) {
			same++
		}
	}
	if same > 1200 { // independent draws agree about half the time
		t.Errorf("styles agree for %d of 2000 learners between waves: they should be drawn independently", same)
	}
}

func TestLeader(t *testing.T) {
	tests := []struct {
		name  string
		stats [NumStyles]Stat
		want  int
	}{
		{"nothing yet: the plain style", [NumStyles]Stat{}, StyleDirect},
		{"tie: the plain style", [NumStyles]Stat{{50, 4}, {50, 4}}, StyleDirect},
		{"glance clicked more", [NumStyles]Stat{{50, 2}, {50, 5}}, StyleGlance},
		{"direct clicked more", [NumStyles]Stat{{50, 5}, {50, 2}}, StyleDirect},
		{"rates, not counts: fewer sends with a higher rate wins", [NumStyles]Stat{{100, 6}, {20, 3}}, StyleGlance},
		{"a style with no sends cannot lead on no evidence", [NumStyles]Stat{{100, 8}, {0, 0}}, StyleDirect},
	}
	for _, tc := range tests {
		if got := Leader(tc.stats); got != tc.want {
			t.Errorf("%s: Leader = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestWaveShapes(t *testing.T) {
	if WaveSize(1) != 50 || WaveSize(2) != 50 || WaveSize(3) != -1 {
		t.Errorf("wave sizes = %d %d %d, want 50 50 -1", WaveSize(1), WaveSize(2), WaveSize(3))
	}
	if s := Shares(1, StyleDirect); s != [NumStyles]int{500, 500} {
		t.Errorf("wave 1 shares = %v, want an even split", s)
	}
	if s := Shares(2, StyleGlance); s != [NumStyles]int{450, 550} {
		t.Errorf("wave 2 shares with glance leading = %v", s)
	}
	if s := Shares(3, StyleDirect); s != [NumStyles]int{600, 400} {
		t.Errorf("wave 3 shares with direct leading = %v", s)
	}
}

func TestMessagesFitTheirChannels(t *testing.T) {
	for seg := 1; seg <= Segments; seg++ {
		m := For(seg)
		if n := utf8.RuneCountInString(m.Telegram); n == 0 || n > 1024 {
			t.Errorf("segment %d: Telegram caption is %d characters, want 1..1024", seg, n)
		}
		for style, b := range m.Buttons {
			if b == "" || utf8.RuneCountInString(b) > 40 {
				t.Errorf("segment %d style %d: button %q is empty or too long", seg, style, b)
			}
		}
		for _, field := range []string{m.Subject, m.Preheader, m.Heading, m.Lead, m.Telegram} {
			if strings.Contains(field, "{{") {
				t.Errorf("segment %d: %q contains a template action", seg, field)
			}
		}
		if m.Lesson == "" || m.Image == "" || m.Subject == "" || m.Lead == "" {
			t.Errorf("segment %d is missing a field: %+v", seg, m)
		}
		// The swallow's tour is promised exactly where there is a finished lesson behind the person.
		tour := strings.Contains(m.Telegram, "ласточка") && strings.Contains(m.Telegram, "5 зёрнышек")
		if tour != (seg >= 3) {
			t.Errorf("segment %d: tour mentioned in Telegram text = %v", seg, tour)
		}
		if (m.Box != "") != (seg >= 3) {
			t.Errorf("segment %d: e-mail tour box present = %v", seg, m.Box != "")
		}
	}
}

func TestEveryButtonStyleDiffersInEverySegment(t *testing.T) {
	for seg := 1; seg <= Segments; seg++ {
		b := For(seg).Buttons
		if b[StyleDirect] == b[StyleGlance] {
			t.Errorf("segment %d: both styles read %q", seg, b[StyleDirect])
		}
	}
}

func TestLessonsMatchTheSegments(t *testing.T) {
	want := map[int]string{1: "00", 2: "00", 3: "01", 4: "02"}
	for seg, lesson := range want {
		if got := For(seg).Lesson; got != lesson {
			t.Errorf("segment %d opens lesson %q, want %q", seg, got, lesson)
		}
	}
	if got := LessonURL("https://ucimo.ru/", 3, "abcd"); got != "https://ucimo.ru/lesson/01?c=abcd" {
		t.Errorf("LessonURL = %q", got)
	}
	if got := PhotoURL("https://x.test/photos/", 4); got != "https://x.test/photos/return-4.jpg" {
		t.Errorf("PhotoURL = %q", got)
	}
	if got := ImageURL("https://ucimo.ru", 1); got != "https://ucimo.ru/campaign/return-1.jpg" {
		t.Errorf("ImageURL = %q", got)
	}
}

// The pictures the messages point at must exist: one file per segment, served by the site for the e-mail and read from the repository
// by Telegram.
func TestPicturesExist(t *testing.T) {
	for seg := 1; seg <= Segments; seg++ {
		p := filepath.Join("..", "..", "..", "landing", "public", "campaign", For(seg).Image+".jpg")
		if st, err := os.Stat(p); err != nil || st.Size() == 0 {
			t.Errorf("segment %d: picture %s is missing or empty (%v)", seg, p, err)
		}
	}
}

func TestMarkupBecomesTelegramHTML(t *testing.T) {
	got := TelegramHTML("👋 **Ćao (привет)!** a < b & «c»\n\n• x")
	want := "👋 <b>Ćao (привет)!</b> a &lt; b &amp; «c»\n\n• x"
	if got != want {
		t.Errorf("TelegramHTML = %q, want %q", got, want)
	}
	if got := PlainText("**a** b **c**"); got != "a b c" {
		t.Errorf("PlainText = %q", got)
	}
}

// Every Telegram caption must fit Telegram's 1024 characters once it is HTML, and must not leave a stray bold mark.
func TestCaptionsAreWellFormedHTML(t *testing.T) {
	for seg := 1; seg <= Segments; seg++ {
		h := TelegramHTML(For(seg).Telegram)
		if strings.Count(h, "<b>") != strings.Count(h, "</b>") || strings.Contains(h, "**") {
			t.Errorf("segment %d: unbalanced bold marks: %q", seg, h)
		}
		// Telegram counts the characters of the text after the tags are parsed.
		if n := utf8.RuneCountInString(PlainText(For(seg).Telegram)); n > 1024 {
			t.Errorf("segment %d: caption is %d characters", seg, n)
		}
	}
}

// The texts are meant to be read: every message has the greeting emoji or an accent on its first line, and a message carries several.
func TestMessagesHaveEmoji(t *testing.T) {
	count := func(s string) int {
		n := 0
		for _, r := range s {
			if r >= 0x1F300 || (r >= 0x2600 && r <= 0x27BF) {
				n++
			}
		}
		return n
	}
	for seg := 1; seg <= Segments; seg++ {
		m := For(seg)
		if c := count(m.Telegram); c < 3 {
			t.Errorf("segment %d: Telegram text has %d emoji, want at least 3", seg, c)
		}
		if c := count(m.Heading + m.Lead + m.Box); c < 3 {
			t.Errorf("segment %d: e-mail has %d emoji, want at least 3", seg, c)
		}
	}
}
