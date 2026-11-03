#!/usr/bin/env python3
"""Озвучка истории (scripts/story_specs/<slug>.yaml): клипы по предложениям + метки слов.

Для каждого предложения делает два клипа (сербский голосом говорящего и русский),
обрезает хвостовую тишину edge-tts (около 0,9 с) по концу последней метки слова и пишет
manifest.json с длительностями и метками. Загрузки в хранилище здесь нет: это следующий шаг.

Результат: scripts/.cache/stories/<slug>/ (в git не попадает)
    s01_sr.mp3  s01_ru.mp3  ...  manifest.json

Usage:
    pip install edge-tts pyyaml
    python3 scripts/story_assets.py drago-mi-je
    python3 scripts/story_assets.py drago-mi-je --force     # переозвучить всё

Движок голоса изолирован в synthesize(): замена edge-tts на платный движок не должна менять
ни спеку, ни манифест, ни плеер (нужны только mp3 и метки слов в мс).

Готовые записи (ElevenLabs, живой диктор): положить mp3 одной реплики в scripts/.cache/stories/<slug>/voice/
под именем s01_sr.mp3 (русская: s01_ru.mp3). Такая запись заменяет синтез этого клипа: скрипт находит начало и конец
речи, подрезает тишину, выравнивает громкость и строит метки слов по длине слов (приблизительно, помечено в json).
"""
from __future__ import annotations

import argparse
import asyncio
import json
import os
import re
import subprocess
import sys

import yaml

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from tts import sr_lat_to_cyr  # noqa: E402

try:
    import edge_tts
except ImportError:
    sys.exit("edge-tts не установлен — pip install edge-tts")

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SPECS = os.path.join(ROOT, "scripts", "story_specs")
OUT = os.path.join(ROOT, "scripts", ".cache", "stories")
TAIL_MS = 150          # сколько оставить после конца последнего слова
FAST_RATE = 1.0        # playbackRate третьего прохода: обычная скорость (быстрый вариант 1,25 выбирался на слух, образец D)
SLOW_RATE = 0.9        # playbackRate первого прохода: первое прослушивание помедленнее
CONCURRENCY = 4
VOICE_DIR = "voice"      # готовые записи по репликам (s01_sr.mp3), заменяют синтез
LEAD_MS = 60           # тишина перед речью в обработанной записи
# Готовые записи: мягкий компрессор срезает пики внутри реплики, двухпроходный loudnorm выравнивает громкость реплик (−16 LUFS), лимитер
# оставляет запас под mp3-кодек, чтобы на телефоне и в наушниках не трещало.
PRE_FILTER = "acompressor=threshold=-18dB:ratio=2.5:attack=8:release=100:makeup=1"
LIMIT = "alimiter=limit=0.79:level=disabled"


async def synthesize(text: str, voice: str) -> tuple[bytes, list[dict]]:
    """Единственное место, где знают про движок. Возвращает mp3 и метки слов:
    [{"text", "start_ms", "end_ms"}] в порядке слов (по пробелам)."""
    comm = edge_tts.Communicate(text, voice, boundary="WordBoundary")
    audio = bytearray()
    marks: list[dict] = []
    async for ch in comm.stream():
        if ch["type"] == "audio":
            audio += ch["data"]
        elif ch["type"] == "WordBoundary":
            start = ch["offset"] / 10000
            marks.append({"text": ch["text"], "start_ms": round(start), "end_ms": round(start + ch["duration"] / 10000)})
    return bytes(audio), marks


def trim(src: str, dst: str, end_ms: int) -> None:
    subprocess.run(
        ["ffmpeg", "-y", "-loglevel", "error", "-i", src, "-t", f"{(end_ms + TAIL_MS) / 1000:.3f}",
         "-ar", "24000", "-ac", "1", "-b:a", "48k", dst],
        check=True,
    )


def display_words(text: str) -> list[str]:
    """Слова для показа: по пробелам, без знаков по краям (как режет edge-tts)."""
    return [w.strip(".,!?;:—–-«»\"'()") for w in text.split() if w.strip(".,!?;:—–-«»\"'()")]


def probe_ms(path: str) -> int:
    out = subprocess.run(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=nw=1:nk=1", path],
                         capture_output=True, text=True, check=True).stdout
    return round(float(out.strip()) * 1000)


