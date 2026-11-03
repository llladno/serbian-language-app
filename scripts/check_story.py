#!/usr/bin/env python3
"""Проверка спеки истории (scripts/story_specs/*.yaml) до озвучки.

1. Орфография, иекавизм, диакритика, смешение алфавитов: те же проверки, что у `make lint-sr`
   (lint_serbian.Checker), на сербском тексте предложений и вопросов.
2. «Слово встречалось раньше»: каждое сербское слово истории должно быть в текстах уроков
   фаз курса до уровня истории включительно (теория, чтение, диалоги, задания) или в словаре
   этих уроков. Иначе новичок услышит слово, которого ещё не видел.
3. Структура: номера предложений подряд, сцены есть, вопросы ссылаются на существующие
   предложения, ответ входит в варианты.

Usage:
    python3 scripts/check_story.py                    # все спеки
    python3 scripts/check_story.py drago-mi-je        # одна
    python3 scripts/check_story.py --coverage prvi-dan  # граница: доля знакомых слов после каждого урока
"""
from __future__ import annotations

import glob
import os
import sys
import unicodedata

import yaml

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import lint_serbian as L  # noqa: E402

SPECS = os.path.join(L.ROOT, "scripts", "story_specs")
NEW_WORDS_MAX = 0.15  # share of spoken words that may be absent from the level's lessons (spec field new_words)


def level_corpus(phase_id: str, through_lesson: str | None = None) -> set[str]:
    """Нижний регистр: все слова из файлов уроков фаз курса от первой до этой включительно (история уровня 2 вправе
    пользоваться словами уровня 1, ученик их уже видел) и латиница их словаря. through_lesson (поле спеки): уроки
    идут только до этого включительно, для истории, которая посильна уже в середине уровня."""
    course = yaml.safe_load(open(os.path.join(L.CONTENT, "course.yaml"), encoding="utf-8"))
    lessons: list[str] = []
    for phase in course["phases"]:
        lessons += [str(x) for x in phase["lessons"]]
        if str(phase["id"]) == str(phase_id):
            break
    else:
        raise SystemExit(f"в course.yaml нет фазы {phase_id!r}")
    if through_lesson is not None:
        if str(through_lesson) not in lessons:
            raise SystemExit(f"through_lesson {through_lesson!r}: такого урока нет в фазах до {phase_id!r}")
        lessons = lessons[: lessons.index(str(through_lesson)) + 1]
    words: set[str] = set()

    def feed(text: str) -> None:
        text = unicodedata.normalize("NFC", text)
        for m in L.TOKEN_RE.finditer(text):
            for part in m.group(0).split("-"):
                words.add(part.lower())

    for lid in lessons:
        manifest = os.path.join(L.CONTENT, "lessons", f"{lid}.yaml")
        if os.path.exists(manifest):
            feed(open(manifest, encoding="utf-8").read())
        for md in glob.glob(os.path.join(L.CONTENT, "lessons", lid, "*.md")):
            feed(open(md, encoding="utf-8").read())
    for w in yaml.safe_load(open(os.path.join(L.CONTENT, "vocab.yaml"), encoding="utf-8")):
        if str(w.get("lesson")) in lessons:
            feed(w.get("latin", ""))
    return words


def serbian_items(spec: dict):
    for s in spec["sentences"]:
        yield L.Item("must", f"{spec['slug']}:{s['n']}", s["sr"])
    for q in spec.get("quiz_inline") or []:
        for o in q["options"]:
            yield L.Item("must", f"{spec['slug']}:inline@{q['after']}", o)
    for i, q in enumerate(spec.get("quiz_final") or [], 1):
        loc = f"{spec['slug']}:final{i}"
        if q["kind"] == "word":
            yield L.Item("must", loc, q["prompt"])
        elif q["kind"] == "blank":
            for o in q["options"]:
                yield L.Item("must", loc, o)
        elif q["kind"] == "bank":
            for w in q.get("extra") or []:
                yield L.Item("must", loc, w)
        elif q["kind"] == "match":
            for sr, _ru in q["pairs"]:
                yield L.Item("must", loc, sr)


