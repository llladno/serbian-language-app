# История «Prvi dan»: промты кадров

Вторая история идёт с **новыми героями**: **Майя** и **Милош** (врачи в белых халатах) выглядят как ребята с присланного фото: у Майи длинные прямые светло-русые волосы, серёжки-гвоздики, жемчужная нить с цепочкой и бантиком, цветочное платье и бордовая сумка; у Милоша короткие тёмно-русые растрёпанные волосы с чёлкой набок, белая футболка. Пиксель-арт передаёт причёску, одежду и украшения, а лицо только условно. Ана и Марко остаются героями первой истории.

Метод тот же, что у первой: **одна камера на все кадры**, **кадр это правка готового плейта**, всё стоит у стены, позиции в процентах ширины. Отличия:

- **Сцена:** коридор больницы в Белграде, одно утро. Вечернего плейта нет: плейт один.
- **Пять кадров** (пять сцен в спеке), на каждый кадр один читаемый жест, без вывернутых шей и поднятых над головой рук.
- **Ласточка** в каждом кадре одна, на новом месте, в плейте её нет.
- Блоки стиля, героев, сцены и света лежат в `docs/story-image-style.md` (раздел «История «Prvi dan»: герои и сцена»), промты собраны скриптом из них дословно.

## Порядок работы

1. **Лист героев** (промт 0): Майя, Милош, ласточка, папка. Сохрани как `refs/sheet.png`. Проверь сходство с фото: волосы, жемчуг с бантиком, цветочное платье, белый халат, сумка на ремне.
2. **Плейт «утро»** (промт 1). Сделай 3–4 варианта и выбери самый чистый: окно слева с видом на город, стулья, доска, растение, кулер, двустворчатая дверь справа, пол внизу пустой. Этот файл дальше не пересоздаётся. Сохрани как `refs/plate.png`.
3. **Кадры 1–5**: правка плейта. В генератор по порядку: 1) плейт, 2) лист героев, 3) `web/src/assets/invite-scene.webp` (если можно), и вставь промт кадра.
4. Сохрани кадры как `s1.png`…`s5.png` в `scripts/.cache/stories/prvi-dan/frames/` и положи пять рядом: окно, стулья, доска, растение, кулер и дверь должны стоять на одних и тех же местах. Затем `cd /Users/grisha/plans/serbian-app && python3 scripts/story_publish.py prvi-dan --dev` (нужен `make dev`).

Если правка двигает фон, план Б: герои рисуются отдельно на однотонном фоне, а я накладываю их на плейт скриптом.

## Промт 0. Лист героев

```
REFERENCES (if a reference image is attached): it is the app's style reference. Match its pixel size and its rich colour, and draw the swallow exactly like the swallows in it.

TASK: a character reference sheet (model sheet) for a pixel-art story, landscape 4:3, on a flat, plain cream background (#F7F9F7) with no scenery. Only a tiny soft oval shadow under each figure. Two rows with generous space between the figures. Top row, left to right: MAJA full-body front view, MAJA full-body three-quarter view, MILOŠ full-body three-quarter view, MILOŠ full-body front view. Bottom row, left to right: the SWALLOW perched on a small twig seen from the side, the SWALLOW in flight with wings spread, and a blank pale-blue PATIENT FOLDER seen from the front. Every figure is drawn at the same scale and the same pixel size as it will be in the final scenes, neutral friendly expressions, the same outfit in every view. No labels, no arrows, no captions, no text.

STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

MAJA (a Russian-speaking young woman doctor in her mid-twenties, new at a hospital in Belgrade): chibi, about 3 heads tall. Long, straight, smooth light-brown hair with soft golden-blonde highlights (#B98A5E, lighter #D9B27F), worn down and falling past her shoulders on both sides, a soft parting near the centre and a few strands framing her face. Fair skin with warm rosy cheeks, thin light-brown eyebrows, small friendly black dot eyes with a white highlight pixel, a small gentle closed-mouth smile. Tiny silver stud earrings, a short single-strand necklace of cream pearls and, hanging below it on a thin silver chain, a tiny sparkling bow-shaped pendant (a few light-blue and white pixels). She wears an open white doctor's coat (#F4F6F5, shadow #C9D3D3) over a cream-white dress with a sweetheart neckline, puffy sleeves and a small scattered floral print (tiny dusty-pink, brown and green flowers); the dress and the necklace are clearly visible at the front. The strap of a burgundy (#7A1F2B) leather shoulder bag crosses one shoulder, the bag hangs at her hip. Clean white sneakers.

MILOŠ (a young man doctor in his mid-twenties, Maja's new colleague): chibi, about 3 heads tall, a little taller than Maja. Short, thick, tousled dark-brown hair (#4A3326, lighter #6B4B38) with a longer fringe swept to one side and falling over his forehead, no beard, clean-shaven. Fair skin with a warm tint, thick dark-brown eyebrows, small calm black dot eyes with a white highlight pixel, a relaxed closed-mouth half smile. He wears an open white doctor's coat (#F4F6F5, shadow #C9D3D3) over a plain white crew-neck T-shirt with a faint small red-pink print on one sleeve, dark navy trousers and clean white sneakers, and a deep-teal (#176B68) stethoscope around his neck.

SWALLOW (the UCIMO mascot): exactly one small swallow in the whole image, drawn exactly like the swallows in the style reference: deep teal head, back and wings (#176B68, darker wing tips #0B3F3D), a cream belly (#F7F9F7), a golden-yellow V-shaped bib at the throat (#E3B04B), a small orange-gold beak and orange-gold feet, a round black eye with a white highlight, a long forked tail. It is a real bird about the size of a pigeon (roughly one third of a character's height), with no clothes, no human arms and no human pose. There are no other birds anywhere in the image.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: extra people, scenery, props other than the twig and the folder, mixed pixel sizes, smooth gradients, glossy plastic shine, anything that looks like writing.
```

