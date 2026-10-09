# Зёрнышки — промты для генерации изображений валюты

Четыре картинки в одном стиле: иконка одного зёрнышка и три ступени для
баланса в кошельке: **несколько зёрнышек → горсть → мешок**. Стиль — как у
маскота UCIMO (пиксель-арт, flat cel-shading, палитра бренда), см.
`ucimo-content-admin/.claude/skills/ucimo-content/references/mascot-prompt.md`.
Эталон формы — семечка подсолнуха с полосками: золотистая с тёмно-серыми
полосами, тёмно-коричневый пиксельный контур.

Файлы в приложении (`web/src/assets/currency/`), по балансу: `seed.webp` — одно
зёрнышко (0), `seeds-few.webp` — несколько (до 100), `seeds-handful.webp` —
горсть (до 1000), `seeds-sack.webp` — мешок (от 1000). Выбор делает `pileFor` в
`web/src/lib/seeds.ts`. Сырые картинки лежат в `docs/seed-effects/raw/`.

Маскота на картинках **нет**. Теней, искорок, текста и фона-сцены нет.
Тёмный пиксельный контур — есть (он держит форму в мелком размере).

## Как генерировать

1. Сначала **иконка одного зёрнышка** (промт 1) — приложи свою картинку как
   референс и прогони до результата, который нравится. Это эталон формы и цвета.
2. Промты 2–4 генерировать **с иконкой как референсом** («same seed design as
   the reference image»), иначе модель каждый раз нарисует другое зёрнышко.
3. Формат 1:1, фон ровный белый — потом вырезается скриптом (ниже).

## Палитра зёрнышка

Снята с твоей картинки.

| Роль | Hex |
|---|---|
| золотистая заливка | `#F0C870` |
| блик | `#F8E0A0` |
| тёмные полоски | `#584838` |
| тень на полосках | `#483820` |
| контур | `#382008` (тёмно-коричневый) |

Для мешка дополнительно (ткань, нужна, чтобы мешок отличался от золота зёрен):

| Роль | Hex |
|---|---|
| мешковина, основной тон | `#C9A56A` |
| тень на мешковине | `#A67F45` |
| блик на мешковине | `#E0C48E` |
| контур мешка | `#382008` |

Фон — чистый белый `#FFFFFF`.

## Общий хвост стиля (вшит в каждый промт)

> Crisp pixel art with clearly visible square pixels, flat cel-shading in 2–3 tones
> per surface, 1-pixel dark brown outline, no smooth vector gradients, no
> anti-aliasing blur, no photorealism. Perfectly flat plain pure-white background,
> nothing else in the frame. No shadow, no sparkles, no glow, no text, no coin,
> no characters. Square 1:1 composition, subject centered with generous margin.

---

## 1. Одно зёрнышко (иконка валюты)

Основной глиф: рядом с балансом, в магазине, в наградах за квесты. Должен
читаться в чипе ~20px, поэтому форма крупная и простая. Если картинка уже есть,
этот шаг можно пропустить и использовать её как эталон.

```
A single striped sunflower seed, a game-currency icon in the style of the UCIMO
teal pixel-art swallow mascot. Classic teardrop seed silhouette standing upright
and tilted a few degrees: pointed tip at the top, rounded wider base at the
bottom. Warm golden-yellow body (#F0C870) with three or four curved dark
grey-brown stripes (#584838) running from the tip to the base, following the
curve of the seed; a light highlight (#F8E0A0) on the upper-left side of the
golden parts, a darker shading tone (#483820) on the lower-right side of the
dark stripes. Dark brown outline (#382008), one pixel thick. The silhouette must
stay recognizable as a sunflower seed when scaled down to 20 pixels, so keep the
stripes bold and few.
Crisp pixel art with clearly visible square pixels, flat cel-shading in 2-3 tones
per surface, no smooth vector gradients, no anti-aliasing blur, no photorealism.
Perfectly flat plain pure-white background, nothing else in the frame. No shadow,
no sparkles, no glow, no text, no coin, no characters. Square 1:1 composition,
subject centered with generous margin.
```

---

## 2. Ступень 1: несколько зёрнышек (3–5)