def structure_problems(spec: dict) -> list[str]:
    bad: list[str] = []
    ns = [s["n"] for s in spec["sentences"]]
    if ns != list(range(1, len(ns) + 1)):
        bad.append(f"номера предложений не идут подряд: {ns}")
    if spec.get("difficulty", 2) not in (1, 2, 3):
        bad.append(f"difficulty {spec.get('difficulty')!r}: нужно 1, 2 или 3")
    scene_ids = {sc["id"] for sc in spec["scenes"]}
    for s in spec["sentences"]:
        if s["scene"] not in scene_ids:
            bad.append(f"предложение {s['n']}: нет сцены {s['scene']}")
        if s["who"] not in spec["voices"]:
            bad.append(f"предложение {s['n']}: нет голоса для {s['who']}")
    scenes = [s["scene"] for s in spec["sentences"]]
    if scenes != sorted(scenes):
        bad.append("сцены идут не по порядку")
    for q in spec.get("quiz_inline") or []:
        if q["after"] not in ns:
            bad.append(f"быстрый вопрос после несуществующего предложения {q['after']}")
        if q["answer"] not in q["options"]:
            bad.append(f"быстрый вопрос после {q['after']}: ответа нет среди вариантов")
        if not 2 <= len(q["options"]) <= 3:
            bad.append(f"быстрый вопрос после {q['after']}: нужно 2–3 варианта")
    by_n = {s["n"]: s for s in spec["sentences"]}
    for i, q in enumerate(spec.get("quiz_final") or [], 1):
        kind = q["kind"]
        if kind not in ("plot", "word", "listen", "who", "blank", "bank", "match"):
            bad.append(f"финальный вопрос {i}: неизвестный вид {kind}")
            continue
        if kind in ("listen", "who", "blank", "bank") and q.get("sentence") not in by_n:
            bad.append(f"финальный вопрос {i} ({kind}): нет предложения {q.get('sentence')}")
        if kind in ("plot", "word", "listen", "who"):
            if q.get("answer") not in q.get("options", []):
                bad.append(f"финальный вопрос {i} ({kind}): ответа нет среди вариантов")
        if kind == "blank":
            sent = by_n.get(q.get("sentence"), {}).get("sr", "")
            if q["mask"].lower() not in sent.lower():
                bad.append(f"финальный вопрос {i} (blank): «{q['mask']}» нет в предложении")
            if q["mask"] not in q["options"]:
                bad.append(f"финальный вопрос {i} (blank): слова нет среди вариантов")
        if kind == "match":
            if not 2 <= len(q["pairs"]) <= 5:
                bad.append(f"финальный вопрос {i} (match): нужно 2–5 пар")
            text = " ".join(s["sr"] for s in spec["sentences"]).lower()
            for sr, _ru in q["pairs"]:
                if sr.lower() not in text:
                    bad.append(f"финальный вопрос {i} (match): «{sr}» не звучит в истории")
    return bad


def lesson_order(phase_id: str) -> list[str]:
    course = yaml.safe_load(open(os.path.join(L.CONTENT, "course.yaml"), encoding="utf-8"))
    out: list[str] = []
    for phase in course["phases"]:
        out += [str(x) for x in phase["lessons"]]
        if str(phase["id"]) == str(phase_id):
            break
    return out


def coverage(path: str) -> int:
    """Граница истории: после какого урока знакомо не меньше 85% слов (то есть незнакомых не больше NEW_WORDS_MAX).
    Берётся самый ранний такой урок; его пишут в спеку как recommended_lesson."""
    spec = yaml.safe_load(open(path, encoding="utf-8"))
    spoken = [p.lower() for s in spec["sentences"] for m in L.TOKEN_RE.finditer(unicodedata.normalize("NFC", s["sr"]))
              for p in m.group(0).split("-")]
    print(f"== {spec['slug']}: {len(spoken)} слов в репликах (recommended_lesson сейчас: {spec.get('recommended_lesson', 'нет')})")
    first = None
    for lid in lesson_order(spec["level"]):
        known = level_corpus(spec["level"], lid)
        share = sum(1 for w in spoken if w in known) / max(len(spoken), 1)
        mark = ""
        if share >= 1 - NEW_WORDS_MAX and first is None:
            first, mark = lid, "  <- граница: с этого урока историю можно слушать без проблем"
        print(f"  после {lid:5} знакомо {share:6.1%}{mark}")
    if first is None:
        print("  ни после одного урока не набирается 85%: в истории слишком много незнакомых слов")
        return 1
    print(f"  recommended_lesson: \"{first}\"")
    return 0


