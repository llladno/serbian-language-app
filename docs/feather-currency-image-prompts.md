# Пёрышки — промты для генерации изображений валюты

Четыре картинки в одном стиле: иконка одного пёрышка и горка в три ступени
(для баланса в кошельке: мало → горсть → куча). Стиль — как у маскота UCIMO
(пиксель-арт, flat cel-shading, палитра бренда), см.
`ucimo-content-admin/.claude/skills/ucimo-content/references/mascot-prompt.md`.

Маскота на этих картинках **нет**. Теней, искорок, текста и фона-сцены нет.
Тёмный пиксельный контур — есть (он держит форму в мелком размере).

## Как генерировать

1. Сначала **иконка одного пёрышка** (промт 1) — прогнать до результата, который
   нравится. Это эталон формы и цвета.
2. Промты 2–4 (горки) генерировать **с иконкой как референсом** («same feather
   design as the reference image»), иначе модель каждый раз нарисует другое перо.
3. Формат 1:1, фон ровный белый — потом вырезается скриптом (ниже).

## Палитра пера

| Роль | Hex |
|---|---|
| основная заливка (gold) | `#E3B04B` |
| блик | `#F6D88A` |
| тень на пере | `#C98F2E` |
| контур | `#7A4B12` (тёмно-янтарный) |

Фон — чистый белый `#FFFFFF`. Cream `#F7F9F7` как фон не просим: он близок к
золотому свету и хуже вырезается.

## Общий хвост стиля (уже вшит в каждый промт)

> Crisp pixel art with clearly visible square pixels, flat cel-shading in 2–3 tones
> per surface, 1-pixel dark amber outline, no smooth vector gradients, no
> anti-aliasing blur, no photorealism. Perfectly flat plain pure-white background,
> nothing else in the frame. No shadow, no sparkles, no glow, no text, no coin,
> no characters. Square 1:1 composition, subject centered with generous margin.

---

## 1. Одно пёрышко (иконка валюты)

Основной глиф: рядом с балансом, в магазине, в наградах за квесты. Должен
читаться в чипе ~20px, поэтому форма крупная и простая.

```
A single golden feather, a game-currency icon in the style of the UCIMO teal
pixel-art swallow mascot. Classic feather silhouette tilted about 45 degrees:
the tip points to the upper right, the bare thin quill points to the lower left.
Leaf-shaped vane filled with warm gold (#E3B04B), a thin central shaft, three or
four small visible notches along the edge of the vane, one light highlight stripe
(#F6D88A) along the upper side, a slightly darker shading tone (#C98F2E) on the
lower side. Dark amber outline (#7A4B12), one pixel thick. The silhouette must
stay recognizable as a feather when scaled down to 20 pixels.
Crisp pixel art with clearly visible square pixels, flat cel-shading in 2-3 tones
per surface, no smooth vector gradients, no anti-aliasing blur, no photorealism.
Perfectly flat plain pure-white background, nothing else in the frame. No shadow,
no sparkles, no glow, no text, no coin, no characters. Square 1:1 composition,
subject centered with generous margin.
```

---

## 2. Горка, ступень 1: несколько пёрышек (3–5)

Баланс — единицы или первые десятки.

```
A small loose pile of four golden feathers lying flat and slightly overlapping, a
game-currency icon in the style of the UCIMO teal pixel-art swallow mascot. Same
feather design as the reference image: leaf-shaped gold vane (#E3B04B), thin
central shaft, a few small edge notches, light highlight stripe (#F6D88A), darker
shading tone (#C98F2E), dark amber one-pixel outline (#7A4B12). The feathers are
fanned out at different angles, a couple of quills crossing each other, forming a
compact cluster. Depth comes only from overlap: feathers underneath are a touch
darker than the ones on top. Every feather stays individually readable.
Crisp pixel art with clearly visible square pixels, flat cel-shading in 2-3 tones
per surface, no smooth vector gradients, no anti-aliasing blur, no photorealism.
Perfectly flat plain pure-white background, nothing else in the frame. No shadow,
no sparkles, no glow, no text, no coin, no characters. Square 1:1 composition,
subject centered with generous margin.
```

---

## 3. Горка, ступень 2: горсть (~10)

Баланс — сотни.

```
A modest heap of about ten golden feathers piled together, a game-currency icon in
the style of the UCIMO teal pixel-art swallow mascot. Same feather design as the
reference image: leaf-shaped gold vane (#E3B04B), thin central shaft, a few small
edge notches, light highlight stripe (#F6D88A), darker shading tone (#C98F2E),
dark amber one-pixel outline (#7A4B12). The feathers lie criss-crossed in a
rounded low mound, tips pointing in many directions, some quills sticking out at
the edges. Depth comes only from overlap: lower feathers are slightly darker, the
top ones brightest. The overall silhouette is a soft mound, still clearly made of
separate feathers rather than a gold blob.
Crisp pixel art with clearly visible square pixels, flat cel-shading in 2-3 tones
per surface, no smooth vector gradients, no anti-aliasing blur, no photorealism.
Perfectly flat plain pure-white background, nothing else in the frame. No shadow,
no sparkles, no glow, no text, no coin, no characters. Square 1:1 composition,
subject centered with generous margin.
```

---

## 4. Горка, ступень 3: большая пышная куча

Баланс — тысячи, «богатство».

```
A large, lush mountain of golden feathers, twenty or more, heaped high in a tall
rounded pyramid, a game-currency icon in the style of the UCIMO teal pixel-art
swallow mascot. Same feather design as the reference image: leaf-shaped gold vane
(#E3B04B), thin central shaft, a few small edge notches, light highlight stripe
(#F6D88A), darker shading tone (#C98F2E), dark amber one-pixel outline (#7A4B12).
Feathers overlap in layers, tips and quills pointing outward in all directions so
the edge of the heap looks fluffy and abundant. Depth comes only from overlap:
the lowest layer is darkest, the top layer brightest, with a few of the top
feathers catching light highlights. The heap feels generous and rich but the
outer feathers remain individually readable.
Crisp pixel art with clearly visible square pixels, flat cel-shading in 2-3 tones
per surface, no smooth vector gradients, no anti-aliasing blur, no photorealism.
Perfectly flat plain pure-white background, nothing else in the frame. No shadow,
no sparkles, no glow, no text, no coin, no characters. Square 1:1 composition,
subject centered with generous margin.
```

---

## Если модель ошибается

| Проблема | Что дописать в промт |
|---|---|
| рисует монету/медаль вместо пера | `It is a feather, not a coin or medal; there is no circular disc anywhere.` |
| перо слишком реалистичное/пушистое | `Stylized chunky game-icon feather, simplified shapes, bold readable outline.` |
| сглаженные края, «размытый» пиксель | `Hard pixel edges only, every pixel a solid flat colour, no soft transitions.` |
| появилась тень или свечение | `Absolutely no cast shadow, no drop shadow, no halo, no glow behind the object.` |
| горка слипается в золотое пятно | `Each feather clearly separated by its own dark outline where they overlap.` |
| фон не белый | `Background is solid pure white #FFFFFF with no vignette, gradient or texture.` |

## После генерации

Сохранить сырые картинки (белый фон) и вырезать фон тем же скриптом, что и для
поз маскота:

```bash
python3 ucimo-content-admin/scripts/strip-bg.py in.jpg out.png
```

Проверить результат на тонких деталях: кончик очина и зубчики по краю пера
легче всего «съедаются» при вырезании. Золотое на белом вырезается чище, чем
кремовое брюшко маскота, но тонкий светлый блик стоит проверить отдельно.
