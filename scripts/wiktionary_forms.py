#!/usr/bin/env python3
"""Serbo-Croatian data from en.wiktionary: part of speech, gender, English glosses, every inflected form.

The library behind `check_translations.py` and `check_grammar.py` (a dictionary, not a guess: every
form comes with the case / person tag Wiktionary itself prints in the table). The pages are read once
through the MediaWiki API and cached in `scripts/.cache/wiktionary-forms.json`.

Wiktionary leans to the Croatian norm and has gaps, so a missing word or form means "check by hand",
never "the course is wrong".

    pip install pyyaml
    python3 scripts/wiktionary_forms.py sestra imati        # print what is known about words
    python3 scripts/wiktionary_forms.py --refresh sestra    # read the page again
"""
from __future__ import annotations

import html as htmllib
import json
import os
import re
import sys
import time
import unicodedata
import urllib.parse
import urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CACHE_DIR = os.path.join(ROOT, "scripts", ".cache")
CACHE = os.path.join(CACHE_DIR, "wiktionary-forms.json")
API = "https://en.wiktionary.org/w/api.php"
UA = "ucimo-course-check/1.0 (https://ucimo.ru; man30968@gmail.com)"

POS_HEADINGS = {
    "Noun", "Verb", "Adjective", "Adverb", "Pronoun", "Numeral", "Preposition", "Conjunction",
    "Interjection", "Particle", "Determiner", "Proper noun", "Phrase", "Contraction",
}


_ACCENT_MARKS = {"\u0300", "\u0301", "\u0304", "\u0306", "\u030f", "\u0311", "\u0303", "\u0329", "\u0361"}
_KEEP = set("čćšžđČĆŠŽĐ")


def strip_accents(s: str) -> str:
    """Wiktionary prints accents (sèstra, sestárā); the course has none. The letters č ć š ž đ stay.

    NFC first, so `ć` is one letter; then every other character loses its accent marks (an accented
    vowel such as `ȕ` decomposes to `u` + mark and the mark goes).
    """
    out = []
    for ch in unicodedata.normalize("NFC", s):
        if ch in _KEEP:
            out.append(ch)
            continue
        out.append("".join(c for c in unicodedata.normalize("NFD", ch) if c not in _ACCENT_MARKS))
    return unicodedata.normalize("NFC", "".join(out))


def _api(**params) -> dict:
    params.update(format="json", formatversion="2")
    req = urllib.request.Request(API + "?" + urllib.parse.urlencode(params), headers={"User-Agent": UA})
    last = None
    for attempt in range(4):
        try:
            return json.load(urllib.request.urlopen(req, timeout=60))
        except Exception as e:      # network hiccup or a rate limit: wait and try again
            last = e
            time.sleep(2 + 3 * attempt)
    raise RuntimeError(f"Wiktionary API failed: {last}")


def _text(fragment: str) -> str:
    fragment = re.sub(r"<(ul|dl|ol)\b.*?</\1>", "", fragment, flags=re.S)       # examples and sub-senses
    fragment = re.sub(r"<style.*?</style>", "", fragment, flags=re.S)
    fragment = re.sub(r"<[^>]+>", "", fragment)
    return re.sub(r"\s+", " ", htmllib.unescape(fragment)).strip()


