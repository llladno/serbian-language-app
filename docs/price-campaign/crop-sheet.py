#!/usr/bin/env python3
"""Cut a generated sheet of stickers into separate transparent pictures.

The sheet is what the image generator returns for the prompts in docs/price-campaign-image-prompts.md: ten or so
objects, each with a thick white die-cut border, on a flat near-black plate, with wide empty margins between them.
The plate colour is read from the corners. A picture is whatever is not plate; pictures are told apart by the gaps,
numbered in reading order (rows from the top, left to right inside a row), and cut out with a soft edge. Dark parts
inside a picture (a clock hand, the hole of a ring) stay: only what is outside the sticker border becomes transparent.

    python3 docs/price-campaign/crop-sheet.py docs/price-campaign/raw/sheet-symbols.jpg symbols

Writes docs/price-campaign/cut/<name>-NN.webp (lossless, RGBA) and docs/price-campaign/cut/<name>-preview.png (the numbered
pictures on a dark and on a light plate, to choose from). Copy the chosen ones into web/src/assets/promo/.
"""
import os
import sys

import numpy as np
from PIL import Image, ImageDraw
from scipy import ndimage

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "cut")

# Distance from the plate colour (RGB units) at which a pixel is certainly part of a picture, and the width of the soft edge.
T = 70
SOFT = 70
# Gaps narrower than this many pixels do not separate two pictures (a sticker shadow, the dots of a percent sign).
JOIN = 2
# Smaller specks are noise, not pictures.
MIN_AREA = 600
PAD = 6


def main(src: str, name: str) -> None:
    img = Image.open(src).convert("RGB")
    a = np.asarray(img).astype(np.int16)
    h, w, _ = a.shape
    corners = np.array([a[0, 0], a[0, w - 1], a[h - 1, 0], a[h - 1, w - 1]])
    plate = corners.mean(axis=0)
    dist = np.sqrt(((a - plate) ** 2).sum(axis=2))

    mask = dist > T
    joined = ndimage.binary_dilation(mask, iterations=JOIN)
    labels, n = ndimage.label(joined)
    boxes = []
    for i, sl in enumerate(ndimage.find_objects(labels), start=1):
        area = int((labels[sl] == i).sum())
        if area < MIN_AREA:
            continue
        boxes.append((sl, i))

    # A sticker whose dark outline is as dark as the plate falls into two pieces, the border and the shape inside it: the inner
    # piece lies within the box of the other and is not a picture of its own.
    def inside(a, b):
        return a[0].start >= b[0].start and a[0].stop <= b[0].stop and a[1].start >= b[1].start and a[1].stop <= b[1].stop

    boxes = [b for b in boxes if not any(o is not b and inside(b[0], o[0]) for o in boxes)]

    # reading order: group by rows (centres of similar height), then left to right
    centres = [((sl[0].start + sl[0].stop) / 2, (sl[1].start + sl[1].stop) / 2) for sl, _ in boxes]
    order = sorted(range(len(boxes)), key=lambda k: centres[k][0])
    rows, row = [], [order[0]]
    for k in order[1:]:
        if centres[k][0] - centres[row[0]][0] > h * 0.12:
            rows.append(row)
            row = [k]
        else:
            row.append(k)
    rows.append(row)
    ordered = [k for r in rows for k in sorted(r, key=lambda k: centres[k][1])]

    os.makedirs(OUT, exist_ok=True)
    pics = []
    for num, k in enumerate(ordered, start=1):
        sl, label = boxes[k]
        y0, y1 = max(0, sl[0].start - PAD), min(h, sl[0].stop + PAD)
        x0, x1 = max(0, sl[1].start - PAD), min(w, sl[1].stop + PAD)
        own = labels[y0:y1, x0:x1] == label
        # the picture itself: what is not plate, inside this picture's own area, with its holes filled
        d = dist[y0:y1, x0:x1]
        solid = ndimage.binary_fill_holes((d > T) & own)
        # The border of a sticker is thick and white, so the edge can be pulled in a little: that drops the blend with the dark
        # plate (a dark fringe on a light page) and a soft edge is put back by blurring the mask.
        alpha = ndimage.gaussian_filter(ndimage.binary_erosion(solid, iterations=2).astype(float), 0.9) * 255
        rgba = np.dstack([np.asarray(img)[y0:y1, x0:x1], alpha.astype(np.uint8)])
        pic = Image.fromarray(rgba)
        pic.save(os.path.join(OUT, f"{name}-{num:02d}.webp"), lossless=True)
        pics.append(pic)

    # a numbered preview on a dark and a light plate
    cell = 220
    cols = 6
    rows_n = (len(pics) + cols - 1) // cols
    sheet = Image.new("RGB", (cols * cell, rows_n * cell * 2), "#0B1211")
    d = ImageDraw.Draw(sheet)
    for i, pic in enumerate(pics):
        x, y = (i % cols) * cell, (i // cols) * cell * 2
        d.rectangle([x, y + cell, x + cell, y + 2 * cell], fill="#F7F9F7")
        thumb = pic.copy()
        thumb.thumbnail((cell - 30, cell - 30))
        ox, oy = x + (cell - thumb.width) // 2, y + (cell - thumb.height) // 2
        sheet.paste(thumb, (ox, oy), thumb)
        sheet.paste(thumb, (ox, oy + cell), thumb)
        d.text((x + 8, y + 6), str(i + 1), fill="#FFFFFF")
    sheet.save(os.path.join(OUT, f"{name}-preview.png"))
    print(f"{len(pics)} pictures -> {OUT}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    main(sys.argv[1], sys.argv[2])
