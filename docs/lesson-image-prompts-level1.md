# Картинки к урокам: уровень 1 (промты)
Одна картинка на урок. Показывается в начале урока и в конце (на итоговом экране). К отдельным заданиям картинок нет.

Файлы класть в `content/images/lessons/<id>.webp`, например `02d.webp`. Поле `image` в YAML урока и показ в приложении пока не сделаны.

## Как пользоваться

Каждый промт ниже самодостаточный: стиль, правила и сцена. Копировать один блок целиком, ничего склеивать не надо.

## Принципы

- **Люди разные.** Возраст, телосложение, причёски (кудри, лысина, платок, длинные седые, кепка), одежда. Одинаковых причёсок и нарядов в одной картинке нет, каре по умолчанию не рисуется. Постоянных персонажей между уроками тоже нет.
- **Люди только там, где без них нельзя.** Это уроки 02, 02b (маленькие фигурки вдали) и 03, 04 (крупно только руки, без лиц). Остальные 13 картинок это места и предметы, они сами рассказывают идею урока. Постоянных персонажей нет.
- **У каждой картинки одна мысль**, она написана в строке «Идея».
- **Ласточка (маскот UCIMO) везде одна**, размером с голубя, без одежды и без человеческой позы. Других птиц нет. Если ласточки в сцене нет, в промте сказано «птиц нет».
- **Руки описаны явно**: что делает каждая рука и какой у неё рукав, чтобы они не получались одинаковыми.

