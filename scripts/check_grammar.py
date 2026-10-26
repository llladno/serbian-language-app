#!/usr/bin/env python3
"""Grammar check of the Serbian phrases of the course against the forms Wiktionary prints.

A spell checker says `sestru` is a word. It cannot say that after `nemam` it must be `sestre`. This script
takes every CORRECT Serbian phrase of the chosen lessons (slides, dialogue lines, readings, answers, accepted
answers, dictations, matching pairs, filled-in blanks) and checks the places where Serbian grammar is strict
enough to test by rule:

  NEMAM   `nemam / nema / nemaš…` + noun        → genitive            Nemam sestre (not sestru)
  IMAM    `imam / imaš…` + noun                 → accusative           Imam sestru
  DAJTE   `dajte mi / treba mi` + noun          → accusative / genitive Dajte mi vodu
  PREP    `od, do, iz, bez, kod, pored…` + noun → genitive; `u, na` → accusative or locative; `sa` → instrumental
  NUM     2–4 + noun → genitive singular; 5+ → genitive plural; …1 → singular   dva dinara, pet dinara, dvadeset jednu godinu
  POSS    `moj / tvoj / njegov / njen / naš / vaš` + noun, adjective + noun → gender agrees        moja mama, moj tata
  COP     `Ovo je / To je` + noun               → nominative
  PERS    pronoun + verb                        → person and number agree          ja sam, ti si, mi smo

Every form is looked up in the tables of en.wiktionary (see wiktionary_forms.py), so a verdict rests on a dictionary,
not on a guess. A word that Wiktionary does not know is skipped, never failed. A finding is "to read": the rules are
simple, and Serbian has exceptions (a partitive genitive `Imam vremena`, a vocative). The point is that nothing
slips through unread.

    pip install pyyaml
    python3 scripts/check_grammar.py --levels 1,1b           # lessons of phases 1 and 1b
    python3 scripts/check_grammar.py --lesson 13 13b         # these lessons
    python3 scripts/check_grammar.py --all
    python3 scripts/check_grammar.py --self-test             # the checker must flag the planted mistakes
    python3 scripts/check_grammar.py --levels 1b --fetch     # first read the Wiktionary pages of the lessons' words
"""
from __future__ import annotations

import argparse
import glob
import os
import re
import sys

import yaml

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import wiktionary_forms as W   # noqa: E402

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CONTENT = os.path.join(ROOT, "content")

LET = "A-Za-zČčĆćŠšŽžĐđ"
TOKEN = re.compile(rf"[{LET}]+(?:-[{LET}]+)*")
NOUN_TAG = re.compile(r"^(nom|gen|dat|acc|voc|loc|ins)\|(s|p)$")
ADJ_TAG = re.compile(r"^(nom|gen|dat|acc|voc|loc|ins)\|(s|p)\|([mfn])\|(indef|def|comp|sup)$")
VERB_TAG = re.compile(r"^([123])\|(s|p)\|(pres|imp)$")

# ------------------------------------------------------------------ small hand-made tables ----
# The dictionary has no tables for pronouns and numerals, and they are closed classes: a short list is exact.

# possessive pronouns: form → the (case, gender) readings that matter for agreement with a singular noun
POSS_FORMS: dict[str, set[tuple[str, str]]] = {}


def _add_poss(form: str, *readings: tuple[str, str]) -> None:
    POSS_FORMS.setdefault(form, set()).update(readings)


for _core in ("mo", "tvo", "svo"):                 # moj, tvoj, svoj: moj / moja / moje / moju / mojoj / mojom / mog / mom
    _add_poss(_core + "j", ("nom", "m"), ("acc", "m"), ("voc", "m"))
    _add_poss(_core + "ja", ("nom", "f"), ("voc", "f"))
    _add_poss(_core + "je", ("nom", "n"), ("acc", "n"), ("voc", "n"), ("gen", "f"))
    _add_poss(_core + "ju", ("acc", "f"))
    _add_poss(_core + "joj", ("dat", "f"), ("loc", "f"))
    _add_poss(_core + "jom", ("ins", "f"))
    for _f in (_core + "g", _core + "ga"):
        _add_poss(_f, ("gen", "m"), ("gen", "n"), ("acc", "m"))
    for _f in (_core + "m", _core + "me", _core + "mu"):
        _add_poss(_f, ("dat", "m"), ("loc", "m"), ("ins", "m"), ("dat", "n"), ("loc", "n"), ("ins", "n"))
