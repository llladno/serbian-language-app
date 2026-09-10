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

// RenderVerify builds the email-verification message. baseURL has no trailing
// slash; the verify link points at the backend endpoint from part 3.
func RenderVerify(baseURL, token, name string) (subject, text, html string) {
	d := linkData{Name: name, Link: baseURL + "/api/auth/verify?token=" + url.QueryEscape(token)}
	return subjectVerify, exec(verifyText, d), exec(verifyHTML, d)
}

// RenderReset builds the password-reset message. The reset link points at the
// front-end route from part 3.
func RenderReset(baseURL, token, name string) (subject, text, html string) {
	d := linkData{Name: name, Link: baseURL + "/reset?token=" + url.QueryEscape(token)}
	return subjectReset, exec(resetText, d), exec(resetHTML, d)
}

// RenderAlreadyRegistered builds the message sent when someone tries to
// register with an address that already has an account. It never states
// outright that an account exists; it uses "если это был ты…" framing and
// points at the login and password-recovery routes.
func RenderAlreadyRegistered(baseURL, name string) (subject, text, html string) {
	d := accountData{Name: name, LoginLink: baseURL + "/login", ForgotLink: baseURL + "/forgot"}
	return subjectAlreadyRegistered, exec(alreadyText, d), exec(alreadyHTML, d)
}

// RenderPasswordChanged builds the security notice sent after a successful
// password change.
func RenderPasswordChanged(baseURL, name string) (subject, text, html string) {
	d := accountData{Name: name, ForgotLink: baseURL + "/forgot"}
	return subjectPasswordChanged, exec(passwordChangedText, d), exec(passwordChangedHTML, d)
}

// linkData feeds the templates that carry a single tokenised action link.
type linkData struct {
	Name string
	Link string
}

// accountData feeds the templates that link to login / password recovery.
type accountData struct {
	Name       string
	LoginLink  string
	ForgotLink string
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
// friendly and short. Each HTML message has exactly one styled anchor button
// plus a plain-text fallback link.

const verifyTextSrc = `Привет, {{.Name}}!

Осталось подтвердить почту — открой ссылку:
{{.Link}}

Если ты не регистрировался, просто удали это письмо.
`

const verifyHTMLSrc = `<!doctype html>
<html lang="ru">
<body style="margin:0;padding:24px;font-family:-apple-system,Segoe UI,Roboto,Arial,sans-serif;color:#1f2937;line-height:1.5">
<p>Привет, {{.Name}}!</p>
<p>Осталось подтвердить почту:</p>
<p><a href="{{.Link}}" style="display:inline-block;padding:12px 22px;background:#2563eb;color:#ffffff;text-decoration:none;border-radius:8px">Подтвердить почту</a></p>
<p style="color:#6b7280;font-size:13px">Если кнопка не работает, открой ссылку: {{.Link}}</p>
<p style="color:#6b7280;font-size:13px">Если ты не регистрировался, просто удали это письмо.</p>
</body>
</html>
`

const resetTextSrc = `Привет, {{.Name}}!

Ты просил сбросить пароль. Задай новый по ссылке:
{{.Link}}

Ссылка скоро перестанет работать. Если это был не ты — ничего делать не нужно.
`

const resetHTMLSrc = `<!doctype html>
<html lang="ru">
<body style="margin:0;padding:24px;font-family:-apple-system,Segoe UI,Roboto,Arial,sans-serif;color:#1f2937;line-height:1.5">
<p>Привет, {{.Name}}!</p>
<p>Ты просил сбросить пароль. Задай новый:</p>
<p><a href="{{.Link}}" style="display:inline-block;padding:12px 22px;background:#2563eb;color:#ffffff;text-decoration:none;border-radius:8px">Задать новый пароль</a></p>
<p style="color:#6b7280;font-size:13px">Если кнопка не работает, открой ссылку: {{.Link}}</p>
<p style="color:#6b7280;font-size:13px">Ссылка скоро перестанет работать. Если это был не ты — ничего делать не нужно.</p>
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
`

const alreadyHTMLSrc = `<!doctype html>
<html lang="ru">
<body style="margin:0;padding:24px;font-family:-apple-system,Segoe UI,Roboto,Arial,sans-serif;color:#1f2937;line-height:1.5">
<p>Привет, {{.Name}}!</p>
<p>Кто-то только что попробовал зарегистрироваться с этой почтой.</p>
<p>Если это был ты — возможно, ты уже заходил сюда раньше:</p>
<p><a href="{{.LoginLink}}" style="display:inline-block;padding:12px 22px;background:#2563eb;color:#ffffff;text-decoration:none;border-radius:8px">Войти</a></p>
<p style="color:#6b7280;font-size:13px">Не помнишь пароль — восстанови доступ: {{.ForgotLink}}</p>
<p style="color:#6b7280;font-size:13px">Если это был не ты, просто удали это письмо.</p>
</body>
</html>
`

const passwordChangedTextSrc = `Привет, {{.Name}}!

Пароль от твоего аккаунта только что изменили.

Если это был ты — всё в порядке, делать ничего не нужно.

Если это был не ты — сразу восстанови доступ:
{{.ForgotLink}}
`

const passwordChangedHTMLSrc = `<!doctype html>
<html lang="ru">
<body style="margin:0;padding:24px;font-family:-apple-system,Segoe UI,Roboto,Arial,sans-serif;color:#1f2937;line-height:1.5">
<p>Привет, {{.Name}}!</p>
<p>Пароль от твоего аккаунта только что изменили.</p>
<p>Если это был ты — всё в порядке, делать ничего не нужно.</p>
<p>Если это был не ты — сразу восстанови доступ:</p>
<p><a href="{{.ForgotLink}}" style="display:inline-block;padding:12px 22px;background:#dc2626;color:#ffffff;text-decoration:none;border-radius:8px">Восстановить доступ</a></p>
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
