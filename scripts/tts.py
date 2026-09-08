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

Writes content/audio/<id>.mp3 for every entry in content/vocab.yaml and for
every `type: listen` exercise (keyed by exercise id, e.g. 01-E-1.mp3).
Re-run any time; existing files are skipped unless --force.

The sr-RS voices are Cyrillic-trained and mispronounce Latin text, so
everything is synthesized from Cyrillic — vocab via its `cyrillic` field,
listen exercises via sr_lat_to_cyr() on the Latin `say`.
"""
import argparse
import asyncio
import os
import re
import sys

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
DEFAULT_VOICE = "sr-RS-SophieNeural"
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
    os.makedirs(AUDIO_DIR, exist_ok=True)
    sem = asyncio.Semaphore(CONCURRENCY)
    tasks, planned = [], []
    for e in entries:
        eid = e["id"]
        dest = os.path.join(AUDIO_DIR, eid + ".mp3")
        if os.path.exists(dest) and not force:
            continue
        text = speech_text(e)
        if not text:
            print(f"  ! {eid}: no speakable text, skipped")
            continue
        planned.append((eid, text))
        tasks.append(synth(sem, voice, text, dest))

    if not tasks:
        print("nothing to do — all audio present (use --force to regenerate)")
        return 0

    print(f"generating {len(tasks)} file(s) with {voice}\n")
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
    return 1 if errs else 0


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--voice", default=DEFAULT_VOICE)
    ap.add_argument("--force", action="store_true", help="regenerate existing files")
    ap.add_argument("--only", action="append", default=[], metavar="ID", help="limit to these ids (repeatable)")
    args = ap.parse_args()

    with open(VOCAB, encoding="utf-8") as f:
        entries = [e for e in yaml.safe_load(f) if isinstance(e, dict) and e.get("id")]
    entries += listen_entries()
    if args.only:
        want = set(args.only)
        entries = [e for e in entries if e["id"] in want]
        missing = want - {e["id"] for e in entries}
        for m in sorted(missing):
            print(f"  ! id not in vocab/exercises: {m}")

    return asyncio.run(run(entries, args.voice, args.force))


if __name__ == "__main__":
    sys.exit(main())
