# Демо-история «Drago mi je»: промты кадров (версия 2, неподвижная сцена)

Первая версия просила генератор нарисовать каждый кадр заново, с референсами. Фон при этом «плыл»: в промтах у каждого кадра
был свой ракурс (шире, ближе, снизу), а скамья стояла в центре двора, поэтому модель так её и ставила. Теперь иначе:

- **Одна камера на все кадры.** Меняются только герои и свет.
- **Скамья у стены** (и вообще всё стоит у стены), посреди пола ничего нет.
- **Кадр это правка готового фона**, а не новая картинка: модели сказано «фон не трогать, только добавить героев».
- **Позиции в процентах ширины**, чтобы можно было проверить глазами.

## Порядок работы

1. **Лист героев** (промт 0). Готов: Ана, Марко, ласточка и велосипед получились. Сохрани как `refs/sheet.png`.
2. **Плейт «день»** (промт 1). Сделай 3–4 варианта и выбери самый чистый. Проверь: скамья у стены, а не в середине; арка
   слева, подъезд справа, пол внизу пустой. **Этот файл дальше не пересоздаётся**, все кадры идут от него. Сохрани как `refs/plate_day.png`.
3. **Плейт «вечер»** (промт 2): правка плейта «день», загрузи его первой картинкой. Сохрани как `refs/plate_evening.png`.
4. **Кадры 1–3**: правка плейта «день». В генератор загрузи по порядку: 1) плейт «день», 2) лист героев, 3) `invite-scene.webp`
   (если можно), и вставь промт. **Кадр 4**: то же, но первым идёт плейт «вечер».
5. Сохрани кадры как `s1.png`…`s4.png` в `scripts/.cache/stories/drago-mi-je/frames/` и положи четыре рядом: арка, скамья,
   дерево и дверь должны стоять на одних и тех же местах.

Если правка всё равно двигает фон (или скамья «уезжает»), **план Б**: герои и реквизит рисуются отдельно на однотонном фоне, а
я накладываю их на плейт скриптом. Тогда фон совпадает по построению. Напиши, подготовлю промты слоёв.

## Промт 0. Лист героев (готов)

Оставлен для справки и на случай перегенерации.

```
REFERENCES (if a reference image is attached): it is the app's style reference. Match its pixel size and its rich colour, and draw the swallow exactly like the swallows in it.

TASK: a character reference sheet (model sheet) for a pixel-art story, landscape 4:3, on a flat, plain cream background (#F7F9F7) with no scenery. Only a tiny soft oval shadow under each figure. Two rows with generous space between the figures. Top row, left to right: ANA full-body front view, ANA full-body three-quarter view, MARKO full-body three-quarter view, MARKO full-body front view. Bottom row, left to right: the SWALLOW perched on a small twig seen from the side, the SWALLOW in flight with wings spread, and MARKO's BICYCLE in side view. Every figure is drawn at the same scale and the same pixel size as it will be in the final scenes, neutral friendly expressions, the same outfit in every view. No labels, no arrows, no captions, no text.

STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

ANA (a Russian woman in her late twenties who has just moved to Serbia): chibi, about 3 heads tall. A long thick chestnut-brown braid (#7A4A2B, lighter #A06A3F on top) that always hangs over her LEFT shoulder, with a small teal hair tie at the end and a short side-swept fringe. A deep teal scarf (#176B68) wrapped twice around her neck, the fringed ends hanging in front. A cream knee-length wool coat (#F2E9D8, shadow #D9CBB0) with two large buttons, dark navy jeans, clean white sneakers. Fair skin with light rosy cheeks, thin brown eyebrows, small friendly black dot eyes with a white highlight pixel, a small gentle smile.

MARKO (a local neighbour in his early thirties, warm and talkative): chibi, about 3 heads tall, a little taller and broader than Ana. Short black curly hair, a short neat dark beard, round glasses with thin dark frames (plain lenses, the dot eyes stay visible), warm medium skin, thick dark eyebrows, small friendly black dot eyes with a white highlight pixel, a wide easy smile. A rust-orange (#C4622D, shadow #9A4720) hooded sweatshirt with a front pocket and drawstrings, blue jeans, brown leather boots. His bicycle: a classic deep-teal (#176B68) city bike with a brown leather saddle, a wicker basket on the front and a rear rack, no lettering on the frame.

SWALLOW (the UCIMO mascot): exactly one small swallow in the whole image, drawn exactly like the swallows in the style reference: deep teal head, back and wings (#176B68, darker wing tips #0B3F3D), a cream belly (#F7F9F7), a golden-yellow V-shaped bib at the throat (#E3B04B), a small orange-gold beak and orange-gold feet, a round black eye with a white highlight, a long forked tail. It is a real bird about the size of a pigeon (roughly one third of a character's height), with no clothes, no human arms and no human pose. There are no other birds anywhere in the image.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: extra people, scenery, props other than the bicycle and the twig, mixed pixel sizes, smooth gradients, glossy plastic shine, anything that looks like writing.
```

