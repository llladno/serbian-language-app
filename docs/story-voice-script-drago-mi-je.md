# Демо-история «Drago mi je»: текст для озвучки

Одна страница для двух задач: **заказ живых дикторов** (письмо на английском ниже, его можно отправлять как есть) и
**сравнение движков синтеза** (те же 14 реплик в кириллице: синтезатор получает именно кириллицу, латиница читается хуже).
Реплики берутся из `scripts/story_specs/drago-mi-je.yaml`, если текст там изменится, этот файл надо обновить.

Нечётные реплики говорит Марко (мужской голос), чётные Ана (женский). Русские реплики второго прохода диктора не нужны.
Условия оплаты и прав в письме подставьте свои.

## Письмо диктору (English)

> **Subject: Serbian voice-over, 14 short lines (about 1 minute), male or female voice**
>
> Hi! We are building UCIMO, a Serbian language-learning app for Russian speakers who are moving to Serbia. We need
> native-speaker recordings for a short dialogue between two neighbours in Novi Sad: **Ana** (a woman in her late twenties)
> and **Marko** (a man in his early thirties). It is 14 lines, about 45 words, around one minute of audio in total.
> We need two different voices. If you have only one, please record just your character's lines: Marko is lines
> 1, 3, 5, 7, 9, 11, 13 and Ana is lines 2, 4, 6, 8, 10, 12, 14.
>
> **Language.** Standard Serbian (ekavian), neutral everyday pronunciation, no strong regional dialect. The listeners are
> beginners, so the diction must be clear and natural: not slow-motion, not theatrical. Please read the text exactly as
> written (it is attached in Cyrillic and Latin). If a line sounds unnatural in everyday speech, or you would say it
> differently, please tell us. We will gladly change the text.
>
> **Technical.** WAV, 48 kHz, 24-bit, mono. A dry recording: no reverb, music, noise reduction, compression or EQ, peaks
> around -6 dB. A quiet room, the same microphone position for all lines. One file per line, with about 0.5 seconds of
> silence at the start and at the end. Two takes per line, please. File names: `01_Marko_take1.wav`, `01_Marko_take2.wav`,
> `02_Ana_take1.wav` and so on.
>
> **Rights.** A perpetual, worldwide licence to use the recordings in our web app, our Telegram Mini App and our promo
> videos and social media. The recordings will not be used to train or clone voices.
>
> Please send a flat price for the whole set and your turnaround time. Thank you!

## Реплики и ремарки

| № | Кто | Кириллица | Латиница | Значение | Как сказать |
|---|---|---|---|---|---|
| 1 | Марко | Здраво! | Zdravo! | Hello! | Бодро, дружелюбно, чуть громче: он машет через двор |
| 2 | Ана | Ћао! Како си? | Ćao! Kako si? | Hi! How are you? | Тепло, с улыбкой, легко |
| 3 | Марко | Добро, хвала. А ти? | Dobro, hvala. A ti? | Fine, thanks. And you? | Расслабленно, радостно |
| 4 | Ана | Може. Како се зовеш? | Može. Kako se zoveš? | So-so. What's your name? | «Може» с лёгкой шутливой интонацией «так себе», вопрос живо и с интересом |
| 5 | Марко | Зовем се Марко. | Zovem se Marko. | My name is Marko. | Естественно и чуть быстро, как местному, но разборчиво |
| 6 | Ана | Извини, не разумем. Полако, молим. | Izvini, ne razumem. Polako, molim. | Sorry, I don't understand. Slowly, please. | Смущённо, виновато-мило; «Полако, молим» мягко и вежливо |
| 7 | Марко | Марко. Зовем се Марко. Драго ми је. | Marko. Zovem se Marko. Drago mi je. | Marko. My name is Marko. Nice to meet you. | **Медленно и чётко**, пауза около 0,4 с после каждой фразы, с улыбкой, без «сюсюканья» |
| 8 | Ана | Аха! Ја сам Ана. Такође. | Aha! Ja sam Ana. Takođe. | Aha! I'm Ana. Likewise. | «Аха!» как озарение, облегчённо и радостно |
| 9 | Марко | Одакле си? | Odakle si? | Where are you from? | Любопытно, дружелюбно |
| 10 | Ана | Из Русије. Живим у Новом Саду. | Iz Rusije. Živim u Novom Sadu. | From Russia. I live in Novi Sad. | Просто, с тихой гордостью |
| 11 | Марко | Говориш српски? | Govoriš srpski? | Do you speak Serbian? | Приятно удивлённо |
| 12 | Ана | Мало. Учим. | Malo. Učim. | A little. I'm learning. | Скромно, с улыбкой, короткая пауза между словами |
| 13 | Марко | Добро! Видимо се! | Dobro! Vidimo se! | Good! See you! | Подбадривающе, тепло |
| 14 | Ана | Ћао! | Ćao! | Bye! | Весёлое прощание, как будто машет рукой |

