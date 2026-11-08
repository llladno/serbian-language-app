package mail

import (
	"html"
	htmltmpl "html/template"
	"regexp"
	"strings"
)

// CampaignMail is one study e-mail of the «return» campaign. Heading is plain text; Lead and Box are in the campaign markup (a blank
// line starts a paragraph, a line that starts with «• » is a list item, **double stars** are bold). Both are escaped here. Image is
// the address of the scene across the top, Link where the button leads, Unsubscribe the address of the unsubscribe page.
type CampaignMail struct {
	Subject     string
	Preheader   string
	Heading     string
	Lead        string
	Box         string
	BtnLabel    string
	Image       string
	Link        string
	Unsubscribe string
}

type campaignData struct {
	Image, Link, Unsubscribe string
}

var boldMarks = regexp.MustCompile(`\*\*([^*]+)\*\*`)

// paragraphs turns text in the campaign markup into HTML for a plain (non-markup) field: escaped, a blank line a paragraph break.
// The result is template source, so the double-brace sequence that would open an action is neutralised first.
func paragraphs(s string) string {
	s = strings.ReplaceAll(html.EscapeString(strings.TrimSpace(s)), "{{", "{&#8203;{")
	return strings.ReplaceAll(s, "\n\n", "<br><br>")
}

// rich turns text in the campaign markup into HTML: a blank line starts a paragraph, single line breaks stay line breaks, a line that
// starts with «• » is a list item with a hanging indent, **double stars** are bold. Everything is escaped first.
func rich(s string) string {
	s = strings.ReplaceAll(html.EscapeString(strings.TrimSpace(s)), "{{", "{&#8203;{")
	s = boldMarks.ReplaceAllString(s, `<b style="color:`+brandFG+`;">$1</b>`)
	var out strings.Builder
	for _, block := range strings.Split(s, "\n\n") {
		var plain, items []string
		flushPlain := func() {
			if len(plain) > 0 {
				out.WriteString(`<p style="margin:0 0 14px;">` + strings.Join(plain, "<br>") + `</p>`)
				plain = nil
			}
		}
		flushItems := func() {
			if len(items) > 0 {
				out.WriteString(`<div style="margin:0 0 14px;">`)
				for _, it := range items {
					out.WriteString(`<div style="padding:3px 0 3px 18px;text-indent:-14px;">•&nbsp;&nbsp;` + it + `</div>`)
				}
				out.WriteString(`</div>`)
				items = nil
			}
		}
		for _, line := range strings.Split(block, "\n") {
			if item, ok := strings.CutPrefix(line, "• "); ok {
				flushPlain()
				items = append(items, item)
			} else {
				flushItems()
				plain = append(plain, line)
			}
		}
		flushPlain()
		flushItems()
	}
	return out.String()
}

// RenderCampaign builds the HTML and plain-text bodies of a campaign e-mail on the same branded card as the transactional
// messages: mint band with the wordmark and the swallow, a heading with a lead, one big button, an optional note box, and small
// print with the unsubscribe link. The page reads without the picture (the image carries no information).
func RenderCampaign(m CampaignMail) (subject, text, htmlBody string) {
	src := htmlMsg{
		Preheader: paragraphs(m.Preheader),
		Heading:   paragraphs(m.Heading),
		Lead:      rich(m.Lead), FullImage: true, NoImage: m.Image == "",
		LeadStyle: "margin:0;font-family:" + fontSans + ";font-size:15px;line-height:1.6;color:" + brandMuted + ";text-align:left;",
		BtnLabel:  paragraphs(m.BtnLabel), BtnLink: "{{.Link}}", BtnColor: brandAccent,
		Box:   rich(m.Box),
		Small: `Вы получили это письмо, потому что зарегистрировались в ucimo. <a href="{{.Unsubscribe}}" style="color:` + brandAccent + `;">Отписаться</a>`,
	}.src()
	t := htmltmpl.Must(htmltmpl.New("campaign").Parse(src))
	d := campaignData{Image: m.Image, Link: m.Link, Unsubscribe: m.Unsubscribe}
	htmlBody = exec(t, d)

	var b strings.Builder
	b.WriteString(strings.TrimSpace(m.Heading))
	b.WriteString("\n\n")
	b.WriteString(boldMarks.ReplaceAllString(strings.TrimSpace(m.Lead), "$1"))
	b.WriteString("\n\n")
	if m.Box != "" {
		b.WriteString(boldMarks.ReplaceAllString(strings.TrimSpace(m.Box), "$1"))
		b.WriteString("\n\n")
	}
	b.WriteString(m.BtnLabel + ": " + m.Link + "\n\n— ucimo\n\n")
	b.WriteString("Отписаться от писем об учёбе: " + m.Unsubscribe + "\n")
	return m.Subject, b.String(), htmlBody
}