def parse_section(page_html: str) -> dict:
    """The rendered Serbo-Croatian section → {"entries": [{"pos", "gender", "glosses"}], "forms": {form: [tags]}}."""
    entries: list[dict] = []
    # headings: <h3 id="Noun">…, headword line: <p><strong class="Latn headword" …>, then <ol>senses</ol>
    pos_iter = list(re.finditer(r'<h[345][^>]*\bid="([^"]+)"', page_html))
    for i, m in enumerate(pos_iter):
        name = re.sub(r"_\d+$", "", m.group(1)).replace("_", " ")
        if name not in POS_HEADINGS:
            continue
        end = pos_iter[i + 1].start() if i + 1 < len(pos_iter) else len(page_html)
        chunk = page_html[m.end():end]
        gender = ""
        g = re.search(r'<span class="gender">(.*?)</span>', chunk, re.S)
        if g:
            marks = re.findall(r"<abbr[^>]*>([^<]+)</abbr>", g.group(1))
            gender = ",".join(sorted({x for x in marks if x in ("m", "f", "n", "impf", "pf")}))
        ol = re.search(r"<ol>(.*?)</ol>", chunk, re.S)
        glosses = []
        if ol:
            for li in re.findall(r"<li>(.*?)(?=<li>|$)", ol.group(1), re.S):
                t = _text(li)
                if t:
                    glosses.append(t)
        entries.append({"pos": name, "gender": gender, "glosses": glosses})

    forms: dict[str, list[str]] = {}
    for sp in re.finditer(r'<span class="([^"]*form-of[^"]*)"[^>]*\blang="sh"[^>]*>(.*?)</span>', page_html, re.S):
        classes = htmllib.unescape(sp.group(1))
        if "Latn" not in classes:
            continue
        tag = next((c[: -len("-form-of")] for c in classes.split() if c.endswith("-form-of") and c != "form-of"), "")
        if not tag:
            continue
        form = strip_accents(_text(sp.group(2))).strip()
        if form and " " not in form:
            forms.setdefault(form, [])
            if tag not in forms[form]:
                forms[form].append(tag)
    for row in re.finditer(r"<tr>(.*?)</tr>", page_html, re.S):
        ths = [_text(x) for x in re.findall(r"<th[^>]*>(.*?)</th>", row.group(1), re.S)]
        tds = re.findall(r"<td[^>]*>(.*?)</td>", row.group(1), re.S)
        head = " ".join(ths).strip().lower()
        if head.endswith("present") and len(tds) == 6:
            tags = ["1|s|pres", "2|s|pres", "3|s|pres", "1|p|pres", "2|p|pres", "3|p|pres"]
        elif head.endswith("imperative") and len(tds) == 3:
            tags = ["2|s|imp", "1|p|imp", "2|p|imp"]
        else:
            continue
        for tag, td in zip(tags, tds):
            for form in re.findall(r'<span class="Latn"[^>]*lang="sh"[^>]*>(.*?)</span>', td, re.S):
                for alt in _text(form).split("/"):
                    _add(forms, alt, tag)
    for tm in re.finditer(r"<table[^>]*>(.*?)</table>", page_html, re.S):
        _grid_table(tm.group(1), forms)
        _noun_grid_table(tm.group(1), forms)
    return {"entries": entries, "forms": forms}


_CASES = {"nominative": "nom", "genitive": "gen", "dative": "dat", "accusative": "acc",
          "vocative": "voc", "locative": "loc", "instrumental": "ins"}


def _alts(span_html: str) -> list[str]:
    """The spelling variants inside one cell: `<br>` and `/` separate them; a slash inside brackets does not
    (`novom(u/e)` is one entry). Tags come off before the split."""
    t = _text(re.sub(r"<br\s*/?>", "/", span_html, flags=re.I))
    t = re.sub(r"\(([^)]*)\)", lambda m: "(" + m.group(1).replace("/", "\x00") + ")", t)
    return [x.replace("\x00", "/") for x in t.split("/")]


def _add(forms: dict, raw: str, tag: str) -> None:
    """Register `raw` (maybe `dobrim(a)`) under `tag`; accents come off, and `x(a)` gives both spellings."""
    raw = strip_accents(raw).strip()
    if not raw or " " in raw:
        return
    variants = [raw]
    m = re.fullmatch(r"(.+?)\(([\w/]+)\)", raw)
    if m:                                     # dobrim(a) → dobrim, dobrima;  novom(e/u) → novom, novome, novomu
        variants = [m.group(1)] + [m.group(1) + x for x in m.group(2).split("/")]
    for f in variants:
        forms.setdefault(f, [])
        if tag not in forms[f]:
            forms[f].append(tag)


def _noun_grid_table(table_html: str, forms: dict) -> None:
    """Noun tables written without per-form markup: header cells «singular» / «plural», rows are cases."""
    if "form-of" in table_html:
        return
    cap = re.search(r"<caption[^>]*>(.*?)</caption>", table_html, re.S)
    if cap and " forms" in _text(cap.group(1)).lower():
        return                                   # adjective-style, handled by _grid_table
    numbers: list[str] = []
    for row in re.finditer(r"<tr[^>]*>(.*?)</tr>", table_html, re.S):
        ths = [_text(x).lower() for x in re.findall(r"<th[^>]*>(.*?)</th>", row.group(1), re.S)]
        tds = re.findall(r"<td[^>]*>(.*?)</td>", row.group(1), re.S)
        if not tds:
            found = [("s" if t == "singular" else "p") for t in ths if t in ("singular", "plural")]
            if found:
                numbers = found
            continue
        case = next((_CASES[t] for t in ths if t in _CASES), None)
        if case is None or not numbers or len(tds) != len(numbers):
            continue
        for number, td in zip(numbers, tds):
            for form in re.findall(r'<span class="Latn"[^>]*lang="sh"[^>]*>(.*?)</span>', td, re.S):
                for alt in _alts(form):
                    _add(forms, alt, f"{case}|{number}")


