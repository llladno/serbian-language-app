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

const verifyHTMLSrc = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
</head>
<body style="margin:0;padding:0;background:` + brandBG + `;">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;mso-hide:all;">Осталось подтвердить почту — одна ссылка, и можно начинать заниматься&#8203;</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:` + brandBG + `;">
<tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:440px;width:100%;">
<tr><td align="center" style="padding-bottom:20px;">
<span style="font-family:` + fontSerif + `;font-size:22px;font-weight:600;color:` + brandAccent + `;letter-spacing:0.02em;">ucimo</span>
</td></tr>
<tr><td style="background:` + brandCard + `;border:1px solid ` + brandBorder + `;border-radius:20px;padding:32px 28px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding-bottom:16px;">
<img src="{{.Image}}" width="120" alt="ucimo" style="display:block;width:120px;max-width:120px;height:auto;">
</td></tr></table>
<p style="margin:0 0 12px;font-family:` + fontSans + `;font-size:17px;font-weight:700;color:` + brandFG + `;">Привет, {{.Name}}!</p>
<p style="margin:0 0 24px;font-family:` + fontSans + `;font-size:14px;color:` + brandMuted + `;">Осталось подтвердить почту, чтобы начать заниматься сербским.</p>
<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="border-radius:12px;background:` + brandAccent + `;">
<a href="{{.Link}}" style="display:inline-block;padding:13px 26px;font-family:` + fontSans + `;font-size:15px;font-weight:700;color:#ffffff;text-decoration:none;border-radius:12px;">Подтвердить почту</a>
</td></tr></table>
<p style="margin:24px 0 0;font-family:` + fontSans + `;font-size:13px;color:` + brandMuted + `;">Если кнопка не работает, открой ссылку напрямую:<br><a href="{{.Link}}" style="color:` + brandAccent + `;word-break:break-all;">{{.Link}}</a></p>
<p style="margin:16px 0 0;font-family:` + fontSans + `;font-size:13px;color:` + brandMuted + `;">Если ты не регистрировался, просто удали это письмо.</p>
</td></tr>
<tr><td align="center" style="padding-top:20px;">
<p style="margin:0;font-family:` + fontSans + `;font-size:12px;color:` + brandMuted + `;">— ucimo · сербский с нуля</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>
`

const resetTextSrc = `Привет, {{.Name}}!

Ты просил сбросить пароль. Задай новый по ссылке:
{{.Link}}

Ссылка скоро перестанет работать. Если это был не ты — ничего делать не нужно.

