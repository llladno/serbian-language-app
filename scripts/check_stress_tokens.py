#!/usr/bin/env python3
"""Сверка ударений ВСЕХ слов курса с Wiktionary, а не только лемм из словаря.

`check_stress.py` сравнивает одиночные леммы `vocab.yaml`. Слова, которые стоят в
записи словаря рядом с другими формами («godinu / godine / godina») или живут
только в теории, он не видит: так `godina` восемь раз подряд была записана
`годи́на` вместо `го́дина`. Этот скрипт берёт каждое слово из русских транскрипций
(словарь, ложные друзья, `[…]` после `` `латиницы` `` в теории), ищет его
статью в en.wiktionary и сверяет ударный слог.

Уровень A — все ударения Wiktionary для слова нисходящие (ȁ ȃ): это надёжно для
обеих норм, расхождение считаем ошибкой курса. Уровень B — восходящие: сербская и
хорватская норма в позиции расходятся, нужен носитель; только отчёт.

    python3 scripts/check_stress_tokens.py            # отчёт, код 1 при уровне A
    python3 scripts/check_stress_tokens.py --all      # + список уровня B
    python3 scripts/check_stress_tokens.py --refresh  # перекачать статьи

Исключение — `keep_words:` в content/stress-decisions.yaml, с обязательным
`source` (кто и по чему проверил). Кэш общий с check_stress.py.
"""
from __future__ import annotations

import argparse
import collections
import glob
import json
import os
import re
import sys
import unicodedata

import yaml

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import check_stress as cs  # noqa: E402
import check_transcriptions as ct  # noqa: E402
import translit_ru as tr  # noqa: E402

ROOT = cs.ROOT
PAIR_RE = re.compile(r"`([^`]+)`\s*(?:>\s*)?\[([^\]]+)\]")


def event_for(core: str, acute_idx):
    """Номер ударного «события» движка для слова, чей акут стоит на acute_idx."""
    if acute_idx is None:
        return 0
    for k in range(1, 16):
        if ct._acute_index(tr.translit_word(core, k)) == acute_idx:
            return k
    return None


def course_stress() -> dict[str, collections.Counter]:
    """слово -> Counter{ударный слог (1..), 0 = без ударения} по всему курсу."""
    cnt: dict[str, collections.Counter] = collections.defaultdict(collections.Counter)

    def feed(latin: str, ru: str):
        latin = re.sub(r"\s*\n>\s*", " ", latin).replace("\n", " ")
        lt, ot = latin.split(), ru.split()
        if len(lt) != len(ot):
            return
        for l, o in zip(lt, ot):
            m, mo = tr.WORD_RE.search(l), ct.CYR_RUN.search(o)
            if not m or not mo or "/" in l:
                continue
            k = event_for(m.group(0), ct._acute_index(mo.group(0)))
            if k is not None:
                cnt[m.group(0).lower()][k] += 1

    for e in yaml.safe_load(open(cs.VOCAB, encoding="utf-8")) or []:
        if e.get("transcription"):
            lat = re.sub(r"[()/]", " ", e["latin"]).split()
            ru = re.sub(r"[()/]", " ", e["transcription"]).split()
            if len(lat) == len(ru):
                feed(" ".join(lat), " ".join(ru))
    ff = yaml.safe_load(open(os.path.join(ROOT, "content", "false-friends.yaml"), encoding="utf-8")) or []
    for e in ff:
        if e.get("transcription"):
            feed(e["sr"].replace("/", " "), e["transcription"].replace("/", " "))
    for path in glob.glob(os.path.join(ROOT, "content", "lessons", "*", "*.md")):
        for para in open(path, encoding="utf-8").read().split("\n\n"):
            for m in PAIR_RE.finditer(para.replace("\n", " ")):
                if re.search(r"[а-яё]", m.group(2).lower()):
                    feed(ct._unwrap(m.group(1)), ct._unwrap(m.group(2)))
    return cnt


def keep_words() -> set[str]:
    raw = yaml.safe_load(open(cs.DECISIONS, encoding="utf-8")) or {} if os.path.exists(cs.DECISIONS) else {}
    out = set()
    for d in raw.get("keep_words") or []:
        if not (d.get("source") or "").strip():
            sys.exit(f"{cs.DECISIONS}: у «{d.get('word')}» нет source — кто/откуда проверил?")
        out.add(d["word"].lower())
    return out


