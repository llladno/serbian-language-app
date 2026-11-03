package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/grisha/serbian-app/server/internal/economy"
)

// Story tiers. The price of a story depends on its length (docs/STORIES_PLAN.md): three steps, not a free
// number per story. The prices are placeholders to be calibrated against what learners earn between level
// purchases (a short lesson pays 2 зёрнышка, level 1 pays 52 in all).
const (
	StoryTierShort  = "short"
	StoryTierMedium = "medium"
	StoryTierLong   = "long"

	StoryStatusDraft     = "draft"
	StoryStatusPublished = "published"
)

// StoryPrice is the default price in зёрнышки of a paid story by tier. A story that sets its own price
// (StoryContent.Price, decision of 2026-11-03) is sold for that instead: see StoryRow.PriceZ.
var StoryPrice = map[string]int64{StoryTierShort: 10, StoryTierMedium: 20, StoryTierLong: 30}

// StoryReward is what finishing a story pays, once. Stories pay nothing (decision of 2026-10-30): a story is a
// reward in itself, and a payout would make the paid ones a way to earn зёрнышки. The payout code in
// SaveStoryProgress stays and starts working again if this ever returns more than 0.
func StoryReward(tier string, isDemo bool) int64 {
	return 0
}

// StoryEventKinds are the events the player reports.
var StoryEventKinds = map[string]bool{
	"open": true, "pass_start": true, "pass_end": true, "quiz_answer": true,
	"complete": true, "buy_view": true, "replay_sentence": true,
	// The warning before a story that asks the learner to go through the lessons first.
	"lessons_warn_view": true, "lessons_warn_lesson": true, "lessons_warn_continue": true,
}

var (
	// ErrStoryNotFound means no story has that slug.
	ErrStoryNotFound = errors.New("story not found")
	// ErrStoryInvalid wraps every reason a story document is refused.
	ErrStoryInvalid = errors.New("invalid story")
)

// StoryWord is one spoken word with its place in the clip, for highlighting and for hiding the word of a question.
type StoryWord struct {
	W       string `json:"w"`
	StartMS int    `json:"start_ms"`
	EndMS   int    `json:"end_ms"`
}

// StorySentence is one line of the story: the Serbian and the Russian clip with their marks.
type StorySentence struct {
	N       int         `json:"n"`
	Who     string      `json:"who"`
	Scene   int         `json:"scene"`
	SR      string      `json:"sr"`
	RU      string      `json:"ru"`
	SRAudio string      `json:"sr_audio"`
	SRMS    int         `json:"sr_ms"`
	SRWords []StoryWord `json:"sr_words"`
	RUAudio string      `json:"ru_audio"`
	RUMS    int         `json:"ru_ms"`
	RUWords []StoryWord `json:"ru_words"`
}

// StoryScene is one frame; the player cross-fades between scenes with a slow zoom.
type StoryScene struct {
	ID    int    `json:"id"`
	Image string `json:"image"`
}

// StoryChapter is a part of a story. A short story has one; the split is there so audiobooks can reuse the model.
type StoryChapter struct {
	Title     string          `json:"title,omitempty"`
	Scenes    []StoryScene    `json:"scenes"`
	Sentences []StorySentence `json:"sentences"`
}

// StoryInlineQuiz is a quick question on the third pass: the audio pauses after sentence After.
type StoryInlineQuiz struct {
	After   int      `json:"after"`
	Hint    string   `json:"hint"`
	Options []string `json:"options"`
	Answer  string   `json:"answer,omitempty"`
	// Mask is the part of the sentence the player hides while the question is open. It defaults to the answer and,
	// unlike Answer, is sent to the client: the subtitle cannot be blanked without knowing what to blank. The right
	// answer of a quick question is therefore not a secret; a learner who peeks only cheats at listening practice.
	Mask string `json:"mask,omitempty"`
}

// Kinds of the questions after the three passes. The first two are plain choices; the rest copy the lesson tasks:
// listen and pick the translation, who said it, fill the gap, build the sentence from words, match the pairs.
const (
	QuizPlot   = "plot"   // a question about the story, options in Russian
	QuizWord   = "word"   // a Serbian phrase, options are its translation
	QuizListen = "listen" // the clip of a sentence plays, options are its translation
	QuizWho    = "who"    // the clip plays, options are the speakers
	QuizBlank  = "blank"  // a sentence with a gap, options are words
	QuizBank   = "bank"   // build the sentence from word tiles
	QuizMatch  = "match"  // match Serbian phrases and their translations
)