def speech_span(path: str) -> tuple[int, int]:
    """Начало и конец речи в записи (мс) по тишине: без неё в начале и конце, внутренние паузы остаются."""
    total = probe_ms(path)
    err = subprocess.run(["ffmpeg", "-hide_banner", "-nostats", "-i", path, "-af", "silencedetect=noise=-40dB:d=0.1", "-f", "null", "-"],
                         capture_output=True, text=True).stderr
    starts = [float(x) for x in re.findall(r"silence_start: (-?[\d.]+)", err)]
    ends = [float(x) for x in re.findall(r"silence_end: ([\d.]+)", err)]
    start, end = 0, total
    if starts and starts[0] <= 0.05:
        if not ends:
            raise SystemExit(f"{path}: в записи нет речи")
        start = round(ends[0] * 1000)
    if starts and (len(starts) > len(ends) or ends[-1] >= (total - 30) / 1000) and starts[-1] > 0.05:
        end = round(starts[-1] * 1000)
    return start, end


def estimate_marks(shown: str, lead_ms: int, speech_ms: int) -> list[dict]:
    """Метки слов по длине слов, когда движок их не отдаёт: речь делится между словами пропорционально буквам, после слова с
    запятой, точкой или многоточием остаётся пауза. Подсветка может расходиться с голосом на десятые доли секунды."""
    tokens = [t for t in shown.split() if t.strip(".,!?;:—–-«»\"'()")]
    letters = [max(2, len(re.sub(r"[^\w]", "", t))) for t in tokens]
    pauses = [4 if t.endswith(("...", "…")) else 3 if t.endswith((".", "!", "?")) else 2 if t.endswith(",") else 0 for t in tokens]
    pauses[-1] = 0
    unit = speech_ms / (sum(letters) + sum(pauses))
    out, at = [], 0.0
    for n, p in zip(letters, pauses):
        out.append({"start_ms": lead_ms + round(at * unit), "end_ms": lead_ms + round((at + n) * unit)})
        at += n + p
    out[-1]["end_ms"] = lead_ms + speech_ms
    return out


def level_filter(rec: str, ss: float, t: float, offset_db: float = 0.0) -> str:
    """Цепочка выравнивания громкости для одной записи. loudnorm в два прохода: первый измеряет, второй применяет линейно,
    поэтому реплики получают одну громкость без «дыхания». На совсем коротких репликах измерение не выходит, тогда проход один."""
    cut = ["-ss", f"{ss:.3f}", "-t", f"{t:.3f}", "-i", rec]
    target = f"I={-16 + offset_db:g}:TP=-0.5:LRA=11"
    err = subprocess.run(["ffmpeg", "-hide_banner", "-nostats", *cut, "-af", f"{PRE_FILTER},loudnorm={target}:print_format=json", "-f", "null", "-"],
                         capture_output=True, text=True).stderr
    blocks = re.findall(r"\{[^{}]*\}", err)
    try:
        j = json.loads(blocks[-1])
        i = float(j["input_i"])
        if i > -45:
            return (f"{PRE_FILTER},loudnorm={target}:measured_I={j['input_i']}:measured_TP={j['input_tp']}:measured_LRA={j['input_lra']}"
                    f":measured_thresh={j['input_thresh']}:offset={j['target_offset']}:linear=true,{LIMIT}")
    except (IndexError, ValueError, KeyError):
        pass
    return f"{PRE_FILTER},loudnorm={target},{LIMIT}"


def from_recording(rec: str, path: str, meta_path: str, key: str, shown: str, offset_db: float = 0.0) -> dict:
    start, end = speech_span(rec)
    lead = min(start, LEAD_MS)
    begin = start - lead
    ss, t = begin / 1000, (end + TAIL_MS - begin) / 1000
    subprocess.run(
        ["ffmpeg", "-y", "-loglevel", "error", "-ss", f"{ss:.3f}", "-t", f"{t:.3f}", "-i", rec,
         "-af", level_filter(rec, ss, t, offset_db), "-ar", "24000", "-ac", "1", "-b:a", "48k", path],
        check=True,
    )
    speech_ms = end - start
    words = display_words(shown)
    marks = estimate_marks(shown, lead, speech_ms)
    meta = {
        "file": f"{key}.mp3",
        "ms": lead + speech_ms + TAIL_MS,
        "words": [{"w": w, "start_ms": m["start_ms"], "end_ms": m["end_ms"]} for w, m in zip(words, marks)],
        "source": "recording", "marks": "estimated", "offset_db": offset_db,
    }
    json.dump(meta, open(meta_path, "w", encoding="utf-8"), ensure_ascii=False)
    return meta