Баланс — единицы или первые десятки.

```
A small loose scatter of four striped sunflower seeds lying close together at
different angles, a game-currency icon in the style of the UCIMO teal pixel-art
swallow mascot. Same seed design as the reference image: teardrop shape, golden
body (#F0C870), bold curved dark grey-brown stripes (#584838), light highlight
(#F8E0A0), darker shading tone (#483820), dark brown one-pixel outline (#382008).
Two seeds lie side by side, one rests partly over them, one lies a little apart;
tips pointing in different directions. Depth comes only from overlap: seeds
underneath are a touch darker than the ones on top. Every seed stays
individually readable.
Crisp pixel art with clearly visible square pixels, flat cel-shading in 2-3 tones
per surface, no smooth vector gradients, no anti-aliasing blur, no photorealism.
Perfectly flat plain pure-white background, nothing else in the frame. No shadow,
no sparkles, no glow, no text, no coin, no characters. Square 1:1 composition,
subject centered with generous margin.
```

---

## 3. Ступень 2: горсть (~12–15)

Баланс — сотни.

```
A modest heap of about fourteen striped sunflower seeds piled together in a
small rounded mound, a game-currency icon in the style of the UCIMO teal
pixel-art swallow mascot. Same seed design as the reference image: teardrop
shape, golden body (#F0C870), bold curved dark grey-brown stripes (#584838),
light highlight (#F8E0A0), darker shading tone (#483820), dark brown one-pixel
outline (#382008). The seeds lie criss-crossed at different angles, tips
pointing in many directions, a few seeds slid down at the base of the mound.
Depth comes only from overlap: lower seeds are slightly darker, the top ones
brightest. The overall silhouette is a soft low mound, still clearly made of
separate seeds with visible stripes rather than a yellow-brown blob.
Crisp pixel art with clearly visible square pixels, flat cel-shading in 2-3 tones
per surface, no smooth vector gradients, no anti-aliasing blur, no photorealism.
Perfectly flat plain pure-white background, nothing else in the frame. No shadow,
no sparkles, no glow, no text, no coin, no characters. Square 1:1 composition,
subject centered with generous margin.
```

---

## 4. Ступень 3: мешок зёрнышек

Баланс — тысячи, «богатство». Это мешок, а не куча: он сразу читается как «очень
много» и хорошо смотрится в малом размере.

```
A plump burlap sack full of striped sunflower seeds, standing upright, a
game-currency icon in the style of the UCIMO teal pixel-art swallow mascot. The
sack is rounded and bulging, made of coarse tan burlap (#C9A56A) with a darker
shading tone (#A67F45) on the lower-right side and a light highlight (#E0C48E)
on the upper-left, a few simple pixel stitch marks along one seam. The mouth of
the sack is rolled open at the top and brimming with seeds that heap up above
the rim; a handful of seeds spill over the edge and a few lie on the ground at
the foot of the sack. Same seed design as the reference image: teardrop shape,
golden body (#F0C870), bold curved dark grey-brown stripes (#584838), light
highlight (#F8E0A0), dark brown one-pixel outline (#382008) around both the sack
and each seed. The seeds on top and on the ground stay individually readable.
The sack has no label, no writing and no rope.
Crisp pixel art with clearly visible square pixels, flat cel-shading in 2-3 tones
per surface, no smooth vector gradients, no anti-aliasing blur, no photorealism.
Perfectly flat plain pure-white background, nothing else in the frame. No shadow,
no sparkles, no glow, no text, no coin, no characters. Square 1:1 composition,
subject centered with generous margin.
```

---

---

## 5. «Пригласи друга» (картинка в окне приглашения)

Не валюта, а иллюстрация для окна «Позови друга»: два маскота UCIMO на ветке,
один протягивает другому зёрнышко. Это целая сцена с фоном (не вырезанный
силуэт): в окне она стоит сверху квадратом со скруглёнными углами. Генерировать
**с референсом маскота** и иконки зёрнышка, иначе модель нарисует другую птицу.

Первая попытка дала лишнее (третье) крыло и белый фон, поэтому в промте крылья
пересчитаны явно, а фон описан.

