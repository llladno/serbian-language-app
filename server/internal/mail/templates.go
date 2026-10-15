package mail

import (
	"bytes"
	htmltmpl "html/template"
	"io"
	"net/url"
	texttmpl "text/template"
)

// Subjects for the four transactional messages.
const (
	subjectVerify            = "Подтверди почту"
	subjectReset             = "Сброс пароля"
	subjectAlreadyRegistered = "Кто-то регистрируется с этой почтой"
	subjectPasswordChanged   = "Пароль изменён"
)

// Brand tokens shared by every HTML template below, lifted from the app's
// default (teal) palette in web/src/style.css so transactional mail matches
// the product instead of looking like a generic system notice.
const (
	brandBG     = "#f7f9f7"
	brandPage   = "#eef4f2" // page background behind the card
	brandHero   = "#d8eeeb" // mint hero band
	brandSubtle = "#8a9496" // small print
	brandCard   = "#ffffff"
	brandBorder = "#e2f1ed"
	brandFG     = "#172326"
	brandMuted  = "#546063"
	brandAccent = "#176b68"
	brandDanger = "#d95555"
	fontSans    = "Manrope,-apple-system,'Segoe UI',Roboto,Arial,sans-serif"
	fontSerif   = "Lora,Georgia,'Times New Roman',serif"
)

// Mascot images. Each message gets its own pose of the "ucimo" pixel-art
// swallow, served by the landing site's static /mascot/ folder (same domain
// as baseURL, see landing/app/pages/index.vue for the existing convention).
const (
	mascotGreeting = "/mascot/05-greeting.webp" // verify: welcoming a new user
	mascotKey      = "/mascot/13-key.webp"      // reset: handing over a fresh key
	mascotSitting  = "/mascot/11-sitting.webp"  // already-registered: calm, unbothered — "no worries"
	mascotShield   = "/mascot/14-shield.webp"   // password-changed: reassuring, protective
)

// RenderVerify builds the email-verification message. baseURL has no trailing
// slash; the verify link points at the backend endpoint from part 3.
func RenderVerify(baseURL, token, name string) (subject, text, html string) {
	d := linkData{Name: name, Link: baseURL + "/api/auth/verify?token=" + url.QueryEscape(token), Image: baseURL + mascotGreeting}
	return subjectVerify, exec(verifyText, d), exec(verifyHTML, d)
}

// RenderReset builds the password-reset message. The reset link points at the
// front-end route from part 3.
func RenderReset(baseURL, token, name string) (subject, text, html string) {
	d := linkData{Name: name, Link: baseURL + "/reset?token=" + url.QueryEscape(token), Image: baseURL + mascotKey}
	return subjectReset, exec(resetText, d), exec(resetHTML, d)
}

// RenderAlreadyRegistered builds the message sent when someone tries to
// register with an address that already has an account. It never states
// outright that an account exists; it uses "если это был ты…" framing and
// points at the login and password-recovery routes.
func RenderAlreadyRegistered(baseURL, name string) (subject, text, html string) {
	d := accountData{Name: name, LoginLink: baseURL + "/login", ForgotLink: baseURL + "/forgot", Image: baseURL + mascotSitting}
	return subjectAlreadyRegistered, exec(alreadyText, d), exec(alreadyHTML, d)
}

// RenderPasswordChanged builds the security notice sent after a successful
// password change.
func RenderPasswordChanged(baseURL, name string) (subject, text, html string) {
	d := accountData{Name: name, ForgotLink: baseURL + "/forgot", Image: baseURL + mascotShield}
	return subjectPasswordChanged, exec(passwordChangedText, d), exec(passwordChangedHTML, d)
}

// linkData feeds the templates that carry a single tokenised action link.
type linkData struct {
	Name  string
	Link  string
	Image string
}

// accountData feeds the templates that link to login / password recovery.
type accountData struct {
	Name       string
	LoginLink  string
	ForgotLink string
	Image      string
}

// executor is the common subset of *text/template.Template and
// *html/template.Template used here.
type executor interface {
	Execute(wr io.Writer, data any) error
}

// exec runs a template that is a package-level constant. A failure means the
// template source is broken, which is a programmer error caught by tests.
func exec(t executor, data any) string {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		panic("mail: render template: " + err.Error())
	}
	return buf.String()
}

