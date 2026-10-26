# Баннер «Уровень» в профиле — промты для двух иконок

Дата: 2026-10-26. В баннере три строки с маленькими иконками слева (`web/src/components/TariffsPromo.vue`):

| Строка | Иконка сейчас |
|---|---|
| Навсегда, без подписки | линейная иконка «слои» (lucide), **нужна картинка** |
| Повторения без лимита | линейная иконка «повтор» (lucide), **нужна картинка** |
| Зёрнышки — на другие уровни | настоящее зёрнышко `assets/currency/seed.webp` на золотой плашке, готово |

Рядом с «Hvala lepo» стоит пиксельное сердечко из карточки «Поддержать проект» (`assets/donate/heart.png`).
Остаётся нарисовать две иконки в том же пиксельном стиле, чтобы три строки выглядели как одно семейство.

## Куда они попадут

Иконка показывается в кружке 1.5rem (24 px), сама примерно 14–18 px, с `image-rendering: pixelated`.
Поэтому: **крупные блоки пикселей, простая форма, никаких мелких деталей**. Если форма не читается
в 16 px, она не годится. Золото в баннере занято (зёрнышки и кнопка), поэтому новые иконки **бирюзовые**,
цвета ласточки с картинки баннера.

## Палитра (снята с крыла ласточки на `tariffs-promo-scene.webp`)

| Роль | Hex |
|---|---|
| контур, самый тёмный | `#082E37` |
| тёмная тень | `#0A424D` |
| основной тон | `#095E67` |
| светлее основного | `#2C9890` |
| блик | `#5AC4AA` |

Фон картинки — чистый белый `#FFFFFF`, потом вырезается.

## Общий хвост стиля (вшит в оба промта)

> Crisp pixel art on a coarse square grid, big clearly visible square pixels (about 16x16
> blocks for the whole icon), flat cel-shading in 3 tones per surface, 1-block dark outline
> (#082E37), no smooth gradients, no anti-aliasing blur, no photorealism. Perfectly flat
> plain pure-white background, nothing else in the frame. No shadow, no sparkles, no glow,
> no text, no characters, no coin. Square 1:1 composition, the icon centered with generous
> margin. Same chunky pixel style and block size as the reference image.

**Референс:** приложи `web/src/assets/donate/heart.png` (или `seed.webp`) и пиши «same chunky pixel
style as the reference». Без референса модель нарисует мелкий пиксель-арт, который в 16 px развалится.

## 1. «Навсегда, без подписки»: знак бесконечности

> Pixel art icon of an infinity symbol (lemniscate, the sideways figure eight): one thick
> continuous band forming two equal round loops joined with a clean crossing in the middle,
> built from chunky square blocks, easy to read at 16 pixels. Flat teal body #095E67, a
> lighter teal #2C9890 strip along the upper-left side of each loop, a small mint #5AC4AA
> highlight on the upper-left of the left loop, darker teal #0A424D shading on the lower-right,
> dark outline #082E37. [общий хвост стиля]

**Запасная идея, если ∞ покажется слишком «математическим»:** открытый замок (дужка поднята, ключевая
скважина кремовая `#F3E6C4`), смысл «уровень открыт навсегда». В промте замени первую фразу на:
«Pixel art icon of an open padlock with the shackle raised, chunky square blocks, readable at 16 pixels»
и оставь палитру и хвост.

## 2. «Повторения без лимита»: две стрелки по кругу

> Pixel art icon of the "repeat" symbol: two chunky curved arrows chasing each other around a
> circle, a ring made of two thick bands, each band ending in a blocky triangular arrowhead,
> one arrowhead at the upper right pointing down, the other at the lower left pointing up, a
> clear gap between each arrow tail and the next arrowhead, easy to read at 16 pixels. Flat
> teal body #095E67, lighter teal #2C9890 along the outer upper-left edge of the ring, mint
> #5AC4AA highlight on the upper-left, darker teal #0A424D shading on the lower-right, dark
> outline #082E37. [общий хвост стиля]

Не добавляй внутрь кольца ∞ или цифры: в 16 px это превратится в кашу.

## Как довести до приложения (сделано 2026-10-26)

Сгенерированные иконки уже в баннере: сырые в `docs/promo-icons/raw/`, спрайты делает
`python3 docs/promo-icons/crop-icons.py`. Ниже исходный план.

1. Сгенерируй по 2–3 варианта каждой иконки, выбери лучшие.
2. Сырые картинки положи в `docs/promo-icons/raw/` (папку создай) и пришли мне. Я вырежу фон, ужму до
   спрайта (как `docs/seed-effects/crop-seeds.py` делает для зёрнышек), сохраню в
   `web/src/assets/promo/forever.webp` и `repeat.webp` и подставлю вместо линейных иконок.
3. Проверю в баннере в светлой и тёмной теме. Если иконка в кружке теряется, кружок можно сделать светлее.
