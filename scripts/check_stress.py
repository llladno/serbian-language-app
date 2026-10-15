#!/usr/bin/env python3
"""Сверка ударений в `vocab.yaml` с Wiktionary (en.wiktionary.org, раздел Serbo-Croatian).

Правильность места ударения нельзя вывести из правил, и ни одна другая проверка
курса её не видит (см. docs/SERBIAN_ERRORS.md, класс S1). Этот скрипт берёт
ударение из статьи Wiktionary — `{{sh-IPA|kȕća}}` / `{{sh-noun|kȕća|…}}` — и
сравнивает НОМЕР УДАРНОГО СЛОГА с русской транскрипцией слова.

ВАЖНО: Wiktionary описывает прежде всего хорватскую норму, а в Сербии у части
слов ударение другое. Поэтому расхождение — это НЕ приговор, а очередь на
проверку носителем/сербским источником. Скрипт сам ничего не правит.

Что сделать с расхождением:
  * Wiktionary прав  → исправь `transcription` в vocab.yaml (через translit_ru.py),
    расхождение исчезнет само;
  * курс прав (сербская норма) → добавь запись в `content/stress-decisions.yaml`
    (кто/откуда проверил) — слово уйдёт из очереди. Решение действует, пока
    транскрипция в vocab.yaml не изменилась.

    pip install pyyaml
    python3 scripts/check_stress.py                 # отчёт в консоль
    python3 scripts/check_stress.py --write-queue   # + docs/stress-queue.md
    python3 scripts/check_stress.py --refresh       # перекачать статьи (кэш: scripts/.cache/)
    python3 scripts/check_stress.py --strict        # код возврата 1, если есть нерешённые
"""
from __future__ import annotations

import argparse
import collections
import datetime
import json
import os
import re
import sys
import time
import unicodedata
import urllib.parse
import urllib.request
from dataclasses import dataclass

import yaml

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
VOCAB = os.path.join(ROOT, "content", "vocab.yaml")
DECISIONS = os.path.join(ROOT, "content", "stress-decisions.yaml")
CACHE_DIR = os.path.join(ROOT, "scripts", ".cache")
CACHE = os.path.join(CACHE_DIR, "wiktionary-stress.json")
QUEUE = os.path.join(ROOT, "docs", "stress-queue.md")
API = "https://en.wiktionary.org/w/api.php"
UA = "ucimo-stress-check/1 (https://ucimo.ru; man30968@gmail.com)"

STRESS_MARKS = {"̀", "́", "̏", "̑"}  # grave, acute, double grave, inverted breve
LAT_VOWELS = "aeiou"
CYR_VOWELS = "аеёиоуыэюя"


# ------------------------------------------------------------------ разбор слов --

FALLING_MARKS = {"\u030f", "\u0311"}   # ȁ ȃ — нисходящее; а grave/acute (à á) — восходящее


def parse_accented(s: str) -> tuple[str, int | None, str | None]:
    """Слово с акцентами Wiktionary → (слово без знаков ударения, индекс ударной
    буквы, тип акцента "falling"/"rising").

    Идём по символам NFC, а не по общему разложению: в NFD `ć` = `c` + акут, и
    общий разбор принял бы букву за ударение. Ударение — только на гласной или на
    слоговом `r`; знак долготы (макрон) отбрасываем, это не ударение.
    """
    s = unicodedata.normalize("NFC", s)
    out: list[str] = []
    stress: int | None = None
    kind: str | None = None
    for ch in s:
        if ch in STRESS_MARKS and out and out[-1].lower() == "r":   # r̀ — r + отдельный знак
            stress = len(out) - 1
            kind = "falling" if ch in FALLING_MARKS else "rising"
            continue
        if unicodedata.combining(ch):    # макрон и пр. — отдельным символом
            continue
        d = unicodedata.normalize("NFD", ch)
        base, marks = d[0], set(d[1:])
        if base.lower() in LAT_VOWELS + "r" and marks & STRESS_MARKS:
            stress = len(out)
            kind = "falling" if marks & FALLING_MARKS else "rising"
            out.append(base)
        elif base.lower() in LAT_VOWELS + "r" and marks == {"̄"}:
            out.append(base)                 # ā ī ū — долгота без тона
        else:
            out.append(ch)
    return "".join(out), stress, kind


def lat_nuclei(word: str) -> list[int]:
    """Позиции ядер слогов латинского слова: гласные и слоговое r (между не-гласными)."""
    w = word.lower()
    nuc = []
    for i, c in enumerate(w):
        if c in LAT_VOWELS:
            nuc.append(i)
        elif c == "r":
            prev = w[i - 1] if i else ""
            nxt = w[i + 1] if i + 1 < len(w) else ""
            if prev not in LAT_VOWELS and nxt not in LAT_VOWELS:
                nuc.append(i)
    return nuc