// Template sources. Comments in code are English; user-facing copy is Russian,
// friendly and short. Each HTML message is a single branded card (teal accent,
// serif "ucimo" wordmark, Manrope body — the app's own default palette, not
// its pink/blue alternates) with one styled anchor button plus a plain-text
// fallback link, built on a table skeleton so it survives Outlook/Gmail.

const verifyTextSrc = `Привет, {{.Name}}!

Осталось подтвердить почту — открой ссылку:
{{.Link}}

Если ты не регистрировался, просто удали это письмо.

— ucimo
`

const resetTextSrc = `Привет, {{.Name}}!

Ты просил сбросить пароль. Задай новый по ссылке:
{{.Link}}

Ссылка скоро перестанет работать. Если это был не ты — ничего делать не нужно.

— ucimo
`

const alreadyTextSrc = `Привет, {{.Name}}!

Кто-то только что попробовал зарегистрироваться с этой почтой.

Если это был ты — возможно, ты уже заходил сюда раньше. Просто войди:
{{.LoginLink}}

Не помнишь пароль — восстанови доступ:
{{.ForgotLink}}

Если это был не ты, просто удали это письмо.

— ucimo
`

const passwordChangedTextSrc = `Привет, {{.Name}}!

Пароль от твоего аккаунта только что изменили.

Если это был ты — всё в порядке, делать ничего не нужно.

Если это был не ты — сразу восстанови доступ:
{{.ForgotLink}}

— ucimo
`

// Shared HTML layout for every message: mint hero with the wordmark and the
// mascot, a heading with a short lead, one big button, an optional note box and
// a small-print footer. Arguments are template sources (they may reference
// {{.Name}}, {{.Link}} ...). Table skeleton + inline styles so it survives
// Outlook/Gmail.
type htmlMsg struct {
	Preheader string
	Heading   string
	Lead      string
	BtnLabel  string
	BtnLink   string
	BtnColor  string
	Box       string // optional note box content (HTML), "" to skip
	Small     string // small print under the card (HTML)
}

func (m htmlMsg) src() string {
	const f = "font-family:" + fontSans + ";"
	box := ""
	if m.Box != "" {
		box = `<tr><td style="padding:24px 32px 0;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:` + brandBG + `;border-radius:16px;"><tr><td style="padding:16px 20px;` + f + `font-size:14px;line-height:1.5;color:` + brandMuted + `;">` + m.Box + `</td></tr></table>
</td></tr>
`
	}
	return `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
</head>
<body style="margin:0;padding:0;background:` + brandPage + `;">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;mso-hide:all;">` + m.Preheader + `&#8203;</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:` + brandPage + `;">
<tr><td align="center" style="padding:36px 16px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:480px;width:100%;background:` + brandCard + `;border-radius:24px;overflow:hidden;">
<tr><td align="center" style="background:` + brandHero + `;padding:28px 24px;">
<div style="font-family:` + fontSerif + `;font-size:20px;font-weight:600;color:` + brandAccent + `;letter-spacing:0.04em;padding-bottom:20px;">ucimo</div>
<img src="{{.Image}}" width="130" alt="" style="display:block;width:130px;max-width:130px;height:auto;margin:0 auto;padding:6px;">
</td></tr>
<tr><td style="padding:32px 32px 0;">
<h1 style="margin:0 0 10px;` + f + `font-size:24px;line-height:1.25;font-weight:800;color:` + brandFG + `;text-align:center;">` + m.Heading + `</h1>
<p style="margin:0;` + f + `font-size:15px;line-height:1.55;color:` + brandMuted + `;text-align:center;">` + m.Lead + `</p>
</td></tr>
<tr><td align="center" style="padding:24px 32px 0;">
<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="border-radius:14px;background:` + m.BtnColor + `;">
<a href="` + m.BtnLink + `" style="display:inline-block;padding:15px 34px;` + f + `font-size:16px;font-weight:700;color:#ffffff;text-decoration:none;border-radius:14px;">` + m.BtnLabel + `</a>
</td></tr></table>
</td></tr>
` + box + `<tr><td style="padding:20px 32px 28px;` + f + `font-size:12px;line-height:1.5;color:` + brandSubtle + `;text-align:center;">` + m.Small + `</td></tr>
</table>
<p style="margin:18px 0 0;` + f + `font-size:12px;color:` + brandSubtle + `;">ucimo · сербский с нуля</p>
</td></tr>
</table>
</body>
</html>
`
}

