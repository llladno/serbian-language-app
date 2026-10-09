#!/usr/bin/env python3
"""Generate Serbian pronunciation audio for vocab entries with edge-tts.

edge-tts uses Microsoft Edge's online "Read Aloud" service — free, no API key,
no account. Serbian neural voices: sr-RS-SophieNeural (f), sr-RS-NicholasNeural (m).

Usage:
    pip install edge-tts pyyaml
    python3 scripts/tts.py                 # fill in missing files
    python3 scripts/tts.py --force         # regenerate everything
    python3 scripts/tts.py --voice sr-RS-NicholasNeural
    python3 scripts/tts.py --only zdravo --only hleb

Writes content/audio/<id>.mp3 for every entry in content/vocab.yaml, for
every `type: listen` exercise (keyed by exercise id, e.g. 01-E-1.mp3) and for
every dialogue turn (keyed <step id>-t<N>, e.g. 05.9-t1.mp3 — the two speakers
get different voices).
Phrases shown in lessons (teach-text code spans, exercise options/answers,
reading lines) go to content/audio/p/<hash>.mp3 + index.json (see phrase_key).
Re-run any time; existing files are skipped unless --force.

The sr-RS voices are Cyrillic-trained and mispronounce Latin text, so
everything is synthesized from Cyrillic — vocab via its `cyrillic` field,
listen exercises via sr_lat_to_cyr() on the Latin `say`.
"""
from __future__ import annotations

import argparse
import asyncio
import json
import os
import re
import sys
import unicodedata

import yaml

try:
    import edge_tts
except ImportError:
    sys.exit("edge-tts not installed — run: pip install edge-tts")

import glob

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
VOCAB = os.path.join(ROOT, "content", "vocab.yaml")
EXERCISES_GLOB = os.path.join(ROOT, "content", "exercises", "*.yaml")
LESSONS_GLOB = os.path.join(ROOT, "content", "lessons", "*.yaml")
AUDIO_DIR = os.path.join(ROOT, "content", "audio")
VOICE_F = "sr-RS-SophieNeural"
VOICE_M = "sr-RS-NicholasNeural"
DEFAULT_VOICE = VOICE_F
CONCURRENCY = 4


# Serbian is a 1:1 Latin<->Cyrillic script pair. The sr-RS neural voices are
# trained on Cyrillic and mangle Latin input (English-ish phonetics), so every
# string handed to the engine must be Cyrillic — hence speech_text prefers the
# `cyrillic` field for vocab, and listen exercises get transliterated here.
_SR_DIGRAPHS = [
    ("DŽ", "Џ"), ("Dž", "Џ"), ("dž", "џ"),
    ("LJ", "Љ"), ("Lj", "Љ"), ("lj", "љ"),
    ("NJ", "Њ"), ("Nj", "Њ"), ("nj", "њ"),
]
_SR_MAP = str.maketrans({
    "a": "а", "b": "б", "c": "ц", "č": "ч", "ć": "ћ", "d": "д", "đ": "ђ",
    "e": "е", "f": "ф", "g": "г", "h": "х", "i": "и", "j": "ј", "k": "к",
    "l": "л", "m": "м", "n": "н", "o": "о", "p": "п", "r": "р", "s": "с",
    "š": "ш", "t": "т", "u": "у", "v": "в", "z": "з", "ž": "ж",
    "A": "А", "B": "Б", "C": "Ц", "Č": "Ч", "Ć": "Ћ", "D": "Д", "Đ": "Ђ",
    "E": "Е", "F": "Ф", "G": "Г", "H": "Х", "I": "И", "J": "Ј", "K": "К",
    "L": "Л", "M": "М", "N": "Н", "O": "О", "P": "П", "R": "Р", "S": "С",
    "Š": "Ш", "T": "Т", "U": "У", "V": "В", "Z": "З", "Ž": "Ж",
})


def sr_lat_to_cyr(s: str) -> str:
    """Transliterate Serbian Latin to Cyrillic. Digraphs (lj/nj/dž) first; other
    characters (spaces, digits, punctuation) pass through unchanged. Does not
    handle the rare non-digraph d+ž / n+j / l+j sequences — none occur in the
    current content, add a say_cyrillic override if that ever changes."""
    for lat, cyr in _SR_DIGRAPHS:
        s = s.replace(lat, cyr)
    return s.translate(_SR_MAP)


