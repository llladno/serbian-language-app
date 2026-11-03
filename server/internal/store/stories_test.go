package store

import (
	"strings"
	"testing"
	"time"
)

func TestStoryFullMSCountsThreePassesAndQuestions(t *testing.T) {
	c := StoryContent{
		SlowRate: 0.5, FastRate: 2,
		Chapters:   []StoryChapter{{Sentences: []StorySentence{{SRMS: 1000, RUMS: 800}}}},
		QuizInline: []StoryInlineQuiz{{}, {}},
		QuizFinal:  []StoryFinalQuiz{{}},
	}
	// pass 1: 1000/0.5 = 2000; pass 2: 1000+800; pass 3: 1000/2 = 500; four pauses; two cards; 2 quick + 1 final.
	want := 2000 + 1800 + 500 + 4*storyGapMS + 2*storyCardMS + 2*storyInlineMS + storyFinalMS
	if got := c.FullMS(); got != want {
		t.Fatalf("FullMS = %d, want %d", got, want)
	}
}

func TestStoryApproxMinutesOverridesTheEstimate(t *testing.T) {
	c := StoryContent{ApproxMinutes: 3, Chapters: []StoryChapter{{Sentences: []StorySentence{{SRMS: 1000, RUMS: 800}}}}}
	if got := c.FullMS(); got != 180000 {
		t.Fatalf("FullMS = %d, want 180000", got)
	}
}

func TestStoryRecommendedLessonMustLookLikeALessonId(t *testing.T) {
	for _, id := range []string{"a b", "x/y", "0123456789012"} {
		c := StoryContent{RecommendedLesson: id}
		if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "recommended_lesson") {
			t.Errorf("RecommendedLesson %q: err = %v, want a recommended_lesson error", id, err)
		}
	}
	ok := StoryContent{RecommendedLesson: "04b"}
	if err := ok.Validate(); err != nil && strings.Contains(err.Error(), "recommended_lesson") {
		t.Errorf("04b refused: %v", err)
	}
}

func TestStoryDifficultyIsZeroToThree(t *testing.T) {
	for _, d := range []int{-1, 4} {
		c := StoryContent{Difficulty: d}
		if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "difficulty") {
			t.Errorf("Difficulty %d: err = %v, want a difficulty error", d, err)
		}
	}
	for _, d := range []int{0, 1, 2, 3} {
		c := StoryContent{Difficulty: d}
		if err := c.Validate(); err != nil && strings.Contains(err.Error(), "difficulty") {
			t.Errorf("Difficulty %d refused: %v", d, err)
		}
	}
}

func TestAStoryMadeFreeStopsBeingSold(t *testing.T) {
	st := newStore(t)
	now := time.Now()
	row := StoryRow{Slug: "early", TitleSR: "Rano", TitleRU: "Рано", Level: "1", Tier: StoryTierShort, Status: StoryStatusPublished,
		Content: StoryContent{Chapters: []StoryChapter{{Scenes: []StoryScene{{ID: 1, Image: "/x.webp"}}, Sentences: []StorySentence{
			{N: 1, Who: "A", Scene: 1, SR: "Zdravo!", RU: "Привет!", SRAudio: "/a.mp3", RUAudio: "/b.mp3", SRMS: 500, RUMS: 500}}}}}}
	if err := st.UpsertStory(row, now); err != nil {
		t.Skipf("the minimal story document is not valid here: %v", err)
	}
	if got, _ := st.DebugQuery("SELECT active FROM products WHERE kind = 'story_unlock' AND ref = 'early'"); len(got) != 1 || got[0] != "1" {
		t.Fatalf("paid story product = %v, want one active row", got)
	}
	row.IsDemo = true
	if err := st.UpsertStory(row, now); err != nil {
		t.Fatal(err)
	}
	if got, _ := st.DebugQuery("SELECT active FROM products WHERE kind = 'story_unlock' AND ref = 'early'"); len(got) != 1 || got[0] != "0" {
		t.Fatalf("after becoming a demo the product = %v, want one inactive row", got)
	}
}
