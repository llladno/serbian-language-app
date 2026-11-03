# История «Dobro veče»: промты кадров

Эта история идёт **в том же дворе и в тот же вечер, что и «Drago mi je»**, с теми же героями (Ана и Марко): значит, нужны **готовые материалы первой истории**, новых лист героев и фон рисовать не надо. Кадр это правка готового вечернего фона: один и тот же фон и одна камера во всех четырёх кадрах (подробно о методе в `docs/story-image-style.md`).

## Порядок работы

1. Возьми **лист героев** (Ана, Марко, ласточка, велосипед) и **плейт «вечер»** из первой истории. Если файлов нет, их промты 0, 1 и 2 лежат в `docs/story-image-prompts-drago-mi-je.md`: плейт «вечер» это правка плейта «день» (промт 2).
2. Для каждого из четырёх кадров загрузи по порядку: **1)** плейт «вечер», **2)** лист героев, **3)** `web/src/assets/invite-scene.webp` (если можно), и вставь промт. Все четыре кадра идут от плейта «вечер» (свет уже вечерний).
3. Сохрани кадры как `s1.png`…`s4.png` в `scripts/.cache/stories/dobro-vece/frames/` и положи четыре рядом с кадром 4 первой истории: арка, скамья, дерево и дверь должны стоять на тех же местах.
4. Скажи мне, что кадры лежат (или пришли их сюда): подключу и опубликую.

Если правка двигает фон или «уезжает» скамья, смотри раздел «Если что-то не получается» в файле первой истории (и план Б с отдельными слоями героев).

**Руки и стороны.** Во всех кадрах руки описаны стороной кадра («рука со стороны Марко»), а не «правая/левая»: «правую» генератор читает то по герою, то по картинке, и руки в рукопожатии выходили наоборот. Кадр 4 повторяет композицию кадра 4 первой истории (Марко уходит в арку спиной, машет одна Ана) нарочно: это самое надёжное прощание.

## Кадр 1. Dobro veče!

**Звучит:** 1–2: Dobro veče! / Dobro veče! Kako ste?

**Идея:** Вечер. Ана только вышла из подъезда и остановилась на верхней ступени, Марко ведёт велосипед по двору и приветственно машет ей. Они только что заметили друг друга.

**Основа:** плейт «вечер» (первая картинка), потом лист героев.

**Проверить:** Марко слева на булыжнике (около 30%) ведёт велосипед, поднятая рука приветствует; Ана справа на верхней ступени (около 74%), руки в карманах пальто; между ними пустой пол; скамья пустая у стены; вечерний свет, окна жёлтые; ласточка одна, на верёвке с бельём.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Ana, Marko, the bicycle and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall, the arch, the linden tree, the bench against the wall at 32-60% of the width, the entrance with its steps and door, the balcony, the windows, the clothesline, the cobblestones, the sky and the light. The output has the same size and aspect ratio as image 1, every background object stays at the same position, and the bench is never moved. The camera is the same in all four pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 1 of 4, evening, 'good evening!'):
- Marko is on the open cobblestones on the left (about 30% of the width, in front of the arched passage and the linden), walking his deep-teal bicycle with one hand on the handlebar. His body is in three-quarter view facing Ana (to the right), his head facing the same way as his body. His other arm is raised in a big friendly wave, his mouth is open in a cheerful hello. A blank paper bakery bag with a bread roll sticks out of the bicycle's basket.
- Ana stands on the top step in front of the door at about 74% of the width, in three-quarter view facing Marko (to the left), her head facing the same way as her body, both hands in the pockets of her cream coat. She smiles politely and warmly, her lips a little parted as if she is about to ask something. The door behind her stays closed.
- There is clear empty floor between the two characters, so it reads as two neighbours who have just noticed each other. The bench stays empty.
- Both faces are at about 40% of the image height.

CHARACTERS: Ana and Marko exactly as in image 2, the character sheet (Ana: chestnut braid over her left shoulder, deep teal scarf, cream coat; Marko: black curls, dark beard, round glasses, rust-orange hoodie), and Marko's deep-teal bicycle with a wicker basket exactly as in image 2. Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the clothesline above the bench (about 50% of the width), a small silhouette against the evening sky, looking down at the two of them. The swallow is added in this picture only, it is not part of the background.

The background already has its evening light, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, a second bicycle, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Кадр 2. Как вы? Нормально

**Звучит:** 3–4: Dobro, hvala. A vi? / Može. Ne žalim se.

**Идея:** Вежливая беседа во дворе: оба на булыжнике ближе к центру лицом друг к другу, Марко тепло отвечает и спрашивает в ответ, Ана слегка пожимает плечами и улыбается («нормально, не жалуюсь»).