def listen_entries() -> list[dict]:
    """Collect {id, cyrillic} rows for every `type: listen` exercise with a
    `say:` field, so they get an audio clip keyed by exercise id (01-E-1.mp3
    for the legacy model, 01.8.1.mp3 for the manifest model). `say` is authored
    in Latin (like the rest of the content) and transliterated to Cyrillic so
    the Serbian voice pronounces it correctly."""
    out, seen = [], set()

    def add(ex):
        if ex.get("type") == "listen" and ex.get("say") and ex.get("id") not in seen:
            seen.add(ex["id"])
            out.append({"id": ex["id"], "cyrillic": sr_lat_to_cyr(ex["say"])})

    for path in sorted(glob.glob(EXERCISES_GLOB)):        # legacy: exercises/NN.yaml
        if os.path.basename(path) == "_TEMPLATE.yaml":
            continue
        doc = yaml.safe_load(open(path, encoding="utf-8")) or {}
        for block in doc.get("blocks") or []:
            for ex in block.get("exercises") or []:
                add(ex)

    for path in sorted(glob.glob(LESSONS_GLOB)):          # manifest: lessons/NN.yaml
        if os.path.basename(path) == "_TEMPLATE.yaml":
            continue
        doc = yaml.safe_load(open(path, encoding="utf-8")) or {}
        for step in doc.get("steps") or []:
            for ex in step.get("exercises") or []:
                add(ex)

    return out


def dialogue_entries() -> list[dict]:
    """Collect {id, cyrillic, voice} rows for every dialogue turn, keyed
    <step id>-t<N> (05.9-t1.mp3). The other speaker uses the step's `voice`
    (f by default) and the learner's own lines always take the opposite one,
    so the chat has two distinct voices."""
    out = []
    for path in sorted(glob.glob(LESSONS_GLOB)):
        if os.path.basename(path) == "_TEMPLATE.yaml":
            continue
        doc = yaml.safe_load(open(path, encoding="utf-8")) or {}
        for step in doc.get("steps") or []:
            if step.get("kind") != "dialogue":
                continue
            npc = VOICE_M if step.get("voice") == "m" else VOICE_F
            me = VOICE_F if npc == VOICE_M else VOICE_M
            for i, turn in enumerate(step.get("turns") or [], start=1):
                if not turn.get("sr"):
                    continue
                out.append({
                    "id": f"{step['id']}-t{i}",
                    "cyrillic": sr_lat_to_cyr(turn["sr"]),
                    "voice": npc if turn.get("who") == "npc" else me,
                })
    return out


# ---- phrase clips ---------------------------------------------------------
# Every Serbian phrase shown in a lesson (teach text, exercise options and
# answers, reading lines) gets content/audio/p/<key>.mp3, where <key> is a
# hash of the normalised text. The web app computes the same hash
# (web/src/lib/phraseAudio.ts) and shows a speaker button when the key is in
# content/audio/p/index.json — so no id plumbing, and a changed phrase simply
# gets a new clip instead of playing a stale one.
PHRASE_DIR = os.path.join(AUDIO_DIR, "p")
LESSON_MD_GLOB = os.path.join(ROOT, "content", "lessons", "*", "*.md")


def _phrase_norm(text: str) -> str:
    t = unicodedata.normalize("NFC", text).lower()
    return " ".join("".join(c if c.isalnum() else " " for c in t).split())


def phrase_key(text: str) -> str:
    """FNV-1a 64 of the normalised text, 16 hex chars. Keep in sync with
    phraseKey() in web/src/lib/phraseAudio.ts."""
    h = 0xCBF29CE484222325
    for b in _phrase_norm(text).encode("utf-8"):
        h = ((h ^ b) * 0x100000001B3) & 0xFFFFFFFFFFFFFFFF
    return f"{h:016x}"


def phrase_speech(text: str) -> str:
    """Cyrillic string for the voice: all slash-variants are read out;
    parentheticals, ellipses and blanks are dropped."""
    t = re.sub(r"\([^)]*\)", "", text)
    t = t.replace("…", "").replace("...", "").replace("___", "")
    t = re.sub(r"\s*/\s*", ", ", t)
    t = re.sub(r"\s+([?!.,;:])", r"\1", t)
    t = re.sub(r"\s+", " ", t).strip(" -–—,")
    return sr_lat_to_cyr(t)


_SOUND_ONLY = {"nj", "lj", "dž", "dj", "ž", "š", "č", "ć", "đ"}
_LATIN = re.compile(r"[A-Za-zČčĆćŠšŽžĐđ]")
_BAD_CHARS = re.compile(r"[+=→<>*_\n\d]|[Ѐ-ӿ]")


