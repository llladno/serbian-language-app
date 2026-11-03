# История «SIM kartica»: текст для озвучки

Озвучка только в ElevenLabs (живых дикторов не нанимаем). Реплики берутся из `scripts/story_specs/sim-kartica.yaml`; если текст там изменится, этот файл надо обновить (его собирает скрипт из спеки).

Реплики Продавщицы (женский голос): 1, 3, 5, 7, 9, 11, 13, 15. Реплики Алекса (мужской голос): 2, 4, 6, 8, 10, 12, 14, 16; они чередуются (нечётные Продавщица, чётные Алекс). **Реплика 11 говорится быстро, цифры подряд, реплика 13 медленно по группам по три цифры**: это смысл сцены, следи за разницей. Русские реплики читают те же по полу голоса, тон нейтральный и ясный (это перевод, а не игра).

## Реплики и ремарки

| № | Кто | Кириллица | Латиница | Значение | Как сказать |
|---|---|---|---|---|---|
| 1 | Продавщица | Добар дан! Изволите? | Dobar dan! Izvolite? | Добрый день! Чем могу помочь? | Приветливо и по-деловому: встречает покупателя |
| 2 | Алекс | Добар дан! Треба ми СИМ картица. | Dobar dan! Treba mi SIM kartica. | Добрый день! Мне нужна сим-карта. | Вежливо и спокойно, просто просит |
| 3 | Продавщица | Треба ми пасош, молим вас. | Treba mi pasoš, molim vas. | Мне нужен ваш паспорт, пожалуйста. | Деловито, но вежливо: просит документ |
| 4 | Алекс | Ево, изволите. | Evo, izvolite. | Вот, пожалуйста. | Дружелюбно, протягивает паспорт |
| 5 | Продавщица | Из Русије сте? | Iz Rusije ste? | Вы из России? | Увидела паспорт: лёгкое удивление и интерес |
| 6 | Алекс | Да, али сада живим овде. | Da, ali sada živim ovde. | Да, но теперь я живу здесь. | Спокойно, чуть застенчиво: «ali sada živim ovde» мягко |
| 7 | Продавщица | Добро. СИМ и интернет, да? | Dobro. SIM i internet, da? | Хорошо. Сим-карту и интернет, да? | Быстро и приветливо: «Dobro» и вопрос про пакет |
| 8 | Алекс | Да, да. Колико кошта? | Da, da. Koliko košta? | Да, да. Сколько стоит? | Легко, согласно; «Koliko košta?» вопрос |
| 9 | Продавщица | Триста динара. Кешом или картицом? | Trista dinara. Kešom ili karticom? | Триста динаров. Наличными или картой? | Быстро и по-деловому: цена и вопрос про оплату |
| 10 | Алекс | Картицом. | Karticom. | Картой. | Коротко и спокойно |
| 11 | Продавщица | Само мало… Ево! Ово је број: нула шест четири седам осам пет девет један три. | Samo malo… Evo! Ovo je broj: nula šest četiri sedam osam pet devet jedan tri. | Секунду… Вот! Это номер: ноль шесть четыре семь восемь пять девять один три. | БЫСТРО, цифры подряд без пауз: «Samo malo… Evo!» легко, номер скороговоркой |
| 12 | Алекс | Извините, не разумем. Полако, молим вас. | Izvinite, ne razumem. Polako, molim vas. | Извините, не понимаю. Помедленнее, пожалуйста. | Вежливо, с извиняющейся улыбкой; «Polako, molim vas» мягко |
| 13 | Продавщица | Нула шест четири… седам осам пет… девет један три. | Nula šest četiri… sedam osam pet… devet jedan tri. | Ноль шесть четыре… семь восемь пять… девять один три. | МЕДЛЕННО и чётко, по группам по три цифры, паузы на многоточии |
| 14 | Алекс | Сад разумем. Хвала! | Sad razumem. Hvala! | Теперь понимаю. Спасибо! | С облегчением и улыбкой, коротко |
| 15 | Продавщица | Изволите пасош. Добро дошли у Србију! | Izvolite pasoš. Dobro došli u Srbiju! | Вот ваш паспорт. Добро пожаловать в Сербию! | Тепло, радушно, с улыбкой; «Dobro došli u Srbiju!» от души |
| 16 | Алекс | Хвала! Довиђења! | Hvala! Doviđenja! | Спасибо! До свидания! | Тепло и благодарно, коротко |

## Основной список: без тегов, теги только в особых местах (брать первым)

v3 и v4 играют: эмоциональные теги (`delighted`, `triumphant`, `proud`) и восклицательные знаки дают «аниме-голос». Поэтому **обычные реплики идут без тегов**, а `!` во входном тексте заменены на `.` (слова те же, что в приложении, меняется только пунктуация во входе озвучки). **Тег стоит только там, где он нужен сцене:** реплика 11, реплика 13, реплика 15. Если и так играют: Stability на максимум (Natural или Robust), Style Exaggeration 0, по одной реплике. Каждый голос читает весь список, из записи берутся только реплики своего героя (номера выше).

**Сербский (латиница):**

