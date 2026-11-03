# История «Dobro veče»: текст для озвучки

Одна страница для двух задач: **заказ живых дикторов** (письмо на английском ниже, его можно отправлять как есть) и **озвучка в ElevenLabs** (списки с тегами ниже). Реплики берутся из `scripts/story_specs/dobro-vece.yaml`; если текст там изменится, этот файл надо обновить (его собирает скрипт из спеки).

Нечётные реплики говорит Марко (мужской голос), чётные Ана (женский). Русские реплики второго прохода живого диктора не требуют. Условия оплаты и прав в письме подставьте свои.

## Письмо диктору (English)

> **Subject: Serbian voice-over, 8 very short lines (about 15 seconds), male or female voice**
>
> Hi! We are building UCIMO, a Serbian language-learning app for Russian speakers who are moving to Serbia. We need native-speaker recordings for a tiny dialogue between two neighbours who meet in a courtyard in the evening: **Ana** (a woman in her late twenties) and **Marko** (a man in his early thirties). It is 8 lines, about 25 words, around 15 seconds of audio in total. We need two different voices. If you have only one, please record just your character's lines: Marko is lines 1, 3, 5, 7 and Ana is lines 2, 4, 6, 8.
>
> **Language.** Standard Serbian (ekavian), neutral everyday pronunciation, no strong regional dialect. The listeners are complete beginners, so the diction must be clear and natural: not slow-motion, not theatrical. Please read the text exactly as written (it is attached in Cyrillic and Latin). The neighbours address each other politely ("vi"). If a line sounds unnatural in everyday speech, or you would say it differently, please tell us. We will gladly change the text.
>
> **Technical.** WAV, 48 kHz, 24-bit, mono. A dry recording: no reverb, music, noise reduction, compression or EQ, peaks around -6 dB. A quiet room, the same microphone position for all lines. One file per line, with about 0.5 seconds of silence at the start and at the end. Two takes per line, please. File names: `01_Marko_take1.wav`, `01_Marko_take2.wav`, `02_Ana_take1.wav` and so on.
>
> **Rights.** A perpetual, worldwide licence to use the recordings in our web app, our Telegram Mini App and our promo videos and social media. The recordings will not be used to train or clone voices.
>
> Please send a flat price for the whole set and your turnaround time. Thank you!

## Реплики и ремарки

| № | Кто | Кириллица | Латиница | Значение | Как сказать |
|---|---|---|---|---|---|
| 1 | Марко | Добро вече! | Dobro veče! | Добрый вечер! | Тепло, приветливо, на выдохе: сосед увидел знакомое лицо во дворе вечером |
| 2 | Ана | Добро вече! Како сте? | Dobro veče! Kako ste? | Добрый вечер! Как вы? | Вежливо, с улыбкой; «Kako ste?» искренне, с интересом |
| 3 | Марко | Добро, хвала. А ви? | Dobro, hvala. A vi? | Хорошо, спасибо. А вы? | Просто и тепло; «A vi?» встречный вопрос, с интересом |
| 4 | Ана | Може. Не жалим се. | Može. Ne žalim se. | Нормально. Не жалуюсь. | Спокойно, чуть устало, с улыбкой; «Može» с лёгким пожатием плеч, «Ne žalim se» мягко и шутливо |
| 5 | Марко | Видимо се? | Vidimo se? | Увидимся? | С лёгкой надеждой, вопросительно, как приглашение: интонация вверх |
| 6 | Ана | Може! Чујемо се? | Može! Čujemo se? | Можно! Созвонимся? | «Može!» с радостью, сразу «Čujemo se?» лёгким воодушевлённым вопросом |
| 7 | Марко | Добро! Лаку ноћ! | Dobro! Laku noć! | Хорошо! Спокойной ночи! | Довольно и тепло; «Laku noć!» мягко и чуть тише |
| 8 | Ана | Лаку ноћ! Довиђења! | Laku noć! Doviđenja! | Спокойной ночи! До свидания! | Тепло и негромко, вечерняя интонация; «Doviđenja!» с улыбкой |

## Русская версия (для второго прохода)

Тот же текст по-русски, он звучит после каждой сербской реплики. Лучше два голоса, по полу героя: мужской для Марко (реплики 1, 3, 5, 7) и женский для Аны (2, 4, 6, 8). Тон нейтральный и ясный: это перевод, а не игра. Русский поддерживают все модели ElevenLabs, включая Multilingual v2.

**Марко:**

```
Добрый вечер!
Хорошо, спасибо. А вы?
Увидимся?
Хорошо! Спокойной ночи!
```

**Ана:**

```
Добрый вечер! Как вы?
Нормально. Не жалуюсь.
Можно! Созвонимся?
Спокойной ночи! До свидания!
```

## Всё одним списком, с тегами (так вставляется в голос ElevenLabs)

Каждый голос читает весь список, из записи берутся только реплики своего героя (Марко нечётные, Ана чётные). Нужна модель v3 или v4: на других теги прочитаются вслух, а сербского Multilingual v2 и Flash не знают (в интерфейсе на них только хорватский, и интонация неверная). Теги описывают голос, а не действие; паузы дают пунктуация и многоточие.

**Сербский (латиница):**

```
[warm, friendly, relaxed evening voice] Dobro veče!
[polite, warm, smiling voice] Dobro veče! Kako ste?
[simple, warm, friendly voice] Dobro, hvala. A vi?
[calm, slightly tired, smiling voice] Može. Ne žalim se.
[hopeful, light, questioning voice] Vidimo se?
[pleased, bright voice, then a light question] Može! Čujemo se?
[content, soft, warm voice] Dobro! Laku noć!
[gentle, warm, evening voice] Laku noć! Doviđenja!
```

**Русский:**

```
[warm, friendly, relaxed evening voice] Добрый вечер!
[polite, warm, smiling voice] Добрый вечер! Как вы?
[simple, warm, friendly voice] Хорошо, спасибо. А вы?
[calm, slightly tired, smiling voice] Нормально. Не жалуюсь.
[hopeful, light, questioning voice] Увидимся?
[pleased, bright voice, then a light question] Можно! Созвонимся?
[content, soft, warm voice] Хорошо! Спокойной ночи!
[gentle, warm, evening voice] Спокойной ночи! До свидания!
```

## Для сравнения движков

Только кириллица, по одной реплике на строку (так их получает синтезатор). Голоса: женский для чётных, мужской для нечётных.

```
1  Добро вече!
2  Добро вече! Како сте?
3  Добро, хвала. А ви?
4  Може. Не жалим се.
5  Видимо се?
6  Може! Чујемо се?
7  Добро! Лаку ноћ!
8  Лаку ноћ! Довиђења!
```
