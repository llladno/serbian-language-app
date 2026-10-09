#!/usr/bin/env python3
"""Generate the pixel-art sparkle set (4 sizes x 4 warm colours).

Draws each sparkle at native resolution, saves it at 1x and 8x (nearest
neighbour, no smoothing), plus 1x/8x sprite sheets. Run from anywhere:

    python3 docs/seed-effects/gen_sparkles.py
"""
import math
import os

from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "sparkles")

# name -> half-length k of the arms; the sprite is (2k+1) x (2k+1) pixels.
SIZES = {"s": 2, "m": 3, "l": 4, "xl": 6}

# name -> (arm colour, bright core colour). All from the seed palette.
COLORS = {
    "pale": ("#F6D88A", "#FFF3D1"),
    "gold": ("#E3B04B", "#F6D88A"),
    "amber": ("#C98F2E", "#E3B04B"),
    "cream": ("#FFF3D1", "#FFFFFF"),
}

SCALE = 8


def rgb(h):
    h = h.lstrip("#")
    return tuple(int(h[i : i + 2], 16) for i in (0, 2, 4)) + (255,)


def sparkle(k, arm, core):
    """4-point star: concave astroid-like shape, brighter plus-shaped core."""
    n = 2 * k + 1
    img = Image.new("RGBA", (n, n), (0, 0, 0, 0))
    px = img.load()
    # Astroid sqrt(dx)+sqrt(dy) <= sqrt(k): concave, so arms taper to a point.
    limit = math.sqrt(k) + (0.45 if k == 4 else 0.3)
    for y in range(n):
        for x in range(n):
            dx, dy = abs(x - k), abs(y - k)
            if math.sqrt(dx) + math.sqrt(dy) <= limit:
                bright = (dx + dy == 0) if k <= 2 else (dx + dy <= 1)
                if k >= 6:
                    bright = dx + dy <= 2 and (dx <= 1 and dy <= 1 or dx + dy <= 1)
                px[x, y] = rgb(core if bright else arm)
    return img


def up(img, s):
    return img.resize((img.width * s, img.height * s), Image.NEAREST)


def main():
    os.makedirs(OUT, exist_ok=True)
    sprites = {}
    for cname, (arm, core) in COLORS.items():
        for sname, k in SIZES.items():
            im = sparkle(k, arm, core)
            sprites[(cname, sname)] = im
            im.save(os.path.join(OUT, f"{sname}-{cname}.png"))
            up(im, SCALE).save(os.path.join(OUT, f"{sname}-{cname}@{SCALE}x.png"))

    # Sprite sheet: rows = colours, columns = sizes, 15px cells (fits XL=13).
    cell = 15
    sheet = Image.new("RGBA", (cell * len(SIZES), cell * len(COLORS)), (0, 0, 0, 0))
    for r, cname in enumerate(COLORS):
        for c, sname in enumerate(SIZES):
            im = sprites[(cname, sname)]
            sheet.paste(
                im,
                (c * cell + (cell - im.width) // 2, r * cell + (cell - im.height) // 2),
            )
    sheet.save(os.path.join(OUT, "sheet.png"))
    up(sheet, SCALE).save(os.path.join(OUT, f"sheet@{SCALE}x.png"))
    print(f"wrote {len(sprites) * 2 + 2} files to {OUT}")


if __name__ == "__main__":
    main()
