// Package content loads the structured course content (lessons, exercises,
// vocabulary, false friends) from a directory of YAML and Markdown files.
package content

// Course is an in-memory snapshot of everything under a content directory.
type Course struct {
	Title        string
	Phases       []Phase
	Lessons      map[string]*Lesson         // key: lesson id ("01")
	Exercises    map[string][]ExerciseBlock // key: lesson id
	Vocab        []Vocab
	FalseFriends []FalseFriend
}

// Phase groups lessons (курс делится на фазы A/B/C).
type Phase struct {
	ID      string
	Title   string
	Lessons []string
}

// Lesson is one lesson's metadata plus its theory markdown.
type Lesson struct {
	ID           string
	Title        string
	Subtitle     string
	Planned      bool   // true when no markdown file exists yet
	MarkdownPath string // relative path, "" when planned
	Markdown     string // "" when planned
}

// ExerciseBlock is a titled group of exercises (блок A/B/C… в уроке).
type ExerciseBlock struct {
	ID          string
	Title       string
	Instruction string
	Exercises   []Exercise
}

// Exercise is a single practice item.
type Exercise struct {
	ID      string
	Type    string // translate | fill_blank | fix_error | conjugate | free
	Prompt  string
	Explain string
	Sample  string // free: образец ответа
	Meta    string // conjugate: подпись (напр. "тип I")

	Accept      []string   // auto types except conjugate: принимаемые ответы
	Forms       []string   // conjugate: подписи форм (ja/ti/on…)
	AcceptForms [][]string // conjugate: принимаемые ответы по форме
}

// Vocab is one dictionary entry.
type Vocab struct {
	ID       string
	Latin    string
	Cyrillic string
	RU       string
	Note     string
	Lesson   string
	POS      string
	Gender   string
	Aspect   string
	Tags     []string
	Emoji    string // optional
	Image    string // optional filename under content/images/
}

// FalseFriend is one RU↔SR false-friend entry.
type FalseFriend struct {
	ID      string
	SR      string
	Means   string
	Not     string
	Correct string
	Group   string // top | shop | small
	Emoji   string
	Image   string
}
