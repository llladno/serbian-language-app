# История «SIM kartica»: промты кадров

«SIM kartica»: герои **Алекс** (русский парень в джинсовой куртке с оливковым рюкзаком, красный паспорт без надписей) и **Продавщица** (чёрные волосы в пучке, коралловый жилет). Сцена: небольшой салон связи в Белграде, день, один плейт. Блоки стиля, «без текста», композиции и ласточки лежат в `docs/story-image-style.md`, а герои и сцена в разделе «Истории уровня 2». Особенность: **прилавок через весь кадр**, продавщица стоит за ним (видна по пояс), покупатель на полу перед ним (во весь рост). Пустые упаковки, коробки и бумажки: нейросети искажают текст.

## Порядок работы

1. **Лист героев** (промт 0). Сохрани как `refs/sheet.png`. Проверь: приметы героев, ласточка одна.
2. **Плейт** (промт 1). Сделай 3–4 варианта и выбери самый чистый: все предметы на прилавке на своих местах, место за прилавком и пол пустые. Этот файл дальше не пересоздаётся. Сохрани как `refs/plate.png`.
3. **Кадры 1–5**: правка плейта. В генератор по порядку: 1) плейт, 2) лист героев, 3) `web/src/assets/invite-scene.webp` (если можно), и вставь промт кадра.
4. Сохрани кадры как `s1.png`…`s5.png` в `scripts/.cache/stories/sim-kartica/frames/` и положи 5 рядом: навес/стена, прилавок и все предметы на нём должны стоять на одних и тех же местах. Затем `cd /Users/grisha/plans/serbian-app && python3 scripts/story_publish.py sim-kartica --dev` (нужен `make dev`).

Если правка двигает фон, план Б: герои рисуются отдельно на однотонном фоне, а я накладываю их на плейт скриптом.

Говорят: **Продавщица** (женщина) и **Алекс** (мужчина).

## Промт 0. Лист героев

```
REFERENCES (if a reference image is attached): it is the app's style reference. Match its pixel size and its rich colour, and draw the swallow exactly like the swallows in it.

TASK: a character reference sheet (model sheet) for a pixel-art story, landscape 4:3, on a flat, plain cream background (#F7F9F7) with no scenery. Only a tiny soft oval shadow under each figure. Two rows with generous space between the figures. Top row, left to right: ALEKS full-body front view, ALEKS full-body three-quarter view, the SALESWOMAN (PRODAVAČICA) full-body front view, the SALESWOMAN full-body three-quarter view. Bottom row, left to right: the SWALLOW perched on a small twig seen from the side, the SWALLOW in flight with wings spread, a plain RED PASSPORT seen from the front (no lettering), a blank teal-and-gold SIM-CARD PACK and a blank white slip of paper. Every figure is drawn at the same scale and the same pixel size as it will be in the final scenes, neutral friendly expressions, the same outfit in every view. No labels, no arrows, no captions, no text.

STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

ALEKS (a Russian man in his late twenties who has just moved to Belgrade): chibi, about 3 heads tall. Short, slightly messy dark-blond hair (#B08A55, lighter #CBA774), light stubble, fair skin with a warm tint, medium-thick brown eyebrows, small friendly black dot eyes with a white highlight pixel, a relaxed easy smile. A faded blue denim jacket (#4C6FA5, shadow #355079) over a plain grey T-shirt, dark-grey jeans, white sneakers, a small olive-green (#6B7B3A) backpack worn on both shoulders (the straps are visible on his chest). His passport is a small plain red booklet (#B3262E) with one tiny blank gold pictogram on the cover and no lettering.

PRODAVAČICA (the saleswoman, a quick, friendly, confident woman in her mid-thirties): chibi, about 3 heads tall. Glossy black hair in a neat high bun with a few strands at her temples, warm olive skin, thin dark eyebrows, small bright black dot eyes with a white highlight pixel, a confident warm smile with soft coral lips. A coral-red (#C8503D, shadow #9C3A2B) sleeveless work vest over a crisp white blouse with rolled sleeves, a blank white lanyard badge with no writing, small gold hoop earrings. She is seen from the waist up above the counter.

SWALLOW (the UCIMO mascot): exactly one small swallow in the whole image, drawn exactly like the swallows in the style reference: deep teal head, back and wings (#176B68, darker wing tips #0B3F3D), a cream belly (#F7F9F7), a golden-yellow V-shaped bib at the throat (#E3B04B), a small orange-gold beak and orange-gold feet, a round black eye with a white highlight, a long forked tail. It is a real bird about the size of a pigeon (roughly one third of a character's height), with no clothes, no human arms and no human pose. There are no other birds anywhere in the image.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: extra people, scenery, props other than the twig, the passport, the SIM pack and the paper, mixed pixel sizes, smooth gradients, glossy plastic shine, anything that looks like writing.
```

