# Кампания возврата: промты картинок

Четыре картинки к четырём сообщениям кампании (`docs/superpowers/specs/2026-11-08-return-campaign-design.md`): по одной на сегмент, одна и та же
картинка в Telegram (фото над текстом) и в письме (на всю ширину сверху). Стиль тот же, что у `web/src/assets/tariffs-promo-scene.webp` и
`invite-scene.webp` и у кадров историй (`docs/story-image-style.md`): цельная пиксельная сцена с тёплым светом, а не птица на плоском фоне.
Все промты ниже **полные**: стиль, запрет текста, композиция, ласточка (а в четвёртом ещё Ана и Марко) уже вписаны, ничего собирать не надо.

## Как генерировать

- **Референсы** к каждому запросу: `web/src/assets/tariffs-promo-scene.webp` (стиль, цвет, ласточка) и `web/src/assets/invite-scene.webp`. Для четвёртой картинки
  ещё лист героев Ана и Марко, если он остался от историй (`docs/story-image-prompts-drago-mi-je.md`, промт 0).
- **Формат** 4:3, чем больше, тем лучше (хотя бы 1600×1200). Все лица и главный предмет в центральных 60% по ширине: в Telegram кадр целиком, а в письме он ужимается
  до 480 px по ширине и на телефоне иногда подрезается сверху и снизу.
- **Серия.** Лучше делать все четыре в одном чате или с теми же референсами, чтобы освещение «золотого часа» и размер пикселя совпали. Картинки 1 и 2 это одна и та же комната, просто другое время дня.
- **Куда класть.** `landing/public/campaign/return-1.jpg` … `return-4.jpg`: JPEG 1280×960, около 300 КБ (PNG такого кадра весит 3 МБ, это слишком много для письма). Другие имена не подойдут, на них ссылается код.
- **Про текст.** Генераторы коверкают буквы, поэтому во всех промтах «без текста»: вместо букв значки (рука, солнце, галочка-«шапочка» над пустой плиткой).
  Если хотите буквы š, č, ć на картинке 3, допишите их руками поверх.

## Проверка перед вставкой в проект

- Одна ласточка на кадре, нет других птиц, нет букв и цифр.
- Лицо и главный предмет в центре, по краям спокойный фон.
- Фон сплошной, размер 4:3, цвет не выбивается из канала (бирюза, золото, тёплое небо).
- Четыре картинки выглядят как серия: одно и то же освещение, тот же размер пикселя.

---

## 1. «Первый урок ждёт» (ничего не делали)

Файл: `landing/public/campaign/return-1.jpg`

Мысль: тихое утро, закрытая книга и приоткрытое окно, ласточка зовёт начать. Тепло, без давления.

Полный промт, копировать целиком:

```
STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

SCENE: A calm early morning in a small sunlit room in Serbia, seen straight-on. A wooden desk stands against the wall under a wide open window. On the desk lies one closed book with a deep teal cover (#176B68) and a golden ribbon bookmark (#E3B04B) hanging out of it, next to a cream coffee cup with a little steam and a small pot with a red geranium. Through the open window: pastel rooftops of an old Serbian town, a church tower with a tiled spire, a pink and gold dawn sky with stepped colour bands, and light rays falling across the desk. The swallow perches on the corner of the closed book, turned towards the viewer with its head slightly tilted, as if inviting. The mood: quiet, warm and welcoming, the very beginning of something. Warm gold light, cool teal-violet shadows.

SWALLOW (the UCIMO mascot): exactly one small swallow in the whole image, drawn exactly like the swallows in the style reference: deep teal head, back and wings (#176B68, darker wing tips #0B3F3D), a cream belly (#F7F9F7), a golden-yellow V-shaped bib at the throat (#E3B04B), a small orange-gold beak and orange-gold feet, a round black eye with a white highlight, a long forked tail. It is a real bird about the size of a pigeon, with no clothes, no human arms and no human pose. There are no other birds anywhere in the image.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags, books and walls are blank or carry only simple pictograms.

COMPOSITION RULES (the picture is shown in a chat and at the top of an e-mail, so it may be trimmed a little at the top and bottom): keep the swallow, any characters and the key object inside the central 60% of the width and between 20% and 80% of the height. Leave about 6% of calm margin at every edge. Sky, roofs and floor go in the top and bottom bands.

OUTPUT: one single finished illustration, 4:3, edge to edge, no frame, no border, no caption, no second panel.
```