**Основа:** плейт «вечер» (первая картинка), потом лист героев.

**Проверить:** Марко около 38% лицом вправо, ладонь открыта к Ане, велосипед у его левой руки; Ана около 62% лицом влево, плечи слегка подняты, руки в карманах, лёгкая улыбка; оба стоят на булыжнике, скамья за ними пустая; ласточка одна, на перилах балкона.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Ana, Marko, the bicycle and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall, the arch, the linden tree, the bench against the wall at 32-60% of the width, the entrance with its steps and door, the balcony, the windows, the clothesline, the cobblestones, the sky and the light. The output has the same size and aspect ratio as image 1, every background object stays at the same position, and the bench is never moved. The camera is the same in all four pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 2 of 4, evening, a polite chat):
- Marko stands on the open cobblestones at about 38% of the width, in three-quarter view facing Ana (to the right), his head facing the same way as his body. He holds his deep-teal bicycle upright with one hand on the handlebar, the other hand open with the palm up toward Ana, as if asking back. An open, friendly smile.
- Ana stands on the cobblestones at about 62% of the width, in three-quarter view facing Marko (to the left), her head facing the same way as her body, both hands in the pockets of her cream coat. Her shoulders are slightly raised in a small calm shrug, a gentle smile: 'so-so, no complaints'.
- There is a clear distance of about one body width between them. The bench stays empty behind them, against the wall.
- Both faces are at about 40% of the image height.

CHARACTERS: Ana and Marko exactly as in image 2, the character sheet (Ana: chestnut braid over her left shoulder, deep teal scarf, cream coat; Marko: black curls, dark beard, round glasses, rust-orange hoodie), and Marko's deep-teal bicycle with a wicker basket exactly as in image 2. Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the wrought-iron railing of the first-floor balcony above the door (about 80% of the width), looking toward the two of them. The swallow is added in this picture only, it is not part of the background.

The background already has its evening light, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, a second bicycle, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Кадр 3. Увидимся? Созвонимся?

**Звучит:** 5–6: Vidimo se? / Može! Čujemo se?

**Идея:** Марко с надеждой предлагает увидеться, Ана радостно соглашается и сама предлагает созвониться: над ней облачко с пиктограммой телефона (пустой экран, волны звонка), без букв.

**Основа:** плейт «вечер» (первая картинка), потом лист героев.

**Проверить:** Марко около 38%, ладонь вверх, брови приподняты; Ана около 62%, большой палец вверх рукой со стороны Марко, яркая улыбка; над Аной одно облачко с телефоном без букв и цифр; скамья пустая; ласточка одна, на ветке липы.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Ana, Marko, the bicycle and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall, the arch, the linden tree, the bench against the wall at 32-60% of the width, the entrance with its steps and door, the balcony, the windows, the clothesline, the cobblestones, the sky and the light. The output has the same size and aspect ratio as image 1, every background object stays at the same position, and the bench is never moved. The camera is the same in all four pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 3 of 4, evening, 'see you? call each other?'):
- Marko stands at about 38% of the width, in three-quarter view facing Ana (to the right), his head facing the same way as his body. One hand holds the handlebar of the bicycle, the other hand is held out with the palm up and open, his eyebrows are raised, a hopeful smile.
- Ana stands at about 62% of the width, in three-quarter view facing Marko (to the left), her head facing the same way as her body. She gives a clear thumbs-up with the hand on the side facing Marko, held at chest height (not raised high), a bright happy smile; her other hand is in the pocket of her coat.
- Above Ana, at about 62% of the width and between 12% and 28% of the height, one round speech bubble with a small pictogram of a smartphone with a plain dark blank screen and two little curved sound waves beside it, no letters and no numbers anywhere.
- The bench stays empty behind them.
- Both faces are at about 40% of the image height.

CHARACTERS: Ana and Marko exactly as in image 2, the character sheet (Ana: chestnut braid over her left shoulder, deep teal scarf, cream coat; Marko: black curls, dark beard, round glasses, rust-orange hoodie), and Marko's deep-teal bicycle with a wicker basket exactly as in image 2. Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on a linden branch above the bench (about 40% of the width), looking down at the two of them. The swallow is added in this picture only, it is not part of the background.

The background already has its evening light, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, a second bicycle, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Кадр 4. Спокойной ночи!

**Звучит:** 7–8: Dobro! Laku noć! / Laku noć! Doviđenja!

**Идея:** Прощаются: Марко уходит в арку, ведя велосипед, мы видим его со спины; Ана на верхней ступени у двери лицом к нему машет рукой. Тот же двор и те же места, что в кадре 4 первой истории, но без ключа и коробок.

