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

	allowWords []string // flattened content/allow-words.yaml (lexicon guard)
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
	Markdown     string // "" when planned; reading block stripped out

	// Reading is an optional short Serbian text for reading practice, pulled
	// from a <!-- reading --> block in the lesson markdown. ReadingRU is its
	// Russian translation (may be empty).
	Reading   string
	ReadingRU string

	// Manifest is true when the lesson was loaded from a lessons/NN.yaml
	// manifest (the step model) rather than a single .md file.
	Manifest bool
	// Steps is the ordered lesson flow. It is always populated for a lesson
	// with content: parsed from the manifest, or synthesized for a legacy
	// .md lesson (a teach card, then one practice step per exercise block,
	// then a reading step).
	Steps []Step
	// Teaches lists the vocab ids this lesson introduces (manifest only).
	Teaches []string
}

// Step is one screen of a lesson: a short teach card, a practice block, a
// reading text, or a mixed checkpoint.
type Step struct {
	ID    string
	Kind  string // teach | practice | reading | checkpoint
	Title string

	Markdown   string // teach/practice reminder, or reading (Serbian side)
	MarkdownRU string // reading: Russian translation

	AlsoOK    []string // extra tokens the lexicon guard allows in this step
	Mixed     bool     // practice: opt out of the difficulty-order check
	Exercises []Exercise
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
	Type    string // translate | fill_blank | fix_error | conjugate | free | listen
	Prompt  string
	Explain string
	Sample  string // free: образец ответа
	Meta    string // conjugate: подпись (напр. "тип I")
	Say     string // listen: текст для синтеза речи (клиенту не отдаётся)
	Audio   string // listen: имя файла под content/audio/ (когда файл есть)

	Accept      []string   // auto types except conjugate: принимаемые ответы
	Forms       []string   // conjugate: подписи форм (ja/ti/on…)
	AcceptForms [][]string // conjugate: принимаемые ответы по форме

	Options []string    // choice: shown options (Answer stays hidden from clients)
	Answer  string      // choice: the correct option
	Bank    []string    // word_bank: chips to assemble (Accept holds full answers)
	Pairs   [][2]string // match: [Serbian, Russian] pairs (mapping hidden from clients)
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
	Audio    string // optional filename under content/audio/ (set when the file exists)
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
