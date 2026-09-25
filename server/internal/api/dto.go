package api

import "github.com/grisha/serbian-app/server/internal/checker"

// ---- course / lessons ----

type courseDTO struct {
	Title   string         `json:"title"`
	Phases  []phaseDTO     `json:"phases"`
	Lessons []lessonRefDTO `json:"lessons"`
}

type phaseDTO struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Lessons []string `json:"lessons"`
}

type lessonRefDTO struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Planned  bool   `json:"planned"`
	Status   string `json:"status"` // not_started | in_progress | done
}

type lessonDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle"`
	Planned   bool      `json:"planned"`
	Markdown  string    `json:"markdown"`
	Reading   string    `json:"reading,omitempty"`
	ReadingRU string    `json:"reading_ru,omitempty"`
	Status    string    `json:"status"`
	Steps     []stepDTO `json:"steps,omitempty"`
}

// stepDTO carries a step's metadata only — its exercises are fetched from the
// exercises endpoint, which strips answers.
type stepDTO struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	Markdown    string   `json:"markdown,omitempty"`
	MarkdownRU  string   `json:"markdown_ru,omitempty"`
	ExerciseIDs []string `json:"exercise_ids,omitempty"`
	Status      string   `json:"status"` // not_started | in_progress | done

	Scene string    `json:"scene,omitempty"` // dialogue: Russian setup line
	Voice string    `json:"voice,omitempty"` // dialogue: f | m
	Turns []turnDTO `json:"turns,omitempty"`
}

// turnDTO is one dialogue line. For a "me" turn the line itself (sr/ru/audio)
// travels only once the user has an attempt on its exercise — otherwise the
// correct answer would be readable straight from the page source.
type turnDTO struct {
	Who        string `json:"who"`
	SR         string `json:"sr,omitempty"`
	RU         string `json:"ru,omitempty"`
	Audio      string `json:"audio,omitempty"`
	ExerciseID string `json:"exercise_id,omitempty"`
}

// ---- exercises (accept lists intentionally omitted) ----

type exerciseBlockDTO struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Instruction string        `json:"instruction,omitempty"`
	Exercises   []exerciseDTO `json:"exercises"`
}

type exerciseDTO struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Prompt  string   `json:"prompt"`
	Forms   []string `json:"forms,omitempty"`
	Meta    string   `json:"meta,omitempty"`
	Audio   string   `json:"audio,omitempty"`   // listen: clip under /audio/
	Options []string `json:"options,omitempty"` // choice (correct answer omitted)
	Bank    []string `json:"bank,omitempty"`    // word_bank chips (accept omitted)
	Left    []string `json:"left,omitempty"`    // match: left column
	Right   []string `json:"right,omitempty"`   // match: right column (client shuffles)
}

type attemptDTO struct {
	Answer  string `json:"answer"`
	Correct bool   `json:"correct"`
}

type checkResultDTO struct {
	OK       bool             `json:"ok"`
	Diff     []checker.Chunk  `json:"diff,omitempty"`
	Expected string           `json:"expected,omitempty"`
	Explain  string           `json:"explain,omitempty"`
	Sample   string           `json:"sample,omitempty"`
	Forms    []checker.Result `json:"forms,omitempty"`
	Match    map[string]bool  `json:"match,omitempty"` // match: per-pair correctness
	NearMiss bool             `json:"near_miss,omitempty"`

	// dialogue: the canonical line this answer produces, so the chat can show
	// a bubble without a second request — sent whether the answer was right
	// or wrong.
	Line      string `json:"line,omitempty"`
	LineRU    string `json:"line_ru,omitempty"`
	LineAudio string `json:"line_audio,omitempty"`
}

// ---- auth / profile ----

// sessionUserDTO is the account view returned by the auth and profile
// endpoints (GET /api/auth/session, GET /api/me).
type sessionUserDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Telegram      struct {
		Linked   bool   `json:"linked"`
		Username string `json:"username"`
	} `json:"telegram"`
}

// deviceDTO is one active session in the profile "devices" list. ID is an
// opaque handle (first 12 chars of the token hash) for DELETE /api/me/sessions/{id}.
type deviceDTO struct {
	ID         string `json:"id"`
	UserAgent  string `json:"user_agent"`
	LastSeenAt string `json:"last_seen_at"`
	Current    bool   `json:"current"`
}

// meDTO is the full profile payload: the account plus its active sessions.
type meDTO struct {
	sessionUserDTO
	Sessions []deviceDTO `json:"sessions"`
}

// ---- vocab / false friends ----

