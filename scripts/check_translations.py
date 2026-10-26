#!/usr/bin/env python3
"""Check the Russian meaning of every vocabulary word against two online dictionaries.

For each word of `content/vocab.yaml` (the lessons you pick) this asks:

  RU   ru.wiktionary: the page of the Russian word lists its Serbian translations (`sr = [[доручак]]`).
       The course's Serbian word has to be one of them. This is a bilingual dictionary lookup, the
       strongest check here: «завтрак» → доручак.
  EN   en.wiktionary: the Serbo-Croatian page of the word. Does it exist, is it the part of speech and
       gender the course says, and what does it mean (English). An English gloss is printed for every word
       that RU could not confirm, so a person can compare it with the Russian by eye.

A word is CONFIRMED when RU finds it. Everything else is "to read", never "wrong": the dictionaries have
gaps and lean to the Croatian norm.

    pip install pyyaml
    python3 scripts/check_translations.py --levels 1,1b          # words of the lessons of phases 1 and 1b
    python3 scripts/check_translations.py --lesson 06 06b        # these lessons
    python3 scripts/check_translations.py --all                  # the whole vocabulary
    python3 scripts/check_translations.py --levels 1b --md out.md  # also write a Markdown report
    python3 scripts/check_translations.py --levels 1,1b --sentences  # whole phrases against Tatoeba
    python3 scripts/check_translations.py --refresh              # read the dictionaries again
"""
from __future__ import annotations

import argparse
import glob
import json
import os
import re
import sys
import time
import unicodedata
import urllib.parse
import urllib.request

import yaml

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import wiktionary_forms as W   # noqa: E402

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CONTENT = os.path.join(ROOT, "content")
RU_CACHE = os.path.join(W.CACHE_DIR, "ruwiktionary-sr.json")
RU_API = "https://ru.wiktionary.org/w/api.php"

CYR2LAT = {
    "а": "a", "б": "b", "в": "v", "г": "g", "д": "d", "ђ": "đ", "е": "e", "ж": "ž", "з": "z", "и": "i",
    "ј": "j", "к": "k", "л": "l", "љ": "lj", "м": "m", "н": "n", "њ": "nj", "о": "o", "п": "p", "р": "r",
    "с": "s", "т": "t", "ћ": "ć", "у": "u", "ф": "f", "х": "h", "ц": "c", "ч": "č", "џ": "dž", "ш": "š",
}
POS_MAP = {"noun": "Noun", "verb": "Verb", "adj": "Adjective", "adv": "Adverb", "pron": "Pronoun",
           "num": "Numeral", "prep": "Preposition", "conj": "Conjunction", "interj": "Interjection"}


def sr_to_latin(s: str) -> str:
    s = unicodedata.normalize("NFC", W.strip_accents(s)).lower()
    return "".join(CYR2LAT.get(ch, ch) for ch in s)


# ---------------------------------------------------------------- ru.wiktionary ----

def _load_ru() -> dict:
    return json.load(open(RU_CACHE, encoding="utf-8")) if os.path.exists(RU_CACHE) else {}


def _save_ru(c: dict) -> None:
    os.makedirs(W.CACHE_DIR, exist_ok=True)
    json.dump(c, open(RU_CACHE + ".part", "w", encoding="utf-8"), ensure_ascii=False)
    os.replace(RU_CACHE + ".part", RU_CACHE)


def _links(value: str) -> list[str]:
    """`[[доручак]]; [[ручак|ручак]] {{f}}` → ['доручак', 'ручак']"""
    out = []
    for m in re.finditer(r"\[\[([^\]|#]+)(?:\|[^\]]*)?\]\]", value):
        out.append(m.group(1).strip())
    return out


def parse_ru_page(text: str) -> dict:
    res: dict[str, list[str]] = {"sr": [], "hr": [], "bs": []}
    for m in re.finditer(r"(?:^|\n|\|)\s*(sr|hr|bs)\s*=\s*([^\n|}]*)", text):
        for w in _links(m.group(2)):
            if w not in res[m.group(1)]:
                res[m.group(1)].append(w)
    return res


