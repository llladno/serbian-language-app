# Акция «полцены до 15 ноября»: промты для картинок (ласточка, предметы, символы)

Дата: 2026-11-11. Картинки нужны для окна после урока (`PriceChangeModal`, `PriceChangePoster`), полосы в профиле и маленького баннера на
тарифах. Стиль, который выбрал владелец: **как сами баннеры, современный, весёлый, «education vibe»**: чёрная плашка, крупные белые
стикеры с толстой белой обводкой-вырубкой, жирные округлые формы, лёгкий наклон, плоские цвета. Как у стикеров на референсах
(Skillbox, Яндекс Практикум), только в нашей палитре.

Можно сгенерировать несколько штук отдельно или **10 вариантов на одном холсте** (5 × 2). Ниже три листа: ласточка, предметы,
символы (%, ₽, часы). Я потом разрежу лист скриптом на отдельные картинки.

## Два стиля, и почему так

- **Ласточка остаётся пиксельной** (лист 1): это герой бренда, она уже пиксельная везде в приложении и на баннерах, другую птицу никто не
  узнает. Но она тоже получает белую обводку стикера и сюжет про время и полцены. Гарнитуру с `support-mascot.webp` убираем: она про поддержку.
- **Предметы и символы (листы 2 и 3) в современном плоском стиле стикеров**, не пиксельные: так они выглядят как наши белые стикеры с ценами и
  как стикеры на референсах, а пиксельная птица рядом даёт тот самый «игровой» характер. На плакате это уже так: пиксельная ласточка и плоские
  стикеры в одной композиции.
- Если хочешь **всё в одном стиле**, возьми «Хвост стиля B» и для птицы тоже (в промте листа 1 замени пиксельный хвост на него), но тогда
  ласточка будет не та, что в приложении. Я бы так не делал.

## Палитра (как у баннеров)

| Роль | Hex |
|---|---|
| фон плакатов, почти чёрный | `#0B1211` |
| белый (стикеры, обводка) | `#FFFFFF` |
| фирменный бирюзовый | `#176B68` |
| светлый бирюзовый | `#5CB3AD` |
| мятный фон | `#D8EEEB` |
| тёмный текст | `#172326` |
| тёплый акцент, **очень немного** | `#E3B04B` |

Золото в приложении держим для кнопки и зёрнышек, поэтому на картинках его **только маленькой деталью** (стрелка часов, искра), не заливкой.
Яркий красный, розовый, фиолетовый и радуга не нужны.

## Хвост стиля A: для ласточки (лист 1)

Референс: приложи `web/src/assets/support-mascot.webp` и пиши «same character, same chunky pixel style and block size as the reference, but
without the headset». Палитра птицы: контур `#001818`, бирюзовый `#087880`, блик `#2C9890`, грудка `#F8F0D0`, нагрудник и клюв `#F0A828`.

