# Курс: контент блоков 1–2 (уроки 04–12) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to
> implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Наполнить контентом все уроки уровней 1 и 2 (04–12) в модели
«манифест + шаги», доведя блоки 1–2 до состояния уроков 01–03.

**Architecture:** Движок шагов, лёгкие типы заданий, гвардия лексики и
пошаговый плеер уже готовы (веха `feature/lesson-steps`). Этот план —
только контент: `content/lessons/NN.yaml` + `content/lessons/NN/*.md` +
слова в `content/vocab.yaml` + озвучка. Уроки 04–05 переезжают из легаси
`.md` в манифест; 06–12 пишутся с нуля. 06 и 12 — уроки-проверки
(review + ролёвки, без новых слов).

**Tech Stack:** YAML-манифесты, Markdown-фрагменты, Go-гварды
(`server/internal/content`), `scripts/tts.py` (edge-tts).

**Spec:** `docs/superpowers/specs/2026-09-08-lesson-steps-design.md`
(наполнение 06+ там значится как не-цель предыдущей вехи — этот план её
закрывает). Карта тем — `/Users/grisha/plans/serbian/plan.md` и
`content/course.yaml`.

**Branch:** продолжаем на `feature/lesson-steps` (форк от `main`).

## Global Constraints

- **Гвардия лексики** (`server/internal/content/lexicon_test.go`): в
  манифест-уроке каждый сербский токен в `accept` / `options` / `bank` /
  сербской стороне `pairs` / `say` обязан быть в накопительном словаре к
  этому уроку. Накопительный словарь урока N = все `vocab.yaml` с
  `lesson` ≤ N по порядку `course.yaml` **+** `teaches:` урока **+**
  `also_ok:` шага **+** `content/allow-words.yaml`. `coveredByPrefix`:
  известное слово — литеральный префикс токена ИЛИ общий префикс ≥ 4 рун.
  Нарушение — красная сборка.
- **Порядок сложности** в шаге `kind: practice` без `mixed: true` — ранги
  не убывают: `choice`=1; `match`,`fill_blank`=2; `word_bank`,`fix_error`=3;
  `conjugate`,`listen`=4; `translate`=5; `free`=6. `checkpoint` и
  `mixed: true` — исключены.
- **Прятанье ответов:** `accept` / `answer` / соответствие `pairs` / `say`
  клиенту не уходят — не полагаться на них в `prompt`.
- **`word_bank`:** каждый токен каждого `accept` (после `Normalize`+split)
  обязан быть среди `bank` (можно с лишними фишками-отвлекалками).
- **`choice`:** `answer` обязан быть среди `options`; `options` ≥ 2,
  уникальны.
- **`match`:** 2–6 пар, левые уникальны, правые уникальны.
- **`conjugate`:** ровно 6 форм, `accept` — список из 6 списков.
- **Персона:** ядро уроков — нейтральные Ana / Marko. `{name_latin}` /
  `{city}` / `{city_ru}` / `{job}` / `{job_ru}` / `{native_ru}` — только в
  `md` / `prompt` / `sample`, точечно, где речь «от лица ученика». В
  `accept` / `answer` / `bank` подстановки НЕ работают — если имя нужно в
  ответе, писать явным словом + `also_ok`.
- **Тон теории:** как в `content/lessons/01/1-pozdravi.md` — ≤ ~180 слов,
  одна мысль на `teach`-шаг, каждый сербский пример глоссируется
  по-русски в скобках/курсивом (см. память «Serbian: always translate»).
  Комментарии в коде — по-английски; контент — русский/сербский.
- **Плотность шага** (эталон 01–03): `practice` — 4–5 упражнений от
  `choice`/`match` к `word_bank`; `reading` — 2 вопроса `choice`;
  шаг-диктант `mixed: true` — ровно 3 `listen`; `checkpoint` — 6 упражнений
  (choice → match → fill_blank → word_bank → translate → free).
- **Размер урока:** 04–05 и 07–11 — 9–13 шагов; 06 и 12 — 7–9 шагов.
  Первый шаг всегда `teach`. Каждый урок содержит ≥ 1 `reading`, ровно
  1 шаг-диктант с 3 `listen`, ровно 1 `checkpoint` последним.
- **Cyrillic:** сербская сторона везде латиницей (движок сам
  транслитерирует для TTS); кириллица — только русские глоссы и
  `cyrillic:` в `vocab.yaml`.

---

## File Structure

Создаётся / изменяется:

- `content/vocab.yaml` — **+~180 записей** (новые слова 04–11), несколько
  ретегов служебных слов. Формат записи — как в файле (id, emoji, latin,
  cyrillic, ru, note?, lesson, pos, gender?/aspect?, tags).
- `content/allow-words.yaml` — добавить базовые вопросительные слова и
  повторяющиеся падежные формы/имена собственные.
- `content/course.yaml` — у записей 04–12 проставить
  `file: "lessons/NN.yaml"`.
- `content/lessons/04.yaml … 12.yaml` — **9 манифестов** (создать).
- `content/lessons/04/ … 12/` — **~45 md-фрагментов** (создать).
- `content/lessons/04-brojevi-i-novac.md`, `content/lessons/05-u-prodavnici.md`
  — **удалить** (контент сохранён в git-истории; нужные диалоги/таблицы
  перенести в новые фрагменты).
- `content/exercises/04.yaml`, `content/exercises/05.yaml` — **удалить**.
- `content/audio/*.mp3` — новые файлы от `scripts/tts.py` (слова + `listen`).
- `server/internal/content/real_test.go` — обновить ожидания для 04–12.
- `server/internal/content/manifest_test.go` (или где лежит
  `TestLegacyLessonSynthesizesSteps`) — репойнт легаси-теста на
  `testdata/content` (там легаси-урок `01.md` + `exercises/01.yaml`).
- `/Users/grisha/plans/serbian/plan.md` — отметить 04–12 как готовые.
- `CLAUDE.md` — одна строка: блоки 1–2 полностью в манифестах.

**Interfaces (общие для всех уроков):**

- Каждый `NN.yaml`: `lesson: "NN"`, `title`, `subtitle` (совпадают с
  `course.yaml`), `teaches: [...]` (id из `vocab.yaml`), `steps: [...]`.
- Каждый шаг: `id: "NN.k"` (k по порядку), `kind`, `title`; `teach`/
  `reading` → `md: "NN/k-slug.md"`; `practice`/`checkpoint` → `exercises`.
- Синтетический блок упражнений на шаг: `Course.Exercises["NN"]` получает
  по блоку с `id` = id шага — учитывать при отладке `/exercises`.

---

## Vocabulary reference (новые id по урокам)

