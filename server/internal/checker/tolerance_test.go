package checker

import "testing"

func TestSoftAcceptance(t *testing.T) {
	cases := []struct {
		name, answer string
		accept       []string
		ok, typo     bool
	}{
		// forgiven, but flagged
		{"missing diacritic", "cokolada", []string{"čokolada"}, true, true},
		{"č vs ć (one sound for a Russian ear)", "veće", []string{"veče"}, true, true},
		{"missing š", "sesir", []string{"šešir"}, true, true},
		{"doubled letter", "Radim kao programmer", []string{"Radim kao programer."}, true, true},
		{"dropped letter in the stem", "hvla", []string{"hvala"}, true, true},
		{"dj for đ", "dovidjenja", []string{"doviđenja"}, true, true},
		// one letter anywhere in a word of 4+ letters is forgiven (and flagged), endings included
		{"case ending sin/sina", "sin", []string{"sina"}, true, true},
		{"plural ending godina/godine", "godina", []string{"godine"}, true, true},
		{"last letter of a long word", "razumen", []string{"razumem"}, true, true},
		{"ending -u/-a", "sestra", []string{"sestru"}, true, true},
		{"vowel swap dobor/dobar", "dobor dan", []string{"dobar dan"}, true, true},
		{"missing letter in an ending (roditeli)", "roditeli", []string{"roditelji"}, true, true},
		{"extra last letter", "hvalas", []string{"hvala"}, true, true},
		// short words stay exact: si/su/je/ja are different forms
		{"short word si/su", "su", []string{"si"}, false, false},
		{"three letters stay exact", "san", []string{"sam"}, false, false},
		{"negation nisi/nije is two edits", "nisi", []string{"nije"}, false, false},
		// still wrong
		{"two words wrong", "dobro dan", []string{"dobar dan"}, false, false},
		{"missing word", "hvala", []string{"Hvala, prijatno!"}, false, false},
		{"doubled last letter is just a slip", "hvalaa", []string{"hvala"}, true, true},
	}
	for _, c := range cases {
		r := Check(c.answer, c.accept)
		if r.OK != c.ok || r.Typo != c.typo {
			t.Errorf("%s: Check(%q, %q) ok=%v typo=%v, want ok=%v typo=%v", c.name, c.answer, c.accept, r.OK, r.Typo, c.ok, c.typo)
		}
		if c.typo && r.Expected != c.accept[0] {
			t.Errorf("%s: Expected = %q, want the correct spelling %q", c.name, r.Expected, c.accept[0])
		}
	}
}

func TestCyrillicAnswerAccepted(t *testing.T) {
	for _, c := range []struct{ ans, want string }{
		{"љубав", "ljubav"},
		{"Шешир", "šešir"},
		{"коњ", "konj"},
		{"конj", "konj"}, // Cyrillic letters mixed with a Latin one
		{"Добро јутро!", "Dobro jutro."},
	} {
		if r := Check(c.ans, []string{c.want}); !r.OK {
			t.Errorf("Check(%q) should match %q: %+v", c.ans, c.want, r)
		}
	}
	// Russian is not Serbian: a Russian answer must not be transliterated into a hit.
	if Check("конь", []string{"konj"}).OK {
		t.Error("Russian «конь» must not pass for konj")
	}
	// A Russian answer to a Russian expected text still works as before.
	if !Check("дом", []string{"дом"}).OK {
		t.Error("Russian exact match broke")
	}
}

func TestSerbianCyrillicToLatin(t *testing.T) {
	if got, ok := SerbianCyrillicToLatin("Његов џем, љубав, ђак"); !ok || got != "njegov džem, ljubav, đak" {
		t.Errorf("got %q %v", got, ok)
	}
	if _, ok := SerbianCyrillicToLatin("привет, я"); ok {
		t.Error("Russian letters must not count as Serbian Cyrillic")
	}
	// a word made only of letters both alphabets share IS valid Serbian Cyrillic
	if got, ok := SerbianCyrillicToLatin("привет"); !ok || got != "privet" {
		t.Errorf("shared-letter word: %q %v", got, ok)
	}
	if _, ok := SerbianCyrillicToLatin("zdravo"); ok {
		t.Error("pure Latin has nothing to transliterate")
	}
}

func TestConjugationKeepsEndingsStrict(t *testing.T) {
	// In a conjugation table every cell differs from its neighbours by the ending:
	// forgiving one letter there would make the exercise meaningless.
	res := CheckForms([]string{"imaš", "imam"}, [][]string{{"imam"}, {"imaš"}})
	for i, r := range res {
		if r.OK {
			t.Errorf("cell %d: a one-letter ending slip must stay wrong in a conjugation table: %+v", i, r)
		}
	}
	if r := CheckForms([]string{"imam"}, [][]string{{"imam"}}); !r[0].OK {
		t.Errorf("exact cell must pass: %+v", r[0])
	}
}

func TestCheckStrictProtectsTheEnding(t *testing.T) {
	for _, c := range []struct{ typed, want string }{
		{"Imam brat.", "Imam brata."},
		{"Moj mama.", "Moja mama."},
		{"Imam trideset godinu.", "Imam trideset godina."},
	} {
		if r := CheckStrict(c.typed, []string{c.want}); r.OK {
			t.Errorf("CheckStrict(%q, %q) passed: the unchanged wrong sentence must stay wrong", c.typed, c.want)
		}
		if r := Check(c.typed, []string{c.want}); !r.OK || !r.Typo {
			t.Errorf("Check(%q, %q) = %+v: the everyday rule forgives one letter", c.typed, c.want, r)
		}
	}
	// Diacritics and doubled letters are still forgiven strictly.
	if r := CheckStrict("sesir", []string{"šešir"}); !r.OK {
		t.Errorf("a missing diacritic is forgiven everywhere: %+v", r)
	}
}