## Общий блок стиля (уже вставлен в каждый промт)

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes.
```

## Надписи на картинках

Нейросети часто искажают буквы, особенно š ž č ć đ. Надписи заданы только в уроках 00, 01 и 01b, буква в букву. После генерации проверить каждую. Если есть ошибка, лучше наложить текст в редакторе поверх картинки, чем перегенерировать.

## Промты

### 00. Кириллица и латиница

**Идея:** Одно название улицы написано двумя алфавитами. Видно, что это одно и то же.

**Что на картинке:** Угол старой улицы. На стене две синие таблички друг над другом: «Железничка» кириллицей и «Železnička» латиницей. Ласточка на проводе рядом.

**Надписи: «Железничка» (кириллица) и «Železnička» (латиница, галочки над Ž и č).**

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a quiet corner of an old street in Novi Sad on a late afternoon. Foreground: cobblestone sidewalk, a potted red geranium on a windowsill, a wooden bench. Middle ground: a cream stucco building corner with a green wooden door and a lit window. On the wall at the corner, two rectangular blue enamel street signs with white borders hang one above the other, same size, clearly legible, in the center of the picture. The top sign reads exactly "Железничка" in Cyrillic letters. The bottom sign reads exactly "Železnička" in Latin letters, with a caron (hacek) over the Ž and over the č. Both use a simple bold white sans-serif font, centered, no other text. Background: pastel townhouses, a tram stop pole, warm low sun with long soft shadows on the wall. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, perched on a thin electrical wire directly beside the signs, head tilted as if comparing them. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 01. Особые звуки

**Идея:** Прилавок кондитерской с четырьмя предметами, у каждого карточка со словом. Буквы с галочками видны сразу.

**Что на картинке:** Прилавок кондитерской: торт, чевапи, шляпа на крючке, лягушка-игрушка, у каждого карточка с названием.

**Надписи: «čokolada», «ćevapi», «šešir», «žaba».**

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: the counter of a small Serbian pastry shop, seen from the customer side, nobody behind it. On the wooden counter, left to right, four items, each with a small cream cardboard label standing in front of it in bold, clearly legible sans-serif: a round chocolate cake with glossy dark icing, label exactly "čokolada"; a wide oval plate of grilled cevapi sausages with chopped raw onion and a flatbread, label exactly "ćevapi"; a straw hat hanging on a wooden hook at the right end of the counter, label exactly "šešir"; a small green toy frog with big round eyes sitting on a doily, label exactly "žaba". Every diacritic must be present and correct: č, ć, š, š, ž. Behind the counter: shelves with bread loaves and jars, a window with a striped awning, a hanging lamp with a warm glow, a black-and-cream checkered floor, a brass cash register. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, standing on the top edge of the glass display case on the counter, looking at the frog. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 01b. Повторение: особые звуки

**Идея:** Полка-коллекция: восемь предметов урока, под каждым подпись. Все слова разом.

**Что на картинке:** Вечерняя деревянная полка с восемью предметами и карточками под ними.

**Надписи слева направо: «čokolada», «ćevapi», «šešir», «žaba», «đak», «džem», «ljubav», «konj».**

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a wide wooden shelf in a cozy room at evening, like a small museum display. Warm lamp light from the left, a dark teal wall behind. Eight items stand in one row from left to right, each on a small wooden stand with a small cream label below it in bold, clearly legible sans-serif. 1: a chocolate bar half unwrapped, label exactly "čokolada". 2: a plate of grilled cevapi with onion, label exactly "ćevapi". 3: a straw hat, label exactly "šešir". 4: a small green toy frog, label exactly "žaba". 5: a small toy schoolboy figurine with a red backpack, label exactly "đak". 6: a glass jar of orange apricot jam with a gingham lid, label exactly "džem". 7: a red heart-shaped cushion, label exactly "ljubav". 8: a small wooden toy horse, label exactly "konj". Every letter must be exactly as written: č, ć, š, š, ž, đ, dž, lj. Below the shelf the edge of a rug and a plant in a pot. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, sitting proudly on the end of the shelf next to the horse, wings folded. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 02. Привет!

**Идея:** Утро, пекарня открыта, соседи здороваются издалека. Главное в кадре: утро и открытая дверь.

**Что на картинке:** Утренняя улица с пекарней. Вдали два маленьких прохожих машут друг другу. Ласточка летит над улицей.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a bright sunny morning on a Serbian street in front of a small bakery with its door wide open, warm light spilling out and a basket of fresh bread rolls with steam. Foreground: cobblestones, a wooden bench, a bicycle leaning on the wall. Middle ground: the bakery with a red-and-white striped awning and a chalkboard easel with no writing. Two small figures far down the street, drawn simply as small distinct people seen from the side: a tall young woman with long curly red hair in a yellow raincoat, and a short elderly man with a bald head, white beard and a green cap, who leans on a cane, facing each other and both raising an arm in a friendly wave. Background: pastel houses, a green street lamp, pale blue sky with soft clouds, golden morning light from the left. Any person shown has a human head, face and hair, exactly two arms and two hands with simplified fingers, and nobody is a bird. People are diverse and look nothing alike: different ages, builds, heights, skin tones, hair colors and hairstyles (curly, braided, buzz cut, bald, long, gray, a cap or a headscarf), and different clothing. No two people in the image share a hairstyle or an outfit. Avoid generic bob haircuts. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, flying above the street, wings spread. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 02b. Вечер и до свидания

**Идея:** Вечер, кафе закрывается, один человек уходит и оглядывается. Свет в окнах, луна.

**Что на картинке:** Сумерки у кафе. Вдали маленький человек в пальто уходит, оглядывается и поднимает руку. Горит свет, виден месяц.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: dusk on a quiet Serbian street in front of a small cafe, deep teal and gold sky with a thin crescent moon and one star. The cafe window glows warm yellow, chairs are stacked on the terrace, a closed-for-the-night feeling. Foreground: a cobblestone sidewalk, a street lamp just turned on with a golden glow. In the middle distance, one small broad-shouldered man with a shaved head, a dark gray beard, a long beige coat and a blue scarf walks away down the sidewalk, seen from the side and slightly from behind, head turned back, one arm raised in a goodbye wave. Details: a chalkboard on an easel with no writing, tram rails glinting, lit windows in the buildings across the street. Any person shown has a human head, face and hair, exactly two arms and two hands with simplified fingers, and nobody is a bird. People are diverse and look nothing alike: different ages, builds, heights, skin tones, hair colors and hairstyles (curly, braided, buzz cut, bald, long, gray, a cap or a headscarf), and different clothing. No two people in the image share a hairstyle or an outfit. Avoid generic bob haircuts. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, sitting on the street lamp, looking at the person leaving. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 02c. Повторение: приветствия

**Идея:** Утро и вечер в одной улице: свет переходит слева направо. «Привет» и «пока» как начало и конец дня.

**Что на картинке:** Улица с часовой башней. Небо слева утреннее, справа вечернее. Люди не нужны.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a wide Serbian town square with a tall clock tower in the exact center. The sky and light transition smoothly from left to right: on the left it is a bright morning with a pale blue sky and a pink-gold sunrise, the left shop windows are dark and a bakery door opens with fresh bread in a basket; on the right it is a warm evening with a deep teal-orange sky, a crescent moon, and the right shop windows glowing yellow and lamps lit. In the center the clock tower with a round white clock. Foreground: cobblestones, a wooden bench on each side, a fountain. A tram stops quietly at the far side. Autumn trees on both edges. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, sitting on top of the clock tower, small against the sky. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 02d. Как дела?

**Идея:** Два стиля общения без людей: слева стол друзей («ты»), справа стол для старших («вы»). Видна разница тона.

**Что на картинке:** Два столика кафе. Слева простой: мороженое, скейт, кроссовки. Справа парадный: скатерть, чайник, трость и газета.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a sunny cafe terrace with two separate tables divided by a tall green lamp post in the exact center, nobody sitting at them. LEFT TABLE (casual): a bare wooden table with three ice cream cones in a cup holder, three soda cans, a skateboard leaning against it, a pair of bright sneakers on the ground, a backpack tossed on a plastic stool, a smartphone with earbuds. RIGHT TABLE (formal): a table covered with a white tablecloth, a porcelain teapot with cups on saucers, a small plate with biscuits, a folded newspaper, a pair of reading glasses, a wooden walking cane hooked on the back of a neatly placed chair, a gray flat cap resting on the table. The mood contrast is clear: left is loud and playful, right is quiet and polite. Background: trees, a fountain, warm afternoon light. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, sitting on the top of the lamp post in the center, looking toward the left table. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 02e. Повторение: как дела

**Идея:** После кафе: две чашки, телефон с отмеченным днём. «Увидимся!»

**Что на картинке:** Столик кафе вечером, на нём две чашки, баклава и телефон с календарём, где один день отмечен золотом.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: an outdoor cafe table in the late afternoon, no one sitting, two chairs slightly pushed back as if just left. On the small round table: two small coffee cups with saucers, a plate with two pieces of baklava, a glass of water, and a smartphone lying with its screen up showing a simple calendar grid with one square circled in gold. A wicker shopping basket beside one chair, a folded gray scarf on the other. Background: a string of warm round bulbs overhead, potted plants, a stone wall, other empty tables, soft low sunlight. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, standing on the back of one chair, looking at the phone. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 03. Пожалуйста, спасибо

**Идея:** Рынок крупным планом: рука продавца передаёт пакет, рука покупателя кладёт монеты. Спасибо и пожалуйста в жесте.

**Что на картинке:** Прилавок рынка крупным планом. Две руки: одна подаёт пакет помидоров, другая протягивает монеты.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a close view of an open-air market stall in Serbia on a sunny morning. Only two pairs of hands are visible, no faces. From the right, the weathered hand of an older seller, tanned, with rolled-up green sleeve and a silver ring, a striped apron edge, holds out a paper bag of red tomatoes. From the left, the slim young hand of a customer, light skin, with a cream coat sleeve and a thin gold bracelet, holds out a few gold coins above the counter, and the customer's left hand is pressed lightly against the chest in a gesture of thanks (only the hand and a bit of the coat visible). Between them on the stall: wooden crates of tomatoes, green peppers, purple eggplants, yellow apples, a basket of eggs, a small hanging scale. Background: other stalls with colorful umbrellas, strings of red peppers, cobblestones, bright warm morning. Any person shown has a human head, face and hair, exactly two arms and two hands with simplified fingers, and nobody is a bird. People are diverse and look nothing alike: different ages, builds, heights, skin tones, hair colors and hairstyles (curly, braided, buzz cut, bald, long, gray, a cap or a headscarf), and different clothing. No two people in the image share a hairstyle or an outfit. Avoid generic bob haircuts. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, standing on a crate of apples on the stall, a gold coin near its feet. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 03b. Повторение: вежливые слова

**Идея:** Столкнулись в дверях, посыпались вещи: сцена «извините». Люди не нужны, всё видно по предметам.

**Что на картинке:** Дверь магазина приоткрыта, на пороге яблоко, корзинка перевернулась, из пакета выпал батон. Всё дружелюбно.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: the entrance of a small neighborhood shop with a glass door held ajar and a brass bell above it, a doorstop in the shape of a small brass cat holding it open. On the stone step in front, a wicker basket has tipped over and one red apple rolls away, a paper grocery bag lies next to it with a baguette sliding out, a dropped umbrella. A bright fallen yellow leaf. The scene looks like a friendly mishap just happened. Details: the shop window with bread and vegetables, a bicycle, a street lamp, a sunny afternoon. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, sitting on the sign bracket above the door, looking down at the apple. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 04. Знакомство

**Идея:** Рукопожатие крупным планом: две разные руки, у каждого бейдж-имя без текста. Первое знакомство.

**Что на картинке:** Две руки пожимают друг друга в центре. Одна в бирюзовом рукаве, другая в зелёной клетчатой рубашке. Бейджи пустые.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a close view of a handshake in a bright living room at a small gathering. Only two torsos from the chest down to the hands and the upper chest are visible, no heads or faces. On the left, a slim person in a deep teal jacket over a cream shirt extends their right hand, a watch on the wrist. On the right, a heavier-set older person in a green checked shirt with rolled sleeves, a hairy forearm and a wedding ring, extends their right hand. The hands meet at the exact center of the picture, clearly visible, a firm friendly handshake. Each of them wears a blank white name tag on the chest, empty, no text. Behind them, softly drawn, a bookshelf, a window with curtains, fairy lights along the wall, a plate of small sandwiches on a table, a potted ficus, two blurred guests holding teacups, one tall with long gray hair, one short with a red headscarf. Any person shown has a human head, face and hair, exactly two arms and two hands with simplified fingers, and nobody is a bird. People are diverse and look nothing alike: different ages, builds, heights, skin tones, hair colors and hairstyles (curly, braided, buzz cut, bald, long, gray, a cap or a headscarf), and different clothing. No two people in the image share a hairstyle or an outfit. Avoid generic bob haircuts. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, standing on top of the bookshelf, watching. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 04b. Быть и откуда я

**Идея:** Квартира человека, который переехал: карта с двумя точками, чемодан, вид из окна на Нови-Сад.

**Что на картинке:** Комната: на стене карта с двумя булавками и пунктиром, раскрытый чемодан, чай у окна с видом на реку.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a warm apartment room in Novi Sad in the evening, nobody in it. A small round wooden table by a large window with a steaming teacup, a laptop, a notebook and a small plant. Through the window: the Danube river, a long bridge and a hilltop fortress in golden evening light. On the wall behind, a large framed map of Europe with two gold pushpins far apart, one in the east and one in the Balkans, connected by a dotted gold string. By the front door: an open suitcase with clothes half unpacked, a pair of shoes, a teal jacket on a hook. A shelf with a few books, a small alarm clock and a radio. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, perched on the top edge of the map frame, looking at the two pins. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 04c. Повторение: знакомство

**Идея:** Набережная на закате: скамейка, две чашки, телефон с картой. Двое только что познакомились и ушли.

**Что на картинке:** Скамейка на набережной, две чашки кофе, телефон с картой и золотой точкой, куртка. Вдали город.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a wooden bench on a riverside promenade at golden hour, nobody sitting on it. Two paper coffee cups on the bench, between them a smartphone lying with its screen up showing a simple map with a gold pin, a folded teal jacket on one side and a wicker basket on the other. Beyond the railing: a wide calm river, ducks swimming, a fortress on a hill and rooftops in orange light, a small sailboat. Details: a lamp post, trees with yellow leaves, a path with fallen leaves. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, sitting on the bench backrest in the middle. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 05. Говорю и понимаю

**Идея:** Слева запутанное облако непонятных слов, справа ясная звезда и горящая лампочка. «Не понимаю… теперь понимаю».

**Что на картинке:** Прилавок: слева облако закорючек, справа чистая золотая звезда и горящая лампочка над раскрытым разговорником. Песочные часы у ласточки.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a grocery shop counter seen from the front, nobody there. On the LEFT part a speech cloud floats in the air filled with abstract tangled squiggles, spirals and zigzags in dark teal, messy and confusing (no letters, no numbers). On the RIGHT part the cloud becomes a smooth clear speech bubble holding one clean gold star, and above it a small hanging light bulb glows bright yellow with soft rays; on the left the same bulb is dim and gray. In the middle on the counter: an open phrasebook with a pencil, a small hourglass, a bowl of olives, a jar of pickles, a wedge of white cheese. Background shelves with jars and cans, warm yellow light. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, standing on the phrasebook on the counter, looking at the hourglass. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 05b. Работаю и учу

**Идея:** Рабочий стол с ноутбуком и разговорником. На стене три картинки профессий.

**Что на картинке:** Стол: ноутбук, раскрытый разговорник, чашка кофе. На стене три постера: врач, официант, учитель.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a cozy home workspace in the afternoon, nobody at the desk. A wooden desk with an open laptop, an open phrasebook with a pencil on it, a steaming coffee cup, a small notebook and a desk lamp. On the wall above the desk, three framed illustrated posters side by side: a doctor with a stethoscope, a waiter carrying a tray with a coffee, and a teacher at a green chalkboard, all drawn in simple pixel style, no text. Window on the right with rooftops and a sunny sky. A bookshelf with colorful books, a small plant on the windowsill. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, standing on the open phrasebook, looking at the pencil. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 05c. Повторение: говорю и работаю

**Идея:** Встреча за круглым столом: четыре места, у каждого своя чашка и свой значок над стулом.

**Что на картинке:** Круглый стол с четырьмя стульями. Над каждым место-пузырь с иконкой: книга, хлеб, кофе, ноутбук.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a language exchange meetup room in a cafe, the people have just gone for a break. A round wooden table seen slightly from above with four chairs around it. At each seat a different item: a cup of tea with an open book, a cup of coffee with a small bread roll, a notebook with a pen and a glass of juice, a laptop with a cookie. Above each chair floats a round speech bubble with one simple icon: a book, a bread loaf, a coffee cup, a laptop. Background: a brick wall with framed pictures, hanging lamps with warm light, a window with raindrops. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, sitting on the pendant lamp above the table. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

### 14. Проверка: первый контакт

**Идея:** Финал уровня: улица, на которой уже всё знакомо. Праздничные флажки, указатель и звезда.

**Что на картинке:** Улица с пекарней, рынком, кафе, трамваем и флажками. Ласточка несёт золотую звезду.

```
Pixel art illustration, 4:3 landscape, with slightly visible chunky pixels (like a 320x240 retro game scaled up, a touch more pixelated than smooth modern pixel art, but still detailed and readable). Flat cel-shading with 2-3 tones per surface, crisp dark outlines, no gradients, no photorealism, no blur, no anti-aliased smooth edges, no film grain. Characters have chibi proportions (about 3 heads tall), simple dot eyes, small readable faces. Brand palette: deep teal #176B68, dark teal outline #0B3F3D, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7, plus natural warm colors for skin, wood, brick and food. A detailed scene with foreground, middle ground and background, warm soft light, cozy Balkan city atmosphere. No text, no letters, no logos, no watermarks, except where the prompt gives exact words in quotes. Scene: a lively sunny Serbian street, a celebratory empty street at golden sunset, the places from the lessons all in one view. Left: the bakery with its red-and-white striped awning and an open door with warm light. Middle: a market stall with colorful umbrellas and crates of tomatoes, and a yellow tram on rails. Right: a cafe terrace with round tables, cups and string lights, and a clock tower behind. Above the street, paper bunting flags in alternating teal, gold and cream (plain colors, no text). In the foreground a wooden signpost with three arrow-shaped boards pointing in different directions (no text), a cobblestone path, falling autumn leaves. There are no people and no humans in the image, the scene is told only through places and objects. Exactly one small teal pixel-art swallow (the UCIMO mascot) as a real bird the size of a pigeon, flying above the street at the center, carrying a small gold star in its beak. It is an animal: no clothes, no human arms, no human pose. There are no other birds anywhere in the image.
```

## Проверка перед использованием

- Ласточка ровно одна и без одежды. Нет других птиц.
- Там, где людей быть не должно, их нет. В уроках 03 и 04 лиц нет, видны только руки, и у рук разные рукава.
- Нет текста, кроме уроков 00, 01 и 01b, и там он совпадает буква в букву.
- Один стиль на все 17 картинок. Если генерация пошла не туда, перегенерировать с референсом из первой удачной.
- Размер: 4:3, не тяжелее ~150 КБ в webp.