## Промт 1. Плейт «день»: пустая сцена

**Идея:** фон для всех кадров: стена с аркой слева, липа и скамья у стены в центре, подъезд справа, пустой пол.
Приложи `invite-scene.webp` как референс стиля.

```
REFERENCES (if a reference image is attached): it is the app's style reference. Match its pixel size and its rich colour. Do not draw any birds.

TASK: the empty stage: a clean, frontal background picture that will be reused as the fixed background for four more pictures, so it must be plain and legible. The camera is straight-on at standing eye level and will never change. Keep the wall flat, frontal and uncluttered, keep the bench clearly visible against the wall, and leave the bottom quarter as open, empty cobblestones for characters to stand on. There are no people, no animals, no birds, no bicycle, no boxes and no vehicles anywhere.

STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

SETTING (a fixed stage that never changes): the inner courtyard (dvorište) of an old three-storey townhouse in Novi Sad, Serbia, seen from ONE fixed camera position: straight-on, at the eye level of a standing person, 4:3. The picture is dominated by one long back wall, a soft sage-mint (#9CC7BE, slightly greyed) stucco facade that runs from the left edge to the right edge of the frame, with a band of cobblestone floor in front of it covering the bottom quarter of the picture. Nothing stands in the middle of the open floor: every object stands against the wall. From left to right along the wall: (1) at the far left, 0-22% of the width, a tall arched carriage passage that leads out to a sunlit street, with a pale street and a pastel house seen through the arch; (2) at 24-32% a low stone-edged bed with the trunk of a big old linden tree growing out of it, its wide round crown filling the upper middle of the picture above the wall; (3) at 32-60% a long wooden bench with a curved backrest standing flat against the wall right beside the linden trunk, its back touching the wall, with one small potted red geranium at its right end; (4) at 66-96% the building's entrance: three stone steps up to a mustard-gold (#E3B04B) double wooden door with an arched glass window above it, a small brass lamp beside the door, a row of blank metal mailboxes on the wall, pots of red geraniums on the steps; (5) above the entrance, on the first floor, a wrought-iron balcony with geraniums. The wall also has white-framed windows with warm-brown wooden shutters, and a clothesline with a few towels between two windows above the bench. Far behind the roofline: a church tower with a tiled spire. The cobblestone floor in front is open and empty.

LIGHT: late morning. The sun is high on the left, shadows are short, soft and fall slightly to the right, the light on the sage-mint wall is warm and clean, shadows are cool violet-teal, and the linden throws a dappled pattern of light on the bench and the cobblestones. Sky: clear light blue with one thin pale cloud, seen above the roofline and through the arch.

COMPOSITION RULES (the picture is shown in an app, cropped on small screens and slowly zoomed in): keep the arch, the bench, the linden trunk and the door fully inside the frame with about 7% of calm margin at every edge, and keep the cobblestone floor in the bottom quarter open and empty.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: people, animals, birds, bicycles, boxes, vehicles, a bench in the middle of the floor, objects standing in the open floor, mixed pixel sizes, smooth gradients, anything that looks like writing.
```

## Промт 2. Плейт «вечер»: правка плейта «день»

**Идея:** тот же фон, меняется только свет. Первой картинкой загрузи плейт «день».

```
EDIT TASK. Image 1 is the finished daytime background of a pixel-art story. Recreate image 1 with ONLY the time of day changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour (except for the light): the camera position and framing, the arch, the linden tree, the bench against the wall, the entrance with its steps and its door, the balcony, the windows, the clothesline, the church tower, the cobblestones. Every object stays at the same position and the same size, and the output has the same size and aspect ratio as image 1. No people, no animals, no birds, no bicycle, no boxes.

STYLE: Pixel art, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, one identical pixel size across the whole image. Rich, warm, atmospheric colour like the style reference: skies in stepped colour bands, colour-shifted cool violet-teal shadows, warm gold lights. Crisp 1-pixel dark-teal (#0B3F3D) outlines on characters, chibi characters about 3 heads tall with black dot eyes with one white highlight pixel. No smooth gradients, no blur, no glossy shine, no 3D look.

LIGHT: golden hour, about six in the evening. Warm orange light from the left, long shadows reaching right across the cobblestones, a stepped sky from teal-blue at the top through peach to gold near the roofs, windows glowing warm yellow (including the arched window of the door), shadows deep violet-teal.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.
```

