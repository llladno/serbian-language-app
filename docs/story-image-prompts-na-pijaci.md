# История «Na pijaci»: промты кадров

Пятая и шестая истории идут с **новыми сценами**. «Na pijaci»: Майя из «Prvi dan», но **в свободный день** (без белого халата, платье, холщовая сумка, бордовая сумка с ремнём) и **Продавец** (полный мужчина за пятьдесят в зелёном фартуке, серой кепке, с густыми усами). Сцена: прилавок на зелёном рынке в Белграде, утро, один плейт. Блоки стиля, «без текста», композиции и ласточки лежат в `docs/story-image-style.md`, а герои и сцена в разделе «Истории уровня 2». Особенность этих сцен: **прилавок через весь кадр**, продавец стоит за ним (виден по пояс), покупатель на полу перед ним (во весь рост).

## Порядок работы

1. **Лист героев** (промт 0). Сохрани как `refs/sheet.png`. Проверь: приметы героев, ласточка одна.
2. **Плейт** (промт 1). Сделай 3–4 варианта и выбери самый чистый: все предметы на прилавке на своих местах, место за прилавком и пол пустые. Этот файл дальше не пересоздаётся. Сохрани как `refs/plate.png`.
3. **Кадры 1–5**: правка плейта. В генератор по порядку: 1) плейт, 2) лист героев, 3) `web/src/assets/invite-scene.webp` (если можно), и вставь промт кадра.
4. Сохрани кадры как `s1.png`…`s5.png` в `scripts/.cache/stories/na-pijaci/frames/` и положи 5 рядом: навес/стена, прилавок и все предметы на нём должны стоять на одних и тех же местах. Затем `cd /Users/grisha/plans/serbian-app && python3 scripts/story_publish.py na-pijaci --dev` (нужен `make dev`).

Если правка двигает фон, план Б: герои рисуются отдельно на однотонном фоне, а я накладываю их на плейт скриптом.

Говорят: **Продавец** (мужчина) и **Майя** (женщина). Майя здесь без халата, как в свободный день; лист героев можно взять из «Prvi dan» только как образец внешности, одежду меняй по блоку «Майя (в свободный день)».

## Промт 0. Лист героев

