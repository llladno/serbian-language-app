# Latin / Cyrillic switch — design

Status: **built 2026-10-19, not deployed.** Decisions were made in chat the same day:
convert at render time everywhere (not rewrite the content), store the choice on the
account, work-through-mistakes ships first and separately. Built straight from this
spec without a separate plan; what changed on the way is in "As built" at the end.

## Goal

A learner can read the whole course in either Serbian script. The course is written in
Latin (`čokolada`); a learner who picks Cyrillic sees `чоколада` everywhere Serbian text
appears: theory, exercises, dialogues, reading, flashcards, the dictionary, check results.
Russian text is untouched (it is already Cyrillic).

Non-goals:
- No second copy of the content, no script that rewrites `content/`.
- No Cyrillic audio: TTS and its clip keys keep working on the Latin source text.
- No change to answer checking. It already accepts Cyrillic input
  (`checker.SerbianCyrillicToLatin`, `tolerance.go`).
- No Cyrillic-first lessons or reading drills (a possible later step).

## The one rule

**Convert at the very last moment, in the component that paints the text. Everything
else keeps the Latin original.**

Components hold Latin strings and compare, hash, look up and submit them as before. Only
the characters that reach the screen go through `toScript()`. This is what keeps three
things from breaking at once:

1. **Choice answers.** `choice`, `word_bank` and `match` send the option text back and
   the server compares it with the Latin answer. A Cyrillic value would be marked wrong.
2. **Audio.** `phraseKey()` hashes the normalised Latin text to find the clip
   (`lib/phraseAudio.ts` ⇄ `scripts/tts.py`). A Cyrillic string finds no clip.
3. **Tap-a-word lookup.** `GlossedText` looks words up by their Latin spelling.

The alternative — converting in the API responses — was rejected for exactly these
reasons: every client-side comparison and every lookup would need a Cyrillic ⇄ Latin
shim.

## Why this is safe to do at render time

In the content, Serbian is always Latin and Russian is always Cyrillic. So "every
Latin-script run in a displayed string is Serbian" holds, apart from a short allowlist of
words that are not Serbian (brand names, English in a few notes) — see below. No field
needs to be tagged by language.

## Converter

`web/src/lib/script.ts`, a pure function plus a small composable:

- `toCyrillic(text)`; `toScript(text, script)` returns `text` unchanged for `'latin'`.
- `useScript()` exposes the reactive `script` (from the session user) and `show(text)`,
  the helper templates call.

Rules (checked against real data, below):

- The Serbian alphabet is a strict one-to-one mapping: `a б c ц č ч ć ћ d д dž џ đ ђ e f ф
  g h х i j ј k l lj љ m n nj њ o p r s š ш t u v z ž ж`.
- `lj`, `nj`, `dž` are single letters. Case: `Lj`, `LJ` → `Љ`; `Nj`, `NJ` → `Њ`; `Dž`, `DŽ` → `Џ`.
- A short exception list for words where the pair is two letters (`injekcija`, `konjunktiv`,
  `nadživeti`, `odžvakati`, `podžupan`). **The list starts empty:** no such word occurs in
  the current course.
- **Stress mark gotcha.** Stress is written as a combining acute (U+0301) after the vowel
  (and after a syllabic `r`). The letter `ć` in decomposed form is `c` + the same U+0301.
  The converter must therefore not run a blanket Unicode normalisation: it decomposes,
  then re-composes `c`+acute → `ć` (a consonant is never stressed), and `c/s/z` + caron →
  `č š ž`, and only then maps. Stress marks on vowels and `r` pass through, attached to the
  Cyrillic letter (`а́`, `р̀`).
- Anything not in the table (digits, punctuation, Cyrillic, markdown) passes through.
- The allowlist `KEEP_LATIN` (non-Serbian words that appear in displayed text) is a small
  constant and starts empty: a scan of the course text found no brand names, and the
  English words it turned up sit in fields that are never displayed (`type`, `kind`). The
  verification script reports Latin words missing from the Serbian dictionary, so the
  list is filled from real findings rather than guessed.

Markdown: `MarkdownView` converts `text` and `code_inline` tokens in a markdown-it core
rule (not code fences, links or URLs). The `[транскрипция]` after a phrase is already
Cyrillic and stays.

### Evidence