## Промт 1. Плейт «утро»: пустая сцена

**Идея:** фон для всех кадров. Приложи `invite-scene.webp` как референс стиля.

```
REFERENCES (if a reference image is attached): it is the app's style reference. Match its pixel size and its rich colour. Do not draw any birds.

TASK: the empty stage: a clean, frontal background picture that will be reused as the fixed background for five more pictures, so it must be plain and legible. The camera is straight-on at standing eye level and will never change. Keep the wall flat, frontal and uncluttered, keep every object clearly visible against the wall, and leave the bottom quarter as open, empty linoleum floor for characters to stand on. There are no people, no animals, no birds, no patients, no beds and no wheelchairs anywhere.

STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

SETTING (a fixed stage that never changes): a corridor (hodnik) of a public hospital in Belgrade, Serbia, seen from ONE fixed camera position: straight-on, at the eye level of a standing person, 4:3. The picture is dominated by one long back wall that runs from the left edge to the right edge of the frame: a soft sage-mint (#9CC7BE, slightly greyed) painted wall, a slightly darker mint lower half, a pale cream upper band, and a thin pale-wood handrail along the wall at hip height. A band of warm grey-beige linoleum floor (not cobblestones) with a little pixel texture covers the bottom quarter of the picture. Nothing stands in the middle of the open floor: every object stands against the wall. From left to right along the wall: (1) at 0-22% of the width a tall window with a pale-wood frame and a white windowsill, open onto a sunny view of Belgrade: rooftops, treetops, and on the horizon the big white dome of a large Orthodox church with a small cross on top, under a light blue sky; (2) at 24-36% a row of three blue-grey plastic waiting chairs on one shared frame, standing flat against the wall; (3) at 38-52% a cork notice board with a few pinned blank coloured papers (pictograms only, no writing), and at 54-60% a tall potted green plant; (4) at 62-68% a water cooler with a clear blue bottle; (5) at 72-96% a pair of white double doors with frosted-glass panels in a mustard-gold (#E3B04B) door frame, a small round green-cross pictogram above them, and a blank hand-sanitiser dispenser on the wall beside the door. Along the very top edge: a plain ceiling with a strip of soft white lights. The linoleum floor in front is open and empty.

LIGHT: morning. Warm sunlight comes in through the window at the left and lays a soft golden patch on the wall and the floor to the right of the window; shadows are short, cool violet-teal; the corridor lights are a clean, slightly cool white. Sky seen through the window: clear light blue with one thin pale cloud.

COMPOSITION RULES (the picture is shown in an app, cropped on small screens and slowly zoomed in): keep the window, the waiting chairs, the notice board, the plant, the water cooler and the double doors fully inside the frame with about 7% of calm margin at every edge, and keep the linoleum floor in the bottom quarter open and empty.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: people, animals, birds, patients, beds, wheelchairs, objects standing in the middle of the floor, mixed pixel sizes, smooth gradients, anything that looks like writing.
```