> Crisp pixel art on a coarse square grid, big clearly visible square pixels, flat cel-shading in 3 tones per surface, 1-block dark outline
> (#001818), no smooth gradients, no anti-aliasing blur. A thick pure-white die-cut sticker border around the whole character. Every picture
> sits on a perfectly flat solid near-black background (#0B1211), nothing else in the frame: no ground, no shadow, no glow, no sparkles, no
> text, no letters, no digits, no logos. Same style and block size as the reference image in every cell.

## Хвост стиля B: для предметов и символов (листы 2 и 3)

> Modern playful education-app sticker illustration, like the stickers on a bold edtech poster: simple chunky rounded shapes, flat solid
> colours with at most one flat darker shade per colour (no gradients, no glossy 3D, no realistic texture, no outlines of other colours than
> the sticker border), every object tilted a little in a different direction, slight confident cartoon charm. A thick pure-white die-cut
> sticker border around every object, and a small hard flat shadow under it. Palette: white, teal #176B68, light teal #5CB3AD, mint #D8EEEB,
> near-black #172326, and a tiny accent of warm yellow #E3B04B. Every picture sits on a perfectly flat solid near-black background
> (#0B1211), nothing else in the frame: no ground, no glow, no sparkles, no text, no letters, no digits, no logos. All objects of one
> sheet share the same drawing style, line weight and corner roundness.

## Лист 1. Ласточка в 10 вариантах (5 × 2), пиксельная

> A sheet of 10 separate pictures arranged in a clean grid of 5 columns and 2 rows, every picture in its own square cell with a very wide
> empty margin around it so the pictures never touch each other or the edge of the sheet. The same cute round pixel-art swallow character
> from the reference (big shiny eyes, teal body, cream belly, orange bib and beak), without the headset, in a different pose and with a
> different object in every cell, whole body visible:
> 1) holding a round pocket stopwatch with a thin golden hand in its feet, looking worried;
> 2) standing next to a tall hourglass with sand running down, one wing raised;
> 3) flying fast to the right with small speed lines, a blank empty price tag on a string in its beak;
> 4) sitting on top of a big round alarm clock (no numbers drawn, just ticks), looking at the viewer;
> 5) holding a sunflower seed cut exactly in half, one half in each wing, cheerful;
> 6) holding up a blank torn-off calendar page with both wings;
> 7) holding a small megaphone to its beak, as if announcing something;
> 8) flying upward with a blank price tag with a down arrow on it hanging from its foot;
> 9) winking, a tiny pair of scissors cutting a price tag in two in its feet;
> 10) sprinting with its wings pulled back like a runner, a tiny stopwatch tied to its leg.
> [хвост стиля A]

**Что смотреть при выборе.** Читается ли сюжет в 100 px. Не сливается ли предмет с телом (бирюзовое на бирюзовом). Нет ли букв и цифр:
текст («−50%», дату, цены) мы печатаем сами. Больше всего нужны номера 1, 2, 5 и 10: время и половина.

## Лист 2. Предметы-стикеры без птицы (5 × 2), современный стиль

Для замены текстового стикера «Скидка 50%», украшения плаката и полосы.

> A sheet of 10 separate sticker objects arranged in a clean grid of 5 columns and 2 rows, every object in its own square cell with a very
> wide empty margin around it so the objects never touch each other or the edge of the sheet, no characters, no people, no animals:
> 1) a round price tag on a string with a big hole, blank, teal, tilted;
> 2) a tall hourglass with light teal sand, the top half almost empty;
> 3) a round stopwatch with a button on top and one thin yellow hand, no numbers, only a few tick marks;
> 4) a sunflower seed cut in half, two halves side by side with a small gap, striped like a real seed, in near-black and cream;
> 5) a torn-off calendar page, blank, with a small torn edge at the top and a bold teal header strip;
> 6) a burst-shaped sale sticker (a starburst with a flat middle), blank, white;
> 7) a pair of scissors with open blades, cutting a thin ribbon;
> 8) an alarm clock with two bells on top and little legs, no numbers;
> 9) a gift tag with a folded corner and a string, blank, mint;
> 10) a half-filled progress bar, a rounded capsule, the left half teal, the right half empty.
> [хвост стиля B]

## Лист 3. Символы и украшения: проценты, рубли, часы (5 × 2), современный стиль

Мелкие знаки для плывущих строк за полосой, соседство с таймером, углы плаката. Показываться будут **маленькими, 16–40 px**, поэтому формы
максимально простые и жирные, и они должны читаться в 24 px. Цифр нет: «15», «50» и цены мы печатаем сами. Процент и рубль просим как
**предмет-знак**, а не как буквы, так ИИ рисует их надёжнее.

