# Истории: стиль картинок

Единый стиль кадров для историй (`docs/STORIES_PLAN.md`). Он продолжает картинки приложения: пиксель-арт с насыщенным
светом, как на `web/src/assets/invite-scene.webp` и `tariffs-promo-scene.webp`, и та же ласточка-маскот. Промты кадров
конкретной истории лежат рядом (`docs/story-image-prompts-drago-mi-je.md`), они собраны из блоков ниже.

## Главное правило: неподвижная сцена

Генератор, получив фон «как референс», рисует новый фон «в духе» референса: скамья переезжает в середину двора, меняются
окна и арка. Поэтому у истории **одна камера на все кадры**, и кадр делается не заново, а **правкой готового фона**:

1. Фон (плейт) без людей: «день», и его же «вечер» (правка света).
2. Кадр = плейт + герои и реквизит (правка: «всё остальное не трогать»).
3. Герои везде одного размера. Приближение для выразительных кадров делается при нарезке (кроп), не в генераторе.
4. Всё в сцене стоит у стен, ничего посреди открытого пола. Скамья у стены, арка слева, подъезд справа: места
   записаны в процентах ширины, чтобы кадры можно было сверить рядом.

## Что отличается от картинок к урокам

- **Постоянные герои.** Зритель должен узнавать Ану и Марко от кадра к кадру, поэтому у них фиксированные приметы с цветами
  (блоки ниже) и лист героев, который прикладывается к каждой правке.
- **Кадр служит сюжету.** У каждого кадра одна мысль и один жест, который читается без слов. У демо-истории есть предыстория,
  которой нет в тексте: **день переезда** (коробки, фургон, растение, ключ). Текст и озвучка не менялись.
- **Формат 4:3, пиксель-арт.** В плеере кадр лежит карточкой сверху, под ним субтитры. Плавность даёт кроссфейд между кадрами
  и медленный зум до 107 % (CSS).
- **Зона безопасности.** На низком экране кадр обрезается по центру до полосы (до 3:1), а зум съедает края. Лица, руки и
  главное действие лежат в центре (60 % ширины, от 25 до 75 % высоты), по краям запас 7 %.
- **Стены не кремовые.** Пальто Аны кремовое, поэтому стена серо-мятная, а дверь горчично-золотая (бренд-золото). Рыжая
  толстовка Марко стоит напротив бирюзы.
- **Ласточка в каждом кадре одна**, размером с голубя, без одежды и человеческой позы, других птиц нет. Её нет в плейте,
  она добавляется в каждом кадре. Рисуется как на `invite-scene.webp`.

## Референсы

1. Лист героев (промт 0 в файле промтов): Ана, Марко, ласточка, велосипед. Прикладывается ко всем кадрам.
2. Плейт «день» (промт 1) и плейт «вечер» (промт 2): основа каждого кадра, **первая картинка в правке**.
3. `web/src/assets/invite-scene.webp`: размер пикселя, насыщенность цвета, ласточка (для листа героев и плейта).

## Блоки (вставляются в каждый промт дословно)

**Стиль**

```
STYLE: Pixel art illustration in the style of a polished modern indie adventure game, 4:3 landscape, drawn on a coarse pixel grid of roughly 320x240 and scaled up with hard nearest-neighbour edges, so the individual pixels are clearly visible (about 4 screen pixels per art pixel). The pixel size is identical across the whole image: characters, props, foliage and sky all sit on the same grid, with no mixed resolutions, no smooth vector curves and no sub-pixel detail. Rich, warm, atmospheric colour: skies built from distinct stepped colour bands with light ordered dithering between the bands, colour-shifted shadows (shadows lean cool teal-violet, lights lean warm gold), soft rim light on the characters, small glowing highlights, foliage drawn as chunky pixel clusters with a light side and a dark side, plaster and cobblestones with a little pixel texture. Crisp 1-pixel dark outlines on characters and key props in dark teal #0B3F3D (never pure black), no outlines on distant background shapes. Characters have chibi proportions (about 3 heads tall), simple black dot eyes each with one white highlight pixel, small readable faces. Accent palette from the UCIMO brand, used sparingly: deep teal #176B68, light teal #8CCBC6, gold #E3B04B, cream #F7F9F7.
```

**Без текста**

```
NO TEXT: There is no text of any kind anywhere in the image: no letters, numbers, signs, labels, logos, watermarks or captions. Boxes, doors, bags and walls are blank or carry only simple pictograms.
```