for _stem, _end in (("naš", "e"), ("vaš", "e"), ("njegov", "o"), ("njen", "o")):   # naš, njegov, njen: regular endings
    _soft = _stem in ("naš", "vaš")
    _add_poss(_stem, ("nom", "m"), ("acc", "m"), ("voc", "m"))
    _add_poss(_stem + "a", ("nom", "f"), ("voc", "f"))
    _add_poss(_stem + ("e" if _soft else "o"), ("nom", "n"), ("acc", "n"), ("voc", "n"))
    _add_poss(_stem + ("e" if _soft else "e"), ("nom", "n"), ("acc", "n"), ("gen", "f"))
    _add_poss(_stem + "u", ("acc", "f"))
    _add_poss(_stem + ("oj"), ("dat", "f"), ("loc", "f"))
    for _f in (_stem + "og", _stem + "oga", _stem + "eg", _stem + "ega"):
        _add_poss(_f, ("gen", "m"), ("gen", "n"), ("acc", "m"))
    for _f in (_stem + "om", _stem + "em", _stem + "omu", _stem + "emu"):
        _add_poss(_f, ("dat", "m"), ("loc", "m"), ("ins", "m"), ("dat", "n"), ("loc", "n"), ("ins", "n"), ("ins", "f"))
POSS_WORDS = set(POSS_FORMS)

ONE = {"jedan", "jedna", "jedno", "jednu", "jednog"}
PAUCAL = {"dva", "dve", "dvije", "tri", "četiri"}
# numerals that take the genitive plural
MANY = {"pet", "šest", "sedam", "osam", "devet", "deset", "jedanaest", "dvanaest", "trinaest", "četrnaest", "petnaest",
        "šesnaest", "sedamnaest", "osamnaest", "devetnaest", "dvadeset", "trideset", "četrdeset", "pedeset", "šezdeset",
        "sedamdeset", "osamdeset", "devedeset", "sto", "dvesta", "trista", "četiristo", "petsto", "šesto", "sedamsto",
        "osamsto", "devetsto", "hiljada", "nula"}
NUMERALS = ONE | PAUCAL | MANY | {"hiljadu", "hiljade", "pola"}

# prepositions → cases they may govern (as tags of a noun form)
PREPS: dict[str, set[str]] = {
    "od": {"gen"}, "do": {"gen"}, "iz": {"gen"}, "bez": {"gen"}, "kod": {"gen"}, "pored": {"gen"}, "iza": {"gen"},
    "ispred": {"gen"}, "oko": {"gen"}, "blizu": {"gen"}, "posle": {"gen"}, "pre": {"gen"}, "umesto": {"gen"},
    "zbog": {"gen"}, "ispod": {"gen"}, "iznad": {"gen"}, "preko": {"gen", "acc"}, "pokraj": {"gen"}, "kraj": {"gen"},
    "osim": {"gen"}, "u": {"acc", "loc"}, "na": {"acc", "loc"}, "o": {"loc", "acc"}, "po": {"loc", "acc"},
    "za": {"acc", "ins", "gen"}, "sa": {"ins", "gen"}, "s": {"ins", "gen"}, "uz": {"acc"}, "kroz": {"acc"},
    "ka": {"dat"}, "prema": {"dat"}, "protiv": {"gen"}, "među": {"ins", "acc", "gen"}, "pod": {"ins", "acc"},
    "nad": {"ins", "acc"}, "pred": {"ins", "acc"}, "pri": {"loc"},
}
HAVE = {"imam", "imaš", "ima", "imamo", "imate", "imaju", "imah", "imao", "imala", "imali"}
NOT_HAVE = {"nemam", "nemaš", "nema", "nemamo", "nemate", "nemaju", "nisam"}
GIVE = {"dajte", "daj", "daću", "dajem"}
PERSON = {"ja": ("1", "s"), "ti": ("2", "s"), "mi": ("1", "p"), "vi": ("2", "p"), "oni": ("3", "p"), "one": ("3", "p")}
BITI = {"sam": ("1", "s"), "si": ("2", "s"), "je": ("3", "s"), "smo": ("1", "p"), "ste": ("2", "p"), "su": ("3", "p")}
STOP = {"i", "ili", "a", "ali", "da", "li", "što", "jer", "pa", "te", "ni"}
# words that Wiktionary also lists as nouns or adjectives but that are used as adverbs or function words here
ADVERBIAL = {"samo", "malo", "mnogo", "dobro", "loše", "lepo", "tačno", "polako", "sitno", "sveže", "jako", "brzo", "ovde",
             "tamo", "kasno", "rano", "popodne", "ujutru", "ujutro", "uveče", "danas", "sutra", "sada", "sad", "još", "baš",
             "ovo", "to", "sve", "nešto", "ništa", "koliko", "kako", "gde", "zato", "zašto"}