```
REFERENCES (if a reference image is attached): it is the app's style reference. Match its pixel size and its rich colour, and draw the swallow exactly like the swallows in it.

TASK: a character reference sheet (model sheet) for a pixel-art story, landscape 4:3, on a flat, plain cream background (#F7F9F7) with no scenery. Only a tiny soft oval shadow under each figure. Two rows with generous space between the figures. Top row, left to right: MAJA full-body front view, MAJA full-body three-quarter view, the STALL OWNER (PRODAVAC) full-body front view, the STALL OWNER full-body three-quarter view. Bottom row, left to right: the SWALLOW perched on a small twig seen from the side, the SWALLOW in flight with wings spread, a round white CHEESE WHEEL with one wedge cut out and a cheese knife, and a grey cardboard EGG TRAY with brown eggs. Every figure is drawn at the same scale and the same pixel size as it will be in the final scenes, neutral friendly expressions, the same outfit in every view. No labels, no arrows, no captions, no text.

STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

MAJA (on her day off, in casual clothes; the same young woman as in the hospital story, a Russian-speaking doctor in her mid-twenties who lives in Belgrade): chibi, about 3 heads tall. Long, straight, smooth light-brown hair with soft golden-blonde highlights (#B98A5E, lighter #D9B27F), worn down and falling past her shoulders on both sides, a soft parting near the centre and a few strands framing her face. Fair skin with warm rosy cheeks, thin light-brown eyebrows, small friendly eyes of a grey-blue-green colour (#7FA39F: grey with blue and green tones, not black, not brown) with a dark pupil pixel and one white highlight pixel, a small gentle closed-mouth smile. Tiny silver stud earrings, a short single-strand necklace of cream pearls and, hanging below it on a thin silver chain, a tiny sparkling bow-shaped pendant (a few light-blue and white pixels). She wears a cream-white dress with a sweetheart neckline, puffy sleeves and a small scattered floral print (tiny dusty-pink, brown and green flowers), and NO doctor's coat. The strap of a burgundy (#7A1F2B) leather shoulder bag crosses one shoulder, the bag hangs at her hip, and she also carries a plain natural-linen shopping tote (blank, no print) in one hand. Clean white sneakers.

PRODAVAC (the stall owner, a stocky, friendly man in his fifties, a market regular): chibi, about 3 heads tall, round and sturdy. Short grey-flecked dark hair under a flat grey cap, a thick grey-black moustache, ruddy cheeks, warm medium skin, thick bushy eyebrows, small kind black dot eyes with a white highlight pixel, a broad generous smile. A forest-green (#3F6B3A, shadow #2C4D29) canvas apron over a white-and-blue checked shirt with the sleeves rolled up, short thick forearms. When he stands behind the counter he is seen from the waist up.

SWALLOW (the UCIMO mascot): exactly one small swallow in the whole image, drawn exactly like the swallows in the style reference: deep teal head, back and wings (#176B68, darker wing tips #0B3F3D), a cream belly (#F7F9F7), a golden-yellow V-shaped bib at the throat (#E3B04B), a small orange-gold beak and orange-gold feet, a round black eye with a white highlight, a long forked tail. It is a real bird about the size of a pigeon (roughly one third of a character's height), with no clothes, no human arms and no human pose. There are no other birds anywhere in the image.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: extra people, scenery, props other than the twig, the cheese and the egg tray, mixed pixel sizes, smooth gradients, glossy plastic shine, anything that looks like writing.
```

## Промт 1. Плейт «утро»: пустая сцена

**Идея:** фон для всех кадров. Приложи `invite-scene.webp` как референс стиля.

```
REFERENCES (if a reference image is attached): it is the app's style reference. Match its pixel size and its rich colour. Do not draw any birds.

TASK: the empty stage: a clean, frontal background picture that will be reused as the fixed background for five more pictures, so it must be plain and legible. The camera is straight-on at standing eye level and will never change. Keep the back wall flat, frontal and uncluttered, keep every object clearly visible and separate from its neighbours, and leave the bottom quarter as open, empty floor for characters to stand on. Keep the stall bright and simple, and make the cheese wheels, the cash box and the egg trays easy to tell apart. There are no people, no animals and no birds anywhere.

STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.

SETTING (a fixed stage that never changes): one stall of a green market (pijaca) in Belgrade, Serbia, seen from ONE fixed camera position: straight-on, at the eye level of a standing person, 4:3. A striped teal-and-cream canvas awning with a scalloped edge runs across the whole top of the picture (0-100% of the width, the top 18% of the height). Behind the stall is a pale warm-cream plaster wall (#EFE3C8) with two simple wooden shelves holding blank glass jars and blank cloth-wrapped packages, and strings of red dried peppers hanging from the awning at 4-10% and 90-96% of the width. In front of the wall runs one long pale-wood counter from 6% to 94% of the width: its top edge is at 62% of the height and its plain-board front panel goes down to 76%. A band of warm grey paving stones with a little pixel texture covers the bottom quarter of the picture and is open and empty: nothing stands on the paving. On the counter, from left to right: (1) at 6-24% a stack of three round white cheese wheels on a wooden board, a big cheese knife leaning beside them; (2) at 50-58% a small blue-grey metal cash box with its lid closed; (3) at 74-90% two grey-cardboard egg trays with brown eggs, side by side, and a small stack of blank brown paper bags at 90-94%. The counter between the cheese wheels and the cash box (26-48% of the width) is bare, empty wood: there is NO scale and nothing standing on it, so that the stall owner's hands can rest on it. The space behind the counter at 34-48% of the width is empty (this is where the stall owner will stand; he is not in the stage). No people, no animals, no birds.

LIGHT: morning. Warm sunlight comes from the upper left and lays a soft golden glow on the cheese and the counter; the shade under the awning is cool violet-teal; the paving in front is bright in the sun; a few warm reflected highlights on the eggs.

COMPOSITION RULES (the picture is shown in an app, cropped on small screens and slowly zoomed in): keep the awning, the cheese wheels, the cash box and the egg trays fully inside the frame with about 7% of calm margin at every edge and keep the floor in the bottom quarter open and empty.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: people, animals, birds, objects standing on the open floor, mixed pixel sizes, smooth gradients, anything that looks like writing.
```

