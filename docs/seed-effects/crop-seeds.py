#!/usr/bin/env python3
"""Turn the raw sunflower-seed renders into the small pixel-art WebPs the app ships.

The generator returns 1200x896 JPGs on a near-white background: pixel art in
look, but not on a real pixel grid (blocks drift between 8 and 13 source
pixels) and smeared by JPEG noise. So instead of trying to recover the grid, we
cut the background, trim to the subject, scale down to the size the art is
actually worth, and re-flatten the colours. The result is a small sprite that
the UI blows back up with `image-rendering: pixelated`, which is what makes the
edges hard again. Saved as lossless WebP: smaller than PNG here and no ringing.

    python3 docs/seed-effects/crop-seeds.py

Reads docs/seed-effects/raw/*.jpg, writes web/src/assets/currency/*.webp.
"""
import os

from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
RAW = os.path.join(HERE, "raw")
OUT = os.path.abspath(os.path.join(HERE, "..", "..", "web", "src", "assets", "currency"))

# name -> width of the sprite we keep. The single seed is an icon (chip, quest
# reward, modal), the other three only ever appear large, so they keep more
# detail; the sizes also keep them visibly bigger than one another.
#   seed          one seed        (balance 0)
#   seeds-few     four seeds      (under 100)
#   seeds-handful a heap of seeds (under 1000)
#   seeds-sack    a sack of seeds (1000 and up)
TARGETS = {"seed": 28, "seeds-few": 56, "seeds-handful": 72, "seeds-sack": 68}

# Colours left after flattening: gold, its highlight, two stripe tones, the
# outline, plus the burlap tones of the sack and the shading between overlaps.
COLORS = 16

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
        dst = os.path.join(OUT, name + ".webp")
        img.save(dst, "WEBP", lossless=True, quality=100, method=6)
        print(f"{name}: {img.size[0]}x{img.size[1]} -> {os.path.relpath(dst)}")


if __name__ == "__main__":
    main()