# ------------------------------------------------------------------ dictionary index ----

class Dict:
    """Form → readings, built from the Wiktionary cache. `noun` readings are (case, number) with the lemma's gender."""

    def __init__(self, cache: dict):
        self.noun: dict[str, list[tuple[str, str, str]]] = {}      # form → [(case, number, gender)]
        self.adj: dict[str, list[tuple[str, str, str]]] = {}       # form → [(case, number, gender)]
        self.verb: dict[str, list[tuple[str, str]]] = {}           # form → [(person, number)]
        self.gender: dict[str, set[str]] = {}                      # noun lemma/form → genders
        self.known: set[str] = set()
        seen_lemmas = set()
        for word, r in cache.items():
            if r.get("status") != "ok":
                continue
            genders = set()
            for e in r["entries"]:
                if e["pos"] == "Noun":
                    genders |= {g for g in e["gender"].split(",") if g in ("m", "f", "n")}
            for form, tags in r["forms"].items():
                self.known.add(form)
                for t in tags:
                    if NOUN_TAG.match(t):
                        c, n = t.split("|")
                        for g in genders or {""}:
                            self.noun.setdefault(form, []).append((c, n, g))
                    m = ADJ_TAG.match(t)
                    if m:
                        self.adj.setdefault(form, []).append((m.group(1), m.group(2), m.group(3)))
                    m = VERB_TAG.match(t)
                    if m and m.group(3) == "pres":
                        self.verb.setdefault(form, []).append((m.group(1), m.group(2)))
            if genders:
                self.gender.setdefault(word, set()).update(genders)
                for form, tags in r["forms"].items():
                    if any(NOUN_TAG.match(t) for t in tags):
                        self.gender.setdefault(form, set()).update(genders)

    def is_noun_form(self, w: str) -> bool:
        return w in self.noun

    def noun_cases(self, w: str) -> set[str]:
        return {c for c, _n, _g in self.noun.get(w, [])}


# ------------------------------------------------------------------ rules ----

class Finding:
    def __init__(self, rule: str, where: str, text: str, msg: str):
        self.rule, self.where, self.text, self.msg = rule, where, text, msg

    def __str__(self):
        return f"{self.rule:6} {self.where:10} «{self.text}»  — {self.msg}"


def _first_noun(tokens: list[str], start: int, d: Dict, limit: int = 4) -> tuple[int, str] | None:
    """The first noun form after `start`, skipping words that cannot be the head noun; stops at a conjunction."""
    for i in range(start, min(len(tokens), start + limit)):
        w = tokens[i]
        if w in STOP or w in PREPS or w in ADVERBIAL and w not in ("sve",):
            return None
        if w in NUMERALS:
            continue
        if d.is_noun_form(w) and not (w in d.adj and not d.is_noun_form(w)):
            # a word that is also a verb form or an adjective is not trusted as the head when a better one follows
            return i, w
        if w in POSS_WORDS or w in d.adj or w in NUMERALS or w in ("sve", "to", "ovo", "ovaj", "ova", "taj", "ta"):
            continue
        # unknown word: it may be a name or a noun the cache lacks; stop rather than guess
        return None
    return None