## Промт 1. Плейт «день»: пустая сцена

**Идея:** фон для всех кадров. Приложи `invite-scene.webp` как референс стиля.

```
REFERENCES (if a reference image is attached): it is the app's style reference. Match its pixel size and its rich colour. Do not draw any birds.

TASK: the empty stage: a clean, frontal background picture that will be reused as the fixed background for five more pictures, so it must be plain and legible. The camera is straight-on at standing eye level and will never change. Keep the back wall flat, frontal and uncluttered, keep every object clearly visible and separate from its neighbours, and leave the bottom quarter as open, empty floor for characters to stand on. Keep the shop bright and simple, and make the shelves, the monitor, the card terminal and the SIM-pack display easy to tell apart. There are no people, no animals and no birds anywhere.

STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

SETTING (a fixed stage that never changes): a small mobile-phone shop in Belgrade, Serbia, seen from ONE fixed camera position: straight-on, at the eye level of a standing person, 4:3. The back wall is a soft sage-mint (#9CC7BE, slightly greyed) painted wall with a pale cream upper band; a warm grey tiled floor with a little pixel texture covers the bottom quarter of the picture and is open and empty. From left to right along the back wall: (1) at 0-22% of the width a big shop window with a pale-wood frame, open onto a sunny Belgrade street: pastel facades, a few treetops, a bright blue sky; (2) at 24-44% a wall shelf unit with rows of blank white phone boxes and two blank phones on small stands (no screens with text); (3) at 46-68% bare wall with one round clock with plain hands and a simple potted plant under it on the counter's far end; (4) at 72-96% a second shelf unit with blank teal-and-gold cardboard SIM-card packs on small hooks, and a mustard-gold (#E3B04B) door frame at the far right edge. In front of the wall runs one long white-and-pale-wood counter from 8% to 92% of the width: its top edge is at 62% of the height and its plain front panel goes down to 76%. On the counter, from left to right: (1) at 22-30% a flat computer monitor seen from the side with its screen turned toward the saleswoman (its screen is not visible) and a keyboard; (2) at 52-58% a small black-and-teal card-payment terminal on a short stand; (3) at 76-90% a small display stand with a few blank SIM-card packs. The space behind the counter at 34-48% of the width is empty (this is where the saleswoman will stand; she is not in the stage). No people, no animals, no birds.

LIGHT: daytime. Bright, slightly cool daylight comes in through the shop window at the left and lays a soft white-gold patch on the counter and the tiles; shadows are short, cool violet-teal; the shop lights are a clean white. Sky seen through the window: clear light blue with one thin cloud.

COMPOSITION RULES (the picture is shown in an app, cropped on small screens and slowly zoomed in): keep the window, the shelf units, the clock, the monitor, the card terminal and the SIM-pack display fully inside the frame with about 7% of calm margin at every edge and keep the floor in the bottom quarter open and empty.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: people, animals, birds, objects standing on the open floor, mixed pixel sizes, smooth gradients, anything that looks like writing.
```

## Кадр 1

**Звучит:** 1–2: Dobar dan! Izvolite? / Dobar dan! Treba mi SIM kartica.

**Идея:** Алекс только что вошёл в салон связи; продавщица за прилавком приветливо встречает его.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Продавщица за прилавком (около 42 %), виден по пояс, открытая ладонь, улыбка; Алекс на полу перед прилавком (около 68 %), лицом влево, рюкзак на двух плечах, свободная рука поднята в приветствии; пол пустой; ласточка одна.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw the characters and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall with the clock, the shop window with the view of the street at 0-22% of the width, the shelf unit with blank phone boxes at 24-44%, the second shelf unit with blank SIM packs and the gold door frame at 72-96%, the long white counter (top edge at 62% of the height), the monitor at 22-30%, the card terminal at 52-58%, the SIM-pack display stand at 76-90%, the tiled floor, and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all 5 pictures of the story, so do not zoom, tilt or change the angle. Characters are drawn at the same scale as the stage: a character standing on the floor is about half of the image height, a character behind the counter is seen from the waist up with the face at about 38% of the image height.

