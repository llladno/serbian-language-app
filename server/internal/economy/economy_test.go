package economy

import "testing"

func TestParseLadderAndDrip(t *testing.T) {
	l, err := ParseLadder(`[[1,1],[30,2],[100,3]]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cases := []struct {
		streak int
		want   int64
	}{
		{0, 0}, {1, 1}, {29, 1}, {30, 2}, {99, 2}, {100, 3}, {500, 3},
	}
	for _, c := range cases {
		if got := l.DripFor(c.streak); got != c.want {
			t.Errorf("DripFor(%d) = %d, want %d", c.streak, got, c.want)
		}
	}
}

func TestParseLadderRejectsGarbage(t *testing.T) {
	for _, in := range []string{``, `[]`, `[[0,1]]`, `[[2,1],[1,1]]`, `nope`} {
		if _, err := ParseLadder(in); err == nil {
			t.Errorf("ParseLadder(%q) succeeded; want error", in)
		}
	}
}

func TestEffectivePriceTakesTheBetterDiscount(t *testing.T) {
	cases := []struct {
		name        string
		base        int64
		sale, promo int
		wantPrice   int64
		wantApplied string
	}{
		{"no discounts", 500, 0, 0, 500, ""},
		{"sale only", 500, 20, 0, 400, "sale"},
		{"promo only", 500, 0, 10, 450, "promo"},
		{"sale wins", 500, 20, 10, 400, "sale"},
		{"promo wins", 500, 10, 20, 400, "promo"},
		{"tie prefers sale", 500, 20, 20, 400, "sale"},
		{"rounds up, never free by accident", 25, 99, 0, 1, "sale"},
		{"full discount still costs nothing less than zero", 25, 100, 0, 0, "sale"},
	}
	for _, c := range cases {
		price, applied := EffectivePrice(c.base, c.sale, c.promo)
		if price != c.wantPrice || applied != c.wantApplied {
			t.Errorf("%s: EffectivePrice(%d,%d,%d) = %d,%q; want %d,%q",
				c.name, c.base, c.sale, c.promo, price, applied, c.wantPrice, c.wantApplied)
		}
	}
}