```
Dobar dan. Izvolite?
Dobar dan. Treba mi SIM kartica.
Treba mi pasoš, molim vas.
Evo, izvolite.
Iz Rusije ste?
Da, ali sada živim ovde.
Dobro. SIM i internet, da?
Da, da. Koliko košta?
Trista dinara. Kešom ili karticom?
Karticom.
[natural, fast, matter-of-fact voice] Samo malo… Evo. Ovo je broj: nula šest četiri sedam osam pet devet jedan tri.
Izvinite, ne razumem. Polako, molim vas.
[natural, slow, clear voice] Nula šest četiri… sedam osam pet… devet jedan tri.
Sad razumem. Hvala.
[warm, welcoming voice] Izvolite pasoš. Dobro došli u Srbiju.
Hvala. Doviđenja.
```

**Русский:**

```
Добрый день. Чем могу помочь?
Добрый день. Мне нужна сим-карта.
Мне нужен ваш паспорт, пожалуйста.
Вот, пожалуйста.
Вы из России?
Да, но теперь я живу здесь.
Хорошо. Сим-карту и интернет, да?
Да, да. Сколько стоит?
Триста динаров. Наличными или картой?
Картой.
[natural, fast, matter-of-fact voice] Секунду… Вот. Это номер: ноль шесть четыре семь восемь пять девять один три.
Извините, не понимаю. Помедленнее, пожалуйста.
[natural, slow, clear voice] Ноль шесть четыре… семь восемь пять… девять один три.
Теперь понимаю. Спасибо.
[warm, welcoming voice] Вот ваш паспорт. Добро пожаловать в Сербию.
Спасибо. До свидания.
```

## Запасная версия: теги на каждой реплике (слишком театральная на v4)

Тег описывает голос, а не действие. Нужна модель v3 или v4: на других теги прочитаются вслух. Бери отдельные реплики отсюда, только если основной список вышел плоским.

**Сербский (латиница):**

```
[friendly, professional voice] Dobar dan! Izvolite?
[polite, calm voice] Dobar dan! Treba mi SIM kartica.
[businesslike, polite voice] Treba mi pasoš, molim vas.
[friendly, helpful voice] Evo, izvolite.
[curious, friendly voice with light surprise] Iz Rusije ste?
[easy, slightly shy voice] Da, ali sada živim ovde.
[brisk, friendly voice] Dobro. SIM i internet, da?
[easy, interested voice] Da, da. Koliko košta?
[brisk, matter-of-fact voice] Trista dinara. Kešom ili karticom?
[calm voice] Karticom.
[fast, brisk, a little rushed voice] Samo malo… Evo! Ovo je broj: nula šest četiri sedam osam pet devet jedan tri.
[polite, apologetic, slightly embarrassed voice] Izvinite, ne razumem. Polako, molim vas.
[slow, clear, gentle voice] Nula šest četiri… sedam osam pet… devet jedan tri.
[relieved, smiling voice] Sad razumem. Hvala!
[warm, welcoming, smiling voice] Izvolite pasoš. Dobro došli u Srbiju!
[grateful, warm voice] Hvala! Doviđenja!
```

**Русский:**

```
[friendly, professional voice] Добрый день! Чем могу помочь?
[polite, calm voice] Добрый день! Мне нужна сим-карта.
[businesslike, polite voice] Мне нужен ваш паспорт, пожалуйста.
[friendly, helpful voice] Вот, пожалуйста.
[curious, friendly voice with light surprise] Вы из России?
[easy, slightly shy voice] Да, но теперь я живу здесь.
[brisk, friendly voice] Хорошо. Сим-карту и интернет, да?
[easy, interested voice] Да, да. Сколько стоит?
[brisk, matter-of-fact voice] Триста динаров. Наличными или картой?
[calm voice] Картой.
[fast, brisk, a little rushed voice] Секунду… Вот! Это номер: ноль шесть четыре семь восемь пять девять один три.
[polite, apologetic, slightly embarrassed voice] Извините, не понимаю. Помедленнее, пожалуйста.
[slow, clear, gentle voice] Ноль шесть четыре… семь восемь пять… девять один три.
[relieved, smiling voice] Теперь понимаю. Спасибо!
[warm, welcoming, smiling voice] Вот ваш паспорт. Добро пожаловать в Сербию!
[grateful, warm voice] Спасибо! До свидания!
```

## Для сравнения движков

Только кириллица, по одной реплике на строку (так их получает синтезатор).

```
1  Добар дан! Изволите?
2  Добар дан! Треба ми СИМ картица.
3  Треба ми пасош, молим вас.
4  Ево, изволите.
5  Из Русије сте?
6  Да, али сада живим овде.
7  Добро. СИМ и интернет, да?
8  Да, да. Колико кошта?
9  Триста динара. Кешом или картицом?
10 Картицом.
11 Само мало… Ево! Ово је број: нула шест четири седам осам пет девет један три.
12 Извините, не разумем. Полако, молим вас.
13 Нула шест четири… седам осам пет… девет један три.
14 Сад разумем. Хвала!
15 Изволите пасош. Добро дошли у Србију!
16 Хвала! Довиђења!
```