// StoryFinalQuiz is a question after the three passes. Which fields matter depends on Kind; the document is checked
// by StoryContent.Validate and sent to the player by Public (answers removed, tiles and columns shuffled by the client).
type StoryFinalQuiz struct {
	Kind     string      `json:"kind"`
	Prompt   string      `json:"prompt"`
	Options  []string    `json:"options,omitempty"`
	Answer   string      `json:"answer,omitempty"`
	Sentence int         `json:"sentence,omitempty"` // listen, who, blank, bank: the sentence the question is about
	Mask     string      `json:"mask,omitempty"`     // blank: the word taken out (it is the answer)
	Extra    []string    `json:"extra,omitempty"`    // bank: tiles that do not belong to the sentence
	Pairs    [][2]string `json:"pairs,omitempty"`    // match: [Serbian, Russian]

	// Filled in for the player only (Public), never stored.
	Audio string `json:"audio,omitempty"` // listen, who: the clip of the sentence
	// Say is the clip that plays once the question is answered, as the lessons say the right phrase: the sentence the
	// question is about. SayMS is its length, so the player waits for it to end before it goes on.
	Say   string   `json:"say,omitempty"`
	SayMS int      `json:"say_ms,omitempty"`
	Bank  []string `json:"bank,omitempty"`  // bank: the tiles
	Left  []string `json:"left,omitempty"`  // match: the Serbian side
	Right []string `json:"right,omitempty"` // match: the Russian side, in another order
}

// StoryContent is the document stored in stories.content.
type StoryContent struct {
	FastRate float64 `json:"fast_rate"`
	SlowRate float64 `json:"slow_rate"` // the first pass, for a beginner's ear; 1 = normal speed
	// ApproxMinutes is the length shown on the card when the author knows better than FullMS's estimate; 0 = use FullMS.
	ApproxMinutes int `json:"approx_minutes,omitempty"`
	// RecommendedLesson is the lesson of the course by which the learner knows most of the story's words (check_story.py
	// computes it from the lessons); before it the list warns that much may be unclear. Empty = no warning.
	RecommendedLesson string `json:"recommended_lesson,omitempty"`
	// Price is what the story costs in зёрнышки, set per story; 0 = the price of its tier (StoryPrice).
	Price int64 `json:"price,omitempty"`
	// Difficulty is the group of the list the story stands in: 1 easy, 2 medium, 3 harder (0 = not set, shown as medium).
	Difficulty int               `json:"difficulty,omitempty"`
	Chapters   []StoryChapter    `json:"chapters"`
	QuizInline []StoryInlineQuiz `json:"quiz_inline"`
	QuizFinal  []StoryFinalQuiz  `json:"quiz_final"`
}