def is_serbian_phrase(text: str) -> bool:
    """True for a code span / quoted string that is plain Serbian Latin text —
    not a suffix (-im), a single sound, a formula (sam + …) or a number."""
    t = text.strip()
    if not t or _BAD_CHARS.search(t) or t.startswith("-") or t.endswith("-"):
        return False
    return len(_LATIN.findall(t)) >= 2 and _phrase_norm(t) not in _SOUND_ONLY


def md_phrases(md: str) -> list[str]:
    """Serbian code spans of a lesson fragment."""
    return [m for m in re.findall(r"`([^`]+)`", md) if is_serbian_phrase(m)]


def prompt_phrases(prompt: str) -> list[str]:
    """Serbian «quoted» segments of an exercise prompt (blanks excluded —
    voicing a gap would give the answer away)."""
    return [m for m in re.findall(r"«([^»]+)»", prompt or "") if "___" not in m and is_serbian_phrase(m)]


def fill_sentence(prompt: str, word: str) -> str | None:
    """Full Serbian sentence of a fill-in-the-blank prompt with the gap filled.
    Keep in sync with fillSentence() in web/src/lib/phraseAudio.ts."""
    if not word or len(re.findall(r"_{2,}", prompt)) != 1:
        return None
    quoted = next((m for m in re.findall(r"«([^»]*)»", prompt) if "___" in m), None)
    if quoted is not None:
        t = quoted
    elif "→" in prompt:
        t = prompt[prompt.rindex("→") + 1:]
    elif " — " in prompt and "___" in prompt[prompt.rindex(" — "):]:
        t = prompt[prompt.rindex(" — ") + 3:]
    else:
        t = prompt
    t = re.sub(r"\([^)]*\)", "", t)
    t = re.sub(r"_{2,}", lambda _: word, t, count=1)
    t = re.sub(r"\s+", " ", t).strip()
    return t if t and not re.search(r"[\u0400-\u04FF]", t) else None


def exercise_phrases(ex: dict) -> list[str]:
    """Serbian phrases of one exercise that may be voiced. Never an option that
    is not the right answer (choice distractors, word-bank chips), and nothing
    for dictations (they have their own clip)."""
    t = ex.get("type")
    out = prompt_phrases(ex.get("prompt", ""))
    if t == "choice":
        out.append(ex.get("answer") or "")
    elif t == "word_bank":
        out += ex.get("accept") or []
    elif t == "match":
        out += [x for pair in (ex.get("pairs") or []) for x in pair]
    elif t == "fill_blank":
        out += [fill_sentence(ex.get("prompt", ""), w) or "" for w in ex.get("accept") or []]
    elif t in ("translate", "fix_error"):
        out += ex.get("accept") or []
    return [x for x in out if isinstance(x, str) and is_serbian_phrase(x)]


def reading_lines(md: str) -> list[str]:
    """Serbian lines of a reading fragment (everything before the first `---`),
    minus the leading dash of a dialogue line."""
    head = re.split(r"^---\s*$", md, maxsplit=1, flags=re.M)[0]
    out = []
    for line in head.splitlines():
        line = re.sub(r"^\s*[—–-]\s*", "", line).strip()
        if line:
            out.append(line)
    return out


def phrase_texts() -> list[str]:
    """Every phrase that should have a clip: first-seen spelling, de-duplicated by key."""
    found: list[str] = []
    for path in sorted(glob.glob(LESSON_MD_GLOB)):
        md = open(path, encoding="utf-8").read()
        found += md_phrases(md)
        if "citanje" in os.path.basename(path) or "reading" in os.path.basename(path):
            found += reading_lines(md)
    for path in sorted(glob.glob(LESSONS_GLOB)):
        if os.path.basename(path) == "_TEMPLATE.yaml":
            continue
        doc = yaml.safe_load(open(path, encoding="utf-8")) or {}
        for step in doc.get("steps") or []:
            exs = list(step.get("exercises") or [])
            for t in step.get("turns") or []:
                if t.get("exercise"):
                    exs.append(t["exercise"])
            for ex in exs:
                found += exercise_phrases(ex)
    seen, out = set(), []
    for t in found:
        k = phrase_key(t)
        if k not in seen and phrase_speech(t):
            seen.add(k)
            out.append(t)
    return out