## Кадр 1

**Звучит:** 1–3: Izvolite, izvolite! Recite! / Dobar dan! Koliko košta sir? / Hiljadu dinara kilo. Domaći je, svež!

**Идея:** Утро на рынке. Продавец радушно зазывает Майю, показывая на сыр. Майя подошла с сумками и с интересом смотрит на головы сыра.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Продавец за прилавком (около 42 %), виден по пояс, улыбка во весь рот, одна рука открытой ладонью к сыру; Майя перед прилавком (около 68 %), лицом влево, лицо и тело в одну сторону; холщовая сумка в руке; пол перед прилавком пустой; ласточка одна.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw the characters and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the striped awning along the top, the cream wall with its shelves and hanging peppers, the long wooden counter (top edge at 62% of the height), the three cheese wheels at 6-24% of the width, the bare counter at 26-48%, the cash box at 50-58%, the egg trays and paper bags at 74-94%, the grey paving, and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all 5 pictures of the story, so do not zoom, tilt or change the angle. Characters are drawn at the same scale as the stage: a character standing on the floor is about half of the image height, a character behind the counter is seen from the waist up with the face at about 38% of the image height.

ADD (frame 1 of 5):
- The stall owner stands behind the counter at about 42% of the width, seen from the waist up, in three-quarter view facing right toward Maja, body and head turned the same way. He is welcoming her with a big warm smile and raised eyebrows; his hand on the side nearer to the viewer lies open, palm up, over the counter in a 'help yourself, look at this' gesture toward the cheese wheels, the other hand rests on the counter.
- Maja stands on the open paving in front of the counter at about 68% of the width, in three-quarter view facing left toward him, head and body turned the same way, a curious pleased smile. Her linen tote hangs from the hand on the side away from the viewer, the other hand holds the strap of her burgundy bag. Her counter-side body is in front of the counter panel.
- Both faces are inside the central 60% of the width. The paving around her feet is clear.