---

## 2. «Вы остановились на первом уроке»

Файл: `landing/public/campaign/return-2.jpg`

Мысль: книга уже открыта, закладка лежит на середине, всё сохранено, можно продолжить. Свет другой, послеполуденный.

Полный промт, копировать целиком:

```
STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

SCENE: The same cosy room in Serbia in the warm light of late afternoon, seen straight-on from the same position as a desk by an open window. On the wooden desk lies an open book with a deep teal cover, its pages cream and blank, the golden ribbon bookmark (#E3B04B) lying across the middle of the open pages. Beside it a half-finished cup of tea with a thin curl of steam, a pencil and a small pot with a red geranium. Through the window: rooftops and a church tower in golden afternoon light with long soft shadows, a few small clouds. The swallow stands on the edge of the open book and holds the end of the golden ribbon in its beak, gently lifting it as if saying "let's carry on from here". The mood: unhurried, safe, "everything is saved". Warm gold light, cool teal-violet shadows.

SWALLOW (the UCIMO mascot): exactly one small swallow in the whole image, drawn exactly like the swallows in the style reference: deep teal head, back and wings (#176B68, darker wing tips #0B3F3D), a cream belly (#F7F9F7), a golden-yellow V-shaped bib at the throat (#E3B04B), a small orange-gold beak and orange-gold feet, a round black eye with a white highlight, a long forked tail. It is a real bird about the size of a pigeon, with no clothes, no human arms and no human pose. There are no other birds anywhere in the image.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags, books and walls are blank or carry only simple pictograms.

COMPOSITION RULES (the picture is shown in a chat and at the top of an e-mail, so it may be trimmed a little at the top and bottom): keep the swallow, any characters and the key object inside the central 60% of the width and between 20% and 80% of the height. Leave about 6% of calm margin at every edge. Sky, roofs and floor go in the top and bottom bands.

OUTPUT: one single finished illustration, 4:3, edge to edge, no frame, no border, no caption, no second panel.
```

---

## 3. «Те самые буквы с галочками» (особые звуки)

Файл: `landing/public/campaign/return-3.jpg`

Мысль: страшные «шапочки» над буквами на самом деле милые и не кусаются. Вместо букв пустые деревянные плитки с маленькими галочками-«шапочками».

Полный промт, копировать целиком:

```
STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

SCENE: A friendly wooden table in the sun, seen straight-on and a little from above. A neat row of eight chunky blank wooden letter tiles stands on a wooden rack in the middle of the picture, in warm light wood with a thin darker edge. Each tile is empty, and above eight of them hover small cute V-shaped "hat" marks (like tiny pointed caps or little gold chevrons, #E3B04B) that glow softly, as if being gently placed on the tiles. Around: a cream cloth, a small teal mug, a tiny brass magnifying glass, a few sparkles. Behind the table a window with a warm pastel Serbian street and a soft teal-violet dusk sky. The swallow stands on the right end of the wooden rack, one wing stretched out and pointing at the tiles, its head tilted with a friendly, reassuring look ("see, nothing scary"). The mood: curious, playful, reassuring. Do not draw any letter or sign on the tiles: they stay blank.

SWALLOW (the UCIMO mascot): exactly one small swallow in the whole image, drawn exactly like the swallows in the style reference: deep teal head, back and wings (#176B68, darker wing tips #0B3F3D), a cream belly (#F7F9F7), a golden-yellow V-shaped bib at the throat (#E3B04B), a small orange-gold beak and orange-gold feet, a round black eye with a white highlight, a long forked tail. It is a real bird about the size of a pigeon, with no clothes, no human arms and no human pose. There are no other birds anywhere in the image.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags, books and walls are blank or carry only simple pictograms.

COMPOSITION RULES (the picture is shown in a chat and at the top of an e-mail, so it may be trimmed a little at the top and bottom): keep the swallow, any characters and the key object inside the central 60% of the width and between 20% and 80% of the height. Leave about 6% of calm margin at every edge. Sky, roofs and floor go in the top and bottom bands.

OUTPUT: one single finished illustration, 4:3, edge to edge, no frame, no border, no caption, no second panel.
```