```
Two UCIMO mascots (the established teal pixel-art swallow character, same style
as its other poses) perched side by side on one thick tree branch, turned toward
each other, in a friendly moment. Each bird has exactly two wings, no more: the
wings are folded neatly against the body at rest. Only ONE wing of the left
swallow is lifted and extended forward like a hand, holding a single striped
sunflower seed out to the right swallow; its other wing stays folded at its side.
The right swallow keeps both wings folded, with a small delighted calm smile,
and tilts its head toward the seed. No spread wings, no flapping, no extra wings
or limbs, no wing behind the body. Same seed design as the reference image:
teardrop shape, golden body (#F0C870), bold curved dark grey-brown stripes
(#584838), light highlight (#F8E0A0), dark brown outline (#382008); the seed is
the brightest spot in the picture. Deep Teal bodies (#176B68) with darker teal
outline (#0B3F3D), lighter teal highlights (#8CCBC6), triangular gold chest patch
(#E3B04B), cream belly (#F7F9F7), small gold beak, simple black pixel eyes with a
white highlight dot. Two separate birds, the right one a touch smaller, slightly
different head tilt.
Background: a calm warm evening over an old Balkan town, drawn in flat pixel
bands: a soft sky going from cream (#F7F9F7) through pale peach to muted teal-grey,
a few simple pixel clouds, far below the branch a row of terracotta roofs with a
church tower and a few warm lit windows. The branch has a few leaves in muted
green. Keep the background soft and low-contrast so the two birds and the seed
stay the clearest shapes.
Crisp pixel art with clearly visible square pixels, flat cel-shading in 2-3 tones
per surface, 1-pixel dark outline on the birds and the seed, no smooth vector
gradients, no anti-aliasing blur, no photorealism. No text, no letters, no
hearts, no sparkles, no frame or border, no logo. Square 1:1 composition, birds
centered in the middle third, generous margin around them.
```

Если снова вырастут лишние крылья, допиши в конец:
`Count the wings: each bird has exactly two, one on each side of its body.`

Готовую картинку положи в `docs/seed-effects/raw/invite.jpg`: я уменьшу её и
поставлю в окно приглашения (`web/src/assets/invite-scene.webp`).

## Если модель ошибается

| Проблема | Что дописать в промт |
|---|---|
| рисует орех, кофейное зерно или фасоль | `It is a striped sunflower seed with a pointed tip, not a bean, nut or coffee grain.` |
| семечка без полосок или полоски расплылись | `Bold readable dark stripes, each stripe clearly separated from the golden parts.` |
| зёрнышко слишком реалистичное | `Stylized chunky game-icon seed, simplified shapes, bold readable outline.` |
| сглаженные края, «размытый» пиксель | `Hard pixel edges only, every pixel a solid flat colour, no soft transitions.` |
| появилась тень или свечение | `Absolutely no cast shadow, no drop shadow, no halo, no glow behind the object.` |
| горсть слипается в коричневое пятно | `Each seed clearly separated by its own dark outline where they overlap.` |
| на мешке появилась надпись или значок | `The sack is completely plain: no label, no letters, no logo, no emblem.` |
| мешок похож на мешок с деньгами (знак $) | `It is a sack of seeds, not money: no dollar sign, no coins.` |
| мешок слишком мелкий или не читается в 20 px | `Large simple sack silhouette, chunky shapes, few details, readable at 20 pixels.` |
| фон не белый | `Background is solid pure white #FFFFFF with no vignette, gradient or texture.` |

## После генерации

Положить сырые JPG (белый фон) в `docs/seed-effects/raw/` под именами `seed.jpg`,
`seeds-few.jpg`, `seeds-handful.jpg`, `seeds-sack.jpg` и запустить:

```bash
python3 docs/seed-effects/crop-seeds.py
```

Скрипт вырезает белый фон, обрезает по предмету, уменьшает до размера спрайта,
схлопывает цвета и сохраняет lossless WebP в `web/src/assets/currency/`.

Проверить на тонких деталях: кончик зёрнышка и тёмные полоски легче всего
«съедаются» при вырезании. Искорки (`sparkles/`) остаются как есть: они
жёлто-золотые и подходят к зёрнышкам.