A prototype of these rules against `content/vocab.yaml`, whose 754 entries carry an
authored `cyrillic` next to `latin`: **732 match exactly.** The other 22 differ only
because the authored field drops a parenthetical note (`biti (jesam / sam / nisam)` →
`бити`), not because the conversion is wrong. A scan of all course text found 158
distinct tokens containing `nj`/`lj`/`dž`; all are ordinary digraphs (`konj`, `kuhinja`,
`počinjem`, `garsonjera`). The first naive prototype produced 56 mismatches; 34 of them
were the `ć`/stress-mark confusion above (the remaining 22 are the note differences),
which is why it is called out as a rule.

## Where it plugs in

The plan lists each component with the exact field, found by grepping for the `serbian`
class and every `v-html`; the set is:

| Area | What is shown |
|---|---|
| `MarkdownView` | theory text, inline phrases |
| `ReadingText`, `GlossedText` | reading and tappable prompts: display converted, lookup on Latin |
| `DialogueBubble` / `DialogueStep` | `sr` lines |
| `ChoiceAnswer`, `WordBankAnswer`, `MatchAnswer`, `ConjugateAnswer`, `TextAnswer` | prompts, options, bank chips, pair columns, form labels, placeholders, and the check result (`expected`, `diff` chunks, `forms`) |
| `ReviewView` | card front/back, example sentence; word shows Cyrillic as the main line and Latin second |
| `VocabView`, word-of-the-day | headword uses the authored `cyrillic` field; replaces the local, unsaved `scriptMode` toggle |
| `LessonView`, `CourseView`, `ProgressDashboard`, `MistakesView` | lesson titles and subtitles (`Привет! (Zdravo!)` → `Привет! (Здраво!)`), weak-exercise prompts, the mistakes screen |

The typed answer stays what the learner typed. `SerbianKeys` swaps its row to
`ђ ћ ј љ њ џ` in Cyrillic mode (`ч ш ж` exist on a Russian keyboard).

## What stays Latin

- **Lessons 00, 01 and 56** teach the two scripts, the special letters and the "cut"
  (ASCII) Latin of chat messages; in Cyrillic the table `č = ч` would read `ч = ч`, and
  `dj` for `đ` would have no meaning. A lesson-manifest flag `script: latin` is loaded into
  the lesson DTO (`keep_latin`); the client ignores the switch inside such a lesson and
  says so in one line.
- Hints and notes elsewhere that name a Latin letter as the subject (`` `c` — это `ц` ``)
  are found by the verification script and either marked or reworded, once.

## Storage and UI

- Account setting, so it follows the learner between the browser and the Telegram Mini
  App (separate storage): column `users.script` (`'latin'` default), migration `029`
  (the same shape as `users.timezone`, migration 013, and `SetUserTimezone`). The account
  DTO gains a `script` field; `PATCH /api/me {script}` sets it, validated to the two
  values and independent of the `name` and `timezone` fields it already takes.
- The session store caches it in `localStorage` for first paint, then trusts the server.
- Controls: Profile → Settings ("Письмо: латиница / кириллица") and a small «Лат | Кир»
  pill in the header of the lesson, the review, the vocabulary and the mistakes screens.

## Verification

`scripts/check_cyrillic.py`, part of `make lint-sr`:

1. Converts every `vocab.yaml` `latin` headword and compares with the authored
   `cyrillic` (the part before any parenthetical or `/` alternative).
2. Converts and back-converts all displayed Serbian text in the course; the round trip
   must reproduce the source (modulo stress marks).
3. Prints every word containing `nj`/`lj`/`dž` with a possible morpheme boundary
   (`in/on/kon + j`, `nad/od/pod + ž`) for a one-time human review; additions go to the
   exception list with a reason.
4. Prints single letters or digraphs in backticks or quotes outside lessons 00–01
   (letters as the subject) and Latin words missing from the Serbian dictionary
   (allowlist candidates).

The TS converter and the Python check use one shared vector file (`scripts/translit_vectors.json`),
the same way `phraseKey` is kept in step with `tts.py`.

Component tests mount each area in Cyrillic mode and assert that no Latin Serbian letters
remain in the visible text. A Go test covers `PATCH /api/me` and the lesson flag.

## Risks

- **A place that still shows Latin.** Caught by the component tests and by a manual pass
  over each screen; a miss is cosmetic and never breaks an answer, because only display
  is converted.
- **Letters as the subject outside 00–01.** Handled by item 4 above.
- **English or brand words in displayed text** would be turned into Cyrillic nonsense.
  Handled by the allowlist, filled from the dictionary check.
