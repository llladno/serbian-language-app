# Split lessons 00–12 into shorter lessons

**Date:** 2026-09-21
**Status:** implemented

## Why

Lessons "00"–"12" (levels 1–2, the only fully-authored part of the course)
had grown to 9–18 steps each — too long for one sitting. The fix is to split
each into 2 lessons (3 for the longest: "04", "05", "09", "10"), without
changing any teach/practice/reading/checkpoint/dialogue content — only
regrouping and renumbering steps into more, shorter lessons.

## Numbering

Full sequential renumbering, chosen over a non-cascading suffix scheme
(`"01"`/`"01b"`) despite the larger blast radius, because a clean `"00"`..`"58"`
sequence matches the precedent already set for levels 3–5 (see plan.md's
2026-09-14 entry) and keeps `course.yaml` easy to read. The trade-off — every
downstream lesson id shifts, and existing user progress in the DB would
otherwise point at the wrong content — is handled by migration `004`
(`server/internal/store/migration_hooks.go`, `migrate004`).

Levels 1–2 grew from 13 lessons (`"00"`–`"12"`) to 30 (`"00"`–`"29"`). Levels
3–5 (`"13"`–`"41"`, unwritten stubs — only titles in `course.yaml`, no
`lessons/NN.yaml` file) shifted by a uniform `+17` to `"30"`–`"58"`.

## Split rule

Each original lesson's steps are `teach`+`practice` pairs (one topic each),
plus a closing run of `reading` → dictation `practice` → `checkpoint` →
(`dialogue`, when present). The closing run always stays with the **last**
new part — it reviews the whole original lesson's vocabulary, so it can't
usefully sit before the material it tests. Topic pairs are split as evenly as
possible across the earlier parts. The two review lessons (old `"06"`,
`"12"`, no `teach` steps) split their `practice` blocks the same way, with
their `reading`/`checkpoint` steps staying in the closing part.

The full old-lesson → new-lesson-id table lives in `course.yaml` (titles) and
as the `lessonSplits` table in `server/internal/store/migration_hooks.go`
(step ranges) — that table is the single source of truth for the mapping; it
is not repeated here since it would drift.

## Vocabulary

`content/vocab.yaml` tags each word with the lesson it's first available
from. Splitting a lesson requires reassigning each of its words to whichever
new part actually uses it first (or, for the "preload" vocabulary — common
verbs/adverbs not tied to a specific `teach` step, tagged to the old lesson
so they're available for exercises many lessons later — the earliest new
part of that split, which is always guard-safe since availability only grows
forward in course order). `server/internal/content/lexicon_test.go`
(`TestLexiconGuardRealContent`, unchanged — it iterates `course.yaml` in
order, not by numeric lesson id) is what actually verifies this; a handful of
misplacements it caught during the split were fixed by moving the word's
`lesson:` tag or adding it to the relevant step's `also_ok`.

## Guard tests

`server/internal/content/real_test.go` (`TestRealContentLoads`) used to
assert `>= 9` steps per lesson with the comment *"broken into small
pieces"* — the literal opposite of this change — plus a lot of hardcoded
per-lesson-id checks. It's rewritten to check structure in aggregate across
`"00"`–`"29"` (total `checkpoint`/`reading`/`dialogue` steps, total `listen`
exercises with audio) instead of per lesson, since a single lesson no longer
necessarily carries its own reading/checkpoint.

## DB migration

Real user progress (`lesson_step_progress`, `lesson_progress`, `attempts`)
in the production Postgres DB is keyed by the old lesson/step/exercise ids.
Migration `004` (a Go hook, no schema change) relabels it using the same
`lessonSplits` table:

- `lesson_step_progress` — exact 1:1 remap (old step → new step).
- `attempts` — pure history log, same exact remap (`lesson`, `block`,
  `exercise_id`).
- `lesson_progress` — one row per whole old lesson has no single new-lesson
  equivalent (a lesson became 2–3), so its status/timestamps are copied to
  every new part. Over-optimistic in the "in_progress" case (a part the
  learner hadn't actually reached now also shows in_progress) but self-heals
  as soon as they interact with any of its steps.

All three tables are read into memory, the old-id rows deleted, then the
remapped rows (re)inserted — so a computed new id that happens to equal an
old id still pending processing in the same table (e.g. new lesson `"04"`
vs. old lesson `"04"`) can never collide mid-migration.