Ниже — минимальный список id и глосс. Полную запись (emoji, cyrillic,
note, pos, tags) заполняем по образцу соседних записей того же
`tags`-класса. `lesson:` = номер урока.

### Урок 04 — числа, цена, время, возраст, телефон

Уже есть: `nula jedan dva tri cetiri pet deset dvadeset sto-num hiljadu
dinar pare novac posto sve-zajedno koliko-dugujem kartica kes sitno-04
kusur racun-04 akcija koliko-godina broj-telefona javiti-se poruka sprat
prizemlje`.

Добавить:
- `sest` (6), `sedam` (7), `osam` (8), `devet` (9)
- `jedanaest` (11), `dvanaest` (12), `petnaest` (15) — образец «-naest»
- `trideset` (30), `cetrdeset` (40), `pedeset` (50)
- `dvesta` (200), `trista` (300)
- `godina` (год / года — возраст и счёт)
- `sat-cas` (`sat` — час; «Koliko je sati?», «dva sata», «pet sati»)
- `koliko-je-sati` (phrase «Koliko je sati?» — сколько времени)
- `u-koliko-sati` (phrase «U koliko sati?» — во сколько)
- `pola-time` (`pola` — половина; «pola tri» = 2:30)
- `podne` (полдень), `ponoc` (полночь)
- `ujutru` (утром), `popodne` (днём), `uvece` (вечером) — общие для 04 и 07
- `tacno-adv` (`tačno` — ровно / точно)

### Урок 05 — покупки

Уже есть весь бытовой набор магазина (см. `vocab.yaml` секция «Урок 05»).
Добавить только материал для упражнений:
- `mleko` (молоко), `sir` (сыр), `jaja` (яйца), `voda` (вода)
- `litar` (литр), `deka` (100 г — dag, на рынке)
- `samo-gledam` (phrase «Samo gledam» — просто смотрю)

### Урок 06 — Провера 1

Новых слов нет.

### Урок 07 — распорядок дня, настоящее время, дни недели

Добавить:
- Глаголы-рутина: `ustati` (вставать — ustajem), `spavati` (спать — spavam),
  `tusirati-se` (принимать душ — tuširam se), `oblaciti-se` (одеваться —
  oblačim se), `dorucкovati`→`doruckovati` (завтракать — doručkujem),
  `spremati-se` (собираться — spremam se), `odmarati` (отдыхать — odmaram),
  `zavrsavati`→`zavrsiti` (заканчивать — završim), `poceti`→`poчeti`… →
  используем `poceti` (начинать — počnem), `morati` (быть должным — moram)
- Приёмы пищи: `dorucak` (завтрак), `rucak` (обед), `vecera` (ужин)
- Место работы: `posao` (работа — «idem na posao», «na poslu»)
- Дни: `ponedeljak utorak sreda cetvrtak petak subota nedelja`
  (`nedelja` = и «воскресенье», и «неделя»), `vikend` (выходные),
  `radni-dan` (будни)
- Частота: `obicno` (обычно), `uvek` (всегда), `ponekad` (иногда),
  `nikad` (никогда), `svaki-dan` (каждый день), `cesto` (часто),
  `retko` (редко), `rano` (рано), `kasno` (поздно)
- `volim-da` (phrase «volim da…» + наст. время), `moram-da` (phrase)

### Урок 08 — дом

Добавить:
- `kuca` (дом), `stan` (квартира), `zgrada` (здание), `ulaz` (подъезд),
  `lift` (лифт)
- Комнаты: `soba` (комната), `kuhinja` (кухня), `kupatilo` (ванная/туалет),
  `spavaca-soba` (спальня), `dnevni-boravak` (гостиная), `hodnik` (коридор),
  `terasa` (терраса), `balkon` (балкон)
- Мебель/вещи: `sto-meb` (`sto` — стол), `stolica` (стул), `krevet` (кровать),
  `orman` (шкаф), `frizider` (холодильник), `sporet` (плита),
  `sudopera` (раковина), `lampa` (лампа), `polica` (полка), `tepih` (ковёр),
  `ogledalo` (зеркало), `slika` (картина)
- Проёмы: `vrata` (дверь), `prozor` (окно), `kljuc` (ключ), `zid` (стена),
  `pod` (пол)
- Глагол: `stanovati` (жить/проживать — stanujem)
- Ретег `gde` из lesson «10» → **lesson «08»** (базовое «где» нужно с дома;
  урок 10 остаётся модулем системы вопросов).

### Урок 09 — еда, напитки, кафе

Добавить:
- Еда: `meso` (мясо), `piletina` (курица), `riba` (рыба), `povrce` (овощи),
  `voce` (фрукты), `jabuka` (яблоко), `paradajz-09`→ уже есть в 05? нет —
  `paradajz` (помидор), `krompir` (картофель), `pirinac` (рис),
  `testenina` (макароны), `supa` (суп), `corba` (чорба — густой суп),
  `salata` (салат), `jogurt` (йогурт), `puter` (масло), `so-salt` (`so` —
  соль), `secer` (сахар), `kolac` (пирожное/торт), `sladoled` (мороженое),
  `palacinke` (блины)
- Напитки: `kafa` (кофе), `caj` (чай), `sok` (сок), `pivo` (пиво),
  `vino` (вино), `rakija` (ракия), `mineralna` (минералка),
  `gazirano` (газированная / negazirano)
- Кафе: `kafic` (кафе), `restoran` (ресторан), `jelovnik` (меню),
  `poruciti` (заказать — poručim), `za-mene` (phrase «Za mene…» — мне),
  `gladan` (голодный), `zedan` (жаждущий), `jesti` (есть — jedem),
  `piti` (пить — pijem), `ukusno` (вкусно), `ljuto` (остро),
  `slano` (солёно), `baksis` (чаевые), `platiti` (заплатить — platim)

### Урок 10 — вопросы (есть) + город

Вопросы уже в `vocab.yaml` секция «Урок 10» (`da-li jel ko sta koji kakav
ciji kuda odakle kada kako koliko zasto zato-sto jer zbog naravno molim-q
nisam-razumeo ponoviti sporije kako-se-kaze sta-znaci pomoci zuriti`).
`gde`, `kada`, `kako`, `koliko` — оставить в 10 ИЛИ ретегнуть точечно (см.
урок 08 про `gde`). Решение: `gde`→08, `koliko`→04, `kako`→01-effect
(добавить в `allow-words`), `kada`→07. Остальные вопросы — в 10.