— ucimo
`

const resetHTMLSrc = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
</head>
<body style="margin:0;padding:0;background:` + brandBG + `;">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;mso-hide:all;">Задай новый пароль — ссылка скоро перестанет работать&#8203;</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:` + brandBG + `;">
<tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:440px;width:100%;">
<tr><td align="center" style="padding-bottom:20px;">
<span style="font-family:` + fontSerif + `;font-size:22px;font-weight:600;color:` + brandAccent + `;letter-spacing:0.02em;">ucimo</span>
</td></tr>
<tr><td style="background:` + brandCard + `;border:1px solid ` + brandBorder + `;border-radius:20px;padding:32px 28px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding-bottom:16px;">
<img src="{{.Image}}" width="120" alt="ucimo" style="display:block;width:120px;max-width:120px;height:auto;">
</td></tr></table>
<p style="margin:0 0 12px;font-family:` + fontSans + `;font-size:17px;font-weight:700;color:` + brandFG + `;">Привет, {{.Name}}!</p>
<p style="margin:0 0 24px;font-family:` + fontSans + `;font-size:14px;color:` + brandMuted + `;">Ты просил сбросить пароль. Задай новый по ссылке ниже.</p>
<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="border-radius:12px;background:` + brandAccent + `;">
<a href="{{.Link}}" style="display:inline-block;padding:13px 26px;font-family:` + fontSans + `;font-size:15px;font-weight:700;color:#ffffff;text-decoration:none;border-radius:12px;">Задать новый пароль</a>
</td></tr></table>
<p style="margin:24px 0 0;font-family:` + fontSans + `;font-size:13px;color:` + brandMuted + `;">Если кнопка не работает, открой ссылку напрямую:<br><a href="{{.Link}}" style="color:` + brandAccent + `;word-break:break-all;">{{.Link}}</a></p>
<p style="margin:16px 0 0;font-family:` + fontSans + `;font-size:13px;color:` + brandMuted + `;">Ссылка скоро перестанет работать. Если это был не ты — ничего делать не нужно.</p>
</td></tr>
<tr><td align="center" style="padding-top:20px;">
<p style="margin:0;font-family:` + fontSans + `;font-size:12px;color:` + brandMuted + `;">— ucimo · сербский с нуля</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>
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

const alreadyHTMLSrc = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
</head>
<body style="margin:0;padding:0;background:` + brandBG + `;">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;mso-hide:all;">Кто-то только что попробовал зарегистрироваться с твоей почтой&#8203;</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:` + brandBG + `;">
<tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:440px;width:100%;">
<tr><td align="center" style="padding-bottom:20px;">
<span style="font-family:` + fontSerif + `;font-size:22px;font-weight:600;color:` + brandAccent + `;letter-spacing:0.02em;">ucimo</span>
</td></tr>
<tr><td style="background:` + brandCard + `;border:1px solid ` + brandBorder + `;border-radius:20px;padding:32px 28px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding-bottom:16px;">
<img src="{{.Image}}" width="120" alt="ucimo" style="display:block;width:120px;max-width:120px;height:auto;">
</td></tr></table>
<p style="margin:0 0 12px;font-family:` + fontSans + `;font-size:17px;font-weight:700;color:` + brandFG + `;">Привет, {{.Name}}!</p>
<p style="margin:0 0 8px;font-family:` + fontSans + `;font-size:14px;color:` + brandMuted + `;">Кто-то только что попробовал зарегистрироваться с этой почтой.</p>
<p style="margin:0 0 24px;font-family:` + fontSans + `;font-size:14px;color:` + brandMuted + `;">Если это был ты — возможно, ты уже заходил сюда раньше.</p>
<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="border-radius:12px;background:` + brandAccent + `;">
<a href="{{.LoginLink}}" style="display:inline-block;padding:13px 26px;font-family:` + fontSans + `;font-size:15px;font-weight:700;color:#ffffff;text-decoration:none;border-radius:12px;">Войти</a>
</td></tr></table>
<p style="margin:24px 0 0;font-family:` + fontSans + `;font-size:13px;color:` + brandMuted + `;">Не помнишь пароль — <a href="{{.ForgotLink}}" style="color:` + brandAccent + `;">восстанови доступ</a>.</p>
<p style="margin:16px 0 0;font-family:` + fontSans + `;font-size:13px;color:` + brandMuted + `;">Если это был не ты, просто удали это письмо.</p>
</td></tr>
<tr><td align="center" style="padding-top:20px;">
<p style="margin:0;font-family:` + fontSans + `;font-size:12px;color:` + brandMuted + `;">— ucimo · сербский с нуля</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>
`

const passwordChangedTextSrc = `Привет, {{.Name}}!

Пароль от твоего аккаунта только что изменили.

Если это был ты — всё в порядке, делать ничего не нужно.

Если это был не ты — сразу восстанови доступ:
{{.ForgotLink}}

— ucimo
`

const passwordChangedHTMLSrc = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
</head>
<body style="margin:0;padding:0;background:` + brandBG + `;">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;mso-hide:all;">Пароль от твоего аккаунта только что изменили&#8203;</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:` + brandBG + `;">
<tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:440px;width:100%;">
<tr><td align="center" style="padding-bottom:20px;">
<span style="font-family:` + fontSerif + `;font-size:22px;font-weight:600;color:` + brandAccent + `;letter-spacing:0.02em;">ucimo</span>
</td></tr>
<tr><td style="background:` + brandCard + `;border:1px solid ` + brandBorder + `;border-radius:20px;padding:32px 28px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding-bottom:16px;">
<img src="{{.Image}}" width="120" alt="ucimo" style="display:block;width:120px;max-width:120px;height:auto;">
</td></tr></table>
<p style="margin:0 0 12px;font-family:` + fontSans + `;font-size:17px;font-weight:700;color:` + brandFG + `;">Привет, {{.Name}}!</p>
<p style="margin:0 0 8px;font-family:` + fontSans + `;font-size:14px;color:` + brandMuted + `;">Пароль от твоего аккаунта только что изменили.</p>
<p style="margin:0 0 24px;font-family:` + fontSans + `;font-size:14px;color:` + brandMuted + `;">Если это был ты — всё в порядке, делать ничего не нужно.</p>
<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="border-radius:12px;background:` + brandDanger + `;">
<a href="{{.ForgotLink}}" style="display:inline-block;padding:13px 26px;font-family:` + fontSans + `;font-size:15px;font-weight:700;color:#ffffff;text-decoration:none;border-radius:12px;">Восстановить доступ</a>
</td></tr></table>
<p style="margin:24px 0 0;font-family:` + fontSans + `;font-size:13px;color:` + brandMuted + `;">Если это был не ты — используй кнопку выше, чтобы сразу сменить пароль.</p>
</td></tr>
<tr><td align="center" style="padding-top:20px;">
<p style="margin:0;font-family:` + fontSans + `;font-size:12px;color:` + brandMuted + `;">— ucimo · сербский с нуля</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>
`

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