def wiktionary_positions(forms: list[str]) -> dict[int, str]:
    pos: dict[int, str] = {}
    for chunk in forms:
        for f in chunk.split(","):   # Wiktionary иногда отдаёт два варианта слитно
            if "<" in f or not f.strip():
                continue
            plain, idx, kind = cs.parse_accented(f.strip())
            if idx is None:
                continue
            s = cs.lat_stress_syllable(plain, idx)
            if s >= 0:
                pos[s + 1] = kind
    return pos


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--all", action="store_true", help="показать и уровень B")
    ap.add_argument("--refresh", action="store_true", help="перекачать статьи")
    ap.add_argument("--write-queue", action="store_true", help="записать docs/stress-queue-tokens.md (очередь для носителя)")
    a = ap.parse_args()

    cnt = course_stress()
    words = sorted(w for w in cnt if len(w) >= 3 and w.isalpha())
    cache = {} if a.refresh else cs.load_cache()
    todo = [w for w in words if w not in cache]
    if todo:
        print(f"качаю из Wiktionary: {len(todo)} слов …", file=sys.stderr)
        cache.update(cs.fetch(todo))
        os.makedirs(cs.CACHE_DIR, exist_ok=True)
        json.dump(cache, open(cs.CACHE, "w", encoding="utf-8"), ensure_ascii=False, indent=0)

    keep = keep_words()
    level_a, level_b, level_c, checked = [], [], [], 0
    for w in words:
        entry = cache.get(w) or {}
        pos = wiktionary_positions(entry.get("forms") or [])
        if not pos:
            continue
        stressed = {k for k in cnt[w] if k > 0}
        if not stressed:
            continue
        checked += 1
        bad = sorted(stressed - set(pos))
        if not bad:
            # Курс совпал с Wiktionary, но восходящее ударение НЕ на первом слоге —
            # в сербской норме позиция может отличаться (нужен носитель).
            if any(k > 1 and pos.get(k) == "rising" for k in stressed):
                level_c.append((w, sorted(stressed), sorted(pos.items()), dict(cnt[w])))
            continue
        if w in keep:
            continue
        row = (w, bad, sorted(pos.items()), dict(cnt[w]))
        (level_a if all(t == "falling" for t in pos.values()) else level_b).append(row)

    print(f"слов с ударением в курсе: {len(words)}; сверено с Wiktionary: {checked}")
    print(f"уровень A (нисходящее, надёжно): {len(level_a)}   уровень B (восходящее, нужен носитель): {len(level_b)}"
          f"   C (курс = Wiktionary, но восходящее не на 1-м слоге): {len(level_c)}")
    for w, bad, pos, c in level_a:
        print(f"  A  {w:16} курс: слог {bad}   Wiktionary: {pos}   (встречается: {c})")
    if a.all:
        for w, bad, pos, c in level_b:
            print(f"  B  {w:16} курс: слог {bad}   Wiktionary: {pos}   (встречается: {c})")
    if a.write_queue:
        write_queue(level_b, level_c)
    return 1 if level_a else 0


def write_queue(level_b: list, level_c: list) -> None:
    """Очередь для носителя: слова, где Wiktionary (хорватоцентричный) даёт восходящее
    ударение в другом слоге, чем курс. Это не приговор: нужен человек, для которого
    сербский родной."""
    path = os.path.join(ROOT, "docs", "stress-queue-tokens.md")
    lines = [
        "# Очередь ударений: слова из теории (уровень B)",
        "",
        "Сгенерировано `python3 scripts/check_stress_tokens.py --write-queue`. Здесь слова,",
        "у которых Wiktionary даёт **восходящее** ударение, а курс ставит другой слог.",
        "Сербская и хорватская норма в таких словах расходятся, поэтому это очередь",
        "для носителя, а не список ошибок. Нисходящие расхождения (уровень A) исправлены",
        "и не допускаются: `make lint-sr` падает.",
        "",
        "| слово | слог в курсе | слог по Wiktionary | где встречается |",
        "|---|---|---|---|",
    ]
    for w, bad, pos, c in sorted(level_b):
        wk = ", ".join(f"{k} ({t})" for k, t in pos)
        lines.append(f"| `{w}` | {', '.join(map(str, bad))} | {wk} | {sum(c.values())} |")
    lines += ["", f"Всего: {len(level_b)}.", "",
              "## Курс совпал с Wiktionary, но ударение восходящее и не на первом слоге",
              "",
              "Здесь курс повторяет Wiktionary, а оно описывает преимущественно хорватскую норму.",
              "В сербском произношении такие слова иногда звучат с ударением на слог раньше —",
              "это стоит показать носителю.",
              "",
              "| слово | слог в курсе | по Wiktionary | где встречается |",
              "|---|---|---|---|"]
    for w, ks, pos, c in sorted(level_c):
        wk = ", ".join(f"{k} ({t})" for k, t in pos)
        lines.append(f"| `{w}` | {', '.join(map(str, ks))} | {wk} | {sum(c.values())} |")
    lines += ["", f"Всего: {len(level_c)}.", ""]
    with open(path, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))
    print(f"записано: {path}")


if __name__ == "__main__":
    sys.exit(main())
