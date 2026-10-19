#!/usr/bin/env python3
"""Is it safe to show this course in Cyrillic?

The app converts Serbian Latin text to Cyrillic at render time
(web/src/lib/script.ts). That is only correct if the text is what the converter
assumes: every Latin run in a displayed string is Serbian, and the pairs lj/nj/dz
really are single letters. This script checks the course for the places where
that is not so, and checks the Python copy of the converter against the same
vectors the TypeScript one is tested with.

  1. vectors      -- scripts/translit_vectors.json (shared with the TS test).
  2. vocab        -- the converter reproduces the authored `cyrillic` of every
                     content/vocab.yaml word (the part before any note in brackets).
  3. roman        -- a roman numeral written with Latin letters inside Russian
                     text ("тип I"): converted it would become "тип И". Write it with
                     the roman-numeral characters instead.
  3b. script-talk -- Russian text that talks about the scripts themselves and has
                     Latin in it: in Cyrillic it would explain nothing. Put the lesson
                     in Latin (`script: latin` in its manifest) or reword. Lessons
                     already marked are skipped.
  4. non-serbian  -- a Latin word with q, w, x or y, which Serbian has no letters
                     for: English or a brand name that would be turned into nonsense.
  5. digraph      -- a word that could hold lj/nj/dz as two letters
                     (in+jekcija, nad+zivjeti). Every one must be looked at once and
                     listed in content/cyrillic.yaml `digraph_ok` with a reason.

  6. dictionary  -- (--dictionary) the independent check: every Latin word of the course,
                     converted, must be a word the real Serbian Cyrillic dictionary
                     (LibreOffice sr) accepts whenever the Latin spelling is accepted by
                     its Latin twin (sr-Latn). It is the only check whose answer does not
                     come from this repo, and it is what found "Prošlost II" -> "ИИ".

Usage:
    python3 scripts/check_cyrillic.py              # everything but the dictionary
    python3 scripts/check_cyrillic.py --dictionary # plus the dictionary check (needs spylls)
    python3 scripts/check_cyrillic.py --self-test  # just the vectors
Exit code 1 on any finding.

See docs/superpowers/specs/2026-10-19-cyrillic-switch-design.md.
"""
import argparse
import glob
import json
import os
import re
import sys
import unicodedata

import yaml

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CONTENT = os.path.join(ROOT, "content")
VECTORS = os.path.join(ROOT, "scripts", "translit_vectors.json")

# ---- the converter (keep in step with web/src/lib/script.ts) ----------------

LETTERS = {
    "a": "а", "b": "б", "c": "ц", "č": "ч", "ć": "ћ", "d": "д", "đ": "ђ", "e": "е", "f": "ф",
    "g": "г", "h": "х", "i": "и", "j": "ј", "k": "к", "l": "л", "m": "м", "n": "н", "o": "о",
    "p": "п", "r": "р", "s": "с", "š": "ш", "t": "т", "u": "у", "v": "в", "z": "з", "ž": "ж",
}
DIGRAPHS = {"lj": "љ", "nj": "њ", "dž": "џ"}
LIGATURES = {
    "ǆ": "џ", "ǅ": "Џ", "Ǆ": "Џ", "ǉ": "љ", "ǈ": "Љ", "Ǉ": "Љ", "ǌ": "њ", "ǋ": "Њ", "Ǌ": "Њ",
}
SPLIT = "\u0000"
TWO_LETTERS = [re.compile(p, re.I) for p in (r"(in)(?=jekc)", r"(kon)(?=junkt)", r"(nad)(?=živ)", r"(od)(?=žvak)", r"(pod)(?=župan)")]
LATIN_RUN = re.compile("(?:[A-Za-zÀ-ſǄ-ǌ][̀-ͯ]*)+")
CARON = {"c": "č", "C": "Č", "s": "š", "S": "Š", "z": "ž", "Z": "Ž"}


