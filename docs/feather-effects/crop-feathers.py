#!/usr/bin/env python3
"""Turn the raw feather renders into the small pixel-art PNGs the app ships.

The generator returns 1200x896 JPGs on a near-white background: pixel art in
look, but not on a real pixel grid (blocks drift between 8 and 13 source
pixels) and smeared by JPEG noise. So instead of trying to recover the grid, we
cut the background, trim to the subject, scale down to the size the art is
actually worth, and re-flatten the colours. The result is a small sprite that
the UI blows back up with `image-rendering: pixelated`, which is what makes the
edges hard again.

    python3 docs/feather-effects/crop-feathers.py

Reads docs/feather-effects/raw/*.jpg, writes web/src/assets/currency/*.png.
"""
import os

from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
RAW = os.path.join(HERE, "raw")
OUT = os.path.abspath(os.path.join(HERE, "..", "..", "web", "src", "assets", "currency"))

# name -> width of the sprite we keep. The single feather is an icon (chip,
# quest reward, modal), the piles only ever appear large, so they keep more
# detail; the sizes also keep the three piles visibly bigger than one another.
TARGETS = {"feather": 40, "pile-1": 72, "pile-2": 88, "pile-3": 104}

# Colours left after flattening. More than the four palette tones because the
# piles shade overlapping feathers against each other.
COLORS = 12

# Distance from the sampled background colour, in RGB units, at which a pixel
# is still background. Generous because JPEG rings the near-white plate.
TOLERANCE = 30


def strip_background(img: Image.Image) -> Image.Image:
    """Make the flat plate behind the subject transparent."""
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
    """Posterize the colours and harden the alpha edge.

    Downscaling averages neighbouring source pixels, which leaves gradients
    where the art had flat fills and a soft halo where it had a hard outline.
    Quantizing the colours and thresholding the alpha puts both back.
    """
    alpha = img.getchannel("A").point(lambda a: 255 if a >= 128 else 0)
    rgb = img.convert("RGB").quantize(colors=COLORS, method=Image.Quantize.MEDIANCUT)
    out = rgb.convert("RGB").convert("RGBA")
    out.putalpha(alpha)
    return out


def main() -> None:
    os.makedirs(OUT, exist_ok=True)
    for name, width in TARGETS.items():
        src = os.path.join(RAW, name + ".jpg")
        img = strip_background(Image.open(src))
        bbox = img.getchannel("A").getbbox()
        if bbox:
            img = img.crop(bbox)
        height = max(1, round(img.height * width / img.width))
        img = flatten(img.resize((width, height), Image.LANCZOS))
        dst = os.path.join(OUT, name + ".png")
        img.save(dst)
        print(f"{name}: {img.size[0]}x{img.size[1]} -> {os.path.relpath(dst)}")


if __name__ == "__main__":
    main()