def ru_lookup(titles: list[str], refresh: bool = False) -> dict[str, dict | None]:
    cache = _load_ru()
    need = [t for t in titles if refresh or t not in cache]
    for i in range(0, len(need), 40):
        batch = need[i:i + 40]
        q = urllib.parse.urlencode({
            "action": "query", "prop": "revisions", "rvprop": "content", "rvslots": "main",
            "format": "json", "formatversion": "2", "titles": "|".join(batch), "redirects": "1",
        })
        req = urllib.request.Request(RU_API + "?" + q, headers={"User-Agent": W.UA})
        data = None
        for attempt in range(4):
            try:
                data = json.load(urllib.request.urlopen(req, timeout=60))
                break
            except Exception:
                time.sleep(2 + 3 * attempt)
        if data is None:
            continue
        back = {}
        for k in ("normalized", "redirects"):
            for r in data.get("query", {}).get(k, []):
                back[r["to"]] = r["from"] if r["to"] not in back else back[r["to"]]
        got = {}
        for p in data["query"]["pages"]:
            src = p["title"]
            while src in back and back[src] != src:
                src = back[src]
            got[src] = None if p.get("missing") else parse_ru_page(p["revisions"][0]["slots"]["main"]["content"])
        for t in batch:
            cache[t] = got.get(t)
        _save_ru(cache)
        print(f"  ru.wiktionary: {min(i + 40, len(need))}/{len(need)}", file=sys.stderr)
        time.sleep(0.5)
    return {t: cache.get(t) for t in titles}


def russian_candidates(ru: str) -> list[str]:
    """The Russian words of a vocabulary gloss to look up: «принимать душ / мыться (в душе)» → several titles."""
    ru = re.sub(r"\([^)]*\)", " ", ru)
    ru = ru.replace("…", " ")
    parts = [p.strip().lower() for p in re.split(r"[/,;]", ru)]
    out: list[str] = []
    for p in parts:
        p = re.sub(r"[^а-яё\- ]", "", p).strip()
        if not p:
            continue
        out.append(p)
        if " " in p:
            out += [w for w in p.split() if len(w) >= 4]
    return list(dict.fromkeys(out))


# ---------------------------------------------------------------- the check ----

def vocab_for(lessons: set[str] | None) -> list[dict]:
    v = yaml.safe_load(open(os.path.join(CONTENT, "vocab.yaml"), encoding="utf-8"))
    return [e for e in v if lessons is None or str(e["lesson"]) in lessons]


def lessons_of_levels(levels: list[str]) -> set[str]:
    c = yaml.safe_load(open(os.path.join(CONTENT, "course.yaml"), encoding="utf-8"))
    ids: set[str] = set()
    for ph in c["phases"]:
        if ph["id"] in levels:
            ids.update(ph["lessons"])
    return ids


def sr_variants(latin: str) -> list[str]:
    out = []
    for part in re.split(r"\s*/\s*", latin):
        part = re.sub(r"\(.*?\)", "", part).strip().lower()
        if part:
            out.append(part)
    return out