## Кадр 1. Привет! Знакомство у подъезда

**Звучит:** 1–4: Zdravo! / Ćao! Kako si? / Dobro, hvala. A ti? / Može. Kako se zoveš?

**Идея:** День переезда. Ана с коробкой на ступенях, Марко идёт по двору с велосипедом и машет ей. Видно, что они только заметили друг друга.

**Основа:** плейт «день» (первая картинка), потом лист героев.

**Проверить:** Ана справа на ступенях с коробкой и растением, Марко слева на булыжнике с велосипедом; между ними пустой пол; скамья пустая и стоит у стены; фургон в арке; ласточка одна.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Ana, Marko, the bicycle and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall, the arch, the linden tree, the bench against the wall at 32-60% of the width, the entrance with its steps and door, the balcony, the windows, the clothesline, the cobblestones, the sky and the light. The output has the same size and aspect ratio as image 1, every background object stays at the same position, and the bench is never moved. The camera is the same in all four pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 1 of 4, moving day, the first hello):
- Ana stands on the second step of the entrance on the right (about 72% of the width), hugging a big blank brown cardboard box with both arms. A leafy green houseplant in a terracotta pot sticks out of the top of the box, its leaves almost touching her chin. Her face is turned toward Marko: eyebrows slightly raised, a shy friendly smile, her braid and scarf swaying a little. Two more blank cardboard boxes and a rolled-up striped rug stand at the foot of the steps next to her.
- Marko is on the open cobblestones on the left (about 30% of the width, in front of the arched passage and the linden), walking his deep-teal bicycle with one hand on the handlebar. His other arm is raised high in a big friendly wave, his mouth is open in a cheerful hello, his body leans a little toward Ana. A blank paper bakery bag with a bread roll sticking out sits in the bicycle's basket.
- There is clear empty floor between the two characters, so it reads as two people who have just noticed each other. The bench stays empty.
- Far to the left, seen through the arched passage, the back of a small white delivery van with its rear doors open (no lettering, no licence plate).
- Both faces are at about 40% of the image height.

CHARACTERS: Ana and Marko exactly as in image 2, the character sheet (Ana: chestnut braid over her left shoulder, deep teal scarf, cream coat; Marko: black curls, dark beard, round glasses, rust-orange hoodie), and Marko's deep-teal bicycle with a wicker basket exactly as in image 2. Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on a linden branch above the bench, looking down at the two of them. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, a second bicycle, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Кадр 2. Не понимаю. Помедленнее, пожалуйста

**Звучит:** 5–8: Zovem se Marko. / Izvini, ne razumem. Polako, molim. / Marko. Zovem se Marko. Drago mi je. / Aha! Ja sam Ana. Takođe.

**Идея:** Марко тараторит и показывает на себя, Ана просит помедленнее. Самый «живой» кадр: смущённая улыбка, поднятая ладонь, поток закорючек вместо речи.

**Основа:** плейт «день» (первая картинка), потом лист героев.

**Проверить:** ладонь Аны поднята, рука Марко на груди, пузырь с закорючками без читаемых букв; вопросительный знак один; велосипед у стены; скамья пустая у стены за Марко; ласточка одна.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Ana, Marko, the bicycle and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall, the arch, the linden tree, the bench against the wall at 32-60% of the width, the entrance with its steps and door, the balcony, the windows, the clothesline, the cobblestones, the sky and the light. The output has the same size and aspect ratio as image 1, every background object stays at the same position, and the bench is never moved. The camera is the same in all four pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 2 of 4, 'too fast, slower please'):
- Marko stands on the open floor at about 40% of the width, in front of the bench (the bench stays against the wall behind him), turned three-quarters toward Ana. Talking very fast: his right hand is pressed flat against his own chest (he is saying his name), his left hand is mid-gesture with spread fingers, his eyebrows are high and his mouth is wide open, with a few small motion lines near his hands. From his mouth comes a round speech bubble filled with a dense chain of tiny wavy squiggles and little dots, like a torrent of fast speech, with no readable letters.
- Ana stands on the floor at the foot of the steps at about 64% of the width, turned three-quarters toward Marko, a little overwhelmed but kind: her right palm is raised toward him in a clear 'wait, slow down' gesture, her other hand tugs at the end of her scarf, her head is tilted, her eyebrows are raised and she has an apologetic crooked smile. A single small pixel question mark floats above her head (a pictogram only, not a letter).
- Marko's deep-teal bicycle is parked against the wall between the arch and the linden (about 20% of the width). Ana's plant box and the other blank boxes stand on the steps behind her, out of the way.
- Both faces are at about 38% of the image height.