ADD (frame 1 of 5):
- The saleswoman stands behind the counter at about 42% of the width, seen from the waist up, three-quarter view facing right toward Aleks, body and head turned the same way. A friendly professional smile; the hand on the side nearer to the viewer lies open on the counter in a welcoming 'how can I help' gesture.
- Aleks stands on the open tiled floor in front of the counter at about 68% of the width, in three-quarter view facing left toward her, head and body turned the same way. He has just walked in: a friendly smile, the hand on the side facing her raised to about shoulder height in a small greeting, the other hand on the strap of his olive backpack.
- Both faces are inside the central 60% of the width.

CHARACTERS: Aleks exactly as in image 2, the character sheet (short messy dark-blond hair, light stubble, faded blue denim jacket over a grey T-shirt, olive backpack worn on both shoulders, plain red passport with no lettering); the saleswoman exactly as in image 2 (black hair in a high bun, coral-red work vest over a white blouse, gold hoop earrings). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the sill of the shop window at the left (about 12% of the width), looking at the two of them. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, a third hand or a third arm on anyone (every person has exactly two arms and two hands, every visible hand belongs to a visible arm), objects held in more than two hands at once, mixed pixel sizes, smooth gradients, the saleswoman standing in front of the counter.
```

## Кадр 2

**Звучит:** 3–8: Treba mi pasoš, molim vas. / Evo, izvolite. / Iz Rusije ste? / Da, ali sada živim ovde. / Dobro. SIM i internet, da? / Da, da. Koliko košta?

**Идея:** Алекс протягивает красный паспорт. Продавщица открывает его, удивлённо и тепло улыбается: он из России, они болтают.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Красный паспорт без надписей, рука Алекса вытянута над прилавком; продавщица держит паспорт, смотрит на Алекса с тёплой улыбкой (голова и тело повёрнуты вправо); у Алекса довольное лицо; ласточка одна, на верхней полке.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw the characters and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall with the clock, the shop window with the view of the street at 0-22% of the width, the shelf unit with blank phone boxes at 24-44%, the second shelf unit with blank SIM packs and the gold door frame at 72-96%, the long white counter (top edge at 62% of the height), the monitor at 22-30%, the card terminal at 52-58%, the SIM-pack display stand at 76-90%, the tiled floor, and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all 5 pictures of the story, so do not zoom, tilt or change the angle. Characters are drawn at the same scale as the stage: a character standing on the floor is about half of the image height, a character behind the counter is seen from the waist up with the face at about 38% of the image height.

ADD (frame 2 of 5):
- The saleswoman stands behind the counter at about 42% of the width, seen from the waist up, three-quarter view toward Aleks, head and body turned the same way. She holds the open red passport in the hand on the side nearer to the viewer and looks up from it at Aleks with a warm, pleasantly surprised smile and raised eyebrows.
- Aleks stands on the tiled floor in front of the counter at about 68% of the width, three-quarter view facing left toward her, head and body turned the same way. His arm on the side facing her is still slightly stretched out over the counter edge after handing the passport over, he smiles with a relaxed, slightly shy nod; the other hand on his backpack strap.
- Both faces are inside the central 60% of the width.

CHARACTERS: Aleks exactly as in image 2, the character sheet (short messy dark-blond hair, light stubble, faded blue denim jacket over a grey T-shirt, olive backpack worn on both shoulders, plain red passport with no lettering); the saleswoman exactly as in image 2 (black hair in a high bun, coral-red work vest over a white blouse, gold hoop earrings). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the top shelf of the shelf unit at the left (about 34% of the width), looking at the passport. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, a third hand or a third arm on anyone (every person has exactly two arms and two hands, every visible hand belongs to a visible arm), objects held in more than two hands at once, mixed pixel sizes, smooth gradients, the saleswoman standing in front of the counter.
```

## Кадр 3

**Звучит:** 9–10: Trista dinara. Kešom ili karticom? / Karticom.

