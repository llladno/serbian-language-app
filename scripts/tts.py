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
every `type: listen` exercise (keyed by exercise id, e.g. 01-D-1.mp3).
Re-run any time; existing files are skipped unless --force.
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
AUDIO_DIR = os.path.join(ROOT, "content", "audio")
DEFAULT_VOICE = "sr-RS-SophieNeural"
CONCURRENCY = 4


def listen_entries() -> list[dict]:
    """Collect {id, cyrillic} rows for every `type: listen` exercise with a
    `say:` field, so they get an audio clip keyed by exercise id (01-D-1.mp3)."""
    out = []
    for path in sorted(glob.glob(EXERCISES_GLOB)):
        if os.path.basename(path) == "_TEMPLATE.yaml":
            continue
        with open(path, encoding="utf-8") as f:
            doc = yaml.safe_load(f) or {}
        for block in doc.get("blocks") or []:
            for ex in block.get("exercises") or []:
                if ex.get("type") == "listen" and ex.get("say"):
                    # reuse speech_text's cleanup via the "latin" slot
                    out.append({"id": ex["id"], "latin": ex["say"]})
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