def _recompose(run):
    s = unicodedata.normalize("NFD", run)
    s = re.sub("([cC])́", lambda m: "ć" if m.group(1) == "c" else "Ć", s)
    return re.sub("([csz])̌", lambda m: CARON[m.group(1)], s, flags=re.I)


def _convert_run(m):
    s = _recompose(m.group(0))
    for rx in TWO_LETTERS:
        s = rx.sub(lambda mm: mm.group(1) + SPLIT, s)
    out, i = [], 0
    while i < len(s):
        ch = s[i]
        if ch == SPLIT:
            i += 1
            continue
        if ch in LIGATURES:
            out.append(LIGATURES[ch])
            i += 1
            continue
        pair = DIGRAPHS.get((ch + (s[i + 1] if i + 1 < len(s) else "")).lower())
        if pair:
            out.append(pair if ch == ch.lower() else pair.upper())
            i += 2
            continue
        low = ch.lower()
        cyr = LETTERS.get(low)
        out.append((cyr if ch == low else cyr.upper()) if cyr else ch)
        i += 1
    return "".join(out)


def to_cyrillic(text):
    return LATIN_RUN.sub(_convert_run, text)


# ---- helpers ----------------------------------------------------------------

CYR = re.compile("[Ѐ-ӿ]")
WORD = re.compile(r"[A-Za-zÀ-ſǄ-ǌ][A-Za-zÀ-ſǄ-ǌ̀-ͯ'’-]*")
STRESS = re.compile("(?<=[aeiouAEIOUаеиоуАЕИОУрР])́")


def strip_stress(s):
    return unicodedata.normalize("NFC", STRESS.sub("", unicodedata.normalize("NFD", s)))


def load_yaml(path):
    with open(path, encoding="utf-8") as f:
        return yaml.safe_load(f)


def reviewed(name):
    """content/cyrillic.yaml: {digraph_ok: {word: reason}, ...}; every entry needs a reason."""
    path = os.path.join(CONTENT, "cyrillic.yaml")
    data = load_yaml(path) if os.path.exists(path) else {}
    out = (data or {}).get(name) or {}
    bad = [k for k, v in out.items() if not (isinstance(v, str) and v.strip())]
    if bad:
        sys.exit(f"content/cyrillic.yaml: {name}: an entry without a reason: {', '.join(bad)}")
    return out


# ---- 1. vectors -------------------------------------------------------------

def check_vectors():
    with open(VECTORS, encoding="utf-8") as f:
        vectors = json.load(f)["vectors"]
    bad = [(v["latin"], v["cyrillic"], to_cyrillic(v["latin"])) for v in vectors if to_cyrillic(v["latin"]) != v["cyrillic"]]
    return [f"vectors: {a!r} -> {got!r}, want {want!r}" for a, want, got in bad], len(vectors)


# ---- 2. vocab ---------------------------------------------------------------

def head(s):
    """The word itself: no note in brackets, no second variant after ' / '."""
    return re.sub(r"\s*\(.*?\)", "", s).split(" / ")[0].replace("'", "").strip()


def check_vocab():
    data = load_yaml(os.path.join(CONTENT, "vocab.yaml"))
    items = data if isinstance(data, list) else next(v for v in data.values() if isinstance(v, list))
    problems, n = [], 0
    for w in items:
        if not isinstance(w, dict) or not w.get("latin") or not w.get("cyrillic"):
            continue
        n += 1
        got = strip_stress(to_cyrillic(head(w["latin"])))
        want = strip_stress(head(w["cyrillic"]))
        if got != want:
            problems.append(f"vocab: {w['latin']!r} converts to {got!r} but the authored cyrillic is {want!r}")
    return problems, n


# ---- gathering displayed text ----------------------------------------------

DISPLAY_KEYS = {"title", "subtitle", "prompt", "hint", "explain", "options", "bank", "left", "forms",
                "meta", "say", "sr", "ru", "scene", "note", "pairs", "text", "means", "not", "correct", "sample"}