const (
	linkFallback = `Кнопка не работает? Скопируй ссылку в браузер:<br><a href="{{.Link}}" style="color:` + brandAccent + `;word-break:break-all;">{{.Link}}</a>`
)

var (
	verifyHTMLSrc = htmlMsg{
		Preheader: "Одна ссылка — и можно начинать учить сербский",
		Heading:   "Добро пожаловать, {{.Name}}!",
		Lead:      "Осталось подтвердить почту — так мы сможем вернуть доступ, если ты забудешь пароль.",
		BtnLabel:  "Подтвердить почту", BtnLink: "{{.Link}}", BtnColor: brandAccent,
		Box:   `<div style="font-weight:700;color:` + brandFG + `;margin-bottom:6px;">Что дальше</div>1&nbsp;&nbsp;Нажми кнопку — почта подтверждена<br>2&nbsp;&nbsp;Пройди первый урок — это пять минут<br>3&nbsp;&nbsp;Заходи каждый день, и серия не прервётся`,
		Small: linkFallback + `<br><br>Если ты не регистрировался, просто удали это письмо.`,
	}.src()

	resetHTMLSrc = htmlMsg{
		Preheader: "Задай новый пароль — ссылка скоро перестанет работать",
		Heading:   "Сброс пароля",
		Lead:      "Привет, {{.Name}}! Ты просил сбросить пароль — задай новый по кнопке ниже.",
		BtnLabel:  "Задать новый пароль", BtnLink: "{{.Link}}", BtnColor: brandAccent,
		Small: linkFallback + `<br><br>Ссылка скоро перестанет работать. Если это был не ты — ничего делать не нужно.`,
	}.src()

	alreadyHTMLSrc = htmlMsg{
		Preheader: "Кто-то только что попробовал зарегистрироваться с твоей почтой",
		Heading:   "Привет, {{.Name}}!",
		Lead:      "Кто-то только что попробовал зарегистрироваться с этой почтой. Если это был ты — возможно, ты уже заходил сюда раньше.",
		BtnLabel:  "Войти", BtnLink: "{{.LoginLink}}", BtnColor: brandAccent,
		Small: `Не помнишь пароль — <a href="{{.ForgotLink}}" style="color:` + brandAccent + `;">восстанови доступ</a>.<br><br>Если это был не ты, просто удали это письмо.`,
	}.src()

	passwordChangedHTMLSrc = htmlMsg{
		Preheader: "Пароль от твоего аккаунта только что изменили",
		Heading:   "Пароль изменён",
		Lead:      "Привет, {{.Name}}! Пароль от твоего аккаунта только что изменили. Если это был ты — всё в порядке, делать ничего не нужно.",
		BtnLabel:  "Это был не я", BtnLink: "{{.ForgotLink}}", BtnColor: brandDanger,
		Small: `Если это был не ты — нажми кнопку выше, чтобы сразу сменить пароль.`,
	}.src()
)

// Templates are parsed once at init; a parse error is a build-time bug.
var (
	verifyText = texttmpl.Must(texttmpl.New("verify.txt").Parse(verifyTextSrc))
	verifyHTML = htmltmpl.Must(htmltmpl.New("verify.html").Parse(verifyHTMLSrc))

	resetText = texttmpl.Must(texttmpl.New("reset.txt").Parse(resetTextSrc))
	resetHTML = htmltmpl.Must(htmltmpl.New("reset.html").Parse(resetHTMLSrc))

	alreadyText = texttmpl.Must(texttmpl.New("already.txt").Parse(alreadyTextSrc))
	alreadyHTML = htmltmpl.Must(htmltmpl.New("already.html").Parse(alreadyHTMLSrc))

	passwordChangedText = texttmpl.Must(texttmpl.New("pwchanged.txt").Parse(passwordChangedTextSrc))
	passwordChangedHTML = htmltmpl.Must(htmltmpl.New("pwchanged.html").Parse(passwordChangedHTMLSrc))
)