def phrase_entries() -> list[dict]:
    return [{"id": "p/" + phrase_key(t), "cyrillic": phrase_speech(t), "exact": True} for t in phrase_texts()]


def prune_phrase_clips() -> int:
    """Delete phrase clips no current lesson text refers to."""
    want = {phrase_key(t) for t in phrase_texts()}
    n = 0
    for f in os.listdir(PHRASE_DIR) if os.path.isdir(PHRASE_DIR) else []:
        if f.endswith(".mp3") and f[:-4] not in want:
            os.remove(os.path.join(PHRASE_DIR, f))
            n += 1
    return n


def write_phrase_index() -> int:
    """List every phrase clip on disk, for the web app."""
    os.makedirs(PHRASE_DIR, exist_ok=True)
    keys = sorted(f[:-4] for f in os.listdir(PHRASE_DIR) if f.endswith(".mp3"))
    with open(os.path.join(PHRASE_DIR, "index.json"), "w") as f:
        json.dump(keys, f)
    return len(keys)


def speech_text(entry: dict) -> str:
    """Pick a clean, speakable string from a vocab entry.

    Prefers Cyrillic (unambiguous for the engine). Takes the first slash-
    variant, drops parentheticals and ellipses, trims stray punctuation.
    """
    raw = (entry.get("cyrillic") or entry.get("latin") or "").strip()
    raw = raw.split("/")[0]                 # "један / једна" -> "један"
    raw = re.sub(r"\([^)]*\)", "", raw)     # "радити (радим)" -> "радити"
    had_ellipsis = "…" in raw or "..." in raw
    raw = raw.replace("…", "").replace("...", "")
    raw = re.sub(r"\s+", " ", raw).strip()
    if had_ellipsis:
        raw = raw.rstrip(" ?!.,")           # "како се каже ?" -> "како се каже"
    return raw.strip(" -–—")


async def synth(sem, voice, text, dest):
    async with sem:
        try:
            await edge_tts.Communicate(text, voice).save(dest)
            return dest, None
        except Exception as e:  # noqa: BLE001
            return dest, e


async def run(entries, voice, force):
    os.makedirs(PHRASE_DIR, exist_ok=True)
    sem = asyncio.Semaphore(CONCURRENCY)
    tasks, planned = [], []
    for e in entries:
        eid = e["id"]
        dest = os.path.join(AUDIO_DIR, eid + ".mp3")
        if os.path.exists(dest) and not force:
            continue
        text = e["cyrillic"] if e.get("exact") else speech_text(e)
        if not text:
            print(f"  ! {eid}: no speakable text, skipped")
            continue
        planned.append((eid, text))
        tasks.append(synth(sem, e.get("voice") or voice, text, dest))

    if not tasks:
        print("nothing to do — all audio present (use --force to regenerate)")
        print(f"phrase index: {write_phrase_index()} clips")
        return 0

    print(f"generating {len(tasks)} file(s)\n")
    ok = errs = 0
    results = await asyncio.gather(*tasks)
    for (eid, text), (dest, err) in zip(planned, results):
        if err:
            errs += 1
            print(f"  ! {eid}: {err}")
        else:
            ok += 1
            print(f"  ok {eid:24s} “{text}”  {os.path.getsize(dest) // 1024} KB")
    print(f"\ndone: {ok} written, {errs} failed")
    print(f"phrase index: {write_phrase_index()} clips")
    return 1 if errs else 0


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--voice", default=DEFAULT_VOICE)
    ap.add_argument("--force", action="store_true", help="regenerate existing files")
    ap.add_argument("--prune", action="store_true", help="delete phrase clips no lesson text uses any more")
    ap.add_argument("--only", action="append", default=[], metavar="ID", help="limit to these ids (repeatable)")
    args = ap.parse_args()

    with open(VOCAB, encoding="utf-8") as f:
        entries = [e for e in yaml.safe_load(f) if isinstance(e, dict) and e.get("id")]
    entries += listen_entries()
    entries += dialogue_entries()
    entries += phrase_entries()
    if args.only:
        want = set(args.only)
        entries = [e for e in entries if e["id"] in want]
        missing = want - {e["id"] for e in entries}
        for m in sorted(missing):
            print(f"  ! id not in vocab/exercises: {m}")

    rc = asyncio.run(run(entries, args.voice, args.force))
    if args.prune:
        print(f"pruned {prune_phrase_clips()} stale phrase clip(s)")
        print(f"phrase index: {write_phrase_index()} clips")
    return rc


if __name__ == "__main__":
    sys.exit(main())