def check_phrase(text: str, where: str, d: Dict) -> list[Finding]:
    out: list[Finding] = []
    # sentences and clauses separately, so that a rule does not run across `.`, `?` or `,`
    for clause in re.split(r"[.?!…;:,—–\-]+(?=\s|$)|\n", text):
        tokens = [t.lower() for t in TOKEN.findall(clause)]
        if len(tokens) < 2:
            continue
        for i, w in enumerate(tokens):
            nxt = i + 1
            # --- NEMAM / IMAM / DAJTE   («Koliko imaš godina?» asks for a quantity: the genitive there is right)
            if "koliko" not in tokens[:i] and (w in NOT_HAVE or w in HAVE or w in GIVE or (w == "treba" and i + 1 < len(tokens) and tokens[i + 1] in ("mi", "ti", "nam", "vam"))):
                j = nxt
                while j < len(tokens) and tokens[j] in ("li", "mi", "ti", "nam", "vam", "da"):
                    j += 1
                # `imam brata` fine, `imam dva brata` is the number rule's business
                if j < len(tokens) and tokens[j] not in NUMERALS:
                    hit = _first_noun(tokens, j, d)
                    if hit:
                        k, noun = hit
                        cases = d.noun_cases(noun)
                        if w in NOT_HAVE and "gen" not in cases:
                            out.append(Finding("NEMAM", where, clause.strip(),
                                               f"after «{w}» the genitive is needed; «{noun}» is {sorted(cases)} only"))
                        elif w in HAVE and "acc" not in cases:
                            out.append(Finding("IMAM", where, clause.strip(),
                                               f"after «{w}» the accusative is needed; «{noun}» is {sorted(cases)} only"))
                        elif w in GIVE and not (any(c == "acc" or (c == "gen" and n == "s") for c, n, _g in d.noun.get(noun, []))):
                            out.append(Finding("DAJTE", where, clause.strip(),
                                               f"after «{w}» the accusative is needed; «{noun}» is {sorted(cases)} only"))
            # --- PREPOSITIONS
            if w in PREPS and nxt < len(tokens):
                hit = _first_noun(tokens, nxt, d, limit=3)
                if hit:
                    k, noun = hit
                    cases = d.noun_cases(noun)
                    if cases and not (cases & PREPS[w]):
                        out.append(Finding("PREP", where, clause.strip(),
                                           f"«{w}» takes {sorted(PREPS[w])}; «{noun}» is {sorted(cases)} only"))
            # --- NUMERALS (the last numeral of a group governs)
            if w in NUMERALS and w != "pola" and (nxt >= len(tokens) or tokens[nxt] not in NUMERALS):
                if nxt < len(tokens):
                    noun = tokens[nxt]
                    if d.is_noun_form(noun) and noun not in NUMERALS and noun not in PREPS and noun not in ADVERBIAL and noun not in STOP:
                        readings = d.noun.get(noun, [])
                        want = None
                        if w in ONE:
                            ok = any(n == "s" for _c, n, _g in readings)
                            want = "singular"
                        elif w in PAUCAL:
                            ok = any(c == "gen" and n == "s" for c, n, _g in readings) or any(c == "nom" and n == "p" for c, n, _g in readings)
                            want = "genitive singular"
                        elif w in ("hiljadu",):
                            ok, want = True, ""
                        elif w in ("hiljade",):
                            ok = any(c == "gen" and n == "p" for c, n, _g in readings)
                            want = "genitive plural"
                        else:
                            ok = any(c == "gen" and n == "p" for c, n, _g in readings)
                            want = "genitive plural"
                        if not ok:
                            out.append(Finding("NUM", where, clause.strip(),
                                               f"«{w}» + «{noun}»: {want} needed, «{noun}» is {sorted({(c, n) for c, n, _g in readings})}"))
            if w == "pola" and nxt < len(tokens):
                noun = tokens[nxt]
                if d.is_noun_form(noun) and "gen" not in d.noun_cases(noun) and noun not in NUMERALS:
                    out.append(Finding("NUM", where, clause.strip(), f"«pola» + genitive needed; «{noun}» is {sorted(d.noun_cases(noun))}"))
            # --- POSSESSIVES / ADJECTIVES + NOUN: gender agreement
            if (w in POSS_FORMS or w in d.adj) and w not in ADVERBIAL and nxt < len(tokens):
                noun = tokens[nxt]
                gens = d.gender.get(noun)
                if gens and d.is_noun_form(noun):
                    noun_readings = d.noun.get(noun, [])
                    nr = {(c, n) for c, n, _g in noun_readings}
                    if w in POSS_FORMS:
                        mod = {(c, g) for c, g in POSS_FORMS[w]}
                        # singular noun readings only; the modifier must offer the noun's gender in a shared case
                        ok = any((c, g) in mod for c, n in nr if n == "s" for g in gens)
                        if nr and any(n == "s" for _c, n in nr) and not ok and w not in ("moje", "tvoje", "svoje"):
                            out.append(Finding("POSS", where, clause.strip(),
                                               f"«{w} {noun}»: «{noun}» is {sorted(gens)}, «{w}» does not fit"))
                        elif nr and any(n == "s" for _c, n in nr) and not ok:
                            out.append(Finding("POSS", where, clause.strip(),
                                               f"«{w} {noun}»: «{noun}» is {sorted(gens)}, «{w}» does not fit"))
                    else:
                        mod = {(c, n, g) for c, n, g in d.adj.get(w, [])}
                        ok = any((c, n, g) in mod for c, n, _g0 in noun_readings for g in gens)
                        if noun_readings and not ok and w not in d.noun:
                            out.append(Finding("POSS", where, clause.strip(),
                                               f"«{w} {noun}»: «{noun}» is {sorted(gens)}, «{w}» is {sorted({(c, n, g) for c, n, g in d.adj.get(w, [])})[:4]}…"))
            # --- «Ovo je» + noun
            if w in ("ovo", "to") and nxt + 1 < len(tokens) and tokens[nxt] in ("je", "su"):
                hit = _first_noun(tokens, nxt + 1, d, limit=3)
                if hit:
                    _k, noun = hit
                    cases = d.noun_cases(noun)
                    if cases and "nom" not in cases and "voc" not in cases:
                        out.append(Finding("COP", where, clause.strip(), f"after «{w} {tokens[nxt]}» the nominative is needed; «{noun}» is {sorted(cases)} only"))
            # --- PRONOUN + VERB: person and number
            if w in PERSON and nxt < len(tokens) and not (w == "mi" and tokens[nxt] == "je"):   # «mi je» is «мне», not «мы»
                v = tokens[nxt]
                if v in BITI:
                    if BITI[v] != PERSON[w]:
                        out.append(Finding("PERS", where, clause.strip(), f"«{w} {v}»: person/number do not agree"))
                elif v in d.verb and v not in d.noun:
                    if PERSON[w] not in d.verb[v]:
                        out.append(Finding("PERS", where, clause.strip(), f"«{w} {v}»: the verb is {sorted(set(d.verb[v]))}"))
    return out