def check(entries: list[dict], refresh: bool = False) -> list[dict]:
    # 1. English Wiktionary for every single-word variant
    singles = sorted({v for e in entries for v in sr_variants(e["latin"]) if re.fullmatch(r"[a-zčćšžđ]+", v)})
    en = W.lookup(singles, refresh=refresh)
    # 2. Russian Wiktionary for every Russian word of the glosses
    titles = sorted({c for e in entries for c in russian_candidates(e["ru"])})
    ruw = ru_lookup(titles, refresh=refresh)

    rows = []
    for e in entries:
        variants = sr_variants(e["latin"])
        ru_hits: list[str] = []      # "завтрак→доручак"
        ru_weak: list[str] = []      # only the Bosnian / Croatian column has it
        ru_has_sr = False            # a Russian page listed Serbian words, and ours was not among them
        ru_lists: list[str] = []     # what the dictionary offers instead: "врач: лекар, доктор"
        for c in russian_candidates(e["ru"]):
            page = ruw.get(c)
            if not page:
                continue
            ru_has_sr = ru_has_sr or bool(page["sr"])
            if page["sr"]:
                ru_lists.append(f"{c}: " + ", ".join(sr_to_latin(x) for x in page["sr"][:5]))
            sr_lat = {sr_to_latin(x) for x in page["sr"]}
            hrbs = {sr_to_latin(x) for x in page["hr"] + page["bs"]}
            for v in variants:
                if v in sr_lat:
                    ru_hits.append(f"{c}→{v}")
                elif v in hrbs:
                    ru_weak.append(f"{c}→{v}")
        # English side
        wk_status, wk_pos, wk_gender, wk_gloss = "n/a", [], set(), []
        wk_ok = False
        for v in variants:
            r = en.get(v)
            if not r:
                continue
            wk_status = r["status"]
            if r["status"] == "ok":
                wk_ok = True
                for ent in r["entries"]:
                    wk_pos.append(ent["pos"])
                    wk_gender |= {g for g in ent["gender"].split(",") if g in ("m", "f", "n")}
                    if ent["glosses"]:
                        wk_gloss.append(ent["glosses"][0][:80])
        flags = []
        pos = POS_MAP.get(e.get("pos", ""))
        if wk_ok and pos and pos not in wk_pos and not (pos == "Adjective" and "Adjective" in wk_pos):
            flags.append(f"pos: курс {pos}, Wiktionary {sorted(set(wk_pos))}")
        g = e.get("gender")
        if wk_ok and g and wk_gender and g not in wk_gender:
            flags.append(f"род: курс {g}, Wiktionary {sorted(wk_gender)}")
        if ru_hits:
            status = "CONFIRMED"
        elif ru_weak:
            status = "WEAK"       # hr / bs column only
        elif ru_has_sr and " " not in e["latin"].strip() and "/" not in e["latin"]:
            status = "CONFLICT"   # the dictionary names other Serbian words for this Russian meaning
        elif ru_has_sr:
            status = "PHRASE"     # several words: a word dictionary cannot confirm them, read them
        elif wk_ok:
            status = "READ"
        else:
            status = "UNKNOWN"
        rows.append({
            "id": e["id"], "lesson": str(e["lesson"]), "latin": e["latin"], "ru": e["ru"],
            "status": status, "ru_hits": ru_hits, "ru_weak": ru_weak, "ru_lists": ru_lists[:3], "wk": wk_status,
            "pos": sorted(set(wk_pos)), "gloss": wk_gloss[:3], "flags": flags,
        })
    return rows



# ---------------------------------------------------------------- Tatoeba (whole phrases) ----

TATOEBA = "https://tatoeba.org/en/api_v0/search"
TAT_CACHE = os.path.join(W.CACHE_DIR, "tatoeba.json")


def _norm_sr(s: str) -> str:
    s = sr_to_latin(s)
    return " ".join(re.findall(r"[a-zčćšžđ]+", s))


def _stems(ru: str) -> set[str]:
    return {w[:5] for w in re.findall(r"[а-яё]+", ru.lower()) if len(w) > 2}


def tatoeba_lookup(phrases: list[str]) -> dict[str, list[str]]:
    """{normalised Serbian phrase: [Russian translations]} for the phrases that Tatoeba has word for word."""
    cache = json.load(open(TAT_CACHE, encoding="utf-8")) if os.path.exists(TAT_CACHE) else {}
    for n, ph in enumerate(phrases, 1):
        if ph in cache:
            continue
        found: list[str] = []
        for lang in ("srp", "hbs"):
            q = urllib.parse.urlencode({"from": lang, "to": "rus", "query": ph})
            try:
                req = urllib.request.Request(TATOEBA + "?" + q, headers={"User-Agent": W.UA})
                data = json.load(urllib.request.urlopen(req, timeout=40))
            except Exception:
                continue
            for r in data.get("results", []):
                if _norm_sr(r["text"]) == ph:
                    for grp in r.get("translations", []):
                        found += [t["text"] for t in grp if t.get("lang") == "rus"]
            time.sleep(0.4)
        cache[ph] = sorted(set(found))
        if n % 25 == 0:
            json.dump(cache, open(TAT_CACHE, "w", encoding="utf-8"), ensure_ascii=False)
            print(f"  Tatoeba: {n}/{len(phrases)}", file=sys.stderr)
    json.dump(cache, open(TAT_CACHE, "w", encoding="utf-8"), ensure_ascii=False)
    return {p: cache.get(p, []) for p in phrases}


