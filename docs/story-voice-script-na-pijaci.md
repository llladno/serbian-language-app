# История «Na pijaci»: текст для озвучки

Озвучка только в ElevenLabs (живых дикторов не нанимаем). Реплики берутся из `scripts/story_specs/na-pijaci.yaml`; если текст там изменится, этот файл надо обновить (его собирает скрипт из спеки).

Реплики Продавца (мужской голос): 1, 3, 4, 7, 9, 11, 13, 15, 17. Реплики Майи (женский голос): 2, 5, 6, 8, 10, 12, 14, 16, 18. Они **не чередуются строго** (4 и 5–6 идут подряд у одного героя), поэтому при нарезке ориентируйся на номера из этого файла, а не на чётность. Майя здесь та же героиня, что в «Prvi dan»: лучше взять тот же женский голос ElevenLabs. Русские реплики читают те же по полу голоса, тон нейтральный и ясный (это перевод, а не игра).

## Реплики и ремарки

| № | Кто | Кириллица | Латиница | Значение | Как сказать |
|---|---|---|---|---|---|
| 1 | Продавец | Изволите, изволите! Реците! | Izvolite, izvolite! Recite! | Прошу, прошу! Слушаю вас! | Громко, радушно, по-рыночному: зазывает, улыбка в голосе; «Recite!» как приглашение, без нажима |
| 2 | Майя | Добар дан! Колико кошта сир? | Dobar dan! Koliko košta sir? | Добрый день! Сколько стоит сыр? | Вежливо и с интересом, чуть неуверенно: иностранка на рынке |
| 3 | Продавец | Хиљаду динара кило. Домаћи је, свеж! | Hiljadu dinara kilo. Domaći je, svež! | Тысяча динаров за килограмм. Домашний, свежий! | С гордостью, как хвалят свой товар; «Domaći je, svež!» бодро |
| 4 | Продавец | Ево, пробајте! | Evo, probajte! | Вот, попробуйте! | Подбадривающе, протягивает кусочек; повтор «Probajte» с улыбкой |
| 5 | Майя | Добар је, добар! | Dobar je, dobar! | Вкусный, вкусный! | Искренне, приятно удивлена; повтор «dobar» как восторг |
| 6 | Майя | Дајте ми двадесет дека, молим вас. | Dajte mi dvadeset deka, molim vas. | Дайте мне двести граммов, пожалуйста. | Решительно и вежливо, довольная вкусом |
| 7 | Продавец | Ево. Још нешто? | Evo. Još nešto? | Вот. Ещё что-нибудь? | Деловито и приветливо: отдаёт сыр, сразу следующий вопрос |
| 8 | Майя | Имате ли јаја? | Imate li jaja? | Яйца есть? | Просто и непринуждённо, вопрос |
| 9 | Продавец | Имам. Колико комада? | Imam. Koliko komada? | Есть. Сколько штук? | Охотно и коротко: «Imam.», потом «Koliko komada?» деловито |
| 10 | Майя | Десет. То је све. | Deset. To je sve. | Десять. Это всё. | Коротко «Deset.», потом «To je sve.» спокойно, чуть небрежно |
| 11 | Продавец | Ево. Триста педесет динара. | Evo. Trista pedeset dinara. | Вот. Триста пятьдесят динаров. | Ровно и дружелюбно: называет итог |
| 12 | Майя | Може картицом? | Može karticom? | Можно картой? | С надеждой и лёгкой неловкостью, вопрос |
| 13 | Продавец | Не, само кеш! | Ne, samo keš! | Нет, только наличные. | Добродушно-категорично, с лёгким смешком; «samo keš» чётко |
| 14 | Майя | Имам само хиљаду. | Imam samo hiljadu. | У меня только тысяча. | Смущённо, виновато: у неё только крупная купюра |
| 15 | Продавец | Хиљаду? Имате ли ситно? | Hiljadu? Imate li sitno? | Тысяча? А помельче нет? | «Hiljadu?» с лёгким удивлением, потом терпеливый вопрос |
| 16 | Майя | Само мало… Имам! Ево, тачно! | Samo malo… Imam! Evo, tačno! | Секунду… Есть! Вот, ровно! | «Samo malo…» роется в сумке, медленно; «Imam!» радостно; «Evo, tačno!» торжествующе |
| 17 | Продавец | Тачно, хвала! Довиђења! | Tačno, hvala! Doviđenja! | Ровно, спасибо! До свидания! | Довольно и тепло; «Doviđenja!» дружелюбно |
| 18 | Майя | Довиђења! | Doviđenja! | До свидания! | Тепло и коротко |

## Основной список: без тегов, теги только в особых местах (брать первым)

v3 и v4 играют: эмоциональные теги (`delighted`, `triumphant`, `proud`) и восклицательные знаки дают «аниме-голос». Поэтому **обычные реплики идут без тегов**, а `!` во входном тексте заменены на `.` (слова те же, что в приложении, меняется только пунктуация во входе озвучки). **Тег стоит только там, где он нужен сцене:** реплика 1, реплика 16. Если и так играют: Stability на максимум (Natural или Robust), Style Exaggeration 0, по одной реплике. Каждый голос читает весь список, из записи берутся только реплики своего героя (номера выше).

