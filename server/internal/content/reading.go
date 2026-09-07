package content

import "strings"

const (
	readingOpen  = "<!-- reading -->"
	readingClose = "<!-- /reading -->"
)

// extractReading pulls an optional reading block out of a lesson's markdown.
//
// The block is delimited by <!-- reading --> / <!-- /reading --> markers, each
// on its own line. Inside, an optional line that is just "---" separates the
// Serbian text (sr) from its Russian translation (ru). The returned body is the
// markdown with the whole block removed.
func extractReading(md string) (body, sr, ru string) {
	lines := strings.Split(md, "\n")
	start, end := -1, -1
	for i, l := range lines {
		switch strings.TrimSpace(l) {
		case readingOpen:
			if start == -1 {
				start = i
			}
		case readingClose:
			if start != -1 && end == -1 {
				end = i
			}
		}
	}
	if start == -1 || end == -1 || end < start {
		return strings.TrimSpace(md), "", ""
	}

	inner := strings.Join(lines[start+1:end], "\n")
	rest := append(append([]string{}, lines[:start]...), lines[end+1:]...)
	body = collapseBlankLines(strings.TrimSpace(strings.Join(rest, "\n")))

	sr, ru = splitOnHR(inner)
	return body, strings.TrimSpace(sr), strings.TrimSpace(ru)
}

// collapseBlankLines turns runs of 3+ newlines (left where a block was cut out)
// into a single blank line.
func collapseBlankLines(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}

// splitOnHR splits text at the first line that is exactly "---".
func splitOnHR(s string) (before, after string) {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) == "---" {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i+1:], "\n")
		}
	}
	return s, ""
}