Добавить (город):
- `grad` (город), `ulica` (улица), `trg` (площадь), `park` (парк),
  `posta` (почта), `banka` (банк), `bolnica` (больница), `skola` (школа),
  `fakultet` (университет), `bioskop` (кинотеатр), `pozoriste` (театр),
  `muzej` (музей), `stanica` (остановка / станция), `autobus` (автобус),
  `most` (мост), `reka` (река), `centar` (центр), `raskrsnica` (перекрёсток),
  `semafor` (светофор), `cosak` (угол)
- Направление: `levo` (налево), `desno` (направо), `pravo` (прямо),
  `blizu` (близко), `daleko` (далеко), `pored` (рядом с), `iza` (за),
  `ispred` (перед), `preko-puta` (напротив), `kod-prep` (`kod` — у / возле)
- Глаголы/фразы: `skrenuti` (свернуть — skrenem), `stici` (добраться —
  stignem), `kako-da-dodjem` (phrase «Kako da dođem do…?»),
  `gde-je` (phrase «Gde je…?»)

### Урок 11 — погода, времена года, нравится

Добавить:
- Погода: `suncano` (солнечно), `oblacno` (облачно), `kisa` (дождь),
  `pada-kisa` (phrase «pada kiša»), `sneg` (снег), `pada-sneg` (phrase),
  `vetar` (ветер), `vetrovito` (ветрено), `magla` (туман), `led` (лёд)
- Температура: `toplo` (тепло), `hladno` (холодно), `vruce` (жарко),
  `stepen` (градус), `napolju` (на улице / снаружи)
- Времена года: `prolece` (весна), `leto` (лето), `jesen` (осень),
  `zima` (зима)
- Нравится: `svidja-mi-se` (phrase «sviđa mi se»), `ne-svidja-mi-se`
  (phrase), `vise-volim` (phrase «više volim» — больше люблю)
- Природа: `priroda` (природа), `more` (море), `planina` (гора),
  `jezero` (озеро), `suma` (лес), `nebo` (небо)

### Урок 12 — Провера 2

Новых слов нет.

### allow-words.yaml — добавить

- `particles`: `kako`, `sto` (в значении «что / который» уже частично),
  `bih` (для «ja bih…» если понадобится — иначе не вводить)
- `names`: падежные формы городов/стран, которые всплывут в примерах
  (`rusiji`, `srbiji`, `novom`, `sadu`, `beogradu` — если ещё не покрыты
  префиксом), плюс `ana`, `marko` уже есть.
- Не добавлять сюда содержательную лексику — только служебное.

---

## Task 1: Словарь и allow-words (общая база)

**Files:**
- Modify: `content/vocab.yaml`
- Modify: `content/allow-words.yaml`

**Interfaces:**
- Produces: id-слова, на которые ссылаются `teaches:` уроков 04–11 и токены
  в упражнениях. Каждый id уникален в файле.

- [ ] **Step 1: Добавить слова урока 04** в `content/vocab.yaml` в конец
  секции «Урок 04» — записи из раздела «Vocabulary reference → Урок 04».
  Каждая запись по образцу соседних (`tags: ["numbers"]` / `["money"]` /
  `["time"]`). `pos: "num"` для чисел, `pos: "noun"` + `gender` для `sat`
  (`m`), `godina` (`f`), `podne`/`ponoc` (`n`/`f`), `pos: "adv"` для
  `ujutru`/`popodne`/`uvece`/`tacno`, `pos: "phrase"` для phrase-id.

- [ ] **Step 2: Добавить слова урока 05** (`mleko sir jaja voda litar deka
  samo-gledam`) в конец секции «Урок 05».

- [ ] **Step 3: Добавить секцию «Урок 07»** со словами из reference (глаголы
  `aspect: "nesv"` кроме `ustati`/`zavrsiti`/`poceti` = `sv`; дни недели
  `pos: "noun" gender: "m"` кроме `sreda`/`subota`/`nedelja` = `f`;
  частотные — `pos: "adv"`).

- [ ] **Step 4: Добавить секции «Урок 08», «Урок 09», «Урок 10 — град»,
  «Урок 11»** аналогично. `vrata` — `pos: "noun" gender: "n"` (pluralia);
  `sto` (стол) id = `sto-meb` чтобы не конфликтовать с `sto-num`.

- [ ] **Step 5: Ретеги.** `gde`: `lesson: "10"` → `lesson: "08"`.
  `koliko`: `lesson: "10"` → `lesson: "04"`. `kada`: `lesson: "10"` →
  `lesson: "07"`. `kako` — оставить `lesson: "10"`, добавить `kako` в
  `allow-words.yaml` → `particles`.

- [ ] **Step 6: allow-words.** Добавить `kako` в `particles`; при
  необходимости — падежные формы имён собственных (проверяется на Task 12,
  правится здесь).

- [ ] **Step 7: Проверка загрузки.**
  Run: `go test ./server/internal/content/ -run TestRealContentLoads -v`
  Expected: PASS по словарю (`vocab >= 60`), уроки 04/05 могут временно
  падать по структуре — это чинит Task 3–4. Достаточно, чтобы YAML
  парсился и не было дублей id (`duplicate vocab id` → фейл загрузки).

- [ ] **Step 8: Commit.**
  ```bash
  git add content/vocab.yaml content/allow-words.yaml
  git commit -m "content: vocabulary for lessons 04-11"
  ```

---

## Task 2: Проводка course.yaml + удаление легаси 04/05

**Files:**
- Modify: `content/course.yaml`
- Delete: `content/lessons/04-brojevi-i-novac.md`,
  `content/lessons/05-u-prodavnici.md`,
  `content/exercises/04.yaml`, `content/exercises/05.yaml`

**Interfaces:**
- Consumes: ничего.
- Produces: у уроков 04–12 в `course.yaml` есть `file: "lessons/NN.yaml"`.
  После этого шага загрузка контента **упадёт** (нет файлов манифестов) —
  это ожидаемо, Task 3 сразу чинит 04.

- [ ] **Step 1:** Проставить `file: "lessons/04.yaml"` … `file:
  "lessons/12.yaml"` в записях 04–12 в `content/course.yaml` (у 06/12
  сейчас нет `file:` вовсе — добавить).

- [ ] **Step 2:** Перенести нужные куски из легаси `.md` во временный
  черновик (таблицы чисел, диалоги в пекаре/магазине) — держать в
  `/private/tmp/claude-501/.../scratchpad/legacy-0405.md` на время
  Task 3–4. Затем удалить 4 легаси-файла.

- [ ] **Step 3: Commit** (частичный, сборка красная — допустимо внутри
  ветки; следующий коммит чинит).
  ```bash
  git add -A content/course.yaml content/lessons content/exercises
  git commit -m "content: wire lessons 04-12 to manifests, drop legacy 04/05"
  ```