CHARACTERS: Ana and Marko exactly as in image 2, the character sheet (Ana: chestnut braid over her left shoulder, deep teal scarf, cream coat; Marko: black curls, dark beard, round glasses, rust-orange hoodie), and Marko's deep-teal bicycle with a wicker basket exactly as in image 2. Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the wrought-iron balcony railing above the door, its head tilted as if it is listening. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, a second bicycle, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Кадр 3. Откуда ты? Немного, учу

**Звучит:** 9–12: Odakle si? / Iz Rusije. Živim u Novom Sadu. / Govoriš srpski? / Malo. Učim.

**Идея:** Перерыв на скамейке у стены: Марко угощает Ану выпечкой, Ана показывает пальцами «чуть-чуть», над ней в пузыре заснеженный собор с луковками (Россия без флагов и надписей).

**Основа:** плейт «день» (первая картинка), потом лист героев.

**Проверить:** скамья на прежнем месте у стены (не в середине двора), герои сидят на ней; пальцы Аны сведены, в пузыре собор с луковками, снежинки и берёза, без флагов и букв; ласточка одна.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Ana, Marko, the bicycle and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall, the arch, the linden tree, the bench against the wall at 32-60% of the width, the entrance with its steps and door, the balcony, the windows, the clothesline, the cobblestones, the sky and the light. The output has the same size and aspect ratio as image 1, every background object stays at the same position, and the bench is never moved. The camera is the same in all four pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 3 of 4, 'where are you from?'):
- Ana and Marko sit side by side on the existing bench, which stays exactly where it is, flat against the wall at 32-60% of the width. Marko sits on the left half, Ana on the right half, turned toward each other, close and relaxed, their feet on the cobblestones in front of the bench. Marko holds a blank paper bakery bag open and offers Ana a golden flaky pastry (like a burek, a coiled or folded pie). Ana has just taken a piece in her left hand, and with her right hand she pinches her thumb and index finger almost together in front of her chest, the 'just a little' gesture, with a modest smile. Marko listens with a warm, interested smile, his head slightly tilted, nodding.
- Above Ana's head floats a pixel thought bubble (a cloud shape with two small circles leading down to her head) with a tiny picture inside: a snowy winter scene with a colourful onion-domed cathedral, a few falling snowflakes and one slim birch tree, drawn simply, with no flags and no text.
- Marko's deep-teal bicycle leans against the wall at about 20% of the width, between the arch and the linden. A stack of blank cardboard boxes stands at the foot of the entrance steps on the right.
- Both faces are at about 45% of the image height (they are sitting).

CHARACTERS: Ana and Marko exactly as in image 2, the character sheet (Ana: chestnut braid over her left shoulder, deep teal scarf, cream coat; Marko: black curls, dark beard, round glasses, rust-orange hoodie), and Marko's deep-teal bicycle with a wicker basket exactly as in image 2. Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the top edge of the bench's backrest at the far right end, near Ana's shoulder, looking at the pastry. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, a second bicycle, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Кадр 4. Увидимся!

**Звучит:** 13–14: Dobro! Vidimo se! / Ćao!

**Идея:** Вечер, переезд закончен. Кадр зеркалит кадр 1 (та же камера, те же места): вместо хаоса коробок тёплый свет в окнах, растение на балконе Аны, ключ в руке. Прощаются до завтра.

**Основа:** плейт «вечер» (первая картинка), потом лист героев.