def strings_in(doc, key=None):
    if isinstance(doc, str):
        if key in DISPLAY_KEYS:
            yield doc
    elif isinstance(doc, dict):
        for k, v in doc.items():
            yield from strings_in(v, k if k in DISPLAY_KEYS else None)
    elif isinstance(doc, list):
        for v in doc:
            yield from strings_in(v, key)


def displayed_texts():
    """Yield (where, text, keep_latin) for everything the learner can read."""
    keep_lessons = set()
    for path in sorted(glob.glob(os.path.join(CONTENT, "lessons", "*.yaml"))):
        if os.path.basename(path).startswith("_"):
            continue
        doc = load_yaml(path)
        lid = str(doc.get("lesson", ""))
        keep = doc.get("script") == "latin"
        if keep:
            keep_lessons.add(lid)
        rel = os.path.relpath(path, ROOT)
        for t in strings_in(doc):
            yield rel, t, keep
        for step in doc.get("steps", []):
            if step.get("md"):
                md = os.path.join(CONTENT, "lessons", step["md"])
                if os.path.exists(md):
                    with open(md, encoding="utf-8") as f:
                        yield os.path.relpath(md, ROOT), f.read(), keep
    for name in ("vocab.yaml", "false-friends.yaml", "grammar.yaml"):
        path = os.path.join(CONTENT, name)
        if os.path.exists(path):
            for t in strings_in(load_yaml(path)):
                yield os.path.relpath(path, ROOT), t, False


def snippet(text, m, width=28):
    a, b = max(0, m.start() - width), min(len(text), m.end() + width)
    return text[a:b].replace("\n", " ")


# ---- 3-5. content checks ----------------------------------------------------

# A roman numeral spelled with Latin letters, with no Latin word next to it (next to
# Latin it is a Serbian word: "I ti?" is "and you?"). A lone I, V or X is a numeral only
# in Russian text.
ROMAN = re.compile(r"(?<![A-Za-z\u00c0-\u017f])(?:[IVX]|II|III|IV|VI|VII|VIII|IX|XI|XII)(?![A-Za-z\u00c0-\u017f])")
# II, III, IV, VII, VIII, IX, XI, XII are never Serbian words, so they are flagged even with
# Latin next to them: "Prošlost II" would become "Прошлост ИИ". (VI is left to ROMAN above:
# in capitals it can be the Serbian "vi", "you".) Written with the roman-numeral
# characters (Ⅱ Ⅲ Ⅳ …) they are left alone by the converter.
ROMAN_SURE = re.compile(r"(?<![A-Za-z\u00c0-\u017f])(?:II|III|IV|VII|VIII|IX|XI|XII)(?![A-Za-z\u00c0-\u017f])")
SCRIPT_TALK = re.compile(r"латиниц|кириллиц|ASCII|диакрит|ћирилиц", re.I)
NON_SERBIAN = re.compile(r"[qwxyQWXY]")
RISK = re.compile(r"^(?:in|kon|van|pan|san|un)j|^(?:nad|od|pod|pred|ad)ž", re.I)