---

## Task 3: Урок 04 — «Бројеви, цена, сати»

**Files:**
- Create: `content/lessons/04.yaml`
- Create: `content/lessons/04/1-nula-deset.md`, `04/3-do-sto.md`,
  `04/5-broj-imenica.md`, `04/7-cena.md`, `04/9-godine.md`,
  `04/11-sati.md`, `04/13-telefon.md`, `04/15-citanje.md`

**Interfaces:**
- Consumes: слова 04 из Task 1; кумулятивно — 01–03.
- Produces: `Lessons["04"]` манифест, `Course.Exercises["04"]` ≥ 5 блоков,
  ≥ 3 `listen`, ≥ 1 reading, 1 checkpoint.

**Steps (манифест, ~16 шагов):**

| id | kind | title | содержимое |
|---|---|---|---|
| 04.1 | teach | Числа 0–10 | таблица nula…deset + глоссы; `jedan/dva` по роду — одна строка |
| 04.2 | practice | Узнай число | choice ×2 (цифра→слово), match (4 числа), fill_blank (слово по цифре) |
| 04.3 | teach | 11–100 | `-naest` (jedanaest, dvanaest, petnaest…), десятки (trideset…pedeset), составные «dvadeset jedan» без «и» |
| 04.4 | practice | Считаем дальше | choice ×2, match (десятки), fill_blank, word_bank («trideset pet») |
| 04.5 | teach | Число + существительное | 1 → ед. (`jedan dinar`), 2–4 → `dva dinara`/`tri kafe`, 5+ → `pet dinara`; только узнавание, падежи — позже |
| 04.6 | practice | Сколько чего | choice ×2 (`dva dinara` vs `dva dinar`), match, fill_blank |
| 04.7 | teach | Цена | `Koliko košta?` / `Pošto?` / `Sve zajedno?` / `Koliko dugujem?` → ответ «X dinara» |
| 04.8 | practice | Спроси цену | choice ×2, fill_blank, word_bank («Koliko košta kafa?») |
| 04.9 | teach | Возраст | `Koliko imaš godina?` → `Imam … godinu / godine / godina` (по правилу 04.5) |
| 04.10 | practice | Сколько лет | choice ×2, fill_blank, word_bank |
| 04.11 | teach | Который час | `Koliko je sati?` → `… sati`; `pola`, `i petnaest`, `do`; `u … sati`, `podne`, `ponoć`, `ujutru/popodne/uveče` |
| 04.12 | practice | Сколько времени | choice ×2, match (время↔запись), fill_blank, word_bank |
| 04.13 | teach | Телефон | `broj telefona`, диктовка по цифрам («nula šest…»), `Javi mi se`, `Poslaću ti poruku` |
| 04.14 | practice | Номер и связь | choice ×2, fill_blank, word_bank |
| 04.15 | reading | В пекаре (цена) | диалог из легаси `04-…md` reading-блока (kafa, cena, kartica, kusur); 2 choice |
| 04.16 | practice | Диктант | `mixed: true`, 3 `listen`: «Koliko košta?», «Imam trideset godina.», «U koliko sati?» |
| 04.17 | checkpoint | Проверка | choice → match → fill_blank → word_bank → translate («Сколько с меня?») → free («Спроси цену кофе, скажи свой возраст и который час — 3 фразы») |

- `also_ok` по шагам: 04.6 `dinara kafe godine`; 04.8 `kosta kafu`;
  04.10 `godinu godina`; 04.12 `sati sata`; 04.15 `dajte kafu`;
  04.16 `godina`; 04.17 `dinara kafa`.
- Числа-цифры («2», «30») уже в allow-words `digits` — использовать в
  `prompt`/`options` свободно; в `accept` слов — только словами.

- [ ] **Step 1:** Написать 8 md-фрагментов (`teach`/`reading`) по образцу
  `content/lessons/01/*.md` и `02/*.md`. Таблицы чисел — из легаси
  `04-brojevi-i-novac.md` (§1–§4), reading — из его reading-блока.
- [ ] **Step 2:** Написать `content/lessons/04.yaml` по таблице выше
  (плотность как в `03.yaml`: practice 4–5 упр., checkpoint 6).
- [ ] **Step 3: Гвардия + структура.**
  Run: `go test ./server/internal/content/ -run 'Lexicon|RealContent|Difficulty' -v`
  Expected: PASS. Фейлы гвардии → добавить токен в `also_ok` шага или в
  `teaches` (если это реально новое слово урока).
- [ ] **Step 4: Commit.**
  ```bash
  git add content/lessons/04.yaml content/lessons/04
  git commit -m "content: lesson 04 (numbers, price, clock) as manifest"
  ```

---

## Task 4: Урок 05 — «Куповина»

**Files:**
- Create: `content/lessons/05.yaml`
- Create: `content/lessons/05/1-gde-kupujem.md`, `05/3-ulazim.md`,
  `05/5-koliko.md`, `05/7-jos-nesto.md`, `05/9-placanje.md`,
  `05/11-reklamacija.md`, `05/13-red.md`, `05/15-citanje.md`

**Interfaces:**
- Consumes: слова 05 + 04 (цена, оплата) + 01–03.
- Produces: `Lessons["05"]` манифест, ≥ 5 блоков, ≥ 3 `listen`.

**Steps (~15):**

| id | kind | title | содержимое |
|---|---|---|---|
| 05.1 | teach | Куда идёшь | `prodavnica pekara pijaca apoteka samoposluga trafika mesara piljara`; ⚠ `magacin` = склад |
| 05.2 | practice | Какой магазин | choice ×2, match (магазин↔что покупаешь), fill_blank |
| 05.3 | teach | Заходишь | `Izvolite?/Recite?` → `Treba mi… / Dajte mi… / Da li imate…? / Samo gledam` |
| 05.4 | practice | Первые слова | choice ×2, fill_blank, word_bank («Treba mi hleb.») |
| 05.5 | teach | Сколько | `kilo, pola kile, komad, pakovanje, litar, deka`; «Koliko?» → «Pola kile.» |
| 05.6 | practice | Мера | choice ×2, match, fill_blank, word_bank |
| 05.7 | teach | Ещё что-нибудь | `Još nešto?` → `To je sve / Još ovo`; `Kesa? Kesa se plaća` |
| 05.8 | practice | Заканчиваем | choice ×2, fill_blank, word_bank |
| 05.9 | teach | Оплата | повтор из 04: `Kešom ili karticom? Imate li sitno? Evo, tačno. Račun, molim.` |
| 05.10 | practice | Плачу | choice ×2, fill_blank, word_bank |
| 05.11 | teach | Что-то не так | `Ovo je pokvareno. Gde piše rok trajanja? Mogu li da vratim / zamenim?` |
| 05.12 | practice | Возврат | choice ×2, fill_blank, word_bank |
| 05.13 | teach | Очередь и часы | `red, Ko je poslednji?, Ja sam za vama, radno vreme, otvoreno/zatvoreno` |
| 05.14 | practice | В очереди | choice ×2, match, fill_blank |
| 05.15 | reading | В пекарне очередь | reading-блок из легаси `05-…md`; 2 choice |
| 05.16 | practice | Диктант | `mixed: true`, 3 `listen`: «Treba mi hleb.», «Koliko dugujem?», «To je sve, hvala.» |
| 05.17 | checkpoint | Проверка | choice → match → fill_blank → word_bank → translate («Дайте мне полкило сыра.») → free («Зайди в магазин, попроси 2 вещи, расплатись — 3–4 фразы») |