def lat_stress_syllable(plain: str, stress_idx: int) -> int:
    nuc = lat_nuclei(plain)
    return nuc.index(stress_idx) if stress_idx in nuc else -1


def ru_stress_syllable(tr: str) -> tuple[int | None, int]:
    """Номер ударного слога и число слогов в русской транскрипции.

    Ударение — U+0301 после буквы. Акут снимаем строкой, а не разложением, чтобы
    `й` (и + краткая) не развалилось и не стало лишней гласной.
    """
    s = unicodedata.normalize("NFC", tr)
    stress = None
    letters: list[str] = []
    for ch in s:
        if ch == "́":
            stress = len(letters) - 1
        else:
            letters.append(ch)
    w = "".join(letters).lower()
    nuc = []
    for i, c in enumerate(w):
        if c in CYR_VOWELS:
            nuc.append(i)
        elif c == "р":
            prev = w[i - 1] if i else ""
            nxt = w[i + 1] if i + 1 < len(w) else ""
            if (prev not in CYR_VOWELS and nxt not in CYR_VOWELS) or i == stress:
                nuc.append(i)
    if stress is None:
        return None, len(nuc)
    return (nuc.index(stress) if stress in nuc else -1), len(nuc)


# ------------------------------------------------------------------ Wiktionary --

HEAD_TEMPLATES = ("sh-IPA", "sh-noun", "sh-verb", "sh-adj", "sh-adv", "sh-prep", "sh-conj",
                  "sh-pron", "sh-intj", "sh-num", "sh-part", "sh-proper-noun", "sh-prop-noun")
TPL_RE = re.compile(r"\{\{(sh-[a-zA-Z-]+)\|([^}]*)\}\}")


def serbo_croatian_section(wikitext: str) -> str | None:
    m = re.search(r"^==Serbo-Croatian==\s*$", wikitext, re.M)
    if not m:
        return None
    rest = wikitext[m.end():]
    nxt = re.search(r"^==[^=].*?==\s*$", rest, re.M)
    return rest[: nxt.start()] if nxt else rest


def accented_forms(wikitext: str) -> list[str] | None:
    """Акцентированные написания заглавного слова из шаблонов sh-*; None — нет раздела SC."""
    sec = serbo_croatian_section(wikitext)
    if sec is None:
        return None
    out: list[str] = []
    for name, args in TPL_RE.findall(sec):
        if name not in HEAD_TEMPLATES:
            continue
        parts = [a.strip() for a in args.split("|") if "=" not in a and a.strip()]
        out += parts if name == "sh-IPA" else parts[:1]
    return sorted(set(out))


def fetch(words: list[str]) -> dict[str, dict]:
    """Скачать статьи пачками по 50 и вернуть {слово: {"status", "forms"}}."""
    res: dict[str, dict] = {}
    for i in range(0, len(words), 50):
        batch = words[i : i + 50]
        q = urllib.parse.urlencode({
            "action": "query", "prop": "revisions", "rvprop": "content", "rvslots": "main",
            "format": "json", "formatversion": "2", "titles": "|".join(batch), "redirects": "1",
        })
        req = urllib.request.Request(f"{API}?{q}", headers={"User-Agent": UA})
        data = json.load(urllib.request.urlopen(req, timeout=60))
        norm = {n["to"]: n["from"] for n in data.get("query", {}).get("normalized", [])}
        redir = {r["to"]: r["from"] for r in data.get("query", {}).get("redirects", [])}
        for p in data["query"]["pages"]:
            src = norm.get(redir.get(p["title"], p["title"]), redir.get(p["title"], p["title"]))
            if p.get("missing"):
                res[src] = {"status": "no_page", "forms": []}
                continue
            forms = accented_forms(p["revisions"][0]["slots"]["main"]["content"])
            if forms is None:
                res[src] = {"status": "no_sh_section", "forms": []}
            else:
                res[src] = {"status": "ok", "forms": forms}
        print(f"  Wiktionary: {min(i + 50, len(words))}/{len(words)}", file=sys.stderr)
        time.sleep(1)    # вежливо к API
    for w in words:
        res.setdefault(w, {"status": "no_page", "forms": []})
    return res


def load_cache() -> dict:
    if os.path.exists(CACHE):
        return json.load(open(CACHE, encoding="utf-8"))
    return {}


# ------------------------------------------------------------------ сравнение --

