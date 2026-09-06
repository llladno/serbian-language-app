package api

import "github.com/grisha/serbian-app/server/internal/checker"

// ---- course / lessons ----

type courseDTO struct {
	Title  string      `json:"title"`
	Phases []phaseDTO  `json:"phases"`
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
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Planned  bool   `json:"planned"`
	Markdown string `json:"markdown"`
	Status   string `json:"status"`
}

// ---- exercises (accept lists intentionally omitted) ----

type exerciseBlockDTO struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Instruction string        `json:"instruction,omitempty"`
	Exercises   []exerciseDTO `json:"exercises"`
}

type exerciseDTO struct {
	ID     string   `json:"id"`
	Type   string   `json:"type"`
	Prompt string   `json:"prompt"`
	Forms  []string `json:"forms,omitempty"`
	Meta   string   `json:"meta,omitempty"`
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
	NearMiss bool             `json:"near_miss,omitempty"`
}

// ---- vocab / false friends ----

type vocabDTO struct {
	ID       string   `json:"id"`
	Latin    string   `json:"latin"`
	Cyrillic string   `json:"cyrillic"`
	RU       string   `json:"ru"`
	Note     string   `json:"note,omitempty"`
	Lesson   string   `json:"lesson,omitempty"`
	POS      string   `json:"pos,omitempty"`
	Gender   string   `json:"gender,omitempty"`
	Aspect   string   `json:"aspect,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

type falseFriendDTO struct {
	ID      string `json:"id"`
	SR      string `json:"sr"`
	Means   string `json:"means"`
	Not     string `json:"not,omitempty"`
	Correct string `json:"correct,omitempty"`
	Group   string `json:"group"`
}

// ---- review ----

type reviewCardDTO struct {
	CardID   string         `json:"card_id"`
	Kind     string         `json:"kind"` // vocab | ff
	Front    string         `json:"front"`
	Cyrillic string         `json:"cyrillic,omitempty"`
	Back     string         `json:"back"`
	Note     string         `json:"note,omitempty"`
	State    string         `json:"state"`
	Preview  map[string]int `json:"preview"` // grade name -> next interval in days
}

type gradeResultDTO struct {
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