**Проверить:** тот же фон, что в кадре 1, только вечерний свет; Марко со спины у арки, лица и поднятой руки нет, ведёт велосипед; Ана на ступени лицом к нему машет рукой, в другой ключ с бирюзовой ленточкой; почти нет коробок; растение на балконе; скамья на месте и пустая; ласточка одна.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Ana, Marko, the bicycle and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall, the arch, the linden tree, the bench against the wall at 32-60% of the width, the entrance with its steps and door, the balcony, the windows, the clothesline, the cobblestones, the sky and the light. The output has the same size and aspect ratio as image 1, every background object stays at the same position, and the bench is never moved. The camera is the same in all four pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 4 of 4, 'see you!', evening, mirroring frame 1):
- The moving-day mess is gone: only one small blank cardboard box stands by the door. Ana's houseplant now stands on the first-floor balcony railing among the geraniums, and the lit window next to the balcony shows a new striped curtain, so her flat looks lived-in. A warm yellow glow shines through the glass of the door's arched window. The door itself stays closed.
- Ana stands on the top step in front of the door at about 72% of the width, her body in three-quarter view facing Marko (to the left), her head facing the same way as her body, a small brass key with a teal ribbon in one hand (held near her chest, clearly visible) and the other hand raised in a wave. She is smiling, relaxed and happy.
- Marko is leaving: he is at the arched passage at about 14% of the width, seen from BEHIND, with the back of his head (black curls), the back of his rust-orange hoodie and the back of his jeans toward the viewer, so no part of his face is visible at all. He walks away into the arch, wheeling his deep-teal bicycle beside him with both hands on the handlebar, the bicycle also seen from behind with its rear rack. His head is straight and is not turned: he does not look back, and no hand of his is raised. Through the arch the sunset street is visible ahead of him. There is a clear distance between him and Ana, as in frame 1. The bench stays empty.
- Both faces are at about 40% of the image height.

CHARACTERS: Ana and Marko exactly as in image 2, the character sheet (Ana: chestnut braid over her left shoulder, deep teal scarf, cream coat; Marko: black curls, dark beard, round glasses, rust-orange hoodie), and Marko's deep-teal bicycle with a wicker basket exactly as in image 2. Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the balcony railing next to Ana's houseplant, a small silhouette against the sunset sky. The swallow is added in this picture only, it is not part of the background.

The background already has its evening light, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, a second bicycle, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Если что-то не получается

- **Фон поменялся даже в правке.** Добавь в начало промта: «This is an image edit. Output image 1 with the additions only; everything not mentioned stays pixel-identical.» Если у генератора есть маска или выделение области, выдели только пол и скамью, куда ставятся герои. Иначе план Б.
- **Скамья уезжает или появляется вторая.** Допиши: «There is exactly one bench, against the wall, at 32-60% of the width, and it never moves.»
- **Герои разные от кадра к кадру.** Лист героев должен быть второй картинкой, и в промт добавь: «Keep Ana's and Marko's faces, hair and clothes identical to image 2.»
- **Голова перевёрнута или шея вывернута** (обычно от «оглядывается через плечо»). Не перегенерируй весь кадр, исправь правкой готового кадра, загрузив кадр первой картинкой и лист героев второй. Вариант А (оглядывается естественно): «Edit this image. Redraw only Marko's head. His body faces away to the left, so turn his head back over his right shoulder at a natural angle of about 90 degrees: we see his head in three-quarter profile with the cheek, one lens of the glasses, the beard and a smile, looking toward Ana, on a normal neck. Same hair, beard and glasses as in image 2. Change nothing else, every other pixel stays identical.» Вариант Б (проще): «...Redraw only Marko's head so that it faces the same direction as his body, looking where he is going, with the same hair, beard and glasses as in image 2. Change nothing else.» Поза «оглядывается» в промтах заменена на «стоит вполоборота лицом к собеседнику».
- **Рука (особенно поднятая) выходит кривой.** Генераторы хуже всего рисуют руки в движении. Не правь руку по кругу: опусти её. На готовом кадре: «Edit this image. Redraw only Marko's raised arm: lower it so that his hand rests in the front pocket of his hoodie (or grips the handlebar as a simple closed fist). Change nothing else, every other pixel stays identical.» В промте кадра 4 Марко уходит в арку спиной к кадру: у него нет ни лица, ни поднятой руки, а прощается рукой одна Ана.
- **Буквы на коробках или в пузыре.** Допиши: «Every box, bag and bubble is completely blank, no markings at all». Если не помогло, замазать в редакторе.
- **Пиксели разного размера.** Могу написать скрипт, который уменьшит кадр до сетки 320×240 и вернёт увеличенным без сглаживания, тогда все кадры получат один размер пикселя.