**Композиция и зона безопасности**

```
COMPOSITION RULES (the picture is shown in an app: on small screens it is cropped to a wide strip and slowly zoomed in): keep all faces, hands and the key action inside the central 60% of the width and between 25% and 75% of the height. Leave about 7% of calm margin at every edge. Put only non-essential things (sky, roofs, cobblestones, the edges of walls) in the top and bottom bands. The characters are large: each one is about half of the image height.
```

**Ана**

```
ANA (a Russian woman in her late twenties who has just moved to Serbia): chibi, about 3 heads tall. A long thick chestnut-brown braid (#7A4A2B, lighter #A06A3F on top) that always hangs over her LEFT shoulder, with a small teal hair tie at the end and a short side-swept fringe. A deep teal scarf (#176B68) wrapped twice around her neck, the fringed ends hanging in front. A cream knee-length wool coat (#F2E9D8, shadow #D9CBB0) with two large buttons, dark navy jeans, clean white sneakers. Fair skin with light rosy cheeks, thin brown eyebrows, small friendly black dot eyes with a white highlight pixel, a small gentle smile.
```

**Марко (и его велосипед)**

```
MARKO (a local neighbour in his early thirties, warm and talkative): chibi, about 3 heads tall, a little taller and broader than Ana. Short black curly hair, a short neat dark beard, round glasses with thin dark frames (plain lenses, the dot eyes stay visible), warm medium skin, thick dark eyebrows, small friendly black dot eyes with a white highlight pixel, a wide easy smile. A rust-orange (#C4622D, shadow #9A4720) hooded sweatshirt with a front pocket and drawstrings, blue jeans, brown leather boots. His bicycle: a classic deep-teal (#176B68) city bike with a brown leather saddle, a wicker basket on the front and a rear rack, no lettering on the frame.
```

**Ласточка (маскот UCIMO)**

```
SWALLOW (the UCIMO mascot): exactly one small swallow in the whole image, drawn exactly like the swallows in the style reference: deep teal head, back and wings (#176B68, darker wing tips #0B3F3D), a cream belly (#F7F9F7), a golden-yellow V-shaped bib at the throat (#E3B04B), a small orange-gold beak and orange-gold feet, a round black eye with a white highlight, a long forked tail. It is a real bird about the size of a pigeon (roughly one third of a character's height), with no clothes, no human arms and no human pose. There are no other birds anywhere in the image.
```

**Сцена (двор, неподвижная)**

```
SETTING (a fixed stage that never changes): the inner courtyard (dvorište) of an old three-storey townhouse in Novi Sad, Serbia, seen from ONE fixed camera position: straight-on, at the eye level of a standing person, 4:3. The picture is dominated by one long back wall, a soft sage-mint (#9CC7BE, slightly greyed) stucco facade that runs from the left edge to the right edge of the frame, with a band of cobblestone floor in front of it covering the bottom quarter of the picture. Nothing stands in the middle of the open floor: every object stands against the wall. From left to right along the wall: (1) at the far left, 0-22% of the width, a tall arched carriage passage that leads out to a sunlit street, with a pale street and a pastel house seen through the arch; (2) at 24-32% a low stone-edged bed with the trunk of a big old linden tree growing out of it, its wide round crown filling the upper middle of the picture above the wall; (3) at 32-60% a long wooden bench with a curved backrest standing flat against the wall right beside the linden trunk, its back touching the wall, with one small potted red geranium at its right end; (4) at 66-96% the building's entrance: three stone steps up to a mustard-gold (#E3B04B) double wooden door with an arched glass window above it, a small brass lamp beside the door, a row of blank metal mailboxes on the wall, pots of red geraniums on the steps; (5) above the entrance, on the first floor, a wrought-iron balcony with geraniums. The wall also has white-framed windows with warm-brown wooden shutters, and a clothesline with a few towels between two windows above the bench. Far behind the roofline: a church tower with a tiled spire. The cobblestone floor in front is open and empty.
```


## Свет

Свет «ведёт» историю: день у кадров 1–3 и золотой час у кадра 4. День один на три кадра специально: меньше разных фонов,
меньше расхождений. Вечер получается правкой плейта «день».

| Кадры | Время | Свет и небо |
|---|---|---|
| 1–3 | позднее утро | высокое солнце слева, короткие мягкие тени, холодные фиолетово-бирюзовые тени, пятна света от липы, голубое небо |
| 4 | 18:00 | оранжевый свет слева, длинные тени, небо ступенями от бирюзы к золоту, окна светятся |