**Идея:** Алекс прикладывает банковскую карту к терминалу; продавщица держит наготове упаковку с сим-картой.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Карта (пустая) у терминала на прилавке, рука Алекса вытянута вперёд-влево; продавщица держит маленькую пустую упаковку сим-карты у груди; оба смотрят на терминал или друг на друга, голова и тело в одну сторону; ласточка одна, на терминале.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw the characters and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall with the clock, the shop window with the view of the street at 0-22% of the width, the shelf unit with blank phone boxes at 24-44%, the second shelf unit with blank SIM packs and the gold door frame at 72-96%, the long white counter (top edge at 62% of the height), the monitor at 22-30%, the card terminal at 52-58%, the SIM-pack display stand at 76-90%, the tiled floor, and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all 5 pictures of the story, so do not zoom, tilt or change the angle. Characters are drawn at the same scale as the stage: a character standing on the floor is about half of the image height, a character behind the counter is seen from the waist up with the face at about 38% of the image height.

ADD (frame 3 of 5):
- The saleswoman stands behind the counter at about 42% of the width, seen from the waist up, three-quarter view toward Aleks, head and body turned the same way. Her hand on the side nearer to the viewer holds a small blank teal-and-gold SIM-card pack up at chest height, her other hand rests on the counter; a friendly businesslike smile.
- Aleks stands on the tiled floor in front of the counter at about 66% of the width, three-quarter view facing left, head and body turned the same way. He holds a plain teal bank card (completely blank, no numbers, no logo) with the hand on the side facing the terminal, tapping it on the card terminal on the counter; a calm smile.
- Both faces are inside the central 60% of the width.

CHARACTERS: Aleks exactly as in image 2, the character sheet (short messy dark-blond hair, light stubble, faded blue denim jacket over a grey T-shirt, olive backpack worn on both shoulders, plain red passport with no lettering); the saleswoman exactly as in image 2 (black hair in a high bun, coral-red work vest over a white blouse, gold hoop earrings). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on top of the card terminal's stand (about 55% of the width), looking at the card. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, a third hand or a third arm on anyone (every person has exactly two arms and two hands, every visible hand belongs to a visible arm), objects held in more than two hands at once, mixed pixel sizes, smooth gradients, the saleswoman standing in front of the counter.
```

## Кадр 4

**Звучит:** 11–14: Samo malo… Evo! Ovo je broj: nula šest četiri sedam osam pet devet jedan tri. / Izvinite, ne razumem. Polako, molim vas. / Nula šest četiri… sedam osam pet… devet jedan tri. / Sad razumem. Hvala!

**Идея:** Продавщица быстро читает номер с бумажки (облачко с волнистыми закорючками), Алекс растерянно поднимает ладонь: помедленнее.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** У продавщицы бумажка (пустая) в руке и облачко-речь с волнистыми линиями без букв; Алекс с ладонью вверх на уровне груди (рука со стороны продавщицы), брови домиком, лёгкая растерянная улыбка; ласточка одна, на мониторе.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw the characters and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall with the clock, the shop window with the view of the street at 0-22% of the width, the shelf unit with blank phone boxes at 24-44%, the second shelf unit with blank SIM packs and the gold door frame at 72-96%, the long white counter (top edge at 62% of the height), the monitor at 22-30%, the card terminal at 52-58%, the SIM-pack display stand at 76-90%, the tiled floor, and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all 5 pictures of the story, so do not zoom, tilt or change the angle. Characters are drawn at the same scale as the stage: a character standing on the floor is about half of the image height, a character behind the counter is seen from the waist up with the face at about 38% of the image height.

ADD (frame 4 of 5):
- The saleswoman stands behind the counter at about 42% of the width, seen from the waist up, three-quarter view toward Aleks, head and body turned the same way. She holds a small blank slip of paper in the hand on the side nearer to the viewer and talks quickly: her mouth is open in a cheerful rush. Next to her head floats a pixel speech bubble filled with three wavy squiggle lines (no letters, no digits) to show fast speech.
- Aleks stands on the tiled floor in front of the counter at about 68% of the width, three-quarter view facing left toward her, head and body turned the same way. He is a little lost but friendly: his eyebrows are slanted up in the middle, a small apologetic smile, the hand on the side facing her is raised open at chest height with the palm toward her in a 'slow down, please' gesture, the other hand on his backpack strap.
- Both faces are inside the central 60% of the width.

CHARACTERS: Aleks exactly as in image 2, the character sheet (short messy dark-blond hair, light stubble, faded blue denim jacket over a grey T-shirt, olive backpack worn on both shoulders, plain red passport with no lettering); the saleswoman exactly as in image 2 (black hair in a high bun, coral-red work vest over a white blouse, gold hoop earrings). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the top edge of the computer monitor (about 26% of the width), looking at the paper slip. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, a third hand or a third arm on anyone (every person has exactly two arms and two hands, every visible hand belongs to a visible arm), objects held in more than two hands at once, mixed pixel sizes, smooth gradients, the saleswoman standing in front of the counter.
```