## Русская версия (для второго прохода)

Тот же текст по-русски, он звучит после каждой сербской реплики. Лучше два голоса, по полу героя: мужской для Марко (реплики 1, 3, 5, 7, 9, 11, 13) и женский для Аны (2, 4, 6, 8, 10, 12, 14). Тон нейтральный и ясный: это перевод, а не игра. Русский поддерживают все модели ElevenLabs, включая Multilingual v2, сербского здесь не нужно.

**Марко:**

```
Привет!
Хорошо, спасибо. А ты?
Меня зовут Марко.
Марко. Меня зовут Марко. Приятно познакомиться.
Откуда ты?
Ты говоришь по-сербски?
Хорошо! Увидимся!
```

**Ана:**

```
Привет! Как дела?
Нормально. Как тебя зовут?
Извини, не понимаю. Помедленнее, пожалуйста.
А! Я Ана. Взаимно.
Из России. Живу в Нови-Саде.
Немного. Учу.
Пока!
```

Если «А!» в реплике 8 прочитается как буква, замени на «Ага!» и скажи мне, я поправлю субтитр.

## Всё одним списком, с тегами (так вставляется в голос ElevenLabs)

Каждый голос читает весь список, из записи берутся только реплики своего героя (Марко нечётные, Ана чётные). Нужна модель v3 или v4: на других теги прочитаются вслух.

**Сербский (латиница):**

```
[bright, friendly voice, a little louder] Zdravo!
[warm, smiling voice] Ćao! Kako si?
[relaxed, cheerful voice] Dobro, hvala. A ti?
[playful, then curious voice] Može. Kako se zoveš?
[friendly, slightly quick voice] Zovem se Marko.
[soft, apologetic, embarrassed voice] Izvini, ne razumem. Polako, molim.
[slow, clear, gentle voice] Marko... Zovem se Marko... Drago mi je.
[relieved, happy voice] Aha! Ja sam Ana. Takođe.
[curious, friendly voice] Odakle si?
[simple, quietly proud voice] Iz Rusije. Živim u Novom Sadu.
[pleasantly surprised voice] Govoriš srpski?
[modest, smiling voice] Malo... Učim.
[warm, encouraging voice] Dobro! Vidimo se!
[cheerful, bright voice] Ćao!
```

**Русский:**

```
[bright, friendly voice, a little louder] Привет!
[warm, smiling voice] Привет! Как дела?
[relaxed, cheerful voice] Хорошо, спасибо. А ты?
[playful, then curious voice] Нормально. Как тебя зовут?
[friendly, slightly quick voice] Меня зовут Марко.
[soft, apologetic, embarrassed voice] Извини, не понимаю. Помедленнее, пожалуйста.
[slow, clear, gentle voice] Марко... Меня зовут Марко... Приятно познакомиться.
[relieved, happy voice] А! Я Ана. Взаимно.
[curious, friendly voice] Откуда ты?
[simple, quietly proud voice] Из России. Живу в Нови-Саде.
[pleasantly surprised voice] Ты говоришь по-сербски?
[modest, smiling voice] Немного... Учу.
[warm, encouraging voice] Хорошо! Увидимся!
[cheerful, bright voice] Пока!
```

## Для сравнения движков

Только кириллица, по одной реплике на строку (так их получает синтезатор). Голоса: женский для чётных, мужской для нечётных.

```
1  Здраво!
2  Ћао! Како си?
3  Добро, хвала. А ти?
4  Може. Како се зовеш?
5  Зовем се Марко.
6  Извини, не разумем. Полако, молим.
7  Марко. Зовем се Марко. Драго ми је.
8  Аха! Ја сам Ана. Такође.
9  Одакле си?
10 Из Русије. Живим у Новом Саду.
11 Говориш српски?
12 Мало. Учим.
13 Добро! Видимо се!
14 Ћао!
```
