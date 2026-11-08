package campaign

import (
	"html"
	"regexp"
	"strings"
)

// The campaign's words (approved by the owner on 2026-11-08, laid out and given more emoji the same day). Segments: 1 did nothing,
// 2 started lesson 00 and left it, 3 finished 00 and never opened 01, 4 finished 01 and never opened 02.
//
// Texts are written in a tiny markup that both channels understand: a blank line starts a paragraph, a line that starts with «• » is a
// list item, **double stars** make bold. Telegram gets it as HTML (TelegramHTML), e-mail renders it itself (see internal/mail).

// Segments is how many segments there are.
const Segments = 4

// Message is everything one segment is told, in Telegram and by e-mail.
type Message struct {
	Lesson string // the lesson the button opens
	Image  string // the picture, file name without extension, in landing/public/campaign (return-1 … return-4)

	Telegram string // caption of the photo

	Subject   string
	Preheader string
	Heading   string
	Lead      string
	Box       string // the swallow's tour, only for segments that have finished a lesson; empty otherwise

	Buttons [NumStyles]string
}

// messages are the words of the four segments. Segments 3 and 4 mention the swallow's tour (5 seeds): it shows when the person next
// opens the course, which is why their stored answer to it is reset when the message goes out.
var messages = [Segments + 1]Message{
	1: {
		Lesson: "00",
		Image:  "return-1",
		Telegram: "👋 **Ćao (привет)!** Вы регистрировались в ucimo, а первый урок так и ждёт.\n\n" +
			"📖 Он короткий и про буквы: кириллицу и латиницу в сербском. После него вы поймёте, как читать сербские слова.\n\n" +
			"🌿 Начнём? Polako (не спеша), мы рядом 💛",
		Subject:   "Первый урок в ucimo ждёт вас",
		Preheader: "Короткий, про буквы. Можно начать прямо сейчас.",
		Heading:   "Первый урок ждёт вас 👋",
		Lead: "Ćao (привет)!\n\n" +
			"Вы зарегистрировались в ucimo, но так и не открыли первый урок. Так бывает: регистрация сегодня, а сербский «потом».\n\n" +
			"📖 **Первый урок короткий.** Он про буквы: в сербском работают сразу два письма, кириллица и латиница, и мы разберём, как они друг с другом дружат. " +
			"Дальше каждый урок строится на предыдущем.\n\n" +
			"🌱 Подписки нет, зёрнышки за прохождение копятся сами.\n\n" +
			"🌿 Polako (не спеша): урок подождёт 💛",
		Buttons: [NumStyles]string{"Открыть первый урок", "Загляну на минутку"},
	},
	2: {
		Lesson: "00",
		Image:  "return-2",
		Telegram: "📖 **Вы остановились на первом уроке**, осталось совсем немного.\n\n" +
			"✅ Мы сохранили, где вы остановились, так что можно просто продолжить.\n\n" +
			"🌿 Sve u redu (всё в порядке): вернуться можно в любой момент 💛",
		Subject:   "Вы остановились на первом уроке",
		Preheader: "Мы сохранили, где вы остановились.",
		Heading:   "Вы остановились на первом уроке 📖",
		Lead: "Ćao (привет)!\n\n" +
			"Вы начали первый урок в ucimo и остановились, не дойдя до конца.\n\n" +
			"✅ **Он всё ещё открыт на том самом месте**, где вы его оставили: ничего не нужно проходить заново.\n\n" +
			"📖 Урок про кириллицу и латиницу. Дальше будут уже настоящие слова, и без этих букв читать их было бы сложнее.\n\n" +
			"🌿 Заглянете, когда будет пара свободных минут? 💛",
		Buttons: [NumStyles]string{"Продолжить урок", "Закончу за пару минут"},
	},
	3: {
		Lesson: "01",
		Image:  "return-3",
		Telegram: "👏 **Первый урок пройден.** Дальше «Особые звуки»: те самые буквы с галочками, из-за которых сербский кажется страшным.\n\n" +
			"🔤 На деле их восемь, и мы разберём каждую:\n" +
			"• š, ž — как русские «ш» и «ж»\n" +
			"• lj, nj — как «ль» и «нь»\n" +
			"• č и ć — главная пара, их различают на слух\n" +
			"• đ и dž — мягкая и твёрдая\n\n" +
			"🐦 Заодно ласточка покажет приложение и подарит 5 зёрнышек 🌱💛",
		Subject:   "Те самые буквы с галочками",
		Preheader: "š, ž, č, ć, đ: разбираем по одной.",
		Heading:   "Те самые буквы с галочками 🔤",
		Lead: "Ćao (привет)!\n\n" +
			"👏 **Вы прошли первый урок**, а дальше как раз то, что многих пугает: буквы с галочками. Выглядит как много, но звуки в основном знакомые:\n\n" +
			"• š, ž — читаются так же, как русские «ш» и «ж»\n" +
			"• lj, nj — как «ль» и «нь»\n" +
			"• č и ć — главная пара, в русском такого различия нет\n" +
			"• đ и dž — мягкая и твёрдая\n\n" +
			"👂 Урок как раз про это, с примерами для слуха 💛",
		Box:     "🐦 Заодно загляните в приложение: ласточка покажет, где что находится, и подарит 5 зёрнышек 🌱",
		Buttons: [NumStyles]string{"К особым звукам", "Послушаю č и ć"},
	},
	4: {
		Lesson: "02",
		Image:  "return-4",
		Telegram: "💪 **Особые звуки позади**, одна из самых непривычных частей начала. Дальше первые настоящие слова: урок «Привет!»\n\n" +
			"Четыре способа поздороваться:\n" +
			"👋 Zdravo — привет\n" +
			"🤝 Ćao — привет и пока\n" +
			"☀️ Dobar dan — добрый день\n" +
			"🌅 Dobro jutro — доброе утро\n\n" +
			"🇷🇸 С ним уже можно здороваться по-сербски 💛\n\n" +
			"🐦 А ласточка покажет приложение и подарит 5 зёрнышек 🌱",
		Subject:   "Урок, после которого можно здороваться по-сербски",
		Preheader: "Zdravo, Ćao, Dobar dan, Dobro jutro.",
		Heading:   "Теперь можно здороваться по-сербски 🇷🇸",
		Lead: "Ćao (привет)!\n\n" +
			"💪 **Вы прошли особые звуки**: это одна из самых непривычных частей начала курса, и она позади.\n\n" +
			"Дальше урок «Привет!», первые настоящие слова. Четыре способа поздороваться:\n\n" +
			"👋 **Zdravo** — привет\n" +
			"🤝 **Ćao** — привет и пока\n" +
			"☀️ **Dobar dan** — добрый день\n" +
			"🌅 **Dobro jutro** — доброе утро\n\n" +
			"Узнаете, когда какое подходит. Ничего сложного: вы уже умеете читать и слышать, чем отличаются č и ć. Остаётся здороваться 💛",
		Box:     "🐦 А ласточка в приложении покажет, что ещё там есть, и подарит 5 зёрнышек 🌱",
		Buttons: [NumStyles]string{"К уроку «Привет!»", "Послушаю Zdravo и Ćao"},
	},
}