CHARACTERS: Maja exactly as in image 2, the character sheet (Maja on her day off: long straight light-brown hair worn down, grey-blue-green eyes, pearl necklace with a tiny bow pendant, NO white coat, cream floral dress, burgundy shoulder-bag strap, natural-linen tote bag); the stall owner exactly as in image 2 (stocky, grey cap, thick moustache, forest-green apron over a checked shirt). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the top of the cheese-wheel stack (about 14% of the width), looking at the two of them. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, a third hand or a third arm on anyone (every person has exactly two arms and two hands, every visible hand belongs to a visible arm), objects held in more than two hands at once, mixed pixel sizes, smooth gradients, the stall owner's lower body floating without a counter.
```

## Кадр 2

**Звучит:** 4–6: Evo, probajte! / Dobar je, dobar! / Dajte mi dvadeset deka, molim vas.

**Идея:** Дегустация: продавец протягивает на кончике ножа кусочек сыра, Майя пробует и довольна. Заказ идёт сразу за этим.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Продавец протягивает нож с маленьким кусочком (рука со стороны Майи), лицо ободряющее; Майя держит кусочек у рта, глаза широкие, счастливая улыбка; нож не направлен ей в лицо; ласточка одна, на доске рядом с сыром.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw the characters and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the striped awning along the top, the cream wall with its shelves and hanging peppers, the long wooden counter (top edge at 62% of the height), the three cheese wheels at 6-24% of the width, the bare counter at 26-48%, the cash box at 50-58%, the egg trays and paper bags at 74-94%, the grey paving, and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all 5 pictures of the story, so do not zoom, tilt or change the angle. Characters are drawn at the same scale as the stage: a character standing on the floor is about half of the image height, a character behind the counter is seen from the waist up with the face at about 38% of the image height.

ADD (frame 2 of 5):
- The stall owner stands behind the counter at about 42% of the width, seen from the waist up, three-quarter view toward Maja, body and head turned the same way. With the hand on the side facing Maja he holds out a small piece of white cheese on the flat of a knife blade, the blade pointing sideways along the counter (not at her face); an encouraging smile, eyebrows raised.
- Maja stands on the paving in front of the counter at about 66% of the width, three-quarter view facing left toward him, head and body turned the same way. She has just taken the piece and holds it near her mouth with the hand on the side facing him; her eyes are wide and bright, her smile is happy and surprised. The linen tote hangs from her other hand.
- Both faces are inside the central 60% of the width.

CHARACTERS: Maja exactly as in image 2, the character sheet (Maja on her day off: long straight light-brown hair worn down, grey-blue-green eyes, pearl necklace with a tiny bow pendant, NO white coat, cream floral dress, burgundy shoulder-bag strap, natural-linen tote bag); the stall owner exactly as in image 2 (stocky, grey cap, thick moustache, forest-green apron over a checked shirt). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the edge of the wooden board under the cheese wheels, next to the knife (about 22% of the width), looking at Maja. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, a third hand or a third arm on anyone (every person has exactly two arms and two hands, every visible hand belongs to a visible arm), objects held in more than two hands at once, mixed pixel sizes, smooth gradients, the stall owner's lower body floating without a counter.
```

## Кадр 3

**Звучит:** 7–10: Evo. Još nešto? / Imate li jaja? / Imam. Koliko komada? / Deset. To je sve.

**Идея:** Сыр уже отрезан и лежит в бумаге. Майя показывает на яйца, продавец поднимает лоток и кладёт десяток в бумажную коробку.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Свёрток сыра на столе рядом с весами (пустой бумажный); Майя показывает рукой, ближней к яйцам (вправо), лицом вправо, голова и тело в одну сторону; продавец держит лоток двумя руками, лицо вниз-вправо; видны яйца; ласточка одна, на столбе справа.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw the characters and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the striped awning along the top, the cream wall with its shelves and hanging peppers, the long wooden counter (top edge at 62% of the height), the three cheese wheels at 6-24% of the width, the bare counter at 26-48%, the cash box at 50-58%, the egg trays and paper bags at 74-94%, the grey paving, and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all 5 pictures of the story, so do not zoom, tilt or change the angle. Characters are drawn at the same scale as the stage: a character standing on the floor is about half of the image height, a character behind the counter is seen from the waist up with the face at about 38% of the image height.

ADD (frame 3 of 5):
- A neat wrapped parcel of cheese in blank white paper now lies on the bare counter at about 32%, to the right of the cheese wheels.
- The stall owner stands behind the counter at about 48% of the width, seen from the waist up, three-quarter view facing right toward the egg trays and Maja, head and body turned the same way. His two hands lift one grey egg tray from the stack and he is placing brown eggs into a blank paper box on the counter; a cheerful, helpful face.
- Maja stands on the paving in front of the counter at about 66% of the width, three-quarter view facing right toward the egg trays, head and body turned the same way. She points at the eggs with the hand on the side nearer to the egg trays, the other hand holds her tote; a bright, interested face.
- Both faces are inside the central 60% of the width.