# ------------------------------------------------------------------ corpus ----

def split_prompt(prompt: str, answer: str) -> str:
    """`«Это брат» → Ovo je ___.` + `brat` → `Ovo je brat.`"""
    p = prompt.split("→", 1)[1] if "→" in prompt else prompt
    return p.replace("___", answer) if "___" in p else answer


LATIN_WORD = re.compile(rf"[{LET}]{{2,}}")


def phrases_of(lid: str) -> list[tuple[str, str]]:
    """(where, serbian phrase) for everything in lesson `lid` that must be correct."""
    path = os.path.join(CONTENT, "lessons", f"{lid}.yaml")
    d = yaml.safe_load(open(path, encoding="utf-8"))
    out: list[tuple[str, str]] = []

    def exercise(e: dict, where: str):
        t = e["type"]
        if t == "choice":
            if e.get("say"):
                out.append((where, e["say"]))
            elif LATIN_WORD.search(e.get("answer", "")) and not re.search(r"[А-Яа-яЁё]", e["answer"]):
                out.append((where, e["answer"]))
        elif t == "match":
            for a, _b in e["pairs"]:
                out.append((where, a))
        elif t in ("translate", "listen", "word_bank", "fix_error"):
            for a in e.get("accept") or []:
                out.append((where, a))
            if t == "listen":
                out.append((where, e["say"]))
        elif t == "fill_blank":
            for a in e.get("accept") or []:
                if " " in a:
                    out.append((where, a))
                else:
                    out.append((where, split_prompt(e["prompt"], a)))

    for s in d["steps"]:
        for e in s.get("exercises") or []:
            exercise(e, e["id"])
        for i, t in enumerate(s.get("turns") or []):
            out.append((f"{s['id']}-t{i + 1}", t["sr"]))
        if s["kind"] == "teach" and s.get("md"):
            txt = open(os.path.join(CONTENT, "lessons", s["md"]), encoding="utf-8").read()
            for m in re.finditer(r"`([^`]+)`", txt):
                out.append((s["id"], m.group(1)))
        if s["kind"] == "reading" and s.get("md"):
            txt = open(os.path.join(CONTENT, "lessons", s["md"]), encoding="utf-8").read()
            out.append((s["id"], txt.split("\n---")[0]))
    return out