- `also_ok`: 05.4 `hleb`; 05.6 `kile sira mleka`; 05.12 `vratim zamenim`;
  05.17 `sira`.

- [ ] **Step 1:** md-фрагменты (таблицы из легаси `05-u-prodavnici.md`).
- [ ] **Step 2:** `05.yaml`.
- [ ] **Step 3:** `go test ./server/internal/content/ -run 'Lexicon|RealContent|Difficulty' -v` → PASS.
- [ ] **Step 4: Commit** `content: lesson 05 (shopping) as manifest`.

---

## Task 5: Урок 06 — «Провера 1»

**Files:**
- Create: `content/lessons/06.yaml`
- Create: `content/lessons/06/1-citanje.md`

**Interfaces:**
- Consumes: только слова 01–05 (новых нет). `teaches: []`.
- Produces: `Lessons["06"]` манифест, `Planned = false`.

**Steps (~8, все `mixed: true` или checkpoint — порядок сложности не давит):**

| id | kind | title | содержимое |
|---|---|---|---|
| 06.1 | reading | День в Нови-Саде | сводный диалог: приветствие → знакомство → пекара (цена) → магазин; 2 choice |
| 06.2 | practice `mixed` | Привет и знакомство | 5 упр. из лексики 01–02 (choice, match, fill_blank, word_bank, translate) |
| 06.3 | practice `mixed` | Семья | 5 упр. из 03 (`imam/nemam`, `moj/tvoj`) |
| 06.4 | practice `mixed` | Числа, цена, время | 5 упр. из 04 |
| 06.5 | practice `mixed` | В магазине | 5 упр. из 05 |
| 06.6 | checkpoint | Большая проверка | 8 упр. вперемешку по всем темам (choice→…→translate) |
| 06.7 | checkpoint | Ролёвки | 3 `free`: (a) «Познакомься на улице: имя, откуда, кем работаешь», (b) «Купи два кофе в пекаре, спроси цену, заплати картой», (c) «В магазине: попроси полкило сыра и хлеб, верни испорченный йогурт» — каждое с `sample` |

- Никаких новых токенов; при фейле гвардии — только `also_ok` с уже
  known-формами.

- [ ] **Step 1:** `06/1-citanje.md` (сводный диалог SR/RU).
- [ ] **Step 2:** `06.yaml`.
- [ ] **Step 3:** `go test ./server/internal/content/ -run 'Lexicon|RealContent|Difficulty' -v` → PASS.
- [ ] **Step 4: Commit** `content: lesson 06 (checkpoint 1)`.

---

## Task 6: Milestone 1 — блок 1 зелёный

**Files:**
- Modify: `server/internal/content/real_test.go`
- Modify: тест-файл с `TestLegacyLessonSynthesizesSteps`

- [ ] **Step 1:** В `real_test.go`: убрать из `{"04","05"}` проверку
  `Planned`/блоков как легаси — теперь это манифесты (`l.Manifest == true`),
  но `c.Exercises[id] >= 5` и `listen >= 3` остаются (манифест их
  заполняет). Для 04/05 добавить проверку `len(l.Steps) >= 9` и
  `kinds["checkpoint"] >= 1`.
- [ ] **Step 2:** `06` больше не `Planned` — заменить проверку на
  `!c.Lessons["06"].Planned && len(c.Lessons["06"].Steps) >= 6`. Список
  «checkpoint planned» → `{"18","24","30"}`.
- [ ] **Step 3:** `TestLegacyLessonSynthesizesSteps` — репойнт на
  `Load("testdata/content")`, урок `"01"` (легаси `.md` + `exercises/01.yaml`),
  проверки те же (teach → practice(A) → reading, ≥ 3 шага).
- [ ] **Step 4: Полный прогон.**
  Run: `make test`
  Expected: PASS (go + vitest). Фронт-тесты не затронуты.
- [ ] **Step 5:** `python3 scripts/tts.py` — озвучка новых слов 04–05 и
  `listen`-упражнений 04/05/06. Проверить, что файлы легли в
  `content/audio/` (напр. `04.16.1.mp3`).
- [ ] **Step 6: Commit.**
  ```bash
  git add server/internal/content content/audio
  git commit -m "test: block 1 (04-06) guards; audio"
  ```
- [ ] **Step 7:** `make build` → успех; `git checkout server/web/dist/.gitkeep`.
  **STOP — чекпойнт для ревью.** Показать пользователю уроки 04–06 в
  браузере (`make dev`), дождаться «ок» перед блоком 2.

---

## Task 7: Урок 07 — «Дан по дан»

**Files:**
- Create: `content/lessons/07.yaml`
- Create: `content/lessons/07/1-prezent.md`, `07/3-moj-dan.md`,
  `07/5-dani.md`, `07/7-koliko-cesto.md`, `07/9-volim-da.md`,
  `07/11-citanje.md`

**Interfaces:**
- Consumes: слова 07 + глаголы из `vocab.yaml` секции 02 (raditi, živeti,
  ići, čitati, …) + 01–06.
- Produces: `Lessons["07"]`, ≥ 5 блоков, ≥ 3 `listen`.

**Steps (~12):**

