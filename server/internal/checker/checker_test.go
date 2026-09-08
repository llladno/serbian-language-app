package checker

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"  Zdravo!  ":     "zdravo",
		"Kako   si?":      "kako si",
		"Iz Rusije sam.":  "iz rusije sam",
		"“Ćao”":           "ćao",
		"Drago mi je . ":  "drago mi je",
		"Zdravo! Kako si?": "zdravo kako si",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckExactAndVariant(t *testing.T) {
	accept := []string{"Zdravo! Kako si?", "Ćao! Kako si?"}
	if !Check("zdravo kako si", accept).OK {
		t.Error("normalized exact should pass")
	}
	if !Check("Ćao! Kako si?", accept).OK {
		t.Error("second variant should pass")
	}
	if Check("Kako si", accept).OK {
		t.Error("partial answer should not pass")
	}
}

func TestCheckWrongProducesDiffAndExpected(t *testing.T) {
	r := Check("Zdravo! Kako sti?", []string{"Zdravo! Kako si?"})
	if r.OK {
		t.Fatal("should fail")
	}
	if r.Expected != "Zdravo! Kako si?" {
		t.Errorf("Expected = %q", r.Expected)
	}
	var wrong int
	for _, c := range r.Diff {
		if !c.OK {
			wrong++
		}
	}
	if wrong == 0 {
		t.Error("expected at least one wrong chunk")
	}
}

func TestCheckPicksClosestExpected(t *testing.T) {
	r := Check("neću kafu", []string{"Ja neću kafu, hvala.", "Neću ništa, hvala."})
	if r.Expected != "Ja neću kafu, hvala." {
		t.Errorf("Expected = %q, want the closer variant", r.Expected)
	}
}

func TestCheckNearMiss(t *testing.T) {
	// one missing diacritic -> near miss
	r := Check("Neću kafu, hvala", []string{"Neću kafu, hvala."})
	_ = r
	miss := Check("Zdravo! Kako si", []string{"Zdravo! Kako si?"})
	if !miss.OK {
		t.Fatalf("trailing punct should still pass")
	}
	typo := Check("Nemam vremana", []string{"Nemam vremena"})
	if typo.OK || !typo.NearMiss {
		t.Errorf("one-letter typo should be a near miss: %+v", typo)
	}
	wayOff := Check("ja ne znam", []string{"Nemam vremena"})
	if wayOff.NearMiss {
		t.Errorf("unrelated answer should not be a near miss")
	}
}

func TestCheckForms(t *testing.T) {
	rs := CheckForms(
		[]string{"govorim", "govoris", "govori"},
		[][]string{{"govorim"}, {"govoriš"}, {"govori"}},
	)
	if !rs[0].OK || rs[1].OK || !rs[2].OK {
		t.Errorf("results = %+v", rs)
	}
}

func TestCheckFormsMissingAnswer(t *testing.T) {
	rs := CheckForms([]string{"idem"}, [][]string{{"idem"}, {"ideš"}})
	if !rs[0].OK || rs[1].OK {
		t.Errorf("results = %+v", rs)
	}
}

func TestCheckChoice(t *testing.T) {
	if r := CheckChoice("Hvala", "hvala"); !r.OK {
		t.Errorf("case-insensitive choice should pass: %+v", r)
	}
	if r := CheckChoice("Molim", "Hvala"); r.OK || r.Expected != "Hvala" {
		t.Errorf("wrong choice: %+v", r)
	}
}

func TestCheckMatch(t *testing.T) {
	pairs := [][2]string{{"Dobro jutro", "Доброе утро"}, {"Laku noć", "Спокойной ночи"}}

	ok, per := CheckMatch(map[string]string{
		"Dobro jutro": "Доброе утро", "Laku noć": "Спокойной ночи",
	}, pairs)
	if !ok || !per["Dobro jutro"] || !per["Laku noć"] {
		t.Errorf("all-correct match failed: ok=%v per=%v", ok, per)
	}

	ok, per = CheckMatch(map[string]string{
		"Dobro jutro": "Спокойной ночи", "Laku noć": "Доброе утро",
	}, pairs)
	if ok || per["Dobro jutro"] {
		t.Errorf("swapped match should fail: ok=%v per=%v", ok, per)
	}

	if ok, _ := CheckMatch(map[string]string{"Dobro jutro": "Доброе утро"}, pairs); ok {
		t.Error("incomplete match should not be ok")
	}
}