## Как писать кадры для следующих историй

1. Придумать **сцену**: одна камера, стена или несколько стен, объекты у стен с позициями в процентах ширины, пустой пол.
2. Сделать плейт и его второй свет (если история длится дольше одного времени суток).
3. Каждый кадр это правка плейта: «оставить всё как есть», «добавить» (кто где стоит в процентах, поза, жест, лицо, реквизит),
   ласточка с местом, «без текста», «избегать». У кадра одна мысль, у героя один читаемый жест. Коробки, вывески и пакеты пустые.

## После генерации

- Проверить: ни одной буквы и надписи, у героев те же приметы, что на листе, ласточка одна, велосипед один, размер пикселей
  одинаковый, лица и руки в центральной полосе.
- Положить четыре кадра рядом: арка, скамья, дерево и дверь на одних и тех же местах, меняются только герои и свет.
- Сжать в `webp`, ширина 1200 px, около 150 КБ: `cwebp -q 82 -resize 1200 0 in.png -o s1.webp`.
- Положить как `s1.png`…`s4.png` в `scripts/.cache/stories/<slug>/frames/`, оттуда их загружает скрипт публикации.


## История «Prvi dan»: герои и сцена

Вторая история («Prvi dan», `docs/story-image-prompts-prvi-dan.md`) идёт с **новыми героями**: Майя и Милош выглядят как ребята с присланного фото
(причёска, одежда, украшения; лицо пиксель-арт передаёт только условно). Ана и Марко остаются героями первой истории. Блоки стиля,
«без текста», композиции и ласточки те же, что выше. Сцена другая: коридор больницы в Белграде, одно утро, один плейт (вечернего
плейта нет). Белый халат героев лежит на мятной стене, поэтому стена серо-мятная, а дверь горчично-золотая (бренд-золото).

**Майя**

```
MAJA (a Russian-speaking young woman doctor in her mid-twenties, new at a hospital in Belgrade): chibi, about 3 heads tall. Long, straight, smooth light-brown hair with soft golden-blonde highlights (#B98A5E, lighter #D9B27F), worn down and falling past her shoulders on both sides, a soft parting near the centre and a few strands framing her face. Fair skin with warm rosy cheeks, thin light-brown eyebrows, small friendly black dot eyes with a white highlight pixel, a small gentle closed-mouth smile. Tiny silver stud earrings, a short single-strand necklace of cream pearls and, hanging below it on a thin silver chain, a tiny sparkling bow-shaped pendant (a few light-blue and white pixels). She wears an open white doctor's coat (#F4F6F5, shadow #C9D3D3) over a cream-white dress with a sweetheart neckline, puffy sleeves and a small scattered floral print (tiny dusty-pink, brown and green flowers); the dress and the necklace are clearly visible at the front. The strap of a burgundy (#7A1F2B) leather shoulder bag crosses one shoulder, the bag hangs at her hip. Clean white sneakers.
```

**Милош**

```
MILOŠ (a young man doctor in his mid-twenties, Maja's new colleague): chibi, about 3 heads tall, a little taller than Maja. Short, thick, tousled dark-brown hair (#4A3326, lighter #6B4B38) with a longer fringe swept to one side and falling over his forehead, no beard, clean-shaven. Fair skin with a warm tint, thick dark-brown eyebrows, small calm black dot eyes with a white highlight pixel, a relaxed closed-mouth half smile. He wears an open white doctor's coat (#F4F6F5, shadow #C9D3D3) over a plain white crew-neck T-shirt with a faint small red-pink print on one sleeve, dark navy trousers and clean white sneakers, and a deep-teal (#176B68) stethoscope around his neck.
```

**Сцена (коридор, неподвижная)**