| id | kind | title | содержимое |
|---|---|---|---|
| 07.1 | teach | Настоящее время: 3 типа | окончания `-am / -im / -em` на знакомых глаголах (`imam, radim, idem`); таблица 6 форм для одного глагола каждого типа |
| 07.2 | practice | Какое окончание | choice ×2, match (глагол↔тип), fill_blank |
| 07.3 | teach | Мой день | `ustajem, tuširam se, doručkujem, idem na posao, vraćam se, spavam`; порядок дня |
| 07.4 | practice | По порядку | choice ×2, match (действие↔перевод), fill_blank, word_bank |
| 07.5 | teach | Дни недели | `ponedeljak…nedelja`, `vikend`; «u ponedeljak», «radnim danima» |
| 07.6 | practice | Какой день | choice ×2, match, fill_blank |
| 07.7 | teach | Как часто | `obično, uvek, ponekad, nikad, često, retko, svaki dan`; место в предложении |
| 07.8 | practice | Частота | choice ×2, fill_blank, word_bank |
| 07.9 | teach | volim da / moram da | `volim da radim`, `moram da idem`, `hoću da spavam` + наст. время |
| 07.10 | practice | Хочу и должен | choice ×2, fill_blank, word_bank |
| 07.11 | reading | Обычный вторник | текст «мой день» от Ana; 2 choice |
| 07.12 | practice | Диктант | `mixed`, 3 `listen`: «Ustajem u sedam.», «Radim svaki dan.», «Volim da čitam.» |
| 07.13 | practice | Спряжение | `mixed: true`: 2 `conjugate` (`raditi` тип I, `ići` тип E) + 1 `translate` |
| 07.14 | checkpoint | Проверка | choice → match → fill_blank → word_bank → translate («Я встаю в семь.») → free («Расскажи свой будний день: 4 фразы») |

- `also_ok`: 07.2 `radim radiš radi idem ideš imam`; 07.4 `ustajem
  tuširam doručkujem vraćam spavam posao`; 07.10 `radim idem spavam
  čitam`; 07.12 `ustajem sedam`; 07.13 формы `raditi`/`ići`.
- Порядок: 07.13 после reading — держать `mixed: true` (conjugate=4 перед
  translate=5 ок, но безопаснее пометить).

- [ ] **Step 1:** md-фрагменты (таблицы спряжения — компактные, 3 типа).
- [ ] **Step 2:** `07.yaml`.
- [ ] **Step 3:** `go test ./server/internal/content/ -run 'Lexicon|RealContent|Difficulty' -v` → PASS.
- [ ] **Step 4: Commit** `content: lesson 07 (daily routine, present tense)`.

---

## Task 8: Урок 08 — «Кући»

**Files:**
- Create: `content/lessons/08.yaml`
- Create: `content/lessons/08/1-kuca-stan.md`, `08/3-sobe.md`,
  `08/5-stvari.md`, `08/7-gde-je.md`, `08/9-ima-nema.md`, `08/11-citanje.md`

**Interfaces:**
- Consumes: слова 08 (+ ретег `gde`) + 01–07.
- Produces: `Lessons["08"]`, ≥ 5 блоков, ≥ 3 `listen`.

**Steps (~12):**

| id | kind | title | содержимое |
|---|---|---|---|
| 08.1 | teach | Дом и квартира | `kuća, stan, zgrada, sprat, ulaz, lift`; «Živim u stanu na trećem spratu» |
| 08.2 | practice | Где живёшь | choice ×2, match, fill_blank |
| 08.3 | teach | Комнаты | `soba, kuhinja, kupatilo, spavaća soba, dnevni boravak, hodnik, terasa` |
| 08.4 | practice | Комната за комнатой | choice ×2, match (комната↔перевод), fill_blank, word_bank |
| 08.5 | teach | Вещи | `sto, stolica, krevet, orman, frižider, šporet, lampa, polica, ogledalo` |
| 08.6 | practice | Что в комнате | choice ×2, match, fill_blank |
| 08.7 | teach | Где это — u / na | `u kuhinji, u sobi, na terasi, na spratu`; «Gde je …?» (локатив только этими формами) |
| 08.8 | practice | u или na | choice ×2, fill_blank («Krevet je ___ spavaćoj sobi»), word_bank |
| 08.9 | teach | ima / nema дома | «U kuhinji ima šporet.», «Nema lifta.», «Da li ima terasa?» |
| 08.10 | practice | Есть или нет | choice ×2, fill_blank, word_bank |
| 08.11 | reading | Мой стан | текст-описание квартиры; 2 choice |
| 08.12 | practice | Диктант | `mixed`, 3 `listen`: «Kuhinja je mala.», «Nema lifta.», «Krevet je u sobi.» |
| 08.13 | checkpoint | Проверка | choice → match → fill_blank → word_bank → translate («В квартире две комнаты.») → free («Опиши свою квартиру: 4 фразы») |

- `also_ok`: 08.2 `stanu spratu`; 08.4 `sobi kuhinji kupatilu`;
  08.7/08.8 `kuhinji sobi terasi spratu spavaćoj`; 08.9 `lifta terase`.
- Локатив — новый концепт: подавать **готовыми формами**, без таблицы
  падежа. Явно сказать «полностью локатив — позже».

- [ ] **Step 1:** md-фрагменты.
- [ ] **Step 2:** `08.yaml`.
- [ ] **Step 3:** guard-тесты → PASS.
- [ ] **Step 4: Commit** `content: lesson 08 (home, u/na)`.

---

## Task 9: Урок 09 — «Храна и пиће»

**Files:**
- Create: `content/lessons/09.yaml`
- Create: `content/lessons/09/1-hrana.md`, `09/3-pice.md`,
  `09/5-gladan-zedan.md`, `09/7-u-kaficu.md`, `09/9-racun.md`,
  `09/11-kakav-je.md`, `09/13-citanje.md`

**Interfaces:**
- Consumes: слова 09 + `konobar` (02) + `hleb` (05) + 01–08.
- Produces: `Lessons["09"]`, ≥ 5 блоков, ≥ 3 `listen`.

**Steps (~14):**

| id | kind | title | содержимое |
|---|---|---|---|
| 09.1 | teach | Еда дома | `meso, piletina, riba, jaja, sir, voće, povrće, jabuka, paradajz, krompir, supa, salata` |
| 09.2 | practice | Что это | choice ×2, match, fill_blank |
| 09.3 | teach | Напитки | `voda, kafa, čaj, sok, pivo, vino, rakija, mineralna, gazirano` |
| 09.4 | practice | Что попить | choice ×2, match, fill_blank, word_bank |
| 09.5 | teach | Голоден / хочу пить | `gladan sam, žedan sam, hoću da jedem, hoću da pijem` (`jesti/jedem`, `piti/pijem`) |
| 09.6 | practice | Есть и пить | choice ×2, fill_blank, word_bank |
| 09.7 | teach | В кафе | `kafić, restoran, jelovnik, konobar`; «Molim vas jednu kafu», «Za mene …», «Još jednu, molim» |
| 09.8 | practice | Заказ | choice ×2, fill_blank, word_bank («Za mene jedno pivo.») |
| 09.9 | teach | Счёт | `Još nešto?` → `To je sve`; `Možemo li da platimo?`, `Račun, molim`, `bakšiš`, `Prijatno!` |
| 09.10 | practice | Платим | choice ×2, fill_blank, word_bank |
| 09.11 | teach | Как тебе | `ukusno, dobro je, slano, ljuto, sveže`; «Kako je …?» |
| 09.12 | practice | Вкусно? | choice ×2, match, fill_blank |
| 09.13 | reading | В кафе на Штранде | диалог заказа; 2 choice |
| 09.14 | practice | Диктант | `mixed`, 3 `listen`: «Molim vas jednu kafu.», «Gladan sam.», «Račun, molim.» |
| 09.15 | checkpoint | Проверка | choice → match → fill_blank → word_bank → translate («Мне, пожалуйста, один чай.») → free («Закажи завтрак в кафе и попроси счёт: 4 фразы») |