---

## 4. «Теперь можно здороваться по-сербски» (урок «Привет!»)

Файл: `landing/public/campaign/return-4.jpg`

Мысль: тёплая уличная сцена, Ана и Марко здороваются, ласточка летит между ними. Вместо слов значки: рука, солнце. Ана и Марко те же, что в историях.

Полный промт, копировать целиком:

```
STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

SCENE: A bright morning on a quiet cobbled street of an old Serbian town, seen straight-on from the eye level of a standing person. Pastel townhouses with white-framed windows, warm-brown wooden shutters and red geraniums on the balconies, a linden tree, a sunny sky with stepped warm bands and soft clouds. On the left stands ANA, waving with one raised hand and smiling. On the right stands MARKO next to his bicycle, waving back with a wide easy smile. Above each of them floats a small round speech bubble with only a pictogram inside: above Ana a little waving hand, above Marko a small rising sun. Between them, in the middle of the picture, the swallow flies with spread wings and a golden sparkle trail. The mood: friendly, bright, "this is how a first greeting feels". Characters are large, each about half the image height.

ANA (a Russian woman in her late twenties who has just moved to Serbia): chibi, about 3 heads tall. A long thick chestnut-brown braid (#7A4A2B, lighter #A06A3F on top) that always hangs over her LEFT shoulder, with a small teal hair tie at the end and a short side-swept fringe. A deep teal scarf (#176B68) wrapped twice around her neck, the fringed ends hanging in front. A cream knee-length wool coat (#F2E9D8, shadow #D9CBB0) with two large buttons, dark navy jeans, clean white sneakers. Fair skin with light rosy cheeks, thin brown eyebrows, small friendly black dot eyes with a white highlight pixel, a small gentle smile.

MARKO (a local neighbour in his early thirties, warm and talkative): chibi, about 3 heads tall, a little taller and broader than Ana. Short black curly hair, a short neat dark beard, round glasses with thin dark frames (plain lenses, the dot eyes stay visible), warm medium skin, thick dark eyebrows, small friendly black dot eyes with a white highlight pixel, a wide easy smile. A rust-orange (#C4622D, shadow #9A4720) hooded sweatshirt with a front pocket and drawstrings, blue jeans, brown leather boots. His bicycle: a classic deep-teal (#176B68) city bike with a brown leather saddle, a wicker basket on the front and a rear rack, no lettering on the frame.

SWALLOW (the UCIMO mascot): exactly one small swallow in the whole image, drawn exactly like the swallows in the style reference: deep teal head, back and wings (#176B68, darker wing tips #0B3F3D), a cream belly (#F7F9F7), a golden-yellow V-shaped bib at the throat (#E3B04B), a small orange-gold beak and orange-gold feet, a round black eye with a white highlight, a long forked tail. It is a real bird about the size of a pigeon, with no clothes, no human arms and no human pose. There are no other birds anywhere in the image.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags, books and walls are blank or carry only simple pictograms.

COMPOSITION RULES (the picture is shown in a chat and at the top of an e-mail, so it may be trimmed a little at the top and bottom): keep the swallow, any characters and the key object inside the central 60% of the width and between 20% and 80% of the height. Leave about 6% of calm margin at every edge. Sky, roofs and floor go in the top and bottom bands.

OUTPUT: one single finished illustration, 4:3, edge to edge, no frame, no border, no caption, no second panel.
```
