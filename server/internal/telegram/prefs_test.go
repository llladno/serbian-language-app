package telegram

import "testing"

func TestPrefsStateEncodeRoundTrip(t *testing.T) {
	for _, st := range []PrefsState{
		{}, {Learning: true}, {Billing: true}, {News: true},
		{Learning: true, Billing: true}, {Learning: true, Billing: true, News: true},
	} {
		got := DecodePrefsState(st.Encode())
		if got != st {
			t.Errorf("DecodePrefsState(%q) = %+v, want %+v", st.Encode(), got, st)
		}
	}
	if got := (PrefsState{Learning: true, News: true}).Encode(); got != "101" {
		t.Errorf("Encode = %q, want 101 (one digit per category in PrefCategories order)", got)
	}
	// A short or damaged target means "off" for what is missing, never a panic.
	if got := DecodePrefsState("1"); got != (PrefsState{Learning: true}) {
		t.Errorf("DecodePrefsState(1) = %+v", got)
	}
	if got := DecodePrefsState(""); got != (PrefsState{}) {
		t.Errorf("DecodePrefsState(empty) = %+v", got)
	}
}

func TestParsePref(t *testing.T) {
	cases := []struct {
		data   string
		cat    PrefCategory
		on, ok bool
	}{
		{"pref:learning:1", PrefLearning, true, true},
		{"pref:billing:0", PrefBilling, false, true},
		{"pref:news:1", PrefNews, true, true},
		{"pref:unknown:1", "", false, false},
		{"pref:news:2", "", false, false},
		{"pref:news", "", false, false},
		{"prefoff:news", "", false, false},
		{"chan_check", "", false, false},
	}
	for _, tc := range cases {
		cat, on, ok := ParsePref(tc.data)
		if cat != tc.cat || on != tc.on || ok != tc.ok {
			t.Errorf("ParsePref(%q) = (%q, %v, %v), want (%q, %v, %v)", tc.data, cat, on, ok, tc.cat, tc.on, tc.ok)
		}
	}
}

func TestParsePrefOff(t *testing.T) {
	if c, ok := ParsePrefOff("prefoff:learning"); !ok || c != PrefLearning {
		t.Errorf("ParsePrefOff(prefoff:learning) = (%q, %v)", c, ok)
	}
	for _, bad := range []string{"prefoff:", "prefoff:nope", "pref:learning:0", ""} {
		if _, ok := ParsePrefOff(bad); ok {
			t.Errorf("ParsePrefOff(%q) ok, want not", bad)
		}
	}
}

func TestIsCommand(t *testing.T) {
	yes := []string{"/stop", "/stop@ucimoappbot", "  /stop  "}
	no := []string{"/stop now", "stop", "/stopp", "/start", "", "/settings"}
	for _, s := range yes {
		if !IsCommand(s, "stop") {
			t.Errorf("IsCommand(%q, stop) = false", s)
		}
	}
	for _, s := range no {
		if IsCommand(s, "stop") {
			t.Errorf("IsCommand(%q, stop) = true", s)
		}
	}
}

// Every toggle must set the OPPOSITE of what it shows: a tick means "on", so
// pressing it sends 0.
func TestPrefsRowsPressSetsTheOpposite(t *testing.T) {
	rows := prefsRows(PrefsState{Learning: true, Billing: false, News: false})
	if len(rows) != len(PrefCategories) {
		t.Fatalf("%d rows, want %d", len(rows), len(PrefCategories))
	}
	want := []string{"pref:learning:0", "pref:billing:1", "pref:news:1"}
	for i, row := range rows {
		if got := row[0]["callback_data"]; got != want[i] {
			t.Errorf("row %d callback_data = %v, want %s", i, got, want[i])
		}
	}
	if text := rows[0][0]["text"].(string); text[:len("✅")] != "✅" {
		t.Errorf("a category that is on should start with a tick, got %q", text)
	}
	if text := rows[1][0]["text"].(string); text[:len("⬜")] != "⬜" {
		t.Errorf("a category that is off should start with an empty box, got %q", text)
	}
}
