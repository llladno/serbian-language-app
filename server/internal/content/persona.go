package content

import (
	"path/filepath"
	"regexp"
)

var personaRe = regexp.MustCompile(`\{([a-z_]+)\}`)

// loadPersona reads persona.yaml (a flat key -> value map). A missing or
// unreadable file yields an empty map and no error: personalization is opt-in.
func loadPersona(dir string) map[string]string {
	var raw map[string]string
	if err := readYAML(filepath.Join(dir, "persona.yaml"), &raw); err != nil {
		return map[string]string{}
	}
	return raw
}

// interpolate replaces {key} tokens with persona values. Unknown keys are
// left untouched.
func interpolate(s string, p map[string]string) string {
	if len(p) == 0 || s == "" {
		return s
	}
	return personaRe.ReplaceAllStringFunc(s, func(m string) string {
		if v, ok := p[m[1:len(m)-1]]; ok {
			return v
		}
		return m
	})
}

// applyPersona interpolates persona tokens into every learner-facing string:
// lesson subtitles, teach/reading markdown, and exercise prompts/samples.
// Answer fields (accept, answer, bank, pairs, say) are deliberately left
// alone so grading stays deterministic.
//
// Steps and Course.Exercises share one backing array per lesson, so mutating
// exercises through Steps updates both views.
func applyPersona(c *Course, p map[string]string) {
	if len(p) == 0 {
		return
	}
	for _, l := range c.Lessons {
		l.Subtitle = interpolate(l.Subtitle, p)
		l.Markdown = interpolate(l.Markdown, p)
		l.Reading = interpolate(l.Reading, p)
		l.ReadingRU = interpolate(l.ReadingRU, p)
		for i := range l.Steps {
			l.Steps[i].Markdown = interpolate(l.Steps[i].Markdown, p)
			l.Steps[i].MarkdownRU = interpolate(l.Steps[i].MarkdownRU, p)
			for j := range l.Steps[i].Exercises {
				l.Steps[i].Exercises[j].Prompt = interpolate(l.Steps[i].Exercises[j].Prompt, p)
				l.Steps[i].Exercises[j].Sample = interpolate(l.Steps[i].Exercises[j].Sample, p)
			}
		}
	}
}