// Validate refuses a document the player could not play. Answers are the server's secret, so every question must
// carry one that is among its options.
func (c *StoryContent) Validate() error {
	bad := func(format string, a ...any) error {
		return fmt.Errorf("%w: %s", ErrStoryInvalid, fmt.Sprintf(format, a...))
	}
	if c.FastRate == 0 {
		c.FastRate = 1.25
	}
	if c.FastRate < 1 || c.FastRate > 2 {
		return bad("fast_rate %v outside 1..2", c.FastRate)
	}
	if c.SlowRate == 0 {
		c.SlowRate = 1
	}
	if c.SlowRate < 0.6 || c.SlowRate > 1 {
		return bad("slow_rate %v outside 0.6..1", c.SlowRate)
	}
	if c.ApproxMinutes < 0 || c.ApproxMinutes > 60 {
		return bad("approx_minutes %d outside 0..60", c.ApproxMinutes)
	}
	if c.Price < 0 || c.Price > 10000 {
		return bad("price %d outside 0..10000", c.Price)
	}
	if c.Difficulty < 0 || c.Difficulty > 3 {
		return bad("difficulty %d outside 0..3", c.Difficulty)
	}
	if len(c.RecommendedLesson) > 12 || strings.ContainsAny(c.RecommendedLesson, "/ ?#") {
		return bad("recommended_lesson %q is not a lesson id", c.RecommendedLesson)
	}
	if len(c.Chapters) == 0 {
		return bad("no chapters")
	}
	sentences := map[int]string{}
	for ci, ch := range c.Chapters {
		if len(ch.Sentences) == 0 {
			return bad("chapter %d has no sentences", ci+1)
		}
		scenes := map[int]bool{}
		for _, sc := range ch.Scenes {
			if sc.Image == "" {
				return bad("scene %d has no image", sc.ID)
			}
			scenes[sc.ID] = true
		}
		for i, s := range ch.Sentences {
			if s.SR == "" || s.RU == "" || s.SRAudio == "" || s.RUAudio == "" || s.SRMS <= 0 || s.RUMS <= 0 {
				return bad("sentence %d of chapter %d is incomplete", i+1, ci+1)
			}
			if !scenes[s.Scene] {
				return bad("sentence %d refers to missing scene %d", s.N, s.Scene)
			}
			if _, dup := sentences[s.N]; dup {
				return bad("sentence number %d repeats", s.N)
			}
			sentences[s.N] = s.SR
		}
	}
	for i := range c.QuizInline {
		q := &c.QuizInline[i]
		text, ok := sentences[q.After]
		if !ok {
			return bad("quick question after missing sentence %d", q.After)
		}
		if q.Mask == "" {
			q.Mask = q.Answer
		}
		if !strings.Contains(strings.ToLower(text), strings.ToLower(q.Mask)) {
			return bad("quick question after %d: %q is not in the sentence", q.After, q.Mask)
		}
		if n := len(q.Options); n < 2 || n > 3 {
			return bad("quick question after %d needs 2-3 options", q.After)
		}
		if !hasString(q.Options, q.Answer) {
			return bad("quick question after %d: answer is not an option", q.After)
		}
	}
	byN := map[int]StorySentence{}
	for _, ch := range c.Chapters {
		for _, sn := range ch.Sentences {
			byN[sn.N] = sn
		}
	}
	for i := range c.QuizFinal {
		q := &c.QuizFinal[i]
		n := i + 1
		sn, hasSentence := byN[q.Sentence]
		needChoice := func() error {
			if len(q.Options) < 2 || len(q.Options) > 4 || !hasString(q.Options, q.Answer) {
				return bad("final question %d (%s): needs 2-4 options with the answer among them", n, q.Kind)
			}
			return nil
		}
		needSentence := func() error {
			if !hasSentence {
				return bad("final question %d (%s): sentence %d does not exist", n, q.Kind, q.Sentence)
			}
			return nil
		}
		switch q.Kind {
		case QuizPlot, QuizWord:
			if q.Sentence != 0 { // optional here: the sentence whose clip is said after the answer
				if err := needSentence(); err != nil {
					return err
				}
			}
			if err := needChoice(); err != nil {
				return err
			}
		case QuizListen, QuizWho:
			if err := needSentence(); err != nil {
				return err
			}
			if err := needChoice(); err != nil {
				return err
			}
		case QuizBlank:
			if err := needSentence(); err != nil {
				return err
			}
			if q.Mask == "" {
				q.Mask = q.Answer
			}
			if q.Answer == "" {
				q.Answer = q.Mask
			}
			blanked, ok := blankOut(sn.SR, q.Mask)
			if !ok {
				return bad("final question %d (blank): %q is not in the sentence", n, q.Mask)
			}
			if q.Prompt == "" {
				q.Prompt = blanked
			}
			if err := needChoice(); err != nil {
				return err
			}
		case QuizBank:
			if err := needSentence(); err != nil {
				return err
			}
			q.Answer = sn.SR
			if q.Prompt == "" {
				q.Prompt = "Соберите по-сербски: «" + sn.RU + "»"
			}
		case QuizMatch:
			if len(q.Pairs) < 2 || len(q.Pairs) > 5 {
				return bad("final question %d (match): needs 2-5 pairs", n)
			}
			sr, ru := map[string]bool{}, map[string]bool{}
			for _, p := range q.Pairs {
				if p[0] == "" || p[1] == "" || sr[p[0]] || ru[p[1]] {
					return bad("final question %d (match): empty or repeated side", n)
				}
				sr[p[0]], ru[p[1]] = true, true
			}
			if q.Prompt == "" {
				q.Prompt = "Соедините пары"
			}
		default:
			return bad("final question %d: kind %q", n, q.Kind)
		}
	}
	return nil
}