@dataclass
class Row:
    latin: str
    tr: str
    status: str            # MATCH | MISMATCH | KEPT | NO_DATA | NO_ACCENT | SYLLABLES | NO_STRESS
    course: int | None = None
    wk: tuple[int, ...] = ()
    wk_forms: tuple[str, ...] = ()
    falling: bool = False      # все ударные слоги Wiktionary — нисходящие (надёжно для обеих норм)


def single_words(vocab: list[dict]) -> list[dict]:
    out = []
    for e in vocab:
        l = unicodedata.normalize("NFC", e.get("latin", "")).strip()
        if l and re.fullmatch(r"[A-Za-zČčĆćŠšŽžĐđ]+", l):
            out.append(e)
    return out


def load_decisions() -> set[tuple[str, str]]:
    if not os.path.exists(DECISIONS):
        return set()
    raw = yaml.safe_load(open(DECISIONS, encoding="utf-8")) or {}
    keep = set()
    for d in raw.get("keep") or []:
        if not (d.get("source") or "").strip():
            sys.exit(f"{DECISIONS}: у «{d.get('latin')}» нет source — кто/откуда проверил?")
        keep.add((d["latin"], unicodedata.normalize("NFC", d["transcription"])))
    return keep


def compare(vocab: list[dict], cache: dict, keep: set[tuple[str, str]]) -> list[Row]:
    rows = []
    for e in single_words(vocab):
        latin = unicodedata.normalize("NFC", e["latin"]).strip()
        tr = unicodedata.normalize("NFC", e.get("transcription", ""))
        info = cache.get(latin)
        if not info or info["status"] != "ok":
            rows.append(Row(latin, tr, "NO_DATA"))
            continue
        wk_syl, wk_forms, kinds = set(), [], set()
        for f in info["forms"]:
            plain, st, kind = parse_accented(f)
            if plain.lower() == latin.lower() and st is not None:
                s = lat_stress_syllable(plain, st)
                if s >= 0:
                    wk_syl.add(s)
                    wk_forms.append(f)
                    kinds.add(kind)
        falling = kinds == {"falling"}
        if not wk_syl:
            rows.append(Row(latin, tr, "NO_ACCENT"))
            continue
        course, n_ru = ru_stress_syllable(tr)
        if n_ru != len(lat_nuclei(latin)):
            rows.append(Row(latin, tr, "SYLLABLES", course, tuple(sorted(wk_syl)), tuple(wk_forms)))
        elif course is None:
            rows.append(Row(latin, tr, "NO_STRESS", None, tuple(sorted(wk_syl)), tuple(wk_forms)))
        elif course in wk_syl:
            rows.append(Row(latin, tr, "MATCH", course, tuple(sorted(wk_syl)), tuple(wk_forms)))
        elif (latin, tr) in keep:
            rows.append(Row(latin, tr, "KEPT", course, tuple(sorted(wk_syl)), tuple(wk_forms)))
        else:
            rows.append(Row(latin, tr, "MISMATCH", course, tuple(sorted(wk_syl)), tuple(wk_forms), falling))
    return rows


def delta(r: Row) -> int:
    return r.course - min(r.wk, key=lambda s: abs(s - r.course))


def report(rows: list[Row]) -> list[Row]:
    cnt = collections.Counter(r.status for r in rows)
    total = len(rows)
    comparable = cnt["MATCH"] + cnt["MISMATCH"] + cnt["KEPT"]
    print(f"однословных лемм в vocab.yaml: {total}")
    print(f"  сравнимо с Wiktionary: {comparable}   (совпало {cnt['MATCH']}, "
          f"расходится {cnt['MISMATCH']}, подтверждено как есть {cnt['KEPT']})")
    print(f"  нет данных в Wiktionary: {cnt['NO_DATA']}   без ударения в статье: {cnt['NO_ACCENT']}")
    print(f"  не сравнивали: число слогов не сошлось {cnt['SYLLABLES']}, "
          f"в транскрипции нет ударения {cnt['NO_STRESS']}")
    mism = sorted((r for r in rows if r.status == "MISMATCH"), key=lambda r: (not r.falling, delta(r), r.latin))
    if mism:
        d = collections.Counter(delta(r) for r in mism)
        print("\nнаправление расхождений (слог курса минус слог Wiktionary): "
              + ", ".join(f"{k:+d}: {v}" for k, v in sorted(d.items())))
        for title, group in (("A. ВЕРОЯТНАЯ ОШИБКА КУРСА — в Wiktionary нисходящее ударение (ȁ ȅ ȉ ȍ ȕ), "
                              "оно одинаково в сербской и хорватской норме",
                              [r for r in mism if r.falling]),
                             ("B. НУЖНА ПРОВЕРКА НОСИТЕЛЕМ — восходящее ударение (à á è é …), "
                              "здесь сербская норма часто отличается от хорватской",
                              [r for r in mism if not r.falling])):
            print(f"\n{title}: {len(group)}")
            for r in group:
                print(f"  {r.latin:<14} {r.tr:<18} курс: слог {r.course + 1}   Wiktionary: "
                      f"{', '.join(r.wk_forms)} (слог {', '.join(str(s + 1) for s in r.wk)})")
    syl = [r for r in rows if r.status == "SYLLABLES"]
    if syl:
        print("\nЧИСЛО СЛОГОВ НЕ СОШЛОСЬ (проверить транскрипцию/разбор):")
        for r in syl:
            print(f"  {r.latin:<14} {r.tr}")
    return mism


