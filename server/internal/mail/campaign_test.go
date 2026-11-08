package mail

import (
	"strings"
	"testing"
)

func TestRenderCampaign(t *testing.T) {
	subject, text, body := RenderCampaign(CampaignMail{
		Subject: "Те самые буквы с галочками", Preheader: "š, ž, č, ć, đ", Heading: "Те самые буквы 🔤",
		Lead: "Ćao (привет)!\n\n👏 **Вы прошли урок** <b>x</b> & дальше:\n\n• š, ž — как «ш» и «ж»\n• **č и ć** — главная пара\n\nИ ещё строка\nвторая строка",
		Box:  "🐦 Ласточка подарит 5 зёрнышек.", BtnLabel: "К особым звукам", Image: "https://ucimo.ru/campaign/return-3.jpg",
		Link: "https://ucimo.ru/lesson/01?c=abc", Unsubscribe: "https://ucimo.ru/api/unsubscribe?t=abc",
	})
	if subject != "Те самые буквы с галочками" {
		t.Errorf("subject = %q", subject)
	}
	for _, want := range []string{
		`href="https://ucimo.ru/lesson/01?c=abc"`,       // the button
		`href="https://ucimo.ru/api/unsubscribe?t=abc"`, // the unsubscribe link
		`src="https://ucimo.ru/campaign/return-3.jpg"`,  // the scene across the top
		`width="480"`,                                       // full-width
		"К особым звукам", "🐦 Ласточка подарит 5 зёрнышек.", // button and box
		"Те самые буквы 🔤",                                                                      // heading with emoji
		`<p style="margin:0 0 14px;">Ćao (привет)!</p>`,                                         // paragraphs are paragraphs
		`👏 <b style="color:#172326;">Вы прошли урок</b> &lt;b&gt;x&lt;/b&gt; &amp; дальше:</p>`, // bold, escaping
		`•&nbsp;&nbsp;š, ž — как «ш» и «ж»</div>`,                                               // list items
		`•&nbsp;&nbsp;<b style="color:#172326;">č и ć</b> — главная пара</div>`,
		`И ещё строка<br>вторая строка</p>`, // single line breaks stay
		"š, ž, č, ć, đ",   // preheader
		"text-align:left", // long text reads left-aligned
	} {
		if !strings.Contains(body, want) {
			t.Errorf("html lacks %q", want)
		}
	}
	for _, want := range []string{"Те самые буквы 🔤", "Ćao (привет)!", "Вы прошли урок", "• š, ž — как «ш» и «ж»", "К особым звукам: https://ucimo.ru/lesson/01?c=abc", "Отписаться от писем об учёбе: https://ucimo.ru/api/unsubscribe?t=abc"} {
		if !strings.Contains(text, want) {
			t.Errorf("text lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "**") {
		t.Errorf("text keeps the bold marks:\n%s", text)
	}
}

func TestRenderCampaignWithoutBoxAndWithTemplateSyntaxInText(t *testing.T) {
	_, _, body := RenderCampaign(CampaignMail{Heading: "H", Lead: "a {{.Link}} b", BtnLabel: "B", Link: "https://x/y", Unsubscribe: "https://x/u"})
	if strings.Count(body, "https://x/y") != 1 {
		t.Errorf("template syntax in the text was run: link appears %d times", strings.Count(body, "https://x/y"))
	}
	if !strings.Contains(body, "{&#8203;{.Link}}") && !strings.Contains(body, "{\u200b{.Link}}") {
		t.Errorf("the double brace in the text should be kept inert, html: %s", body)
	}
}

func TestRenderCampaignWithoutPictureHasNoImageTag(t *testing.T) {
	_, _, body := RenderCampaign(CampaignMail{Heading: "H", Lead: "L", BtnLabel: "B", Link: "https://x/y", Unsubscribe: "https://x/u"})
	if strings.Contains(body, "<img") {
		t.Errorf("a message without a picture should not carry an image tag: %s", body)
	}
}
