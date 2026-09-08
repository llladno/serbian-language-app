package content

import "testing"

func TestInterpolatePersona(t *testing.T) {
	p := map[string]string{"name_latin": "Griša", "city_ru": "Нови-Сад"}
	got := interpolate("Zdravo, ja sam {name_latin}. Živim u {city_ru}. {unknown}", p)
	want := "Zdravo, ja sam Griša. Živim u Нови-Сад. {unknown}"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if interpolate("plain", nil) != "plain" {
		t.Error("nil persona should be a no-op")
	}
}

func TestPersonaWiredIntoLoad(t *testing.T) {
	c, err := Load("testdata/content")
	if err != nil {
		t.Fatal(err)
	}
	// Fixture 90 has no {tokens}; assert interpolation ran without eating text.
	if c.Lessons["90"].Steps[0].Markdown == "" {
		t.Fatal("step markdown lost after interpolation")
	}
}

func TestPersonaInterpolatesStepText(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"course.yaml":        "title: x\nphases: []\nlessons:\n  \"01\": {title: t, file: lessons/01.yaml}\n",
		"lessons/01.yaml":    "lesson: \"01\"\ntitle: t\nsteps:\n  - id: \"01.1\"\n    kind: teach\n    md: \"01/a.md\"\n  - id: \"01.2\"\n    kind: practice\n    exercises:\n      - {id: \"01.2.1\", type: translate, prompt: \"Скажи, что ты {job_ru}\", accept: [\"Ja sam programer\"]}\n",
		"lessons/01/a.md":    "Živiš u {city_ru}.\n",
		"vocab.yaml":         "[]\n",
		"false-friends.yaml": "[]\n",
		"persona.yaml":       "city_ru: \"Нови-Сад\"\njob_ru: \"программист\"\n",
	})
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Lessons["01"].Steps[0].Markdown; got != "Živiš u Нови-Сад." {
		t.Errorf("teach markdown = %q", got)
	}
	if got := c.Lessons["01"].Steps[1].Exercises[0].Prompt; got != "Скажи, что ты программист" {
		t.Errorf("exercise prompt = %q", got)
	}
	// accept must be untouched
	if got := c.Exercises["01"][0].Exercises[0].Accept[0]; got != "Ja sam programer" {
		t.Errorf("accept changed: %q", got)
	}
}