// For returns the words of a segment (1 to 4); anything else gets the first segment's.
func For(segment int) Message {
	if segment < 1 || segment > Segments {
		segment = 1
	}
	return messages[segment]
}

// LessonURL is where the button of a segment's message leads, with the click token in the address.
func LessonURL(base string, segment int, token string) string {
	return strings.TrimRight(base, "/") + "/lesson/" + For(segment).Lesson + "?c=" + token
}

// PhotoURL is the address of a segment's picture as Telegram fetches it, under base (the folder the pictures are published at).
func PhotoURL(base string, segment int) string {
	return strings.TrimRight(base, "/") + "/" + For(segment).Image + ".jpg"
}

// ImageURL is the address of a segment's picture on the site, for the e-mail.
func ImageURL(siteBase string, segment int) string {
	return strings.TrimRight(siteBase, "/") + "/campaign/" + For(segment).Image + ".jpg"
}

var boldMarks = regexp.MustCompile(`\*\*([^*]+)\*\*`)

// TelegramHTML turns a text in the campaign markup into Telegram's HTML: everything is escaped and **bold** becomes <b>. Paragraph
// breaks and list lines are plain line breaks, which Telegram keeps as they are.
func TelegramHTML(s string) string {
	return boldMarks.ReplaceAllString(html.EscapeString(s), "<b>$1</b>")
}

// PlainText is the text without the markup's bold marks, for a plain-text part.
func PlainText(s string) string {
	return boldMarks.ReplaceAllString(s, "$1")
}