```
SETTING (a fixed stage that never changes): a corridor (hodnik) of a public hospital in Belgrade, Serbia, seen from ONE fixed camera position: straight-on, at the eye level of a standing person, 4:3. The picture is dominated by one long back wall that runs from the left edge to the right edge of the frame: a soft sage-mint (#9CC7BE, slightly greyed) painted wall, a slightly darker mint lower half, a pale cream upper band, and a thin pale-wood handrail along the wall at hip height. A band of warm grey-beige linoleum floor (not cobblestones) with a little pixel texture covers the bottom quarter of the picture. Nothing stands in the middle of the open floor: every object stands against the wall. From left to right along the wall: (1) at 0-22% of the width a tall window with a pale-wood frame and a white windowsill, open onto a sunny view of Belgrade: rooftops, treetops, and on the horizon the big white dome of a large Orthodox church with a small cross on top, under a light blue sky; (2) at 24-36% a row of three blue-grey plastic waiting chairs on one shared frame, standing flat against the wall; (3) at 38-52% a cork notice board with a few pinned blank coloured papers (pictograms only, no writing), and at 54-60% a tall potted green plant; (4) at 62-68% a water cooler with a clear blue bottle; (5) at 72-96% a pair of white double doors with frosted-glass panels in a mustard-gold (#E3B04B) door frame, a small round green-cross pictogram above them, and a blank hand-sanitiser dispenser on the wall beside the door. Along the very top edge: a plain ceiling with a strip of soft white lights. The linoleum floor in front is open and empty.
```

**Свет (утро)**

```
LIGHT: morning. Warm sunlight comes in through the window at the left and lays a soft golden patch on the wall and the floor to the right of the window; shadows are short, cool violet-teal; the corridor lights are a clean, slightly cool white. Sky seen through the window: clear light blue with one thin pale cloud.
```


## Истории уровня 2: «Na pijaci» и «SIM kartica» (2026-11-01)

Две истории с **новыми сценами и героями**. Блоки стиля, «без текста», композиции и ласточки те же, что выше. **Маленькие фигуры за прилавком:**
в обеих сценах один длинный прилавок через весь кадр (верх на 62 % высоты, передняя панель до 76 %), продавец стоит за ним (виден по пояс, лицо на 38 % высоты),
покупатель на полу перед прилавком (во весь рост, пол пустой). Одна камера, одно время суток, плейт один (вечера нет). Вывесок и надписей нет: пакеты сим-карт, коробки
телефонов, мешки и баночки пустые. Персонажи: Майя (в рыночной истории без халата, в платье, с холщовой сумкой), Продавец, Алекс, Продавщица. Файлы промтов:
`docs/story-image-prompts-na-pijaci.md`, `docs/story-image-prompts-sim-kartica.md`.

**Майя (в свободный день)**

```
MAJA (on her day off, in casual clothes; the same young woman as in the hospital story, a Russian-speaking doctor in her mid-twenties who lives in Belgrade): chibi, about 3 heads tall. Long, straight, smooth light-brown hair with soft golden-blonde highlights (#B98A5E, lighter #D9B27F), worn down and falling past her shoulders on both sides, a soft parting near the centre and a few strands framing her face. Fair skin with warm rosy cheeks, thin light-brown eyebrows, small friendly eyes of a grey-blue-green colour (#7FA39F: grey with blue and green tones, not black, not brown) with a dark pupil pixel and one white highlight pixel, a small gentle closed-mouth smile. Tiny silver stud earrings, a short single-strand necklace of cream pearls and, hanging below it on a thin silver chain, a tiny sparkling bow-shaped pendant (a few light-blue and white pixels). She wears a cream-white dress with a sweetheart neckline, puffy sleeves and a small scattered floral print (tiny dusty-pink, brown and green flowers), and NO doctor's coat. The strap of a burgundy (#7A1F2B) leather shoulder bag crosses one shoulder, the bag hangs at her hip, and she also carries a plain natural-linen shopping tote (blank, no print) in one hand. Clean white sneakers.
```

**Продавец (рынок)**

```
PRODAVAC (the stall owner, a stocky, friendly man in his fifties, a market regular): chibi, about 3 heads tall, round and sturdy. Short grey-flecked dark hair under a flat grey cap, a thick grey-black moustache, ruddy cheeks, warm medium skin, thick bushy eyebrows, small kind black dot eyes with a white highlight pixel, a broad generous smile. A forest-green (#3F6B3A, shadow #2C4D29) canvas apron over a white-and-blue checked shirt with the sleeves rolled up, short thick forearms. When he stands behind the counter he is seen from the waist up.
```

**Сцена (рынок, неподвижная)**