def _grid_table(table_html: str, forms: dict) -> None:
    """Adjective-style tables: the caption says «… forms», rows are cases, columns m / f / n, blocks singular / plural."""
    cap = re.search(r"<caption[^>]*>(.*?)</caption>", table_html, re.S)
    caption = _text(cap.group(1)).lower() if cap else ""
    if " forms" not in caption or "-form-of" in table_html:
        return
    degree = ("comp" if "comparative" in caption else "sup" if "superlative" in caption else
              "def" if "definite" in caption and "indefinite" not in caption else "indef")
    number = "s"
    for row in re.finditer(r"<tr>(.*?)</tr>", table_html, re.S):
        ths = [_text(x).lower() for x in re.findall(r"<th[^>]*>(.*?)</th>", row.group(1), re.S)]
        tds = re.findall(r"<td[^>]*>(.*?)</td>", row.group(1), re.S)
        if not tds:
            if ths and ths[0] in ("singular", "plural"):
                number = "s" if ths[0] == "singular" else "p"
            continue
        case = next((_CASES[t] for t in ths if t in _CASES), None)
        if case is None or len(tds) != 3:
            continue
        for gender, td in zip("mfn", tds):
            for form in re.findall(r'<span class="Latn"[^>]*lang="sh"[^>]*>(.*?)</span>', td, re.S):
                for alt in _alts(form):
                    _add(forms, alt, f"{case}|{number}|{gender}|{degree}")


_LEMMA_OF = re.compile(r"\bof ([A-Za-zČĆŠŽĐčćšžđ]+):?\s*$")


def lemma_of(entries: list[dict]) -> str | None:
    """`inflection of mal:` → `mal`: a page that is partly a form of another word (`mali` is `mal` too)."""
    lemmas = set()
    for e in entries:
        if not e["glosses"]:
            continue
        g = e["glosses"][0].split("\n")[0].strip()
        m = _LEMMA_OF.search(g)
        if m and ("inflection" in g.lower() or "form of" in g.lower() or " of " in g):
            lemmas.add(m.group(1))
    return lemmas.pop() if len(lemmas) == 1 else None


def fetch_word(word: str) -> dict:
    """One lemma from Wiktionary. status: ok | no_page | no_sh_section."""
    d = _api(action="parse", page=word, prop="sections", redirects="1")
    if "error" in d:
        return {"status": "no_page"}
    secs = d["parse"]["sections"]
    idx = next((s["index"] for s in secs if s["line"] == "Serbo-Croatian" and s["toclevel"] == 1), None)
    if idx is None:
        return {"status": "no_sh_section"}
    d = _api(action="parse", page=word, prop="text", section=idx, redirects="1")
    out = parse_section(d["parse"]["text"])
    out["status"] = "ok"
    return out


def load_cache() -> dict:
    if os.path.exists(CACHE):
        return json.load(open(CACHE, encoding="utf-8"))
    return {}


def save_cache(cache: dict) -> None:
    os.makedirs(CACHE_DIR, exist_ok=True)
    tmp = CACHE + ".part"
    json.dump(cache, open(tmp, "w", encoding="utf-8"), ensure_ascii=False, indent=0)
    os.replace(tmp, CACHE)


def lookup(words: list[str], refresh: bool = False, pause: float = 0.4, quiet: bool = False) -> dict[str, dict]:
    """Cached lookup of many words; saves after every 25 fetched."""
    cache = load_cache()
    fetched = 0
    for w in words:
        if w in cache and not refresh:
            continue
        try:
            r = fetch_word(w)
            lem = lemma_of(r.get("entries", [])) if r["status"] == "ok" else None
            if lem and lem != w:
                lr = cache.get(lem) or fetch_word(lem)
                cache[lem] = lr
                r["lemma"] = lem
                for f, t in lr.get("forms", {}).items():
                    r.setdefault("forms", {}).setdefault(f, [])
                    r["forms"][f] = sorted(set(r["forms"][f]) | set(t))
                r["entries"] = r["entries"] + lr.get("entries", [])
            cache[w] = r
        except Exception as e:      # keep going: one bad page must not lose the rest
            cache[w] = {"status": "error", "error": str(e)[:120]}
        fetched += 1
        if fetched % 25 == 0:
            save_cache(cache)
            if not quiet:
                print(f"  Wiktionary: {fetched} fetched…", file=sys.stderr)
        time.sleep(pause)
    if fetched:
        save_cache(cache)
    return {w: cache.get(w, {"status": "no_page"}) for w in words}


def main() -> None:
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    res = lookup(args, refresh="--refresh" in sys.argv)
    for w, r in res.items():
        print(f"== {w}: {r['status']}")
        for e in r.get("entries", []):
            print(f"   {e['pos']} {e['gender']}: " + " | ".join(e["glosses"])[:200])
        forms = r.get("forms", {})
        if forms:
            print("   forms:", ", ".join(f"{f}[{'/'.join(t)}]" for f, t in list(forms.items())[:40]))


if __name__ == "__main__":
    main()