func hasString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// DurationMS is the length of the first pass: the sentences' Serbian clips back to back.
func (c StoryContent) DurationMS() int {
	n := 0
	for _, ch := range c.Chapters {
		for _, s := range ch.Sentences {
			n += s.SRMS
		}
	}
	return n
}

// SentenceCount is how many sentences the story has, for the card in the list ("16 фраз").
func (c StoryContent) SentenceCount() int {
	n := 0
	for _, ch := range c.Chapters {
		n += len(ch.Sentences)
	}
	return n
}

// What the player adds to the clips, in ms: the pause after each clip, the card before passes 2 and 3 (the same
// numbers as web/src/lib/storyPlayer.ts), and a learner's time on a quick or a final question (7 and 12 seconds: «Prvi dan» takes about 5 minutes in all).
const (
	storyGapMS    = 450
	storyCardMS   = 2200
	storyInlineMS = 7000
	storyFinalMS  = 12000
)

// FullMS is the whole time a learner spends on the story, quizzes included: the three passes with their pauses
// and cards, then every quick question of pass 3 and every final question. The list shows it as "≈ N мин".
func (c StoryContent) FullMS() int {
	if c.ApproxMinutes > 0 {
		return c.ApproxMinutes * 60000
	}
	slow, fast := c.SlowRate, c.FastRate
	if slow <= 0 {
		slow = 1
	}
	if fast <= 0 {
		fast = 1.25
	}
	n := 2*storyCardMS + len(c.QuizInline)*storyInlineMS + len(c.QuizFinal)*storyFinalMS
	for _, ch := range c.Chapters {
		for _, s := range ch.Sentences {
			n += int(float64(s.SRMS)/slow) + int(float64(s.SRMS)/fast) + s.SRMS + s.RUMS + 4*storyGapMS
		}
	}
	return n
}

// Public returns a copy with the answers of the final questions removed and the player's fields filled in: the
// clip of a listening question, the tiles of a bank, the two columns of a match (the second in another order). The
// quick questions keep their mask, see StoryInlineQuiz.
func (c StoryContent) Public() StoryContent {
	out := c
	byN := map[int]StorySentence{}
	for _, ch := range c.Chapters {
		for _, sn := range ch.Sentences {
			byN[sn.N] = sn
		}
	}
	out.QuizInline = make([]StoryInlineQuiz, len(c.QuizInline))
	for i, q := range c.QuizInline {
		q.Answer = ""
		out.QuizInline[i] = q
	}
	out.QuizFinal = make([]StoryFinalQuiz, len(c.QuizFinal))
	for i, q := range c.QuizFinal {
		pub := StoryFinalQuiz{Kind: q.Kind, Prompt: q.Prompt, Options: q.Options}
		if sn, ok := byN[q.Sentence]; ok && q.Kind != QuizMatch {
			pub.Say, pub.SayMS = sn.SRAudio, sn.SRMS
		}
		switch q.Kind {
		case QuizListen, QuizWho:
			pub.Audio = byN[q.Sentence].SRAudio
		case QuizBank:
			tiles := append(wordsOf(byN[q.Sentence].SR), q.Extra...)
			sort.Slice(tiles, func(a, b int) bool { return strings.ToLower(tiles[a]) < strings.ToLower(tiles[b]) })
			pub.Bank = tiles
		case QuizMatch:
			for _, p := range q.Pairs {
				pub.Left = append(pub.Left, p[0])
				pub.Right = append(pub.Right, p[1])
			}
			sort.Strings(pub.Right)
		}
		out.QuizFinal[i] = pub
	}
	return out
}

// StoryVerdict is the result of checking one answer: whether it was right and what the right one is.
type StoryVerdict struct {
	Correct bool
	Answer  string
	Pairs   [][2]string // match: the right pairs
}