CHARACTERS: Maja exactly as in image 2, the character sheet (Maja on her day off: long straight light-brown hair worn down, grey-blue-green eyes, pearl necklace with a tiny bow pendant, NO white coat, cream floral dress, burgundy shoulder-bag strap, natural-linen tote bag); the stall owner exactly as in image 2 (stocky, grey cap, thick moustache, forest-green apron over a checked shirt). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the right awning pole at the edge of the awning (about 90% of the width), looking at the eggs. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, a third hand or a third arm on anyone (every person has exactly two arms and two hands, every visible hand belongs to a visible arm), objects held in more than two hands at once, mixed pixel sizes, smooth gradients, the stall owner's lower body floating without a counter.
```

## Кадр 4

**Звучит:** 11–13: Evo. Trista pedeset dinara. / Može karticom? / Ne, samo keš!

**Идея:** Майя с надеждой показывает банковскую карту. Продавец, смеясь, качает головой и показывает на металлическую коробку с деньгами: только наличные.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Майя держит пустую бирюзовую карту на уровне груди ладонью вверх, надежда на лице; продавец качает головой (лицо повёрнуто к ней), добродушный смех, рука ладонью вверх на кассу; никаких цифр и букв на карте; ласточка одна, на кассе.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw the characters and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the striped awning along the top, the cream wall with its shelves and hanging peppers, the long wooden counter (top edge at 62% of the height), the three cheese wheels at 6-24% of the width, the bare counter at 26-48%, the cash box at 50-58%, the egg trays and paper bags at 74-94%, the grey paving, and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all 5 pictures of the story, so do not zoom, tilt or change the angle. Characters are drawn at the same scale as the stage: a character standing on the floor is about half of the image height, a character behind the counter is seen from the waist up with the face at about 38% of the image height.

ADD (frame 4 of 5):
- The stall owner stands behind the counter at about 46% of the width, seen from the waist up, three-quarter view toward Maja, head and body turned the same way. He laughs good-naturedly and shakes his head 'no': his eyes are crinkled, his mouth is a wide smile; the hand on the side nearer to the viewer is raised open, palm facing Maja at chest height in a friendly 'stop' sign, the other hand rests next to the closed cash box on the counter.
- Maja stands on the paving in front of the counter at about 66% of the width, three-quarter view facing left toward him, head and body turned the same way. A hopeful, slightly awkward smile; with the hand on the side facing him she holds up a small plain teal bank card (completely blank, no numbers, no logo) at chest height, her other hand holds the tote.
- Both faces are inside the central 60% of the width.

CHARACTERS: Maja exactly as in image 2, the character sheet (Maja on her day off: long straight light-brown hair worn down, grey-blue-green eyes, pearl necklace with a tiny bow pendant, NO white coat, cream floral dress, burgundy shoulder-bag strap, natural-linen tote bag); the stall owner exactly as in image 2 (stocky, grey cap, thick moustache, forest-green apron over a checked shirt). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the closed metal cash box (about 54% of the width), looking at the card. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, a third hand or a third arm on anyone (every person has exactly two arms and two hands, every visible hand belongs to a visible arm), objects held in more than two hands at once, mixed pixel sizes, smooth gradients, the stall owner's lower body floating without a counter.
```

## Кадр 5

**Звучит:** 14–18: Imam samo hiljadu. / Hiljadu? Imate li sitno? / Samo malo… Imam! Evo, tačno! / Tačno, hvala! Doviđenja! / Doviđenja!

**Идея:** Майя нашла в сумке мелочь и торжествующе показывает горсть монет; на прилавке лежит большая купюра, свёрток сыра и коробка с яйцами. Продавец довольно показывает большой палец.

**Основа:** плейт (первая картинка), потом лист героев.