def main() -> int:
    args = sys.argv[1:]
    if args and args[0] == "--coverage":
        if len(args) < 2:
            print("usage: check_story.py --coverage <slug>")
            return 1
        return coverage(os.path.join(SPECS, f"{args[1]}.yaml"))
    only = args[0] if args else None
    paths = sorted(glob.glob(os.path.join(SPECS, "*.yaml")))
    if only:
        paths = [p for p in paths if os.path.basename(p) == f"{only}.yaml"]
    if not paths:
        print("нет спек историй")
        return 1
    chk = L.Checker(L.ensure_dictionary(), L.load_config())
    failed = 0
    for path in paths:
        spec = yaml.safe_load(open(path, encoding="utf-8"))
        print(f"== {spec['slug']} ({len(spec['sentences'])} предложений)")
        problems = structure_problems(spec)
        for item in serbian_items(spec):
            for f in L.check_item(chk, item):
                problems.append(f"{f.cat} {f.loc}: «{f.token}» {f.msg}")
        corpus = level_corpus(spec["level"], spec.get("through_lesson"))
        unseen: dict[str, list[str]] = {}
        for item in serbian_items(spec):
            text = unicodedata.normalize("NFC", item.text)
            for m in L.TOKEN_RE.finditer(text):
                for part in m.group(0).split("-"):
                    if part.lower() not in corpus:
                        unseen.setdefault(part.lower(), []).append(item.loc)
        # new_words: unseen words the author accepts (guessable from the translation); everything else unseen still fails,
        # and the allowed ones may not exceed NEW_WORDS_MAX of the words spoken in the sentences.
        allowed = {w.lower() for w in spec.get("new_words") or []}
        spoken = [p.lower() for s in spec["sentences"] for m in L.TOKEN_RE.finditer(unicodedata.normalize("NFC", s["sr"]))
                  for p in m.group(0).split("-")]
        new_spoken = [w for w in spoken if w in allowed and w not in corpus]
        for w, locs in sorted(unseen.items()):
            if w not in allowed:
                where = f"уроках уровня {spec['level']}" + (f" до {spec['through_lesson']}" if spec.get("through_lesson") else "")
                problems.append(f"НЕ ВСТРЕЧАЛОСЬ в {where}: «{w}» ({', '.join(sorted(set(locs)))})")
        if allowed:
            share = len(new_spoken) / max(len(spoken), 1)
            note = f"новых слов в репликах: {len(new_spoken)} из {len(spoken)} ({share:.1%}), разрешено до {NEW_WORDS_MAX:.0%}"
            if share > NEW_WORDS_MAX:
                problems.append("СЛИШКОМ МНОГО НОВЫХ СЛОВ: " + note)
            else:
                print("  " + note)
        # recommended_lesson: the lesson by which at least 85% of the spoken words are known (NEW_WORDS_MAX is the same
        # 15% the author may leave unknown); the list warns learners who have not finished it yet.
        rec = spec.get("recommended_lesson")
        if rec:
            known = level_corpus(spec["level"], str(rec))
            unknown = sum(1 for w in spoken if w not in known)
            share = unknown / max(len(spoken), 1)
            if share > NEW_WORDS_MAX:
                problems.append(f"recommended_lesson {rec}: к нему не знакомо {share:.1%} слов (больше {NEW_WORDS_MAX:.0%}), выбери урок позже")
            else:
                print(f"  recommended_lesson {rec}: к нему знакомо {1 - share:.1%} слов")
        if problems:
            failed += 1
            print("\n".join("  " + p for p in problems))
        else:
            print("  OK")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