type vocabDTO struct {
	ID            string   `json:"id"`
	Latin         string   `json:"latin"`
	Cyrillic      string   `json:"cyrillic"`
	Transcription string   `json:"transcription,omitempty"`
	RU            string   `json:"ru"`
	Note          string   `json:"note,omitempty"`
	Lesson        string   `json:"lesson,omitempty"`
	POS           string   `json:"pos,omitempty"`
	Gender        string   `json:"gender,omitempty"`
	Aspect        string   `json:"aspect,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Emoji         string   `json:"emoji,omitempty"`
	Image         string   `json:"image,omitempty"`
	Audio         string   `json:"audio,omitempty"`
	ExampleSR     string   `json:"example_sr,omitempty"`
	ExampleRU     string   `json:"example_ru,omitempty"`
}

type falseFriendDTO struct {
	ID            string `json:"id"`
	SR            string `json:"sr"`
	Transcription string `json:"transcription,omitempty"`
	Means         string `json:"means"`
	Not           string `json:"not,omitempty"`
	Correct       string `json:"correct,omitempty"`
	Group         string `json:"group"`
	Emoji         string `json:"emoji,omitempty"`
	Image         string `json:"image,omitempty"`
}

// ---- review ----

type reviewCardDTO struct {
	CardID        string `json:"card_id"`
	Kind          string `json:"kind"` // vocab | ff | gram
	Front         string `json:"front"`
	Cyrillic      string `json:"cyrillic,omitempty"`
	Transcription string `json:"transcription,omitempty"`
	Back          string `json:"back"`
	Note          string `json:"note,omitempty"`
	State         string `json:"state"`
	Emoji         string `json:"emoji,omitempty"`
	Image         string `json:"image,omitempty"`
	Audio         string `json:"audio,omitempty"`
	ExampleSR     string `json:"example_sr,omitempty"`
	ExampleRU     string `json:"example_ru,omitempty"`
	// Options carries 4 shuffled candidate translations (Back values, one of
	// them correct) for a first-encounter recognition quiz. Only set for
	// state == "new" — an already-started card uses the plain flip+grade UI.
	Options []string       `json:"options,omitempty"`
	Preview map[string]int `json:"preview"` // grade name -> next interval in days
	// ItemPrompt/ItemIndex are set only for kind == "gram": one item was
	// picked at random from the card's Items, and this is the question to
	// ask ("ti (ты)") plus the index the client echoes back to
	// POST /api/review/grade-gram so the server knows which item's accept
	// list to check against — the accept list itself never reaches the
	// client.
	ItemPrompt string `json:"item_prompt,omitempty"`
	// ItemIndex has no omitempty: 0 (the first item) is a normal value and
	// must round-trip, not vanish — callers should gate on Kind == "gram"
	// to know whether this field means anything, not on its zero value.
	ItemIndex int `json:"item_index"`
}

type gradeResultDTO struct {
	Due          string `json:"due"`
	IntervalDays int    `json:"interval_days"`
	State        string `json:"state"`
}

// gramCheckResultDTO is the response to POST /api/review/grade-gram: the
// answer's correctness (checker.Result, same shape a lesson exercise check
// returns) plus the SM-2 outcome (gradeResultDTO) — checking and grading a
// grammar item happen in the same call, since the grade is derived from
// correctness rather than self-reported.
type gramCheckResultDTO struct {
	OK       bool   `json:"ok"`
	Expected string `json:"expected,omitempty"`
	NearMiss bool   `json:"near_miss,omitempty"`

	Due          string `json:"due"`
	IntervalDays int    `json:"interval_days"`
	State        string `json:"state"`
}

// ---- progress ----

type progressDTO struct {
	Phases        []phaseProgressDTO `json:"phases"`
	SRS           srsProgressDTO     `json:"srs"`
	WeakExercises []weakExerciseDTO  `json:"weak_exercises"`
	StreakDays    int                `json:"streak_days"`
	RecentLessons []recentLessonDTO  `json:"recent_lessons"`
	Activity      []dayActivityDTO   `json:"activity"`
	DailyGoal     int                `json:"daily_goal"`
}

type dayActivityDTO struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type leaderRowDTO struct {
	Name          string `json:"name"`
	LessonsDone   int    `json:"lessons_done"`
	LessonsTotal  int    `json:"lessons_total"`
	CardsKnown    int    `json:"cards_known"`
	TotalCards    int    `json:"total_cards"`
	StreakDays    int    `json:"streak_days"`
	ReviewedToday int    `json:"reviewed_today"`
	LastActive    string `json:"last_active"`
}

type phaseProgressDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

type srsProgressDTO struct {
	DueToday      int `json:"due_today"`
	NewAvailable  int `json:"new_available"`
	ReviewedToday int `json:"reviewed_today"`
	TotalCards    int `json:"total_cards"`
	Known         int `json:"known"`
}

type weakExerciseDTO struct {
	ExerciseID string `json:"exercise_id"`
	Lesson     string `json:"lesson"`
	Prompt     string `json:"prompt"`
	Wrong      int    `json:"wrong"`
	Total      int    `json:"total"`
}

type recentLessonDTO struct {
	Lesson string `json:"lesson"`
	Title  string `json:"title"`
	Status string `json:"status"`
}