- `also_ok`: 09.4 `sok pivo vino vode`; 09.6 `jedem pijem`; 09.8 `kafu
  pivo jednu jedno`; 09.10 `platimo`; 09.15 `caj kafu`.

- [ ] **Step 1:** md-фрагменты.
- [ ] **Step 2:** `09.yaml`.
- [ ] **Step 3:** guard-тесты → PASS.
- [ ] **Step 4: Commit** `content: lesson 09 (food, drink, cafe)`.

---

## Task 10: Урок 10 — «Град око мене»

**Files:**
- Create: `content/lessons/10.yaml`
- Create: `content/lessons/10/1-da-li.md`, `10/3-upitne-reci.md`,
  `10/5-koji-kakav-ciji.md`, `10/7-zasto-jer.md`, `10/9-ne-razumem.md`,
  `10/11-grad.md`, `10/13-kako-da-dodjem.md`, `10/15-citanje.md`

**Interfaces:**
- Consumes: вопросы (`vocab.yaml` секция «Урок 10») + слова-город из
  Task 1 + 01–09.
- Produces: `Lessons["10"]`, ≥ 5 блоков, ≥ 3 `listen`. Самый длинный урок
  (вопросы + город).

**Steps (~16):**

| id | kind | title | содержимое |
|---|---|---|---|
| 10.1 | teach | Да / нет-вопрос | `Da li imate…?` / `Imate li…?` / разг. `Jel' imate…?`; интонация |
| 10.2 | practice | Спроси да/нет | choice ×2, fill_blank, word_bank |
| 10.3 | teach | Вопросительные слова | `ko, šta, gde, kada, kako, koliko, odakle, kuda` (сводка — многие уже знакомы) |
| 10.4 | practice | Какое слово | choice ×2, match (вопрос↔перевод), fill_blank |
| 10.5 | teach | koji / kakav / čiji | «Koji autobus?», «Kakav dan?», «Čiji je ovo ključ?» — по роду |
| 10.6 | practice | koji или kakav | choice ×2, fill_blank, word_bank |
| 10.7 | teach | Почему / потому что | `zašto`, `zato što`, `jer`, `zbog + сущ.` |
| 10.8 | practice | Почему | choice ×2, fill_blank, word_bank |
| 10.9 | teach | Не понял | `Molim? Nisam razumeo/razumela. Možete sporije? Kako se kaže…? Šta znači…?` |
| 10.10 | practice | Переспрос | choice ×2, match, fill_blank |
| 10.11 | teach | Город | `ulica, trg, park, pošta, banka, pijaca, stanica, autobus, centar, most, reka` |
| 10.12 | practice | Что где | choice ×2, match, fill_blank |
| 10.13 | teach | Как пройти | `levo, desno, pravo, blizu, daleko, pored, iza, ispred`; «Izvinite, gde je…?», «Kako da dođem do…?», «Skrenite levo» |
| 10.14 | practice | Направление | choice ×2, match, fill_blank, word_bank |
| 10.15 | reading | Спрашиваю дорогу | прохожий объясняет путь до почты; 2 choice |
| 10.16 | practice | Диктант | `mixed`, 3 `listen`: «Gde je stanica?», «Da li je daleko?», «Skrenite levo.» |
| 10.17 | checkpoint | Проверка | choice → match → fill_blank → word_bank → translate («Извините, где банк?») → free («Останови прохожего и спроси дорогу до рынка: 3–4 фразы») |

- `also_ok`: 10.6 `autobus dan ključ`; 10.8 `kiše posla`; 10.12 `stanice
  pošte`; 10.13/10.14 `skrenite idite pravo`; 10.16 `stanica daleko`.
- Локатив/аккузатив направления — готовыми фразами.

- [ ] **Step 1:** md-фрагменты (материал вопросов частично — из старого
  урока 03 «Pitanja», git-история до коммита `content: replace lesson 03`).
- [ ] **Step 2:** `10.yaml`.
- [ ] **Step 3:** guard-тесты → PASS.
- [ ] **Step 4: Commit** `content: lesson 10 (questions + city)`.

---

## Task 11: Урок 11 — «Време и природа»

**Files:**
- Create: `content/lessons/11.yaml`
- Create: `content/lessons/11/1-vreme-danas.md`, `11/3-toplo-hladno.md`,
  `11/5-godisnja-doba.md`, `11/7-svidja-mi-se.md`, `11/9-priroda.md`,
  `11/11-citanje.md`

**Interfaces:**
- Consumes: слова 11 + `vreme` (02, значение «погода») + 01–10.
- Produces: `Lessons["11"]`, ≥ 5 блоков, ≥ 3 `listen`.

**Steps (~12):**

| id | kind | title | содержимое |
|---|---|---|---|
| 11.1 | teach | Погода сегодня | `Kakvo je vreme?` → `sunčano, oblačno, pada kiša, pada sneg, vetrovito, magla` |
| 11.2 | practice | Что за окном | choice ×2, match, fill_blank |
| 11.3 | teach | Тепло / холодно | `toplo, hladno, vruće, sveže`; `napolju je …`, `… stepeni`, `ispod nule` |
| 11.4 | practice | Сколько градусов | choice ×2, fill_blank, word_bank |
| 11.5 | teach | Времена года | `proleće, leto, jesen, zima`; «Leti je vruće», «Zimi pada sneg» |
| 11.6 | practice | Какое время года | choice ×2, match, fill_blank |
| 11.7 | teach | Нравится / не нравится | `Sviđa mi se…`, `Ne sviđa mi se…`, `Više volim leto nego zimu` |
| 11.8 | practice | Что нравится | choice ×2, fill_blank, word_bank |
| 11.9 | teach | Природа | `priroda, more, planina, jezero, reka, šuma, nebo`; «Idem u prirodu» |
| 11.10 | practice | На природе | choice ×2, match, fill_blank |
| 11.11 | reading | Любимое время года | текст (Ana про осень в Нови-Саде); 2 choice |
| 11.12 | practice | Диктант | `mixed`, 3 `listen`: «Napolju je hladno.», «Pada kiša.», «Volim leto.» |
| 11.13 | checkpoint | Проверка | choice → match → fill_blank → word_bank → translate («Сегодня солнечно и тепло.») → free («Опиши сегодняшнюю погоду и любимое время года: 4 фразы») |