## Кадр 1. Доброе утро! Знакомство

**Звучит:** 1–4: Dobro jutro! Izvolite? / Dobro jutro! Ja sam Maja. Radim kao lekar. / Aha! Dobro došla! Ja sam Miloš. Takođe lekar. / Drago mi je. Danas je prvi dan.

**Идея:** Первое утро Майи. Милош встречает новенькую у окна и протягивает руку, Майя только что пришла с сумкой через плечо. Рукопожатие в центре кадра.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Милош слева (около 34%), Майя справа (около 62%), руки встречаются в центре (его рука идёт слева, её справа, каждая из своего плеча); лица смотрят друг на друга; сумка бордовая на плече Майи; жемчуг и бантик заметны; пол между ними чистый; ласточка одна, на подоконнике.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Maja, Miloš and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall and its handrail, the window with the view of Belgrade at 0-22% of the width, the row of waiting chairs at 24-36%, the notice board at 38-52%, the potted plant at 54-60%, the water cooler at 62-68%, the double doors at 72-96%, the linoleum floor, the ceiling lights and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all five pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 1 of 5):
- Miloš stands on the open floor at about 34% of the width, in three-quarter view facing Maja (to the right), his body and head both turned toward her. He is greeting her with a friendly open smile and raised eyebrows, the arm on the side facing Maja stretched out toward her with an open hand for a handshake (this is his own right arm, the one nearer to the viewer), his other hand relaxed at his side.
- Maja stands at about 62% of the width, in three-quarter view facing Miloš (to the left), her head facing the same way as her body, a shy happy smile. She has just arrived: the strap of the burgundy bag crosses her shoulder, and she is meeting his outstretched hand with the hand on the side facing him (this is her own right arm, the one farther from the viewer, reaching to the left of the picture). A normal right-hand handshake: his hand comes in from the left, hers from the right, they clasp in the middle, and each hand belongs to the arm it comes out of, no swapped or crossed arms. The handshake is in the middle of the picture, at about 48% of the width and about 55% of the height.
- The floor between and around them is clear and empty.
- Both faces are at about 40% of the image height.

CHARACTERS: Maja and Miloš exactly as in image 2, the character sheet (Maja: long straight light-brown hair worn down, pearl necklace with a tiny bow pendant, white coat over a cream floral dress, burgundy shoulder-bag strap; Miloš: short tousled dark-brown hair with a side-swept fringe, no beard, white coat over a white T-shirt, teal stethoscope). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the white windowsill of the window at the left (about 12% of the width), looking toward the two of them. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, swapped or crossed arms, two hands of one person in the handshake, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Кадр 2. Первый день! Как идёт?

**Звучит:** 5–6: Prvi dan! Kako ide? / Dobro. Malo sam umorna, ali ne žalim se!

**Идея:** Милош участливо спрашивает, как идёт первый день. Майя устала, но улыбается и слегка пожимает плечами: не жалуется.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Милош с открытой ладонью, брови подняты; Майя с лёгким пожатием плеч и яркой улыбкой, одна рука держит лямку сумки; ласточка одна, на поручне между ними.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Maja, Miloš and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall and its handrail, the window with the view of Belgrade at 0-22% of the width, the row of waiting chairs at 24-36%, the notice board at 38-52%, the potted plant at 54-60%, the water cooler at 62-68%, the double doors at 72-96%, the linoleum floor, the ceiling lights and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all five pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 2 of 5):
- Miloš stands at about 36% of the width, three-quarter view toward Maja, body and head turned the same way. He asks kindly how it is going: his hand nearer to Maja is open at chest height with the palm up, his eyebrows are slightly raised, a warm interested smile.
- Maja stands at about 62% of the width, three-quarter view toward Miloš, head and body turned the same way. She is a little tired but bright: her shoulders are slightly lowered in a small shrug, one hand holds the strap of her burgundy bag, her smile is wide and cheerful, her eyelids are a little heavy.
- A clear gap of empty floor between them. Both faces are at about 40% of the image height.

CHARACTERS: Maja and Miloš exactly as in image 2, the character sheet (Maja: long straight light-brown hair worn down, pearl necklace with a tiny bow pendant, white coat over a cream floral dress, burgundy shoulder-bag strap; Miloš: short tousled dark-brown hair with a side-swept fringe, no beard, white coat over a white T-shirt, teal stethoscope). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the pale-wood handrail in the gap between the two of them (about 48% of the width), looking at Maja. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Кадр 3. Пациент из Англии

