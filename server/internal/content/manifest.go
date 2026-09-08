package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type manifestFile struct {
	Lesson   string   `yaml:"lesson"`
	Title    string   `yaml:"title"`
	Subtitle string   `yaml:"subtitle"`
	Teaches  []string `yaml:"teaches"`
	Steps    []struct {
		ID        string         `yaml:"id"`
		Kind      string         `yaml:"kind"`
		Title     string         `yaml:"title"`
		MD        string         `yaml:"md"`
		AlsoOK    []string       `yaml:"also_ok"`
		Mixed     bool           `yaml:"mixed"`
		Exercises []exerciseYAML `yaml:"exercises"`
	} `yaml:"steps"`
}

var stepKinds = map[string]bool{"teach": true, "practice": true, "reading": true, "checkpoint": true}

// parseManifest loads a lessons/NN.yaml manifest into a Lesson with Steps.
// Course.Exercises is populated separately (see collectExercises).
func parseManifest(dir, rel string) (*Lesson, error) {
	var mf manifestFile
	if err := readYAML(filepath.Join(dir, rel), &mf); err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	if mf.Lesson == "" {
		return nil, fmt.Errorf("%s: missing lesson id", rel)
	}
	l := &Lesson{
		ID: mf.Lesson, Title: mf.Title, Subtitle: mf.Subtitle,
		Manifest: true, MarkdownPath: rel, Teaches: mf.Teaches,
	}

	seenStep, seenEx := map[string]bool{}, map[string]bool{}
	for _, s := range mf.Steps {
		if !stepKinds[s.Kind] {
			return nil, fmt.Errorf("%s: step %q: unknown kind %q", rel, s.ID, s.Kind)
		}
		if !strings.HasPrefix(s.ID, mf.Lesson+".") {
			return nil, fmt.Errorf("%s: step %q: id must start with %q", rel, s.ID, mf.Lesson+".")
		}
		if seenStep[s.ID] {
			return nil, fmt.Errorf("%s: duplicate step id %q", rel, s.ID)
		}
		seenStep[s.ID] = true

		step := Step{ID: s.ID, Kind: s.Kind, Title: s.Title, AlsoOK: s.AlsoOK, Mixed: s.Mixed}
		if s.MD != "" {
			raw, err := os.ReadFile(filepath.Join(dir, "lessons", s.MD))
			if err != nil {
				return nil, fmt.Errorf("%s: step %s: %w", rel, s.ID, err)
			}
			if s.Kind == "reading" {
				_, step.Markdown, step.MarkdownRU = extractReading(readingOpen + "\n" + string(raw) + "\n" + readingClose)
			} else {
				step.Markdown = strings.TrimSpace(string(raw))
			}
		}
		for _, e := range s.Exercises {
			if seenEx[e.ID] {
				return nil, fmt.Errorf("%s: duplicate exercise id %q", rel, e.ID)
			}
			seenEx[e.ID] = true
			ex, err := decodeExercise(dir, rel, e)
			if err != nil {
				return nil, err
			}
			step.Exercises = append(step.Exercises, ex)
		}

		switch s.Kind {
		case "teach":
			if step.Markdown == "" || len(step.Exercises) > 0 {
				return nil, fmt.Errorf("%s: step %s: teach needs md and no exercises", rel, s.ID)
			}
		case "practice", "checkpoint":
			if len(step.Exercises) == 0 {
				return nil, fmt.Errorf("%s: step %s: %s needs exercises", rel, s.ID, s.Kind)
			}
		case "reading":
			if step.Markdown == "" {
				return nil, fmt.Errorf("%s: step %s: reading needs md", rel, s.ID)
			}
		}
		if err := checkDifficultyOrder(rel, step); err != nil {
			return nil, err
		}
		l.Steps = append(l.Steps, step)
	}
	if len(l.Steps) == 0 {
		return nil, fmt.Errorf("%s: no steps", rel)
	}

	// Legacy-compat view: expose the first reading step through Lesson.Reading
	// so the existing DTO field and content guard-tests keep working.
	for _, s := range l.Steps {
		if s.Kind == "reading" && l.Reading == "" {
			l.Reading, l.ReadingRU = s.Markdown, s.MarkdownRU
		}
	}
	return l, nil
}

// collectExercises rebuilds Course.Exercises for a manifest lesson: one block
// per practice/checkpoint step, block id == step id.
func collectExercises(l *Lesson) []ExerciseBlock {
	var out []ExerciseBlock
	for _, s := range l.Steps {
		if len(s.Exercises) > 0 {
			out = append(out, ExerciseBlock{ID: s.ID, Title: s.Title, Exercises: s.Exercises})
		}
	}
	return out
}

// synthesizeSteps turns a legacy .md lesson into a step flow: a teach card
// holding the whole theory, one practice step per exercise block, then a
// reading step. Step ids reuse block ids so api.findExercise stays consistent.
func synthesizeSteps(l *Lesson, blocks []ExerciseBlock) []Step {
	var steps []Step
	if l.Markdown != "" {
		steps = append(steps, Step{ID: "teach", Kind: "teach", Title: "Теория", Markdown: l.Markdown})
	}
	for _, b := range blocks {
		steps = append(steps, Step{ID: b.ID, Kind: "practice", Title: b.Title, Exercises: b.Exercises})
	}
	if l.Reading != "" {
		steps = append(steps, Step{
			ID: "reading", Kind: "reading", Title: "Текст для чтения",
			Markdown: l.Reading, MarkdownRU: l.ReadingRU,
		})
	}
	return steps
}