def lessons_of_levels(levels: list[str]) -> list[str]:
    c = yaml.safe_load(open(os.path.join(CONTENT, "course.yaml"), encoding="utf-8"))
    return [lid for ph in c["phases"] if ph["id"] in levels for lid in ph["lessons"]]


def all_words(lessons: list[str]) -> list[str]:
    seen = set()
    for lid in lessons:
        for _w, p in phrases_of(lid):
            seen.update(t.lower() for t in TOKEN.findall(p))
    return sorted(seen)


# ------------------------------------------------------------------ self test ----

GOOD = ["Imam sestru.", "Nemam sestre.", "Imam brata.", "Nemam brata.", "Dajte mi vodu.", "Dva dinara.", "Pet dinara.",
        "Imam dvadeset jednu godinu.", "Imam dvadeset dve godine.", "Imam trideset godina.", "Moja mama.", "Moj tata.",
        "Moje dete.", "Ovo je sestra.", "Ja sam iz Rusije.", "Ti si student.", "Mi smo kod kuće.", "Nemam para.",
        "Pola kile sira.", "Dve hiljade dinara.", "Pet hiljada dinara.", "Hiljadu dinara."]
BAD = ["Imam sestra.", "Nemam sestru.", "Nemam račun.", "Moj mama.", "Moja tata.", "Dva dinar.", "Pet dinari.",
       "Imam trideset godine.", "Imam dvadeset dve godina.", "Dajte mi voda.", "Ovo je sestru.", "Ja si iz Rusije.",
       "Mi sam kod kuće.", "Tvoja brat.", "Moje sestra."]


def self_test(d: Dict) -> int:
    bad = 0
    for g in GOOD:
        f = check_phrase(g, "good", d)
        if f:
            bad += 1
            print("FALSE ALARM:", g, "→", f[0])
    for b in BAD:
        f = check_phrase(b, "bad", d)
        if not f:
            bad += 1
            print("MISSED:", b)
    print(f"self-test: {len(GOOD)} good phrases, {len(BAD)} planted mistakes, {bad} problems")
    return 1 if bad else 0


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--levels")
    ap.add_argument("--lesson", nargs="*")
    ap.add_argument("--all", action="store_true")
    ap.add_argument("--fetch", action="store_true", help="read the Wiktionary pages of the lessons' words first")
    ap.add_argument("--self-test", action="store_true")
    a = ap.parse_args()

    if a.self_test:
        words = sorted({t.lower() for s in GOOD + BAD for t in TOKEN.findall(s)})
        W.lookup(words, quiet=True)
        return self_test(Dict(W.load_cache()))
    if a.all:
        lessons = [os.path.basename(p)[:-5] for p in sorted(glob.glob(os.path.join(CONTENT, "lessons", "*.yaml"))) if not os.path.basename(p).startswith("_")]
    elif a.lesson:
        lessons = a.lesson
    elif a.levels:
        lessons = lessons_of_levels(a.levels.split(","))
    else:
        ap.error("choose --levels, --lesson or --all")
    if a.fetch:
        W.lookup(all_words(lessons))
    d = Dict(W.load_cache())
    total = 0
    for lid in lessons:
        seen = set()
        for where, phrase in phrases_of(lid):
            for f in check_phrase(phrase, where, d):
                key = (f.rule, f.text, f.msg)
                if key in seen:
                    continue
                seen.add(key)
                total += 1
                print(f"[{lid}] {f}")
    print(f"{len(lessons)} lessons, {total} findings to read", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