## Кадр 5

**Звучит:** 15–16: Izvolite pasoš. Dobro došli u Srbiju! / Hvala! Doviđenja!

**Идея:** Продавщица возвращает красный паспорт с тёплой улыбкой: добро пожаловать. Алекс берёт паспорт, бумажка и сим-карта лежат на прилавке.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Всего четыре руки в кадре: у продавщицы одна держит паспорт, вторая лежит на прилавке; у Алекса одна берёт паспорт, вторая на лямке рюкзака; бумажка и сим-карта лежат на прилавке, а не в руках; широкая улыбка; ласточка одна, на подоконнике.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw the characters and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall with the clock, the shop window with the view of the street at 0-22% of the width, the shelf unit with blank phone boxes at 24-44%, the second shelf unit with blank SIM packs and the gold door frame at 72-96%, the long white counter (top edge at 62% of the height), the monitor at 22-30%, the card terminal at 52-58%, the SIM-pack display stand at 76-90%, the tiled floor, and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all 5 pictures of the story, so do not zoom, tilt or change the angle. Characters are drawn at the same scale as the stage: a character standing on the floor is about half of the image height, a character behind the counter is seen from the waist up with the face at about 38% of the image height.

ADD (frame 5 of 5):
- The saleswoman stands behind the counter at about 42% of the width, seen from the waist up, three-quarter view toward Aleks, head and body turned the same way. With the hand on the side facing Aleks she holds out the red passport across the counter at the height of the counter top; her other hand simply rests on the counter, relaxed. A big warm smile. She has exactly two hands.
- Aleks stands on the tiled floor in front of the counter at about 66% of the width, three-quarter view facing left toward her, head and body turned the same way. The hand on the side facing her reaches out to take the passport; his other hand holds the strap of his backpack. Nothing else is in his hands. A broad, happy smile. He has exactly two hands.
- On the counter between them, to the left of the passport hand-over, lie a blank slip of paper and one small blank SIM-card pack, flat on the wood.
- Both faces are inside the central 60% of the width. In the whole picture there are exactly four hands: two of the saleswoman and two of Aleks.

CHARACTERS: Aleks exactly as in image 2, the character sheet (short messy dark-blond hair, light stubble, faded blue denim jacket over a grey T-shirt, olive backpack worn on both shoulders, plain red passport with no lettering); the saleswoman exactly as in image 2 (black hair in a high bun, coral-red work vest over a white blouse, gold hoop earrings). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the sill of the shop window at the left (about 12% of the width), looking at the two of them. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, a third hand or a third arm on anyone (every person has exactly two arms and two hands, every visible hand belongs to a visible arm), objects held in more than two hands at once, mixed pixel sizes, smooth gradients, the saleswoman standing in front of the counter.
```

## Если что-то не получается

- **Фон поменялся даже в правке.** Добавь в начало промта: «This is an image edit. Output image 1 with the additions only; everything not mentioned stays pixel-identical.» Если есть маска, выдели только пол и место за прилавком. Иначе план Б.
- **Герои разные от кадра к кадру.** Лист героев должен быть второй картинкой, и в промт добавь: «Keep faces, hair and clothes identical to image 2.»
- **Продавец стоит перед прилавком или без нижней части.** Допиши: «The seller stands BEHIND the counter and is hidden from the waist down by the counter; the buyer stands IN FRONT of the counter on the floor.»
- **Мелкие приметы** (жемчуг и бантик Майи, значок на лямке, паспорт без герба) теряются: в пиксель-арте это 3–6 пикселей. Если исчезли, правь готовый кадр: «Edit this image. Add only ... Change nothing else, every other pixel stays identical.»
- **Руки и пальцы кривые** (карта, монеты, большой палец, паспорт). Опусти руку: «Redraw only that hand so that it simply rests on the counter. Change nothing else.»
- **Буквы и цифры на карте, купюре, упаковках, бумажке.** Допиши: «Every card, banknote, pack, paper and bubble is completely blank, no markings, no numbers, no letters, only simple pictograms». Если не помогло, замазать в редакторе.
- **Пиксели разного размера.** Могу написать скрипт, который уменьшит кадр до сетки 320×240 и вернёт увеличенным без сглаживания.