**Звучит:** 7–9: Imamo pacijenta iz Engleske. On ne govori srpski, a ja govorim malo engleski. / Nema problema. Ja govorim engleski. / Hvala! Dobro govoriš i srpski!

**Идея:** Милош озабочен: пациент из Англии не говорит по-сербски, а он почти не говорит по-английски. Майя спокойно показывает «ок»: проблем нет.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Милош с пустой голубой папкой у груди, брови домиком, свободная рука ладонью вверх; над ним облачко-мысль с красным двухэтажным автобусом и чашкой чая, без флагов и букв; Майя уверенно показывает большой палец на уровне груди; ласточка одна, на верхнем крае доски объявлений.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Maja, Miloš and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall and its handrail, the window with the view of Belgrade at 0-22% of the width, the row of waiting chairs at 24-36%, the notice board at 38-52%, the potted plant at 54-60%, the water cooler at 62-68%, the double doors at 72-96%, the linoleum floor, the ceiling lights and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all five pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 3 of 5):
- Miloš stands at about 36% of the width, three-quarter view toward Maja, body and head turned the same way. He holds a blank pale-blue patient folder against his chest with the arm on the side away from Maja; the hand on the side facing Maja is raised palm up at chest height in a small helpless 'what now' gesture; his eyebrows are slanted up in the middle, a worried but kind face.
- Above Miloš floats a pixel thought bubble (a cloud with two small circles leading down to his head) with a tiny picture inside: a red double-decker bus and a small white teacup, drawn simply, with no flags and no text.
- Maja stands at about 62% of the width, three-quarter view toward Miloš, head and body turned the same way. She gives a clear, confident thumbs-up with the hand on the side facing Miloš, held at chest height (not raised high), a calm reassuring smile, her other hand on the strap of her burgundy bag.
- Both faces are at about 40% of the image height.

CHARACTERS: Maja and Miloš exactly as in image 2, the character sheet (Maja: long straight light-brown hair worn down, pearl necklace with a tiny bow pendant, white coat over a cream floral dress, burgundy shoulder-bag strap; Miloš: short tousled dark-brown hair with a side-swept fringe, no beard, white coat over a white T-shirt, teal stethoscope). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the top edge of the cork notice board (about 45% of the width), looking down at Miloš. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Кадр 4. Откуда ты? Браво!

**Звучит:** 10–13: Ne znam dobro srpski. / Ali govoriš! Odakle si? / Iz Rusije. Ali sada živim u Beogradu. / Ruski, engleski, srpski. Bravo!

**Идея:** Милоша впечатлил её сербский. Майя скромничает, а Милош над собой «считает» три языка: русский, английский, сербский.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Майя скромно, ладонь на груди, лёгкий наклон головы; Милош с поднятыми бровями и широкой улыбкой; над ним три пузыря-пиктограммы в ряд (луковичный купол, красный автобус, белый купол с крестом), без флагов и букв; ласточка одна, на верхней рамке окна.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Maja, Miloš and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall and its handrail, the window with the view of Belgrade at 0-22% of the width, the row of waiting chairs at 24-36%, the notice board at 38-52%, the potted plant at 54-60%, the water cooler at 62-68%, the double doors at 72-96%, the linoleum floor, the ceiling lights and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all five pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 4 of 5):
- Maja stands at about 62% of the width, three-quarter view toward Miloš, head and body turned the same way. She is modest and a little embarrassed: her hand on the viewer's side of her body lies flat on her own chest, her head is tilted slightly, her smile is shy and pleased, her eyes look at Miloš.
- Miloš stands at about 36% of the width, three-quarter view toward Maja, body and head turned the same way. He is impressed: eyebrows high, a wide open smile, one hand in the pocket of his coat, the other relaxed at his side.
- Above Miloš, in a row, float three small round speech-balloon pictograms, each with a tiny simple picture inside and no text: a colourful onion-domed cathedral, a red double-decker bus, a big white dome with a small cross. No flags, no letters.
- Both faces are at about 40% of the image height.