def check_sentences(lessons: list[str]) -> None:
    """Every Serbian = Russian pair of the lessons against Tatoeba. A match is a confirmation; no match proves nothing."""
    import dump_lesson_pairs as D
    pairs: dict[str, set[str]] = {}
    where: dict[str, str] = {}
    for lid in lessons:
        for _kind, sr, ru in D.pairs_of(lid):
            key = _norm_sr(sr)
            if len(key.split()) >= 2:
                pairs.setdefault(key, set()).add(ru)
                where.setdefault(key, lid)
    tat = tatoeba_lookup(sorted(pairs))
    ok = diff = none = 0
    for key in sorted(pairs, key=lambda k: where[k]):
        found = tat.get(key, [])
        if not found:
            none += 1
            continue
        ours = set().union(*(_stems(r) for r in pairs[key]))
        best = max((len(ours & _stems(t)) / max(1, min(len(ours), len(_stems(t)))) for t in found), default=0)
        if best >= 0.5:
            ok += 1
        else:
            diff += 1
            print(f"DIFFERENT [{where[key]}] {key}  наш: {sorted(pairs[key])[:2]}  Tatoeba: {found[:3]}")
    print(f"фраз {len(pairs)}: Tatoeba подтверждает {ok}, расходится {diff}, нет в Tatoeba {none}", file=sys.stderr)


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--levels", help="comma-separated phase ids, e.g. 1,1b")
    ap.add_argument("--lesson", nargs="*", help="lesson ids")
    ap.add_argument("--all", action="store_true")
    ap.add_argument("--refresh", action="store_true")
    ap.add_argument("--sentences", action="store_true", help="check whole phrases against Tatoeba instead of single words")
    ap.add_argument("--md", help="write a Markdown report here")
    ap.add_argument("--only", choices=["CONFIRMED", "WEAK", "CONFLICT", "PHRASE", "READ", "UNKNOWN"], help="print only this status")
    a = ap.parse_args()
    if a.all:
        lessons = None
    elif a.lesson:
        lessons = set(a.lesson)
    elif a.levels:
        lessons = lessons_of_levels(a.levels.split(","))
    else:
        ap.error("choose --levels, --lesson or --all")
    if a.sentences:
        ids = sorted(lessons) if lessons else [os.path.basename(p)[:-5] for p in glob.glob(os.path.join(CONTENT, "lessons", "*.yaml"))]
        check_sentences(ids)
        return 0
    entries = vocab_for(lessons)
    rows = check(entries, a.refresh)
    from collections import Counter
    cnt = Counter(r["status"] for r in rows)
    print(f"{len(rows)} слов: " + ", ".join(f"{k} {v}" for k, v in sorted(cnt.items())), file=sys.stderr)
    lines = []
    for r in rows:
        if a.only and r["status"] != a.only:
            continue
        extra = ""
        if r["ru_hits"]:
            extra += "  ru.wiktionary: " + ", ".join(r["ru_hits"][:3])
        if r["ru_weak"]:
            extra += "  (только hr/bs: " + ", ".join(r["ru_weak"][:3]) + ")"
        if r["status"] == "CONFLICT" and r["ru_lists"]:
            extra += "  словарь даёт: " + " | ".join(r["ru_lists"])
        if r["gloss"] and r["status"] != "CONFIRMED":
            extra += "  en: " + " | ".join(r["gloss"])
        if r["flags"]:
            extra += "  ⚠ " + "; ".join(r["flags"])
        lines.append(f"{r['status']:9} {r['lesson']:4} {r['latin']} = {r['ru']}{extra}")
    print("\n".join(lines))
    if a.md:
        with open(a.md, "w", encoding="utf-8") as f:
            f.write("| статус | урок | сербский | русский | словари |\n|---|---|---|---|---|\n")
            for r in rows:
                f.write(f"| {r['status']} | {r['lesson']} | {r['latin']} | {r['ru']} | "
                        + (", ".join(r["ru_hits"][:2]) or " / ".join(r["gloss"][:2]) or "—") + " |\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