**Сербский (латиница):**

```
[loud, friendly market-seller voice] Izvolite, izvolite. Recite.
Dobar dan. Koliko košta sir?
Hiljadu dinara kilo. Domaći je, svež.
Evo, probajte.
Dobar je, dobar.
Dajte mi dvadeset deka, molim vas.
Evo. Još nešto?
Imate li jaja?
Imam. Koliko komada?
Deset. To je sve.
Evo. Trista pedeset dinara.
Može karticom?
Ne, samo keš.
Imam samo hiljadu.
Hiljadu? Imate li sitno?
[flustered, then relieved voice] Samo malo… Imam. Evo, tačno.
Tačno, hvala. Doviđenja.
Doviđenja.
```

**Русский:**

```
[loud, friendly market-seller voice] Прошу, прошу. Слушаю вас.
Добрый день. Сколько стоит сыр?
Тысяча динаров за килограмм. Домашний, свежий.
Вот, попробуйте.
Вкусный, вкусный.
Дайте мне двести граммов, пожалуйста.
Вот. Ещё что-нибудь?
Яйца есть?
Есть. Сколько штук?
Десять. Это всё.
Вот. Триста пятьдесят динаров.
Можно картой?
Нет, только наличные.
У меня только тысяча.
Тысяча? А помельче нет?
[flustered, then relieved voice] Секунду… Есть. Вот, ровно.
Ровно, спасибо. До свидания.
До свидания.
```

## Запасная версия: теги на каждой реплике (слишком театральная на v4)

Тег описывает голос, а не действие. Нужна модель v3 или v4: на других теги прочитаются вслух. Бери отдельные реплики отсюда, только если основной список вышел плоским.

**Сербский (латиница):**

```
[loud, friendly market-seller voice, smiling] Izvolite, izvolite! Recite!
[polite, curious voice, a little unsure] Dobar dan! Koliko košta sir?
[proud, warm salesman voice] Hiljadu dinara kilo. Domaći je, svež!
[encouraging, friendly voice, smiling] Evo, probajte!
[pleasantly surprised, delighted voice] Dobar je, dobar!
[pleased, polite voice] Dajte mi dvadeset deka, molim vas.
[brisk, friendly voice] Evo. Još nešto?
[light, curious voice] Imate li jaja?
[cheerful, reassuring voice] Imam. Koliko komada?
[casual, brisk voice] Deset. To je sve.
[matter-of-fact, friendly voice] Evo. Trista pedeset dinara.
[hopeful, slightly awkward voice] Može karticom?
[good-natured, firm, amused voice] Ne, samo keš!
[embarrassed, apologetic voice] Imam samo hiljadu.
[slightly surprised, then patient voice] Hiljadu? Imate li sitno?
[flustered, then relieved and triumphant voice] Samo malo… Imam! Evo, tačno!
[satisfied, warm voice] Tačno, hvala! Doviđenja!
[warm, friendly voice] Doviđenja!
```

**Русский:**

```
[loud, friendly market-seller voice, smiling] Прошу, прошу! Слушаю вас!
[polite, curious voice, a little unsure] Добрый день! Сколько стоит сыр?
[proud, warm salesman voice] Тысяча динаров за килограмм. Домашний, свежий!
[encouraging, friendly voice, smiling] Вот, попробуйте!
[pleasantly surprised, delighted voice] Вкусный, вкусный!
[pleased, polite voice] Дайте мне двести граммов, пожалуйста.
[brisk, friendly voice] Вот. Ещё что-нибудь?
[light, curious voice] Яйца есть?
[cheerful, reassuring voice] Есть. Сколько штук?
[casual, brisk voice] Десять. Это всё.
[matter-of-fact, friendly voice] Вот. Триста пятьдесят динаров.
[hopeful, slightly awkward voice] Можно картой?
[good-natured, firm, amused voice] Нет, только наличные.
[embarrassed, apologetic voice] У меня только тысяча.
[slightly surprised, then patient voice] Тысяча? А помельче нет?
[flustered, then relieved and triumphant voice] Секунду… Есть! Вот, ровно!
[satisfied, warm voice] Ровно, спасибо! До свидания!
[warm, friendly voice] До свидания!
```

## Для сравнения движков

Только кириллица, по одной реплике на строку (так их получает синтезатор).

```
1  Изволите, изволите! Реците!
2  Добар дан! Колико кошта сир?
3  Хиљаду динара кило. Домаћи је, свеж!
4  Ево, пробајте!
5  Добар је, добар!
6  Дајте ми двадесет дека, молим вас.
7  Ево. Још нешто?
8  Имате ли јаја?
9  Имам. Колико комада?
10 Десет. То је све.
11 Ево. Триста педесет динара.
12 Може картицом?
13 Не, само кеш!
14 Имам само хиљаду.
15 Хиљаду? Имате ли ситно?
16 Само мало… Имам! Ево, тачно!
17 Тачно, хвала! Довиђења!
18 Довиђења!
```
