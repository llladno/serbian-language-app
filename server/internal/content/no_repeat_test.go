package content

import (
	"fmt"
	"strings"
	"testing"
)

// The lines one exercise screen shows together (the step's title in the header, the
// step's note above the exercise, the prompt, and the explanation under the answer)
// must not repeat each other: the app would print the same line twice. The screens
// themselves are guarded in the web tests (web/src/test/repeatedLines.ts); this is the
// same rule for what the course feeds them. A dialogue is left out: its exercises sit
// between lines of the conversation, which the web test covers.

// repeatedExerciseLines lists every exercise screen that would show a line twice, and
// how many exercises it looked at.
func repeatedExerciseLines(c *Course) (found []string, checked int) {
	norm := func(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
	for id, l := range c.Lessons {
		for _, st := range l.Steps {
			if st.Kind == "dialogue" {
				continue
			}
			for _, e := range st.Exercises {
				checked++
				lines := []struct{ name, text string }{
					{"step title", st.Title}, {"step note", st.Markdown}, {"prompt", e.Prompt}, {"explain", e.Explain},
				}
				for i := range lines {
					for j := i + 1; j < len(lines); j++ {
						if a := norm(lines[i].text); a != "" && a == norm(lines[j].text) {
							found = append(found, fmt.Sprintf("lesson %s, %s: %s and %s are the same line %q",
								id, e.ID, lines[i].name, lines[j].name, lines[i].text))
						}
					}
				}
			}
		}
	}
	return found, checked
}

func TestRealCourseNoLineShownTwiceOnAnExercise(t *testing.T) {
	c, err := Load("../../../content")
	if err != nil {
		t.Fatalf("real content: %v", err)
	}
	found, checked := repeatedExerciseLines(c)
	if checked < 1000 {
		t.Fatalf("checked only %d exercises: the course is not being read", checked)
	}
	for _, f := range found {
		t.Error(f)
	}
}

// The check itself catches a planted repeat, whatever the case and spacing.
func TestRepeatedExerciseLinesCatchesARepeat(t *testing.T) {
	c := &Course{Lessons: map[string]*Lesson{"11": {Steps: []Step{
		{Kind: "practice", Title: "Послушай и выбери", Exercises: []Exercise{
			{ID: "11.1.1", Prompt: "Выбери перевод."},
			{ID: "11.1.2", Prompt: "послушай  и выбери"},
		}},
		{Kind: "dialogue", Title: "Диалог", Exercises: []Exercise{{ID: "11.2.1", Prompt: "Диалог"}}},
	}}}}
	found, checked := repeatedExerciseLines(c)
	if checked != 2 || len(found) != 1 || !strings.Contains(found[0], "11.1.2") {
		t.Errorf("checked %d, found %q; want 2 checked and only 11.1.2 (a dialogue is left out)", checked, found)
	}
}