CHARACTERS: Maja and Miloš exactly as in image 2, the character sheet (Maja: long straight light-brown hair worn down, pearl necklace with a tiny bow pendant, white coat over a cream floral dress, burgundy shoulder-bag strap; Miloš: short tousled dark-brown hair with a side-swept fringe, no beard, white coat over a white T-shirt, teal stethoscope). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the top of the window frame at the left (about 11% of the width), looking at the speech balloons. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Кадр 5. Идём!

**Звучит:** 14–16: Hvala! Pacijent je ovde? / Ovde je. Idemo? / Idemo!

**Идея:** Майя поблагодарила, спросила, здесь ли пациент. Милош показывает на дверь: идём. Оба смотрят на двери справа.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Оба смотрят направо, на двустворчатую дверь; Милош открытой ладонью показывает на дверь, Майя решительно улыбается; голова и тело каждого повёрнуты в одну сторону; дверь закрыта; ласточка одна, над дверью.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Maja, Miloš and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall and its handrail, the window with the view of Belgrade at 0-22% of the width, the row of waiting chairs at 24-36%, the notice board at 38-52%, the potted plant at 54-60%, the water cooler at 62-68%, the double doors at 72-96%, the linoleum floor, the ceiling lights and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all five pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 5 of 5):
- Miloš stands at about 46% of the width, in three-quarter view facing right toward the double doors, his body and head both turned to the right. His arm nearer to the viewer is stretched toward the doors with an open hand, palm up, fingers together, a clear 'this way' gesture; he is smiling at the doors.
- Maja stands at about 62% of the width, a little behind and beside him, also in three-quarter view facing right, her head facing the same way as her body, looking at the doors with a determined, cheerful smile, one hand on the strap of her burgundy bag, the other relaxed at her side.
- The double doors stay closed. Both faces are at about 40% of the image height, and there is clear empty floor ahead of them.

CHARACTERS: Maja and Miloš exactly as in image 2, the character sheet (Maja: long straight light-brown hair worn down, pearl necklace with a tiny bow pendant, white coat over a cream floral dress, burgundy shoulder-bag strap; Miloš: short tousled dark-brown hair with a side-swept fringe, no beard, white coat over a white T-shirt, teal stethoscope). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on top of the mustard-gold door frame, just beside the green-cross pictogram (about 84% of the width), facing the two of them. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Если что-то не получается

- **Фон поменялся даже в правке.** Добавь в начало промта: «This is an image edit. Output image 1 with the additions only; everything not mentioned stays pixel-identical.» Если у генератора есть маска, выдели только пол, куда ставятся герои. Иначе план Б.
- **Герои разные от кадра к кадру.** Лист героев должен быть второй картинкой, и в промт добавь: «Keep Maja's and Miloš's faces, hair and clothes identical to image 2.»
- **Теряются мелкие приметы** (бантик на цепочке, жемчуг, принт платья). В пиксель-арте это 3–6 пикселей: жемчуг просто светлая дуга на шее, бантик светлый блик. Если исчезли совсем, правь готовый кадр: «Edit this image. Add only a short cream pearl necklace and a tiny sparkling bow pendant on Maja's neck. Change nothing else, every other pixel stays identical.»
- **Руки перепутаны (рукопожатие, жесты).** В промтах руки заданы стороной кадра («рука со стороны собеседника», «ближняя к зрителю»), а не «правая/левая»: «правая» генератор читает то по герою, то по картинке. Если в кадре 1 всё равно вышли руки наоборот, добавь второй строкой: «Miloš\'s hand enters the handshake from the left edge of the picture, Maja\'s from the right, both arms are attached to their own bodies, no arm crosses the other person\'s body.»
- **Голова или шея вывернута.** Не перегенерируй весь кадр: кадр первой картинкой, лист героев второй, «Redraw only that head so that it sits upright and faces the same way as the body, with the same hair as in image 2. Change nothing else, every other pixel stays identical.» Не помогло дважды: упрости позу (руки в карманах халата).
- **Рука или большой палец вышли кривыми** (кадр 3 и кадр 5). Опусти руку: «Redraw only Maja's raised hand so that it simply rests on the strap of her bag. Change nothing else.»
- **Буквы на папке, доске, табличке.** Допиши: «Every paper, folder, sign and bubble is completely blank, no markings at all». Если не помогло, замазать в редакторе.
- **Пиксели разного размера.** Могу написать скрипт, который уменьшит кадр до сетки 320×240 и вернёт увеличенным без сглаживания.