def check_content():
    digraph_ok = {k.lower() for k in reviewed("digraph_ok")}
    problems, seen_risk = [], {}
    for where, text, keep in displayed_texts():
        russian = bool(CYR.search(text))
        if russian and not keep:
            for m in ROMAN_SURE.finditer(text):
                problems.append(f"roman: {where}: {m.group(0)!r} in …{snippet(text, m)}…")
            for m in ROMAN.finditer(text):
                before, after = text[: m.start()], text[m.end():]
                if re.search(r"[A-Za-z\u00c0-\u017f][\s\W]*$", before) or re.match(r"[\s\W]*[A-Za-z\u00c0-\u017f]", after):
                    continue
                problems.append(f"roman: {where}: {m.group(0)!r} in …{snippet(text, m)}…")
            for sentence in re.split(r"(?<=[.!?])\s+|\n", text):
                if SCRIPT_TALK.search(sentence) and LATIN_RUN.search(sentence):
                    problems.append(f"script-talk: {where}: {sentence.strip()[:120]!r}")
        for m in WORD.finditer(text):
            w = m.group(0)
            if NON_SERBIAN.search(w):
                problems.append(f"non-serbian: {where}: {w!r} in …{snippet(text, m)}…")
            lw = unicodedata.normalize("NFC", w.lower())
            if RISK.search(lw) and lw not in digraph_ok and not any(rx.search(lw) for rx in TWO_LETTERS):
                seen_risk.setdefault(lw, where)
    for w, where in sorted(seen_risk.items()):
        problems.append(f"digraph: {where}: {w!r} may hold nj/lj/dž as two letters; look once, then add it to digraph_ok in content/cyrillic.yaml")
    return problems


def ensure_cyrillic_dictionary():
    """The Cyrillic twin of the sr-Latn dictionary lint_serbian.py already caches."""
    import urllib.request
    from spylls.hunspell import Dictionary

    cache = os.path.join(ROOT, "scripts", ".cache")
    os.makedirs(cache, exist_ok=True)
    for ext in ("aff", "dic"):
        path = os.path.join(cache, f"sr.{ext}")
        if not os.path.exists(path):
            print(f"downloading the Cyrillic dictionary sr.{ext} …", file=sys.stderr)
            url = f"https://raw.githubusercontent.com/LibreOffice/dictionaries/master/sr/sr.{ext}"
            req = urllib.request.Request(url, headers={"User-Agent": "ucimo-lint/1"})
            with urllib.request.urlopen(req, timeout=120) as r, open(path + ".part", "wb") as f:
                f.write(r.read())
            os.replace(path + ".part", path)
    return Dictionary.from_files(os.path.join(cache, "sr"))


def check_dictionary():
    try:
        sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
        import lint_serbian
    except ImportError:
        sys.exit("--dictionary needs spylls: pip3 install spylls pyyaml")
    lat, cyr = lint_serbian.ensure_dictionary(), ensure_cyrillic_dictionary()
    allowed = {k.lower() for k in reviewed("dictionary_ok")}

    def known(d, w):
        return any(d.lookup(x) for x in (w, w.lower(), w.capitalize()))

    first_seen = {}
    for where, text, keep in displayed_texts():
        if keep:
            continue
        for m in WORD.finditer(text):
            w = unicodedata.normalize("NFC", m.group(0).replace("\u0301", "")).strip("'’-")
            if len(w) >= 2:
                first_seen.setdefault(w, where)
    problems, ok, neither = [], 0, 0
    for w, where in sorted(first_seen.items()):
        cw = to_cyrillic(w)
        if known(cyr, cw):
            ok += 1
        elif not known(lat, w):
            neither += 1  # a name, a loanword, a fragment of an ending: no dictionary can judge it
        elif w.lower() not in allowed:
            problems.append(f"dictionary: {where}: {w!r} is a word, but {cw!r} is not in the Cyrillic dictionary")
    print(f"dictionary: {len(first_seen)} distinct words: {ok} convert to a Cyrillic word the dictionary accepts, {neither} unknown to both dictionaries")
    return problems


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--self-test", action="store_true", help="only check the converter against the shared vectors")
    ap.add_argument("--dictionary", action="store_true", help="also cross-check every word with the Cyrillic dictionary (needs spylls)")
    args = ap.parse_args()

    problems, nvec = check_vectors()
    print(f"vectors: {nvec} checked")
    if not args.self_test:
        vp, nvoc = check_vocab()
        problems += vp
        print(f"vocab: {nvoc} words with an authored cyrillic checked")
        problems += check_content()
        if args.dictionary:
            problems += check_dictionary()
    for p in problems:
        print("  " + p)
    print(f"{len(problems)} finding(s)" if problems else "OK")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