- `also_ok`: 11.4 `stepeni nule`; 11.6 `leti zimi leto zimu`; 11.8 `leto
  zimu more`; 11.10 `prirodu planinu`; 11.12 `kiša`.

- [ ] **Step 1:** md-фрагменты.
- [ ] **Step 2:** `11.yaml`.
- [ ] **Step 3:** guard-тесты → PASS.
- [ ] **Step 4: Commit** `content: lesson 11 (weather, seasons, likes)`.

---

## Task 12: Урок 12 — «Провера 2»

**Files:**
- Create: `content/lessons/12.yaml`
- Create: `content/lessons/12/1-citanje.md`

**Interfaces:**
- Consumes: слова 01–11 (новых нет). `teaches: []`.
- Produces: `Lessons["12"]`, `Planned = false`.

**Steps (~8):**

| id | kind | title | содержимое |
|---|---|---|---|
| 12.1 | reading | Один день | сводный текст: утро (рутина) → кафе (заказ) → город (дорога) → погода; 2 choice |
| 12.2 | practice `mixed` | Мой день | 5 упр. из 07 |
| 12.3 | practice `mixed` | Дом | 5 упр. из 08 |
| 12.4 | practice `mixed` | Еда и кафе | 5 упр. из 09 |
| 12.5 | practice `mixed` | Город и вопросы | 5 упр. из 10 |
| 12.6 | practice `mixed` | Погода | 5 упр. из 11 |
| 12.7 | checkpoint | Большая проверка | 8 упр. вперемешку |
| 12.8 | checkpoint | Ролёвки | 3 `free`: (a) «Расскажи свой будний день», (b) «Закажи обед в кафе и попроси счёт», (c) «Спроси у прохожего дорогу до банка и уточни, далеко ли» — с `sample` |

- [ ] **Step 1:** `12/1-citanje.md`.
- [ ] **Step 2:** `12.yaml`.
- [ ] **Step 3:** guard-тесты → PASS.
- [ ] **Step 4: Commit** `content: lesson 12 (checkpoint 2)`.

---

## Task 13: Milestone 2 — блок 2 зелёный + финал

**Files:**
- Modify: `server/internal/content/real_test.go`
- Modify: `/Users/grisha/plans/serbian/plan.md`, `CLAUDE.md`

- [ ] **Step 1:** `real_test.go`: 07–12 — не `Planned`, `len(Steps) >= 9`
  (12 — `>= 6`), `checkpoint >= 1`; `12` убрать из «planned». Список
  «checkpoint planned» → `{"18","24","30"}`. `len(c.Lessons) >= 28`
  оставить. Добавить: у каждого из 04–12 `kinds["reading"] >= 1` и
  ровно один `mixed`-listen-шаг с 3 `listen`.
- [ ] **Step 2:** `python3 scripts/tts.py` — озвучка всех новых слов
  07–11 и `listen` 07–12.
- [ ] **Step 3:** `make test` → PASS (go + vitest).
- [ ] **Step 4:** `make build` → успех; `git checkout server/web/dist/.gitkeep`.
- [ ] **Step 5:** Прогнать в браузере (`make dev`): открыть уроки 04–12,
  пройти по шагам, проверить аудио, `choice`/`match`/`word_bank`,
  checkpoint + free, глоссы в reading.
- [ ] **Step 6:** `serbian/plan.md` — в разделах «Уровень 1/2» отметить
  04–12 как готовые (снять «черновик»). `CLAUDE.md` — строка: «блоки 1–2
  (уроки 01–12) — полные манифесты».
- [ ] **Step 7: Commit.**
  ```bash
  git add server/internal/content content/audio serbian/plan.md CLAUDE.md
  git commit -m "test: block 2 (07-12) guards; audio; docs"
  ```
- [ ] **Step 8:** REQUIRED SUB-SKILL: `superpowers:finishing-a-development-branch`
  — прогнать тесты, показать меню (merge / PR / keep), выполнить выбор
  пользователя. Напомнить: push в `main` = редеплой прода (Dokploy).

---

## Self-Review

**Spec coverage:**
- Модель шагов, лёгкие типы, гвардия, персона — уже в коде (веха 1);
  этот план их только использует. ✔
- «Наполнение 06+» (не-цель вехи 1) — Tasks 3–12. ✔
- Карта тем `serbian/plan.md` уровни 1–2 — Tasks 3–12 один-в-один
  (04 числа/цена/сати, 05 куповина, 06 провера, 07 дан по дан, 08 кући,
  09 храна, 10 град+вопросы, 11 време, 12 провера). ✔
- «В задании только пройденные слова» — гвардия + `also_ok`, шаг «guard →
  PASS» в каждой задаче. ✔
- Ролёвки в проверках (06, 12) — `checkpoint` с 3 `free`. ✔

**Placeholder scan:** таблицы шагов — конкретные (id, kind, title,
содержимое); точные тексты упражнений пишутся при исполнении по эталону
01–03 (это авторская работа, не placeholder). Список новых слов —
поimённый. ✔

**Type consistency:**
- id шагов везде `NN.k`, растут по порядку; teach/reading нечётные,
  practice чётные — как в 01–03. ✔
- `also_ok` — только known-формы или падежные вариации known-слов. ✔
- id новых слов не конфликтуют: `sto-meb` vs `sto-num`, `so-salt`,
  `sat-cas`, `pola-time` vs существующий `pola-kile`. ✔
- 06/12 `teaches: []`, `Planned=false` после `file:` в `course.yaml`. ✔

**Scope:** 9 уроков — крупно, но однотипно и режется по урокам (каждая
задача = независимый коммит + зелёная гвардия). Milestone 1 (04–06) даёт
рабочую проверяемую поставку до начала блока 2.

**Ambiguity:**
- «Число + существительное» (04.5), локатив (08.7), направление (10.13),
  `sviđa mi se` (11.7) — во всех явно: подаём **готовыми формами**, без
  таблиц падежей, с пометкой «полностью — позже».
- Ретеги служебных слов (`gde`→08, `koliko`→04, `kada`→07) — Task 1
  Step 5, зафиксировано.