def write_queue(rows: list[Row]):
    mism = sorted((r for r in rows if r.status == "MISMATCH"), key=lambda r: (not r.falling, delta(r), r.latin))
    cnt = collections.Counter(r.status for r in rows)
    comparable = cnt["MATCH"] + cnt["MISMATCH"] + cnt["KEPT"]
    lines = [
        "# Очередь проверки ударений",
        "",
        "_Файл генерируется `python3 scripts/check_stress.py --write-queue`; руками не правь._",
        f"_Обновлено: {datetime.date.today().isoformat()}._",
        "",
        "Сверка ударного слога в `vocab.yaml` со статьями Wiktionary (Serbo-Croatian). "
        "**Wiktionary описывает в первую очередь хорватскую норму**, поэтому расхождение — "
        "не приговор, а повод проверить у носителя или в сербском источнике.",
        "",
        f"Сравнимо слов: **{comparable}**; совпало: **{cnt['MATCH']}**; "
        f"расходится: **{cnt['MISMATCH']}**; уже подтверждено: **{cnt['KEPT']}**.",
        "",
        "Как закрыть строку: если прав Wiktionary — поправь `transcription` в `vocab.yaml` "
        "(через `scripts/translit_ru.py`); если прав курс — добавь запись в "
        "`content/stress-decisions.yaml` с указанием источника.",
    ]
    for title, note, group in (
        ("A. Вероятная ошибка курса",
         "В Wiktionary **нисходящее** ударение (ȁ ȃ ȅ ȉ ȍ ȕ). Оно не зависит от сербской/хорватской "
         "нормы, так что расхождение скорее всего реальное. Проверять в первую очередь.",
         [r for r in mism if r.falling]),
        ("B. Нужна проверка носителем",
         "В Wiktionary **восходящее** ударение (à á è é ì í ò ó ù ú). Именно здесь сербская норма часто "
         "отличается от хорватской (`apotéka` ↔ `apòteka`), поэтому без носителя не решить.",
         [r for r in mism if not r.falling]),
    ):
        lines += ["", f"## {title} ({len(group)})", "", note, "",
                  "| Слово | Транскрипция в курсе | Слог в курсе | Wiktionary | Слог в Wiktionary | Δ |",
                  "|---|---|---|---|---|---|"]
        for r in group:
            lines.append(f"| `{r.latin}` | {r.tr} | {r.course + 1} | {', '.join(r.wk_forms)} | "
                         f"{', '.join(str(s + 1) for s in r.wk)} | {delta(r):+d} |")
    open(QUEUE, "w", encoding="utf-8").write("\n".join(lines) + "\n")
    print(f"\nочередь записана: {os.path.relpath(QUEUE, ROOT)}")


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--refresh", action="store_true", help="перекачать все статьи")
    ap.add_argument("--write-queue", action="store_true", help="записать docs/stress-queue.md")
    ap.add_argument("--strict", action="store_true", help="код возврата 1, если есть нерешённые расхождения")
    a = ap.parse_args()

    vocab = yaml.safe_load(open(VOCAB, encoding="utf-8")) or []
    words = sorted({unicodedata.normalize("NFC", e["latin"]).strip() for e in single_words(vocab)})
    cache = {} if a.refresh else load_cache()
    todo = [w for w in words if w not in cache]
    if todo:
        print(f"качаю из Wiktionary: {len(todo)} слов …", file=sys.stderr)
        cache.update(fetch(todo))
        os.makedirs(CACHE_DIR, exist_ok=True)
        json.dump(cache, open(CACHE, "w", encoding="utf-8"), ensure_ascii=False, indent=0)
    rows = compare(vocab, cache, load_decisions())
    mism = report(rows)
    if a.write_queue:
        write_queue(rows)
    return 1 if (a.strict and mism) else 0


if __name__ == "__main__":
    sys.exit(main())
