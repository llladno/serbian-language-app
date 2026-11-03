#!/usr/bin/env python3
"""Публикация истории: клипы и кадры + документ через служебный API.

Берёт то, что собрал scripts/story_assets.py (scripts/.cache/stories/<slug>/manifest.json), кадры из
scripts/.cache/stories/<slug>/frames/s<N>.(webp|png|jpg), копирует файлы туда, откуда их отдаёт сервер, и
отправляет документ истории в PUT /api/service/stories/<slug>.

Режим --dev: файлы копируются в data/story-media/<slug>/ (сервер отдаёт их по /story-media/), документ уходит на
локальный сервер. Режим --prod: файлы заливаются в S3 Timeweb (префикс ucimo-stories/<slug>/, public-read, имена с
хэшем содержимого), документ уходит на STORIES_SERVER; ключи S3 и токен берутся только из окружения.

Если кадра для сцены ещё нет, рисуется заглушка (пиксельный двор в цветах бренда), чтобы плеер можно было
собирать до готовых картинок. Положили свой кадр, запустили скрипт заново: заглушка заменится.

Usage:
    STORIES_SERVICE_TOKEN=dev-stories python3 scripts/story_publish.py drago-mi-je --dev
    python3 scripts/story_publish.py drago-mi-je --dev --status draft
    STORIES_SERVER=https://ucimo.ru STORIES_SERVICE_TOKEN=... S3_BUCKET=... S3_ACCESS_KEY=... S3_SECRET_KEY=... \
        python3 scripts/story_publish.py prvi-dan --prod
Переменные: STORIES_SERVICE_TOKEN (по умолчанию dev-stories в --dev), STORIES_SERVER (по умолчанию http://localhost:8080);
для --prod обязательны STORIES_SERVER, STORIES_SERVICE_TOKEN, S3_BUCKET, S3_ACCESS_KEY, S3_SECRET_KEY, а S3_ENDPOINT
(https://s3.twcstorage.ru) и S3_REGION (ru-1) имеют значения по умолчанию.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
import urllib.error
import urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CACHE = os.path.join(ROOT, "scripts", ".cache", "stories")
MEDIA_DEV = os.path.join(ROOT, "data", "story-media")


def placeholder(path: str, scene: int) -> None:
    """Пиксельный двор 4:3 из плоских блоков: кремовая стена, зелёные ставни, липа, две фигурки."""
    from PIL import Image, ImageDraw

    w, h = 160, 120
    im = Image.new("RGB", (w, h), "#8CCBC6")                      # небо (светлая бирюза бренда)
    d = ImageDraw.Draw(im)
    d.rectangle([0, 20, w, 84], fill="#F7F9F7")                    # стена
    for x in (14, 54, 94, 134):                                    # окна со ставнями
        d.rectangle([x, 34, x + 14, 56], fill="#E3B04B")
        d.rectangle([x - 5, 34, x - 1, 56], fill="#176B68")
        d.rectangle([x + 15, 34, x + 19, 56], fill="#176B68")
    d.rectangle([0, 84, w, h], fill="#B9A58A")                     # брусчатка
    tx = 20 + scene * 22
    d.rectangle([tx, 52, tx + 4, 88], fill="#6B4A2E")              # липа
    d.ellipse([tx - 16, 30, tx + 22, 62], fill="#5D9B6A")
    a = 58 + (scene % 2) * 6
    d.ellipse([a, 72, a + 12, 84], fill="#E8C9A8")                 # Ана: голова, шарф, пальто
    d.rectangle([a - 1, 84, a + 13, 88], fill="#176B68")
    d.rectangle([a, 88, a + 12, 108], fill="#F1E6D2")
    m = 98 - (scene % 3) * 5
    d.ellipse([m, 70, m + 13, 83], fill="#A87A58")                 # Марко: голова, худи
    d.rectangle([m, 83, m + 13, 106], fill="#C8602B")
    d.rectangle([m, 106, m + 13, 112], fill="#2E4A7A")
    d.ellipse([140, 8, 150, 12 + scene], fill="#176B68")           # ласточка где-то в небе
    im.resize((1200, 900), Image.NEAREST).save(path)


def to_webp(src: str, dst: str) -> None:
    subprocess.run(["cwebp", "-quiet", "-q", "82", "-resize", "1200", "0", src, "-o", dst], check=True)


S3_PREFIX = "ucimo-stories"
CONTENT_TYPES = {".mp3": "audio/mpeg", ".webp": "image/webp", ".png": "image/png", ".jpg": "image/jpeg"}


def need_env(*names: str) -> dict[str, str]:
    missing = [n for n in names if not os.environ.get(n)]
    if missing:
        sys.exit("для --prod нужны переменные окружения: " + ", ".join(missing))
    return {n: os.environ[n] for n in names}


def upload_to_s3(slug: str, outdir: str, names: list[str]) -> str:
    """Кладёт файлы в публичный S3 и возвращает адрес папки истории. Имена с хэшем, поэтому кэш у них вечный."""
    import boto3
    from botocore.config import Config

    env = need_env("S3_BUCKET", "S3_ACCESS_KEY", "S3_SECRET_KEY")
    endpoint = os.environ.get("S3_ENDPOINT", "https://s3.twcstorage.ru").rstrip("/")
    client = boto3.client(
        "s3", endpoint_url=endpoint, region_name=os.environ.get("S3_REGION", "ru-1"),
        aws_access_key_id=env["S3_ACCESS_KEY"], aws_secret_access_key=env["S3_SECRET_KEY"],
        config=Config(s3={"addressing_style": "path"}, signature_version="s3v4"))
    for name in names:
        with open(os.path.join(outdir, name), "rb") as f:
            client.put_object(
                Bucket=env["S3_BUCKET"], Key=f"{S3_PREFIX}/{slug}/{name}", Body=f, ACL="public-read",
                ContentType=CONTENT_TYPES[os.path.splitext(name)[1]], CacheControl="public, max-age=31536000, immutable")
    print(f"{slug}: в S3 залито файлов: {len(names)}")
    return f"{endpoint}/{env['S3_BUCKET']}/{S3_PREFIX}/{slug}/"


def stamp(outdir: str, name: str) -> str:
    """Имя с хэшем содержимого (f1.3fa9c2d1.webp): заменённый файл получает новый адрес, и браузер с CDN не отдают старый."""
    path = os.path.join(outdir, name)
    stem, ext = os.path.splitext(name)
    new = f"{stem}.{hashlib.sha1(open(path, 'rb').read()).hexdigest()[:8]}{ext}"
    shutil.copyfile(path, os.path.join(outdir, new))
    return new


def prepare_frames(slug: str, scenes: list[dict], outdir: str) -> tuple[list[dict], int]:
    frames = os.path.join(CACHE, slug, "frames")
    out, fake = [], 0
    for sc in scenes:
        dst = os.path.join(outdir, f"f{sc['id']}.webp")
        real = next((p for ext in ("webp", "png", "jpg", "jpeg")
                     for p in [os.path.join(frames, f"s{sc['id']}.{ext}")] if os.path.exists(p)), None)
        if real:
            to_webp(real, dst)
        else:
            tmp = dst + ".png"
            placeholder(tmp, sc["id"])
            to_webp(tmp, dst)
            os.remove(tmp)
            fake += 1
        out.append({"id": sc["id"], "image": f"f{sc['id']}.webp"})
    return out, fake


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("slug")
    mode = ap.add_mutually_exclusive_group(required=True)
    mode.add_argument("--dev", action="store_true", help="файлы в data/story-media, документ на локальный сервер")
    mode.add_argument("--prod", action="store_true", help="файлы в S3, документ на STORIES_SERVER")
    ap.add_argument("--status", default="published", choices=["draft", "published"])
    args = ap.parse_args()
    if args.prod:
        need_env("STORIES_SERVER", "STORIES_SERVICE_TOKEN")

    src = os.path.join(CACHE, args.slug)
    mf = json.load(open(os.path.join(src, "manifest.json"), encoding="utf-8"))
    outdir = os.path.join(src, "publish") if args.prod else os.path.join(MEDIA_DEV, args.slug)
    os.makedirs(outdir, exist_ok=True)
    base = f"/story-media/{args.slug}/"

    scenes, fake = prepare_frames(args.slug, mf["scenes"], outdir)
    scenes = [{**x, "image": stamp(outdir, x["image"])} for x in scenes]
    for s in mf["sentences"]:
        for key in ("sr_audio", "ru_audio"):
            shutil.copyfile(os.path.join(src, s[key]), os.path.join(outdir, s[key]))
    sentences = []
    for s in mf["sentences"]:
        s = dict(s)
        s["sr_audio"], s["ru_audio"] = base + stamp(outdir, s["sr_audio"]), base + stamp(outdir, s["ru_audio"])
        sentences.append(s)
    if args.prod:
        used = [x["image"] for x in scenes] + [n for s in sentences for n in (s["sr_audio"], s["ru_audio"])]
        base = upload_to_s3(args.slug, outdir, sorted({os.path.basename(n) for n in used}))
        for s in sentences:
            s["sr_audio"], s["ru_audio"] = base + os.path.basename(s["sr_audio"]), base + os.path.basename(s["ru_audio"])
    content = {
        "fast_rate": mf["fast_rate"],
        "slow_rate": mf.get("slow_rate", 1),
        "approx_minutes": mf.get("approx_minutes", 0),
        "price": mf.get("price", 0),
        "recommended_lesson": mf.get("recommended_lesson", ""),
        "difficulty": mf.get("difficulty", 0),
        "chapters": [{"scenes": [{"id": x["id"], "image": base + x["image"]} for x in scenes], "sentences": sentences}],
        "quiz_inline": mf["quiz_inline"],
        "quiz_final": mf["quiz_final"],
    }
    doc = {
        "title_sr": mf["title_sr"], "title_ru": mf["title_ru"], "level": str(mf["level"]), "tier": mf["tier"],
        "is_demo": mf["is_demo"], "status": args.status, "sort_order": 1,
        "cover_url": base + scenes[0]["image"], "checks": "check_story: OK", "content": content,
    }
    token = os.environ.get("STORIES_SERVICE_TOKEN", "dev-stories")
    server = os.environ.get("STORIES_SERVER", "http://localhost:8080")
    req = urllib.request.Request(
        f"{server}/api/service/stories/{args.slug}", data=json.dumps(doc, ensure_ascii=False).encode(),
        method="PUT", headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=20) as r:
            print(f"{args.slug}: {r.status} {r.read().decode()}")
    except urllib.error.HTTPError as e:
        sys.exit(f"{args.slug}: сервер ответил {e.code}: {e.read().decode()}")
    except urllib.error.URLError as e:
        sys.exit(f"{args.slug}: сервер {server} недоступен ({e.reason}). Запущен ли make dev?")
    print(f"файлы: {outdir} (кадров-заглушек: {fake} из {len(scenes)})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
