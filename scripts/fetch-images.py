#!/usr/bin/env python3
"""Fetch curated photos from Wikimedia Commons into content/images/.

Usage: python3 scripts/fetch-images.py
Downloads a ~500px thumbnail per entry in CURATED and writes
content/images/ATTRIBUTIONS.md. Re-run any time; existing files are skipped.
"""
import json
import os
import re
import sys
import urllib.parse
import urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
IMG_DIR = os.path.join(ROOT, "content", "images")
API = "https://commons.wikimedia.org/w/api.php"

# id (matches vocab/false-friend id)  ->  Commons search query
CURATED = {
    "pas": "dog animal",
    "crni-luk": "onion bulb",
    "beli-luk": "garlic bulb",
    "jagoda": "strawberry fruit",
    "dinja": "cantaloupe melon",
    "lubenica": "watermelon",
    "pecurke": "edible mushrooms basket",
    "paradajz": "tomato fruit red",
    "sargarepa": "carrot vegetable",
    "krofna": "doughnut sugar",
    "kifla": "kifli pastry",
    "pasulj": "cooked beans dish",
    "kesa": "plastic shopping bag",
    "racun": "cash register receipt",
    "stolica": "wooden chair furniture",
    "kuca": "residential house exterior",
    "sat": "wristwatch",
    "kupatilo": "bathroom interior",
    "pozoriste": "theatre auditorium stage",
    "banja": "thermal spa bath",
}


def api_get(params):
    q = urllib.parse.urlencode({**params, "format": "json", "action": "query"})
    req = urllib.request.Request(API + "?" + q, headers={"User-Agent": "srpski-app/1.0 (learning app)"})
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.load(r)


def strip_html(s):
    return re.sub(r"<[^>]+>", "", s or "").strip()


def fetch(cid, query):
    dest = os.path.join(IMG_DIR, cid + ".jpg")
    if os.path.exists(dest):
        return None
    d = api_get({
        "prop": "imageinfo",
        "generator": "search",
        "gsrsearch": f"filetype:bitmap {query}",
        "gsrnamespace": "6",
        "gsrlimit": "1",
        "iiprop": "url|extmetadata|mime",
        "iiurlwidth": "560",
    })
    pages = d.get("query", {}).get("pages", {})
    if not pages:
        print(f"  ! no result for {cid} ({query})")
        return None
    ii = list(pages.values())[0]["imageinfo"][0]
    if ii.get("mime") not in ("image/jpeg", "image/png"):
        print(f"  ! skip {cid}: mime {ii.get('mime')}")
        return None
    url = ii.get("thumburl")
    req = urllib.request.Request(url, headers={"User-Agent": "srpski-app/1.0 (learning app)"})
    with urllib.request.urlopen(req, timeout=60) as r:
        data = r.read()
    with open(dest, "wb") as f:
        f.write(data)
    md = ii.get("extmetadata", {})
    return {
        "id": cid,
        "file": ii.get("descriptionshorturl") or ii.get("descriptionurl"),
        "license": strip_html(md.get("LicenseShortName", {}).get("value")),
        "artist": strip_html(md.get("Artist", {}).get("value"))[:120],
        "bytes": len(data),
    }


def main():
    os.makedirs(IMG_DIR, exist_ok=True)
    rows = []
    for cid, q in CURATED.items():
        print(f"- {cid}")
        try:
            info = fetch(cid, q)
        except Exception as e:  # noqa: BLE001
            print(f"  ! {cid}: {e}")
            continue
        if info:
            rows.append(info)
            print(f"  ok {info['bytes']//1024} KB  {info['license']}")

    if rows:
        att = os.path.join(IMG_DIR, "ATTRIBUTIONS.md")
        existing = ""
        if os.path.exists(att):
            existing = open(att).read()
        with open(att, "a" if existing else "w") as f:
            if not existing:
                f.write("# Атрибуция изображений\n\nИсточник — Wikimedia Commons. Лицензии ниже.\n\n")
            for r in rows:
                f.write(f"- **{r['id']}.jpg** — {r['license']} — {r['artist']} — {r['file']}\n")
    print(f"\ndone: {len(rows)} new image(s)")


if __name__ == "__main__":
    sys.exit(main())
