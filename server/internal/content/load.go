package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// ---- YAML file shapes -------------------------------------------------------

type courseFile struct {
	Title  string `yaml:"title"`
	Phases []struct {
		ID      string   `yaml:"id"`
		Title   string   `yaml:"title"`
		Lessons []string `yaml:"lessons"`
	} `yaml:"phases"`
	Lessons map[string]struct {
		Title    string `yaml:"title"`
		Subtitle string `yaml:"subtitle"`
		File     string `yaml:"file"`
	} `yaml:"lessons"`
}

type exerciseFile struct {
	Lesson string `yaml:"lesson"`
	Blocks []struct {
		ID          string `yaml:"id"`
		Title       string `yaml:"title"`
		Instruction string `yaml:"instruction"`
		Exercises   []struct {
			ID      string    `yaml:"id"`
			Type    string    `yaml:"type"`
			Prompt  string    `yaml:"prompt"`
			Explain string    `yaml:"explain"`
			Sample  string    `yaml:"sample"`
			Meta    string    `yaml:"meta"`
			Forms   []string  `yaml:"forms"`
			Accept  yaml.Node `yaml:"accept"`
		} `yaml:"exercises"`
	} `yaml:"blocks"`
}

type vocabFile []struct {
	ID       string   `yaml:"id"`
	Latin    string   `yaml:"latin"`
	Cyrillic string   `yaml:"cyrillic"`
	RU       string   `yaml:"ru"`
	Note     string   `yaml:"note"`
	Lesson   string   `yaml:"lesson"`
	POS      string   `yaml:"pos"`
	Gender   string   `yaml:"gender"`
	Aspect   string   `yaml:"aspect"`
	Tags     []string `yaml:"tags"`
	Emoji    string   `yaml:"emoji"`
	Image    string   `yaml:"image"`
}

type falseFriendFile []struct {
	ID      string `yaml:"id"`
	SR      string `yaml:"sr"`
	Means   string `yaml:"means"`
	Not     string `yaml:"not"`
	Correct string `yaml:"correct"`
	Group   string `yaml:"group"`
	Emoji   string `yaml:"emoji"`
	Image   string `yaml:"image"`
}

// autoTypes are exercise types whose answers are auto-checked against Accept.
var autoTypes = map[string]bool{"translate": true, "fill_blank": true, "fix_error": true}

// Load reads and validates the entire content tree rooted at dir.
func Load(dir string) (*Course, error) {
	c := &Course{
		Lessons:   map[string]*Lesson{},
		Exercises: map[string][]ExerciseBlock{},
	}

	// course.yaml
	var cf courseFile
	if err := readYAML(filepath.Join(dir, "course.yaml"), &cf); err != nil {
		return nil, fmt.Errorf("course.yaml: %w", err)
	}
	c.Title = cf.Title
	for _, p := range cf.Phases {
		c.Phases = append(c.Phases, Phase{ID: p.ID, Title: p.Title, Lessons: p.Lessons})
	}
	for id, le := range cf.Lessons {
		l := &Lesson{ID: id, Title: le.Title, Subtitle: le.Subtitle}
		if le.File == "" {
			l.Planned = true
		} else {
			l.MarkdownPath = le.File
			md, err := os.ReadFile(filepath.Join(dir, le.File))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", le.File, err)
			}
			l.Markdown, l.Reading, l.ReadingRU = extractReading(string(md))
		}
		c.Lessons[id] = l
	}

	// exercises/*.yaml
	exFiles, _ := filepath.Glob(filepath.Join(dir, "exercises", "*.yaml"))
	sort.Strings(exFiles)
	for _, path := range exFiles {
		rel := relPath(dir, path)
		if filepath.Base(path) == "_TEMPLATE.yaml" {
			continue
		}
		var ef exerciseFile
		if err := readYAML(path, &ef); err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		if _, ok := c.Lessons[ef.Lesson]; !ok {
			return nil, fmt.Errorf("%s: exercises target unknown lesson %q", rel, ef.Lesson)
		}
		var blocks []ExerciseBlock
		for _, b := range ef.Blocks {
			block := ExerciseBlock{ID: b.ID, Title: b.Title, Instruction: b.Instruction}
			for _, e := range b.Exercises {
				ex := Exercise{
					ID: e.ID, Type: e.Type, Prompt: e.Prompt,
					Explain: e.Explain, Sample: e.Sample, Meta: e.Meta, Forms: e.Forms,
				}
				switch {
				case e.Type == "conjugate":
					if err := e.Accept.Decode(&ex.AcceptForms); err != nil {
						return nil, fmt.Errorf("%s: exercise %s: accept must be a list of lists: %w", rel, e.ID, err)
					}
					if len(ex.AcceptForms) != len(e.Forms) {
						return nil, fmt.Errorf("%s: exercise %s: %d forms but %d accept rows", rel, e.ID, len(e.Forms), len(ex.AcceptForms))
					}
				case autoTypes[e.Type]:
					if err := e.Accept.Decode(&ex.Accept); err != nil {
						return nil, fmt.Errorf("%s: exercise %s: accept must be a list of strings: %w", rel, e.ID, err)
					}
					if len(ex.Accept) == 0 {
						return nil, fmt.Errorf("%s: exercise %s: %s needs a non-empty accept list", rel, e.ID, e.Type)
					}
				case e.Type == "free":
					// no auto-check
				default:
					return nil, fmt.Errorf("%s: exercise %s: unknown type %q", rel, e.ID, e.Type)
				}
				block.Exercises = append(block.Exercises, ex)
			}
			blocks = append(blocks, block)
		}
		c.Exercises[ef.Lesson] = blocks
	}

	// vocab.yaml
	var vf vocabFile
	if err := readYAML(filepath.Join(dir, "vocab.yaml"), &vf); err != nil {
		return nil, fmt.Errorf("vocab.yaml: %w", err)
	}
	seenV := map[string]bool{}
	for _, v := range vf {
		if seenV[v.ID] {
			return nil, fmt.Errorf("vocab.yaml: duplicate id %q", v.ID)
		}
		seenV[v.ID] = true
		audio := ""
		if _, err := os.Stat(filepath.Join(dir, "audio", v.ID+".mp3")); err == nil {
			audio = v.ID + ".mp3"
		}
		c.Vocab = append(c.Vocab, Vocab{
			ID: v.ID, Latin: v.Latin, Cyrillic: v.Cyrillic, RU: v.RU, Note: v.Note,
			Lesson: v.Lesson, POS: v.POS, Gender: v.Gender, Aspect: v.Aspect, Tags: v.Tags,
			Emoji: v.Emoji, Image: v.Image, Audio: audio,
		})
	}

	// false-friends.yaml
	var ff falseFriendFile
	if err := readYAML(filepath.Join(dir, "false-friends.yaml"), &ff); err != nil {
		return nil, fmt.Errorf("false-friends.yaml: %w", err)
	}
	seenF := map[string]bool{}
	for _, f := range ff {
		if seenF[f.ID] {
			return nil, fmt.Errorf("false-friends.yaml: duplicate id %q", f.ID)
		}
		seenF[f.ID] = true
		c.FalseFriends = append(c.FalseFriends, FalseFriend{
			ID: f.ID, SR: f.SR, Means: f.Means, Not: f.Not, Correct: f.Correct, Group: f.Group,
			Emoji: f.Emoji, Image: f.Image,
		})
	}

	return c, nil
}

func readYAML(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, v)
}

func relPath(dir, path string) string {
	if r, err := filepath.Rel(dir, path); err == nil {
		return r
	}
	return path
}
