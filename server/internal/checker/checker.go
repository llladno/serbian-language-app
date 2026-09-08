// Package checker normalizes and grades free-text exercise answers.
package checker

import (
	"math"
	"strings"
)

// Chunk is one token of a graded answer. OK=false means "highlight as wrong".
type Chunk struct {
	Text string `json:"text"`
	OK   bool   `json:"ok"`
}

// Result is the outcome of checking one answer.
type Result struct {
	OK       bool    `json:"ok"`
	Expected string  `json:"expected"`
	Diff     []Chunk `json:"diff"`
	// NearMiss is true when a wrong answer is within a couple of characters
	// of the closest accepted variant — likely a typo or a missing diacritic.
	NearMiss bool `json:"near_miss,omitempty"`
}

var punct = map[rune]bool{
	'.': true, '!': true, '?': true, ',': true, ';': true, ':': true,
	'"': true, '\'': true, '(': true, ')': true,
	'“': true, '”': true, '„': true, '‟': true, '«': true, '»': true,
	'…': true,
}

// Normalize lower-cases, drops punctuation and quotes, unifies dashes and
// collapses whitespace so cosmetic differences don't fail an answer.
func Normalize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case punct[r]:
			b.WriteRune(' ')
		case r == '–' || r == '—' || r == '-':
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func normTokens(s string) []string {
	n := Normalize(s)
	if n == "" {
		return nil
	}
	return strings.Fields(n)
}

// Check grades answer against the accepted variants.
func Check(answer string, accept []string) Result {
	na := Normalize(answer)
	for _, a := range accept {
		if Normalize(a) == na {
			return Result{OK: true, Expected: a}
		}
	}
	best := ""
	bestDist := math.MaxInt
	at := strings.Fields(na)
	for _, a := range accept {
		d := levenshtein(at, normTokens(a))
		if d < bestDist {
			bestDist = d
			best = a
		}
	}
	nearMiss := false
	if na != "" && best != "" {
		if d := levenshteinRunes([]rune(na), []rune(Normalize(best))); d > 0 && d <= 2 {
			nearMiss = true
		}
	}
	return Result{OK: false, Expected: best, Diff: diffChunks(at, normTokens(best)), NearMiss: nearMiss}
}

// CheckChoice grades a single-choice answer against the one correct option.
func CheckChoice(answer, correct string) Result {
	return Result{OK: Normalize(answer) == Normalize(correct), Expected: correct}
}

// CheckMatch grades a pair-matching answer. got maps each left item to the
// right item the learner picked. ok is true only when every pair is right and
// every pair was answered; per reports each left item individually.
func CheckMatch(got map[string]string, pairs [][2]string) (bool, map[string]bool) {
	per := make(map[string]bool, len(pairs))
	all := len(got) == len(pairs)
	for _, p := range pairs {
		ok := Normalize(got[p[0]]) == Normalize(p[1])
		per[p[0]] = ok
		if !ok {
			all = false
		}
	}
	return all, per
}

// CheckForms grades a conjugation exercise field-by-field.
func CheckForms(answers []string, acceptForms [][]string) []Result {
	out := make([]Result, len(acceptForms))
	for i, accept := range acceptForms {
		ans := ""
		if i < len(answers) {
			ans = answers[i]
		}
		out[i] = Check(ans, accept)
	}
	return out
}

// diffChunks marks each answer token as matched (in the LCS with expected)
// or not.
func diffChunks(a, e []string) []Chunk {
	matched := lcsMatched(a, e)
	out := make([]Chunk, len(a))
	for i, w := range a {
		out[i] = Chunk{Text: w, OK: matched[i]}
	}
	return out
}

func lcsMatched(a, e []string) []bool {
	n, m := len(a), len(e)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == e[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	matched := make([]bool, n)
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == e[j] {
			matched[i] = true
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			i++
		} else {
			j++
		}
	}
	return matched
}

func levenshtein(a, b []string) int {
	n, m := len(a), len(b)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}
	prev := make([]int, m+1)
	for j := 0; j <= m; j++ {
		prev[j] = j
	}
	for i := 1; i <= n; i++ {
		cur := make([]int, m+1)
		cur[0] = i
		for j := 1; j <= m; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[m]
}

func levenshteinRunes(a, b []rune) int {
	n, m := len(a), len(b)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}
	prev := make([]int, m+1)
	for j := 0; j <= m; j++ {
		prev[j] = j
	}
	for i := 1; i <= n; i++ {
		cur := make([]int, m+1)
		cur[0] = i
		for j := 1; j <= m; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[m]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
