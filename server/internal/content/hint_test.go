package content

import (
	"reflect"
	"testing"
)

func TestValidateHintRejectsTheAnswer(t *testing.T) {
	ex := Exercise{ID: "x", Type: "fill_blank", Accept: []string{"sestru"}, Hint: "Imam + винительный: sestra → sestru"}
	if err := validateHint(ex); err == nil {
		t.Error("a hint that spells the answer must be rejected")
	}
	ex.Hint = "После «имею» слово на -a меняет окончание на -u."
	if err := validateHint(ex); err != nil {
		t.Errorf("a nudge must pass: %v", err)
	}
	ch := Exercise{ID: "y", Type: "choice", Options: []string{"u gradu", "u grad"}, Answer: "u gradu", Hint: "Где? — «u gradu» (локатив)"}
	if err := validateHint(ch); err == nil {
		t.Error("choice hint with the answer must be rejected")
	}
	cj := Exercise{ID: "z", Type: "conjugate", AcceptForms: [][]string{{"sam"}, {"si"}}, Hint: "ja → sam"}
	if err := validateHint(cj); err == nil {
		t.Error("conjugate hint with a form must be rejected")
	}
	if err := validateHint(Exercise{ID: "e", Type: "translate", Accept: []string{"Hvala"}}); err != nil {
		t.Errorf("no hint is fine: %v", err)
	}
}

func TestFillSentences(t *testing.T) {
	cases := []struct {
		prompt string
		accept []string
		want   []string
	}{
		{"«Я в центре» → Ja sam u ___.", []string{"centru"}, []string{"Ja sam u centru."}},
		{"У меня есть сын: «Imam ___.»", []string{"sina"}, []string{"Imam sina."}},
		{"Ti ___ student? (ты)", []string{"si"}, []string{"Ti si student?"}},
		{"«Он не студент» → On ___ student.", []string{"nije"}, []string{"On nije student."}},
		{"Утро: «___ jutro!»", []string{"Dobro", "dobro"}, []string{"Dobro jutro!", "dobro jutro!"}},
		{"Что-то без пропуска", []string{"x"}, nil},
		{"два ___ пропуска ___", []string{"x"}, nil},
	}
	for _, c := range cases {
		if got := FillSentences(c.prompt, c.accept); !reflect.DeepEqual(got, c.want) {
			t.Errorf("FillSentences(%q) = %q, want %q", c.prompt, got, c.want)
		}
	}
}