// Check judges an answer to question index of the given kind ("inline" or "final"). choice is the picked option or
// the sentence built from tiles, pairs the learner's pairs of a match (Serbian -> Russian).
func (c StoryContent) Check(kind string, index int, choice string, pairs map[string]string) (StoryVerdict, bool) {
	switch kind {
	case "inline":
		if index < 0 || index >= len(c.QuizInline) {
			return StoryVerdict{}, false
		}
		q := c.QuizInline[index]
		return StoryVerdict{Correct: choice == q.Answer, Answer: q.Answer}, true
	case "final":
		if index < 0 || index >= len(c.QuizFinal) {
			return StoryVerdict{}, false
		}
		q := c.QuizFinal[index]
		switch q.Kind {
		case QuizBank:
			return StoryVerdict{Correct: normalizeAnswer(choice) == normalizeAnswer(q.Answer), Answer: q.Answer}, true
		case QuizMatch:
			ok := len(pairs) == len(q.Pairs)
			for _, p := range q.Pairs {
				if pairs[p[0]] != p[1] {
					ok = false
				}
			}
			return StoryVerdict{Correct: ok, Pairs: q.Pairs}, true
		default:
			return StoryVerdict{Correct: choice == q.Answer, Answer: q.Answer}, true
		}
	}
	return StoryVerdict{}, false
}