```
SETTING (a fixed stage that never changes): one stall of a green market (pijaca) in Belgrade, Serbia, seen from ONE fixed camera position: straight-on, at the eye level of a standing person, 4:3. A striped teal-and-cream canvas awning with a scalloped edge runs across the whole top of the picture (0-100% of the width, the top 18% of the height). Behind the stall is a pale warm-cream plaster wall (#EFE3C8) with two simple wooden shelves holding blank glass jars and blank cloth-wrapped packages, and strings of red dried peppers hanging from the awning at 4-10% and 90-96% of the width. In front of the wall runs one long pale-wood counter from 6% to 94% of the width: its top edge is at 62% of the height and its plain-board front panel goes down to 76%. A band of warm grey paving stones with a little pixel texture covers the bottom quarter of the picture and is open and empty: nothing stands on the paving. On the counter, from left to right: (1) at 6-24% a stack of three round white cheese wheels on a wooden board, a big cheese knife leaning beside them; (2) at 50-58% a small blue-grey metal cash box with its lid closed; (3) at 74-90% two grey-cardboard egg trays with brown eggs, side by side, and a small stack of blank brown paper bags at 90-94%. The counter between the cheese wheels and the cash box (26-48% of the width) is bare, empty wood: there is NO scale and nothing standing on it, so that the stall owner's hands can rest on it. The space behind the counter at 34-48% of the width is empty (this is where the stall owner will stand; he is not in the stage). No people, no animals, no birds.
```

**Свет (рынок)**

```
LIGHT: morning. Warm sunlight comes from the upper left and lays a soft golden glow on the cheese and the counter; the shade under the awning is cool violet-teal; the paving in front is bright in the sun; a few warm reflected highlights on the eggs.
```

**Алекс**

```
ALEKS (a Russian man in his late twenties who has just moved to Belgrade): chibi, about 3 heads tall. Short, slightly messy dark-blond hair (#B08A55, lighter #CBA774), light stubble, fair skin with a warm tint, medium-thick brown eyebrows, small friendly black dot eyes with a white highlight pixel, a relaxed easy smile. A faded blue denim jacket (#4C6FA5, shadow #355079) over a plain grey T-shirt, dark-grey jeans, white sneakers, a small olive-green (#6B7B3A) backpack worn on both shoulders (the straps are visible on his chest). His passport is a small plain red booklet (#B3262E) with one tiny blank gold pictogram on the cover and no lettering.
```

**Продавщица (салон связи)**

```
PRODAVAČICA (the saleswoman, a quick, friendly, confident woman in her mid-thirties): chibi, about 3 heads tall. Glossy black hair in a neat high bun with a few strands at her temples, warm olive skin, thin dark eyebrows, small bright black dot eyes with a white highlight pixel, a confident warm smile with soft coral lips. A coral-red (#C8503D, shadow #9C3A2B) sleeveless work vest over a crisp white blouse with rolled sleeves, a blank white lanyard badge with no writing, small gold hoop earrings. She is seen from the waist up above the counter.
```

**Сцена (салон связи, неподвижная)**

```
SETTING (a fixed stage that never changes): a small mobile-phone shop in Belgrade, Serbia, seen from ONE fixed camera position: straight-on, at the eye level of a standing person, 4:3. The back wall is a soft sage-mint (#9CC7BE, slightly greyed) painted wall with a pale cream upper band; a warm grey tiled floor with a little pixel texture covers the bottom quarter of the picture and is open and empty. From left to right along the back wall: (1) at 0-22% of the width a big shop window with a pale-wood frame, open onto a sunny Belgrade street: pastel facades, a few treetops, a bright blue sky; (2) at 24-44% a wall shelf unit with rows of blank white phone boxes and two blank phones on small stands (no screens with text); (3) at 46-68% bare wall with one round clock with plain hands and a simple potted plant under it on the counter's far end; (4) at 72-96% a second shelf unit with blank teal-and-gold cardboard SIM-card packs on small hooks, and a mustard-gold (#E3B04B) door frame at the far right edge. In front of the wall runs one long white-and-pale-wood counter from 8% to 92% of the width: its top edge is at 62% of the height and its plain front panel goes down to 76%. On the counter, from left to right: (1) at 22-30% a flat computer monitor seen from the side with its screen turned toward the saleswoman (its screen is not visible) and a keyboard; (2) at 52-58% a small black-and-teal card-payment terminal on a short stand; (3) at 76-90% a small display stand with a few blank SIM-card packs. The space behind the counter at 34-48% of the width is empty (this is where the saleswoman will stand; she is not in the stage). No people, no animals, no birds.
```

**Свет (салон связи)**

```
LIGHT: daytime. Bright, slightly cool daylight comes in through the shop window at the left and lays a soft white-gold patch on the counter and the tiles; shadows are short, cool violet-teal; the shop lights are a clean white. Sky seen through the window: clear light blue with one thin cloud.
```