async def one(sem: asyncio.Semaphore, outdir: str, key: str, speech: str, shown: str, voice: str, force: bool, offset_db: float = 0.0) -> dict:
    path = os.path.join(outdir, f"{key}.mp3")
    meta_path = os.path.join(outdir, f"{key}.json")
    rec = os.path.join(outdir, VOICE_DIR, f"{key}.mp3")
    if os.path.exists(rec):
        fresh = os.path.exists(path) and os.path.exists(meta_path) and os.path.getmtime(meta_path) >= os.path.getmtime(rec)
        old = json.load(open(meta_path, encoding="utf-8")) if os.path.exists(meta_path) else {}
        if not force and fresh and old.get("source") == "recording" and old.get("offset_db", 0.0) == offset_db:
            return old
        return from_recording(rec, path, meta_path, key, shown, offset_db)
    if not force and os.path.exists(path) and os.path.exists(meta_path):
        return json.load(open(meta_path, encoding="utf-8"))
    async with sem:
        audio, marks = await synthesize(speech, voice)
    words = display_words(shown)
    if len(marks) != len(words):
        raise SystemExit(f"{key}: меток {len(marks)}, слов {len(words)} ({shown!r}), подсветку по словам не построить")
    raw = path + ".raw.mp3"
    open(raw, "wb").write(audio)
    end = marks[-1]["end_ms"]
    trim(raw, path, end)
    os.remove(raw)
    meta = {
        "file": f"{key}.mp3",
        "ms": end + TAIL_MS,
        "words": [{"w": w, "start_ms": m["start_ms"], "end_ms": m["end_ms"]} for w, m in zip(words, marks)],
    }
    json.dump(meta, open(meta_path, "w", encoding="utf-8"), ensure_ascii=False)
    return meta


async def build(slug: str, force: bool) -> None:
    spec = yaml.safe_load(open(os.path.join(SPECS, f"{slug}.yaml"), encoding="utf-8"))
    outdir = os.path.join(OUT, slug)
    os.makedirs(outdir, exist_ok=True)
    sem = asyncio.Semaphore(CONCURRENCY)
    jobs = []
    for s in spec["sentences"]:
        k = f"s{s['n']:02d}"
        off = spec.get("voice_offsets_db") or {}
        jobs.append(one(sem, outdir, f"{k}_sr", sr_lat_to_cyr(s["sr"]), s["sr"], spec["voices"][s["who"]], force, off.get(f"{s['who']}_sr", 0.0)))
        jobs.append(one(sem, outdir, f"{k}_ru", s["ru"], s["ru"], spec["voices"]["ru"], force, off.get(f"{s['who']}_ru", 0.0)))
    metas = await asyncio.gather(*jobs)
    sentences = []
    for i, s in enumerate(spec["sentences"]):
        sr, ru = metas[2 * i], metas[2 * i + 1]
        sentences.append({
            "n": s["n"], "who": s["who"], "scene": s["scene"],
            "sr": s["sr"], "ru": s["ru"],
            "sr_audio": sr["file"], "sr_ms": sr["ms"], "sr_words": sr["words"],
            "ru_audio": ru["file"], "ru_ms": ru["ms"], "ru_words": ru["words"],
        })
    manifest = {
        "slug": spec["slug"], "title_sr": spec["title_sr"], "title_ru": spec["title_ru"],
        "level": spec["level"], "tier": spec["tier"], "is_demo": spec["is_demo"],
        "fast_rate": FAST_RATE,
        "slow_rate": SLOW_RATE,
        "approx_minutes": spec.get("approx_minutes", 0),
        "price": int(spec.get("price", 0)),
        "recommended_lesson": str(spec.get("recommended_lesson", "")),
        "difficulty": int(spec.get("difficulty", 0)),
        "scenes": spec["scenes"], "sentences": sentences,
        "quiz_inline": spec.get("quiz_inline") or [], "quiz_final": spec.get("quiz_final") or [],
    }
    json.dump(manifest, open(os.path.join(outdir, "manifest.json"), "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    sr_total = sum(x["sr_ms"] for x in sentences)
    ru_total = sum(x["ru_ms"] for x in sentences)
    print(f"{slug}: {len(sentences)} предложений; сербский {sr_total / 1000:.1f} с, русский {ru_total / 1000:.1f} с -> {outdir}")


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("slug")
    ap.add_argument("--force", action="store_true")
    args = ap.parse_args()
    asyncio.run(build(args.slug, args.force))


if __name__ == "__main__":
    main()
