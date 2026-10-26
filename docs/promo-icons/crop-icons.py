#!/usr/bin/env python3
"""Turn the raw pixel-art icon renders of the profile banner into the small WebPs the app ships.

Same idea as docs/seed-effects/crop-seeds.py: the generator returns 2000x2000 JPGs on a
near-white plate, pixel art in look and, here, on a real grid (every block is about 38 source
pixels). So the sprite is made one pixel per block: cut the plate, trim to the icon, scale it
down by the block size with an area average (which keeps flat fills flat), then re-flatten the
colours and harden the alpha edge. The UI draws the sprite back up with `image-rendering:
pixelated`. Saved as lossless WebP.

    python3 docs/promo-icons/crop-icons.py

Reads docs/promo-icons/raw/*.jpg, writes web/src/assets/promo/*.webp.
"""
import os

from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
RAW = os.path.join(HERE, "raw")
OUT = os.path.abspath(os.path.join(HERE, "..", "..", "web", "src", "assets", "promo"))

# Source pixels per block of the pixel art, measured on the staircase edges.
BLOCK = 38.5
# Colours left after flattening: outline, three teal tones, the highlight and a few between.
COLORS = 12
# Distance from the plate colour (RGB units) at which a pixel is still background.
TOLERANCE = 28

# forever.jpg: the infinity sign (line "Навсегда, без подписки")
# repeat.jpg:  two arrows round a ring (line "Повторения без лимита")
NAMES = ["forever", "repeat"]


def strip_background(img: Image.Image) -> Image.Image:
    img = img.convert("RGBA")
    w, h = img.size
    corners = [img.getpixel(p) for p in ((0, 0), (w - 1, 0), (0, h - 1), (w - 1, h - 1))]
    bg = tuple(sum(c[i] for c in corners) // 4 for i in range(3))
    px = img.load()
    for y in range(h):
        for x in range(w):
            r, g, b, _ = px[x, y]
            dist = ((r - bg[0]) ** 2 + (g - bg[1]) ** 2 + (b - bg[2]) ** 2) ** 0.5
            if dist < TOLERANCE:
                px[x, y] = (r, g, b, 0)
            elif dist < TOLERANCE * 2:
                px[x, y] = (r, g, b, int(255 * (dist - TOLERANCE) / TOLERANCE))
    return img


def flatten(img: Image.Image) -> Image.Image:
    alpha = img.getchannel("A").point(lambda a: 255 if a >= 128 else 0)
    rgb = img.convert("RGB").quantize(colors=COLORS, method=Image.Quantize.MEDIANCUT)
    out = rgb.convert("RGB").convert("RGBA")
    out.putalpha(alpha)
    return out


def main() -> None:
    os.makedirs(OUT, exist_ok=True)
    for name in NAMES:
        img = strip_background(Image.open(os.path.join(RAW, name + ".jpg")))
        bbox = img.getchannel("A").getbbox()
        if bbox:
            img = img.crop(bbox)
        size = (max(1, round(img.width / BLOCK)), max(1, round(img.height / BLOCK)))
        img = flatten(img.resize(size, Image.BOX))
        dst = os.path.join(OUT, name + ".webp")
        img.save(dst, "WEBP", lossless=True, quality=100, method=6)
        print(f"{name}: {img.size[0]}x{img.size[1]} -> {os.path.relpath(dst)}")


if __name__ == "__main__":
    main()