**Проверить:** Майя держит над прилавком горсть монет (золотистые пиксели) на ладони, торжествующая улыбка, голова и тело в одну сторону; на прилавке одна пустая бледно-голубая купюра без цифр (рисунок вместо надписи), свёрток сыра и коробка яиц; продавец с большим пальцем и широкой улыбкой; ласточка одна, на навесе.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw the characters and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the striped awning along the top, the cream wall with its shelves and hanging peppers, the long wooden counter (top edge at 62% of the height), the three cheese wheels at 6-24% of the width, the bare counter at 26-48%, the cash box at 50-58%, the egg trays and paper bags at 74-94%, the grey paving, and the light. The output has the same size and aspect ratio as image 1, and every background object stays at the same position and the same size. The camera is the same in all 5 pictures of the story, so do not zoom, tilt or change the angle. Characters are drawn at the same scale as the stage: a character standing on the floor is about half of the image height, a character behind the counter is seen from the waist up with the face at about 38% of the image height.

ADD (frame 5 of 5):
- On the counter, in front of the stall owner at about 52% of the width, lie one blank pale-blue banknote (no numbers, no letters, only a simple pictogram of a bridge), the white parcel of cheese and the paper box of eggs.
- The stall owner stands behind the counter at about 44% of the width, seen from the waist up, three-quarter view toward Maja, head and body turned the same way. He is delighted: a broad smile, eyebrows up, the hand on the side nearer to the viewer gives a clear thumbs-up held at chest height, the other hand rests on the cash box.
- Maja stands on the paving in front of the counter at about 66% of the width, three-quarter view facing left toward him, head and body turned the same way. She triumphantly holds up a little handful of gold-coloured coins on her open palm (the hand on the side facing him, held at chest height above the counter edge), a big proud smile; her burgundy bag is open at her hip, the tote hangs from her other hand.
- Both faces are inside the central 60% of the width.

CHARACTERS: Maja exactly as in image 2, the character sheet (Maja on her day off: long straight light-brown hair worn down, grey-blue-green eyes, pearl necklace with a tiny bow pendant, NO white coat, cream floral dress, burgundy shoulder-bag strap, natural-linen tote bag); the stall owner exactly as in image 2 (stocky, grey cap, thick moustache, forest-green apron over a checked shirt). Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the scalloped edge of the awning above the two of them (about 56% of the width), looking down at the coins. The swallow is added in this picture only, it is not part of the background.

The light is the light of image 1, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, a third hand or a third arm on anyone (every person has exactly two arms and two hands, every visible hand belongs to a visible arm), objects held in more than two hands at once, mixed pixel sizes, smooth gradients, the stall owner's lower body floating without a counter.
```

## Если что-то не получается

- **Фон поменялся даже в правке.** Добавь в начало промта: «This is an image edit. Output image 1 with the additions only; everything not mentioned stays pixel-identical.» Если есть маска, выдели только пол и место за прилавком. Иначе план Б.
- **Герои разные от кадра к кадру.** Лист героев должен быть второй картинкой, и в промт добавь: «Keep faces, hair and clothes identical to image 2.»
- **Продавец стоит перед прилавком или без нижней части.** Допиши: «The seller stands BEHIND the counter and is hidden from the waist down by the counter; the buyer stands IN FRONT of the counter on the floor.»
- **Мелкие приметы** (жемчуг и бантик Майи, значок на лямке, паспорт без герба) теряются: в пиксель-арте это 3–6 пикселей. Если исчезли, правь готовый кадр: «Edit this image. Add only ... Change nothing else, every other pixel stays identical.»
- **Руки и пальцы кривые** (карта, монеты, большой палец, паспорт). Опусти руку: «Redraw only that hand so that it simply rests on the counter. Change nothing else.»
- **Буквы и цифры на карте, купюре, упаковках, бумажке.** Допиши: «Every card, banknote, pack, paper and bubble is completely blank, no markings, no numbers, no letters, only simple pictograms». Если не помогло, замазать в редакторе.
- **Пиксели разного размера.** Могу написать скрипт, который уменьшит кадр до сетки 320×240 и вернёт увеличенным без сглаживания.