- **Screen readers.** Converted spans should carry `lang="sr-Cyrl"`; added where a text
  is painted by a component that already wraps it.

## Order of work

1. Converter, shared vectors, Python check (finds the exception and allowlist inputs).
2. Account storage and the setting UI, `keep_latin` flag.
3. Components, area by area, each with its Cyrillic-mode test.
4. Header pill, vocabulary screen cleanup, final manual pass.

## As built

What matches the design is not repeated. What differs or was learned on the way:

- **Lesson 56 is Latin too.** The first full run of the check found that the "Переписка"
  lesson has a whole step on "cut" Latin (`Dodji` for `Dođi`), which means nothing in
  Cyrillic. It now carries `script: latin` next to 00 and 01.
- **The check is narrower than item 2 and 4 above.** A round trip Latin → Cyrillic →
  Latin cannot catch a wrong digraph (it round-trips either way), and flagging every lone
  Latin letter produced 935 findings, almost all of them grammar endings (`-a`, `-u`),
  which are *right* in Cyrillic (`-а`, `-у`). `scripts/check_cyrillic.py` therefore checks:
  the shared vectors; the converter against the authored `cyrillic` of all 754 vocabulary
  entries; roman numerals in Russian text; Russian sentences that talk about the scripts;
  Latin words with q/w/x/y; and digraph-risk words, each of which needs a reason in
  `content/cyrillic.yaml`. Fed known-bad and known-good strings it catches all the bad and
  flags none of the good. On the course as it stands: the only digraph word was `konj`
  (one letter), two English labels in lesson 23 were reworded in Russian, and nothing else.
- **The Python copy has no round trip either**: it exists to run those checks, and agrees
  with the TypeScript converter on the shared vectors (36).
- **`phraseKey` was not touched.** The audio key is still computed from the Latin text; the
  Markdown renderer keeps the Latin original of a code span in `token.meta` for exactly this.
- **Vocabulary headwords** use the authored `cyrillic` field as the main line (Latin second);
  the dictionary card from a tapped word does the same. A word with no authored Cyrillic (a
  false friend) is converted.
- **Tests got an empty Pinia by default** (`web/src/test/setup.ts`), because the converter
  reads the signed-in learner and many component tests mount without a store.
- **Not done, deliberately:** `lang="sr-Cyrl"` on converted spans (listed under Risks); the
  toggle is not shown in the course list or the dashboard, only where Serbian is read.
- **An independent check was added after the first review.** Everything above compared the
  converter with things written in this repo (the vectors, the authored `cyrillic`), so it
  proved consistency, not Serbian. `make lint-cyr` converts every Latin word of the course
  (2642 distinct words outside the Latin-only lessons) and asks the real LibreOffice
  Cyrillic dictionary: 2602 are accepted, 40 are unknown to both dictionaries (names and
  fragments of endings), none is accepted in Latin and rejected in Cyrillic. It also
  confirms the hand-made exception list (`инјекција`, `надживети`, `поджупан` are accepted
  with it and rejected without it). It found one real bug the earlier checks missed: the
  titles of lessons 31, 47, 63 and 70 ended in a Latin `II` ("Prošlost II") that would
  have read "Прошлост ИИ"; they now use the roman-numeral character Ⅱ, and the roman-numeral
  check no longer excuses a numeral for having a Latin word next to it.


- **A full pass through the app in a browser was run after the dictionary check.** A script
  inside the page opened every step of every lesson in Cyrillic (lessons 02-77 except the
  Latin-only 56; 431 steps counted in lessons 16-77, the steps of 02-15 were scanned the same
  way in an earlier run, with the earlier steps marked done through the API so each step is
  reached by the ordinary resume logic) and collected every Latin letter run in
  `<main>`. A control run in Latin mode flagged every step, so the scanner can see. It found
  the places the unit tests had not been written for: the `meta` note of a conjugation
  exercise (ten steps), Serbian dialogues set in Markdown code blocks (lesson 12), and the
  Russian translation `ru` of a word (vocabulary list and drill, dictionary card, word of the
  day, first-encounter quiz options). All fixed with tests; the affected lessons and pages
  were then re-scanned clean. What it cannot see: dialogue turns after the first (they appear
  as the learner answers) and the answered state of an exercise; those rest on the unit
  tests. Left Latin on purpose: the second line under a Cyrillic headword and the word of
  the day, the email address, and the "Podrži" label of the support card.