**Основа:** плейт «вечер» (первая картинка), потом лист героев.

**Проверить:** Марко со спины у арки (около 14%), лица и поднятой руки нет, ведёт велосипед; Ана на верхней ступени (около 74%) лицом к нему, машет рукой, в другой руке в кармане ничего; окна светятся; скамья пустая; ласточка одна, над дверью.

```
IMAGES: image 1 is the finished background (the stage). Image 2 is the character sheet: draw Ana, Marko, the bicycle and the swallow exactly as there (same faces, hair, clothes, colours, proportions and pixel size). Image 3, if attached, is the app's style reference.

EDIT TASK: recreate image 1 with the additions below and NOTHING else changed. KEEP EXACTLY, do not redraw, restyle, move, crop or recolour: the camera position and framing, the back wall, the arch, the linden tree, the bench against the wall at 32-60% of the width, the entrance with its steps and door, the balcony, the windows, the clothesline, the cobblestones, the sky and the light. The output has the same size and aspect ratio as image 1, every background object stays at the same position, and the bench is never moved. The camera is the same in all four pictures of the story, so do not zoom, tilt or change the angle. Both characters are drawn at the same scale as the stage: each standing character is about half of the image height.

ADD (frame 4 of 4, evening, 'good night!'):
- Ana stands on the top step in front of the door at about 74% of the width, in three-quarter view facing Marko (to the left), her head facing the same way as her body. One hand is raised in a wave, the other hand is in the pocket of her coat. She is smiling, relaxed and happy. A warm yellow glow shines through the glass of the door's arched window; the door itself stays closed.
- Marko is leaving: he is at the arched passage at about 14% of the width, seen from BEHIND, with the back of his head (black curls), the back of his rust-orange hoodie and the back of his jeans toward the viewer, so no part of his face is visible at all. He walks away into the arch, wheeling his deep-teal bicycle beside him with both hands on the handlebar, the bicycle also seen from behind with its rear rack. His head is straight and is not turned: he does not look back, and no hand of his is raised. Through the arch the sunset street is visible ahead of him. There is a clear distance between him and Ana. The bench stays empty.
- Both faces visible in the picture (only Ana's) are at about 40% of the image height.

CHARACTERS: Ana and Marko exactly as in image 2, the character sheet (Ana: chestnut braid over her left shoulder, deep teal scarf, cream coat; Marko: black curls, dark beard, round glasses, rust-orange hoodie), and Marko's deep-teal bicycle with a wicker basket exactly as in image 2. Never copy people or objects from image 1, it has none.

SWALLOW: exactly one UCIMO swallow, drawn like the swallow in image 2 (and in image 3, if attached): teal, cream belly, golden bib, orange beak and feet, a real bird the size of a pigeon, no clothes, no human pose, no other birds. The swallow sits on the top edge of the frame of the door's arched window (about 74% of the width), a small silhouette against the sunset sky. The swallow is added in this picture only, it is not part of the background.

The background already has its evening light, do not change it.

NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.

AVOID: moving, resizing or restyling anything that is in image 1, extra people, extra birds, a second bicycle, anything that looks like writing, extra fingers or limbs, twisted necks, heads turned backwards or upside down, a head that does not face the way the body faces, mixed pixel sizes, smooth gradients.
```

## Если что-то не получается

Всё из файла первой истории (`docs/story-image-prompts-drago-mi-je.md`, раздел «Если что-то не получается») подходит и здесь. Коротко:

- **Фон поменялся.** В начало промта: «This is an image edit. Output image 1 with the additions only; everything not mentioned stays pixel-identical.»
- **Скамья уехала или появилась вторая.** «There is exactly one bench, against the wall, at 32-60% of the width, and it never moves.»
- **Герои разные от кадра к кадру.** Лист героев вторым и фраза: «Keep Ana's and Marko's faces, hair and clothes identical to image 2.»
- **Рука выходит кривой** (большой палец у Аны в кадре 3, взмах в кадрах 1 и 4). Не перегенерируй весь кадр: «Edit this image. Redraw only Ana's raised arm: lower it so that her hand rests in the pocket of her coat. Change nothing else, every other pixel stays identical.» Для большого пальца: замени на руку в кармане, смысл останется в улыбке и пузыре.
- **Голова вывернута.** Правь готовый кадр: «Redraw only that head so that it faces the same way as the body… Change nothing else.»
- **Буквы на пузыре или телефоне.** «Every bubble and screen is completely blank, no markings at all». Если не помогло, замазать в редакторе.
