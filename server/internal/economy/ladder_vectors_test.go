package economy

import "testing"

// The admin panel re-implements this validator in TypeScript, because the two
// services share no package. These vectors are the contract between them: the
// same strings appear in ucimo-content-admin's economy.service.promo.spec.ts.
//
// The contract is one-directional, not equality: anything the admin SAVES, this
// must accept. A ladder this function rejects makes the server fall back to its
// default rate silently, so an admin that accepted it would change the economy
// with nothing in the UI to show it. The admin being stricter is fine — it just
// refuses input it cannot be sure about.
//
// One vector is deliberately asymmetric. "[[1,1,1]]" is accepted here because
// encoding/json drops the extra element when filling a [2]int, so the step
// silently becomes [1,1] — something the author plainly did not write. The admin
// rejects it rather than letting a misread ladder be saved, and that asymmetry
// is the useful direction.
func TestParseLadderSharedVectorsWithTheAdminPanel(t *testing.T) {
	cases := []struct {
		raw          string
		adminAccepts bool
		note         string
	}{
		{"nope", false, "not JSON"},
		{"[]", false, "empty"},
		{"[[2,1],[30,2]]", false, "does not start at day 1"},
		{"[[1,1,1]]", false, "admin is stricter: json would truncate this to [1,1]"},
		{"[[1,1.5]]", false, "non-integer payout"},
		{"[[1,1],[30,-2]]", false, "negative payout"},
		{"[[1,1],[30,2],[10,3]]", false, "days do not increase"},
		{"[[1,1],[30,2],[100,3]]", true, "the seeded ladder"},
	}
	for _, c := range cases {
		_, err := ParseLadder(c.raw)
		if c.adminAccepts && err != nil {
			t.Errorf("the admin panel saves %q (%s) but ParseLadder rejects it: %v", c.raw, c.note, err)
		}
	}
}