// normalizeAnswer lower-cases, drops punctuation and collapses spaces, so "Zovem se Marko." and "zovem se marko" agree.
func normalizeAnswer(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// wordsOf cuts a sentence into its words, without the punctuation around them.
func wordsOf(sentence string) []string {
	var out []string
	for _, w := range strings.Fields(sentence) {
		w = strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

// blankOut replaces the first occurrence of mask (a word or a few) in a sentence with a gap, keeping punctuation.
func blankOut(sentence, mask string) (string, bool) {
	words := strings.Fields(sentence)
	target := strings.Fields(normalizeAnswer(mask))
	if len(target) == 0 {
		return sentence, false
	}
	for i := 0; i+len(target) <= len(words); i++ {
		match := true
		for k, t := range target {
			if normalizeAnswer(words[i+k]) != t {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		out := append([]string{}, words[:i]...)
		tail := strings.TrimLeftFunc(words[i+len(target)-1], func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) })
		out = append(out, "_____"+tail)
		out = append(out, words[i+len(target):]...)
		return strings.Join(out, " "), true
	}
	return sentence, false
}

// PriceZ is what the story costs in зёрнышки: its own price if it has one, else the price of its tier.
func (r StoryRow) PriceZ() int64 {
	if r.Content.Price > 0 {
		return r.Content.Price
	}
	return StoryPrice[r.Tier]
}

// StoryRow is a story as stored.
type StoryRow struct {
	Slug       string
	TitleSR    string
	TitleRU    string
	Level      string
	Tier       string
	IsDemo     bool
	Status     string
	SortOrder  int
	CoverURL   string
	DurationMS int
	Checks     string
	Content    StoryContent
}

// UpsertStory validates and stores a story, and keeps its shop product in step: a paid story is sold as a
// story_unlock product (ref = slug) at the price of its tier. The demo has no product.
func (s *Store) UpsertStory(r StoryRow, now time.Time) error {
	r.Slug = strings.TrimSpace(r.Slug)
	if r.Slug == "" || strings.ContainsAny(r.Slug, "/ ?#") {
		return fmt.Errorf("%w: bad slug %q", ErrStoryInvalid, r.Slug)
	}
	if r.TitleSR == "" || r.TitleRU == "" {
		return fmt.Errorf("%w: titles are required", ErrStoryInvalid)
	}
	if _, ok := StoryPrice[r.Tier]; !ok {
		return fmt.Errorf("%w: tier %q", ErrStoryInvalid, r.Tier)
	}
	if r.Status != StoryStatusDraft && r.Status != StoryStatusPublished {
		return fmt.Errorf("%w: status %q", ErrStoryInvalid, r.Status)
	}
	if err := r.Content.Validate(); err != nil {
		return err
	}
	doc, err := json.Marshal(r.Content)
	if err != nil {
		return fmt.Errorf("upsert story: marshal: %w", err)
	}
	iso := now.UTC().Format(time.RFC3339)
	demo := 0
	if r.IsDemo {
		demo = 1
	}
	if _, err := s.db.Exec(`INSERT INTO stories
		(slug, title_sr, title_ru, level, tier, is_demo, status, sort_order, cover_url, duration_ms, content, checks, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (slug) DO UPDATE SET
			title_sr = excluded.title_sr, title_ru = excluded.title_ru, level = excluded.level, tier = excluded.tier,
			is_demo = excluded.is_demo, status = excluded.status, sort_order = excluded.sort_order,
			cover_url = excluded.cover_url, duration_ms = excluded.duration_ms, content = excluded.content,
			checks = excluded.checks, updated_at = excluded.updated_at`,
		r.Slug, r.TitleSR, r.TitleRU, r.Level, r.Tier, demo, r.Status, r.SortOrder, r.CoverURL,
		r.Content.DurationMS(), string(doc), r.Checks, iso, iso); err != nil {
		return fmt.Errorf("upsert story %s: %w", r.Slug, err)
	}
	if r.IsDemo {
		// A demo is free for everyone: a shop product left from the time the story was paid would still be on sale.
		if _, err := s.db.Exec(`UPDATE products SET active = 0, updated_at = ? WHERE kind = ? AND ref = ?`,
			iso, economy.ProductStoryUnlock, r.Slug); err != nil {
			return fmt.Errorf("upsert story %s: deactivate product: %w", r.Slug, err)
		}
		return nil
	}
	if _, err := s.UpsertProduct(economy.Product{
		Kind: economy.ProductStoryUnlock, Ref: r.Slug, Title: "История «" + r.TitleSR + "»",
		Description: r.TitleRU, Price: r.PriceZ(), Active: r.Status == StoryStatusPublished,
		SortOrder: 200 + r.SortOrder,
	}, now); err != nil {
		return err
	}
	return nil
}

const storyCols = `slug, title_sr, title_ru, level, tier, is_demo, status, sort_order, cover_url, duration_ms, checks, content`

func scanStory(sc interface{ Scan(...any) error }) (StoryRow, error) {
	var r StoryRow
	var demo int
	var doc string
	if err := sc.Scan(&r.Slug, &r.TitleSR, &r.TitleRU, &r.Level, &r.Tier, &demo, &r.Status, &r.SortOrder,
		&r.CoverURL, &r.DurationMS, &r.Checks, &doc); err != nil {
		return StoryRow{}, err
	}
	r.IsDemo = demo == 1
	if err := json.Unmarshal([]byte(doc), &r.Content); err != nil {
		return StoryRow{}, fmt.Errorf("story %s: bad content: %w", r.Slug, err)
	}
	return r, nil
}

// StoryBySlug loads one story, drafts included (the caller decides who may see a draft).
func (s *Store) StoryBySlug(slug string) (StoryRow, error) {
	r, err := scanStory(s.db.QueryRow(`SELECT `+storyCols+` FROM stories WHERE slug = ?`, slug))
	if errors.Is(err, sql.ErrNoRows) {
		return StoryRow{}, ErrStoryNotFound
	}
	if err != nil {
		return StoryRow{}, fmt.Errorf("story %s: %w", slug, err)
	}
	return r, nil
}

// ListStories returns the stories in display order: the demo first, then by sort order and slug.
func (s *Store) ListStories(publishedOnly bool) ([]StoryRow, error) {
	q := `SELECT ` + storyCols + ` FROM stories`
	if publishedOnly {
		q += ` WHERE status = 'published'`
	}
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("list stories: %w", err)
	}
	defer rows.Close()
	var out []StoryRow
	for rows.Next() {
		r, err := scanStory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsDemo != out[j].IsDemo {
			return out[i].IsDemo
		}
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Slug < out[j].Slug
	})
	return out, nil
}

// StoryProgress is how far a learner got in one story.
type StoryProgress struct {
	PassReached int
	Completed   bool
	QuizScore   int
	QuizTotal   int
}

// StoryProgressAll returns the learner's progress for every story they have touched, by slug.
func (u *UserStore) StoryProgressAll() (map[string]StoryProgress, error) {
	rows, err := u.db.Query(`SELECT slug, pass_reached, completed_at, quiz_score, quiz_total FROM story_progress WHERE user_id = ?`, u.user)
	if err != nil {
		return nil, fmt.Errorf("story progress: %w", err)
	}
	defer rows.Close()
	out := map[string]StoryProgress{}
	for rows.Next() {
		var slug, done string
		var p StoryProgress
		if err := rows.Scan(&slug, &p.PassReached, &done, &p.QuizScore, &p.QuizTotal); err != nil {
			return nil, err
		}
		p.Completed = done != ""
		out[slug] = p
	}
	return out, rows.Err()
}

// StoryOwned reports whether the learner may open the story: the demo is open to all, a paid one needs the
// story_unlock entitlement.
func (u *UserStore) StoryOwned(r StoryRow) (bool, error) {
	if r.IsDemo {
		return true, nil
	}
	owned, err := u.Entitlements()
	if err != nil {
		return false, err
	}
	return owned[economy.ProductStoryUnlock+":"+r.Slug] > 0, nil
}

// StoryProductID is the shop product id of a paid story.
func (s *Store) StoryProductID(slug string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM products WHERE kind = ? AND ref = ?`, economy.ProductStoryUnlock, slug).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrProductUnavailable
	}
	if err != nil {
		return 0, fmt.Errorf("story product: %w", err)
	}
	return id, nil
}

// SaveStoryProgress records a finished pass or the whole story. The furthest pass only grows. Finishing pays the
// reward once per learner and story (the ledger key makes a retry harmless) and returns what was paid, 0 for a
// repeat. The score is kept at the best result.
func (u *UserStore) SaveStoryProgress(r StoryRow, pass int, completed bool, score, total int, now time.Time) (int64, error) {
	iso := now.UTC().Format(time.RFC3339)
	tx, err := u.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("story progress: begin: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO story_progress (user_id, slug, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (user_id, slug) DO NOTHING`, u.user, r.Slug, iso); err != nil {
		return 0, fmt.Errorf("story progress: row: %w", err)
	}
	if _, err := tx.Exec(`UPDATE story_progress SET pass_reached = CASE WHEN pass_reached < ? THEN ? ELSE pass_reached END,
		updated_at = ? WHERE user_id = ? AND slug = ?`, pass, pass, iso, u.user, r.Slug); err != nil {
		return 0, fmt.Errorf("story progress: pass: %w", err)
	}
	var paid int64
	if completed {
		if _, err := tx.Exec(`UPDATE story_progress SET
			completed_at = CASE WHEN completed_at = '' THEN ? ELSE completed_at END,
			quiz_score = CASE WHEN ? >= quiz_score THEN ? ELSE quiz_score END,
			quiz_total = ?
			WHERE user_id = ? AND slug = ?`, iso, score, score, total, u.user, r.Slug); err != nil {
			return 0, fmt.Errorf("story progress: complete: %w", err)
		}
		if reward := StoryReward(r.Tier, r.IsDemo); reward > 0 {
			_, err := addLedgerEntryTx(tx, LedgerEntry{
				UserID: u.user, Amount: reward, Kind: "story_reward", Ref: r.Slug,
				IdempotencyKey: "story:" + u.user + ":" + r.Slug, Comment: "story finished",
			}, now)
			switch {
			case err == nil:
				paid = reward
				if _, err := tx.Exec(`UPDATE story_progress SET reward_paid = ? WHERE user_id = ? AND slug = ?`,
					reward, u.user, r.Slug); err != nil {
					return 0, fmt.Errorf("story progress: reward: %w", err)
				}
			case errors.Is(err, ErrDuplicateEntry):
			default:
				return 0, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("story progress: commit: %w", err)
	}
	return paid, nil
}

// RecordStoryEvent appends one raw event of the stories metrics. An unknown kind is refused.
func (s *Store) RecordStoryEvent(userID, slug, kind, detail string, now time.Time) error {
	if !StoryEventKinds[kind] {
		return fmt.Errorf("%w: event kind %q", ErrStoryInvalid, kind)
	}
	if len(detail) > 200 {
		detail = detail[:200]
	}
	if _, err := s.db.Exec(`INSERT INTO story_events (user_id, slug, kind, detail, created_at) VALUES (?, ?, ?, ?, ?)`,
		userID, slug, kind, detail, now.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("story event: %w", err)
	}
	return nil
}
