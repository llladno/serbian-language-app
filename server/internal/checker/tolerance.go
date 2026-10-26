package checker

import "strings"

// Soft acceptance: an answer that is not exact can still be correct if the
// learner only slipped on the SPELLING of one word. We accept it (OK=true) but
// flag it (Typo=true) so the UI tells them how the word is really written.
//
// One letter of difference is forgiven wherever it sits, endings included: the
// learner's job is to know the word, not to spell it. The slip is flagged, and
// the attempt is stored with typo = 1, so "Повторить" brings the exercise back
// until it is written exactly (this is what keeps sin/sina from slipping past
// for good). Two limits remain:
//   - the word is at least 4 letters (si/su/je/ja are different forms, not slips);
//   - conjugation tables (CheckForms) keep the strict rule: every cell there
//     differs from its neighbours by the ending, and the old protection of the
//     last 2 letters (3 for words of 7+) still applies.
// A missing/replaced diacritic (c/č/ć, s/š, z/ž, dj/đ) is forgiven anywhere,
// because for a Russian ear č and ć are one sound.

// serbianCyr maps the Serbian Cyrillic alphabet to Latin. A letter outside
// this set (я, ы, э, ё, щ, ъ, ь, й, ю) means the learner typed Russian, not Serbian.
var serbianCyr = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'ђ': "đ", 'е': "e", 'ж': "ž",
	'з': "z", 'и': "i", 'ј': "j", 'к': "k", 'л': "l", 'љ': "lj", 'м': "m", 'н': "n",
	'њ': "nj", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'ћ': "ć", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "c", 'ч': "č", 'џ': "dž", 'ш': "š",
}

func isCyr(r rune) bool { return r >= 0x0400 && r <= 0x04FF }

// SerbianCyrillicToLatin transliterates s when every Cyrillic letter in it
// belongs to the Serbian alphabet; otherwise (or if s has no Cyrillic) it
// returns s unchanged and false. Latin letters mixed in are kept as typed.
func SerbianCyrillicToLatin(s string) (string, bool) {
	var b strings.Builder
	found := false
	for _, r := range strings.ToLower(s) {
		if !isCyr(r) {
			b.WriteRune(r)
			continue
		}
		lat, ok := serbianCyr[r]
		if !ok {
			return s, false
		}
		found = true
		b.WriteString(lat)
	}
	if !found {
		return s, false
	}
	return b.String(), true
}

var foldPairs = strings.NewReplacer("dž", "dz", "đ", "dj", "č", "c", "ć", "c", "š", "s", "ž", "z")

// fold strips Serbian diacritics so that č/ć/c (etc.) compare equal.
func fold(s string) string { return foldPairs.Replace(s) }

// protectedTail is how many trailing letters carry grammar and so cannot be
// "just a typo".
func protectedTail(n int) int {
	if n >= 7 {
		return 3
	}
	return 2
}

// slip reports whether typed is the expected word with only a spelling slip.
// lenient is the everyday rule (one letter anywhere in a word of 4+ letters);
// strict, for conjugation tables, still protects the ending.
func slip(typed, want string, lenient bool) bool {
	if typed == want {
		return false
	}
	if fold(typed) == fold(want) {
		return true // diacritics only
	}
	w := []rune(want)
	if lenient {
		if len(w) < 4 {
			return false
		}
		return oneEditBefore([]rune(typed), w, len(w)+1) // the limit is past every position
	}
	if len(w) < 5 {
		return false
	}
	limit := len(w) - protectedTail(len(w)) // edits must happen before this index
	return oneEditBefore([]rune(typed), w, limit)
}

// oneEditBefore: is x one substitution / insertion / deletion / adjacent swap
// away from y, with the edit located at an index < limit of y?
func oneEditBefore(x, y []rune, limit int) bool {
	switch {
	case len(x) == len(y):
		var diff []int
		for i := range x {
			if x[i] != y[i] {
				diff = append(diff, i)
			}
		}
		if len(diff) == 1 {
			return diff[0] < limit
		}
		if len(diff) == 2 && diff[1] == diff[0]+1 && x[diff[0]] == y[diff[1]] && x[diff[1]] == y[diff[0]] {
			return diff[1] < limit
		}
	case len(x) == len(y)-1: // a letter is missing in x
		for i := range y {
			if string(append(append([]rune{}, y[:i]...), y[i+1:]...)) != string(x) {
				continue
			}
			if i < limit || doubled(y, i) {
				return true
			}
		}
	case len(x) == len(y)+1: // x has an extra letter
		for i := range x {
			if string(append(append([]rune{}, x[:i]...), x[i+1:]...)) != string(y) {
				continue
			}
			if i < limit || doubled(x, i) {
				return true
			}
		}
	}
	return false
}

// doubled: the letter at i repeats its neighbour («programmer»). A doubled or
// undoubled letter is never a grammatical ending, wherever it sits.
func doubled(r []rune, i int) bool {
	return (i > 0 && r[i] == r[i-1]) || (i+1 < len(r) && r[i] == r[i+1])
}

// softMatch finds an accepted variant that the answer matches up to exactly
// one spelling slip in exactly one word. It returns that variant.
func softMatch(answerTokens []string, accept []string, lenient bool) (string, bool) {
	for _, a := range accept {
		et := normTokens(a)
		if len(et) != len(answerTokens) || len(et) == 0 {
			continue
		}
		bad := -1
		ok := true
		for i := range et {
			if et[i] == answerTokens[i] {
				continue
			}
			if bad >= 0 {
				ok = false // two different words
				break
			}
			bad = i
		}
		if ok && bad >= 0 && slip(answerTokens[bad], et[bad], lenient) {
			return a, true
		}
	}
	return "", false
}