> A sheet of 10 separate sticker symbols arranged in a clean grid of 5 columns and 2 rows, every symbol in its own square cell with a very
> wide empty margin around it so the symbols never touch each other or the edge of the sheet, no characters, no text, no digits. Every symbol
> is very simple, chunky and bold, readable at 24 pixels:
> 1) a big percent sign, two round dots joined by a thick diagonal bar, white with teal dots, tilted;
> 2) a big ruble sign (a letter P with a short double crossbar), thick, white with a dark inner detail;
> 3) a round coin seen from the front with a thick rim and a simple ruble sign in the middle, white and light teal;
> 4) a round wall clock with a thick rim and two thick hands pointing almost to the top (time nearly up), tick marks only, no numerals;
> 5) a circular timer ring, thick, about eighty percent of the ring already filled teal, the rest white, with a tiny yellow dot at the end;
> 6) a thick arrow pointing up with a short stem (the price is going up), teal with a white face;
> 7) a thick multiplication sign (an X made of two rounded bars), white;
> 8) a chunky lightning bolt, white with a small yellow edge (hurry);
> 9) a starburst sale sticker with a thick percent sign in the middle, white starburst and a teal percent sign;
> 10) a blank ribbon banner with curled ends and small folds, mint, empty in the middle.
> [хвост стиля B]

**Что смотреть при выборе.** Узнаётся ли знак в 24 px (проверь, уменьшив картинку). Не слиплись ли два кружка у процента. Не превратился ли
рубль в обычную «Р» без перекладины. Для полосы нужны номера 1, 2, 4 и 8, для плаката 9 и 10.

## Что отдавать мне

1. Лист(ы) в максимальном размере. Положи в `docs/price-campaign/raw/` (папка уже есть), имена `sheet-swallow.jpg`, `sheet-objects.jpg`,
   `sheet-symbols.jpg`. Если хочешь штучные картинки, имена любые, один файл на картинку.
2. Скажи номера понравившихся. Я напишу `docs/price-campaign/crop-sheet.py`: резка листа на ячейки по пустым полям. Фон у картинок
   **чёрный `#0B1211`, как у плаката**, поэтому вырезать его не обязательно: белая обводка-вырубка остаётся от рисунка, а на плакате
   картинка лежит на таком же чёрном. Прозрачность сделаю только там, где картинку надо поставить на другой фон (светлая тема страницы,
   белые стикеры). Результат в `web/src/assets/promo/`.
3. Проверю на тёмном плакате и в светлой теме страницы, подберу размер и положение так, чтобы картинки не закрывали дату и цены.

## Если листом не получилось

ИИ часто путает ячейки и рисует разное число картинок. Тогда проси по **5 штук** (5 × 1) или по одной. Промт тот же, меняются только
число и перечень в списке. Если стиль на листе «плывёт» (то глянец, то плоско), добавь в конец: «all ten drawings must look like the same
artist drew them on the same day».

## Сделано (2026-11-11): лист 3 «Символы»

Лист сгенерирован (`docs/price-campaign/raw/sheet-symbols.jpg`), нарезан `python3 docs/price-campaign/crop-sheet.py docs/price-campaign/raw/sheet-symbols.jpg symbols`
в `docs/price-campaign/cut/symbols-NN.webp` (прозрачные, с превью `symbols-preview.png`). Лишние `symbols-12.webp` и `symbols-13.webp` в этой папке остались от
пробного запуска скрипта, их можно удалить. Владелец выбрал **рубль (№3), молнию (№9) и крестик (№8)**; они в
`web/src/assets/promo/stickers/ruble.webp`, `bolt.webp`, `cross.webp` и стоят:

- в **полосе** (профиль и карта курса): три стикера медленно плывут вверх за текстом, каждый со своим наклоном, вместо прежних плывущих строк с ценами;
- на **большом плакате** (окно после урока): молния между датой и таймером, крестик и рубль в правом нижнем углу, въезжают последними в общей
  анимации появления.

Остальные символы (процент, часы, монета, кольцо, стрелка, звезда, лента) лежат в `cut/` на случай, если захочется добавить. Листы 1 (ласточка) и 2 (предметы) ещё не сгенерированы.
