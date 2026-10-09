<template>
  <main class="page-content">
    <section class="hero">
      <div class="hero__text">
        <p class="eyebrow">Три уровня бесплатно — без подписки и лимитов</p>
        <h1 class="hero__title">Учи сербский язык с нуля</h1>
        <p class="hero__subtitle">
          Личный курс сербского языка: 60 уроков с автопроверкой, SRS-тренажёр слов и дашборд прогресса — от алфавита до падежей и живых диалогов. Кириллица и латиница с первого урока.
        </p>
        <div class="hero__actions">
          <a class="btn btn--primary" href="/register" data-goal="register_hero">Изучить сербский</a>
          <p class="hero__note">
            Первые 42 урока и словарь — бесплатно · Без рекламы · 5–10 минут в день
          </p>
        </div>
      </div>
      <div class="hero__art" aria-hidden="true">
        <img
          src="/mascot/05-greeting.webp"
          width="571"
          height="640"
          alt=""
          fetchpriority="high"
        />
      </div>
    </section>

    <section id="audience" class="section" aria-labelledby="audience-title">
      <h2 id="audience-title" class="reveal">
        Для тех, кто учит сербский самостоятельно
      </h2>
      <p class="section__lead reveal">
        ucimo — не курс «для галочки», а рабочий инструмент для тех, кому
        сербский нужен по-настоящему: для переезда, работы или просто потому
        что язык нравится. Без группы, без расписания, без давления —
        занимаешься в своём темпе.
      </p>
      <div class="cards-row">
        <div class="mini-card reveal">
          <h3>Переезжаете в Сербию</h3>
          <p>
            Уровень 5 отдельно разбирает МУП, пријаву адресе и папирологию —
            бытовой сербский, который реально нужен на боравке.
          </p>
        </div>
        <div class="mini-card reveal">
          <h3>Учите язык сами</h3>
          <p>
            Каждое упражнение проверяется сразу — не нужен ни репетитор, ни
            группа, чтобы понять, где ошибка.
          </p>
        </div>
        <div class="mini-card reveal">
          <h3>Устали от самоучителей</h3>
          <p>
            Вместо таблиц склонений — короткие уроки, живые диалоги и слова,
            которые сами возвращаются на повторение.
          </p>
        </div>
      </div>
    </section>

    <section id="how" class="section section--soft" aria-labelledby="how-title">
      <div class="how__header">
        <h2 id="how-title" class="reveal">Как это работает</h2>
        <img
          class="how__art reveal"
          src="/mascot/06-flying.webp"
          width="541"
          height="640"
          loading="lazy"
          alt="Пиксельная ласточка летит вперёд — иллюстрация процесса обучения в ucimo"
        />
      </div>
      <ol class="steps">
        <li class="steps__item reveal">
          <div class="steps__shot">
            <img
              src="/screens/step-1-lesson.webp"
              width="778"
              height="1198"
              loading="lazy"
              alt="Экран урока ucimo: грамматическое объяснение imam/nemam с таблицей форм"
            />
          </div>
          <span class="steps__num">1</span>
          <h3>Урок с объяснением</h3>
          <p>
            Короткая теория по делу: грамматика и слова сразу с переводом и
            примерами, без лишних терминов.
          </p>
        </li>
        <li class="steps__item reveal">
          <div class="steps__shot">
            <img
              src="/screens/step-2-exercise.webp"
              width="788"
              height="706"
              loading="lazy"
              alt="Экран упражнения ucimo: диалог с выбором варианта ответа и автопроверкой"
            />
          </div>
          <span class="steps__num">2</span>
          <h3>Упражнения с автопроверкой</h3>
          <p>
            Выбор варианта, пропуски, сопоставление, сборка фразы из слов,
            перевод — от лёгкого к сложному, ответ проверяется сразу.
          </p>
        </li>
        <li class="steps__item reveal">
          <div class="steps__shot">
            <img
              src="/screens/step-3-srs.webp"
              width="782"
              height="592"
              loading="lazy"
              alt="Экран SRS-тренажёра ucimo: карточка слова prijatno с переводом"
            />
          </div>
          <span class="steps__num">3</span>
          <h3>SRS-тренажёр слов</h3>
          <p>
            Слово возвращается на повторение именно тогда, когда вот-вот
            забудется — по алгоритму интервальных повторений (SM-2).
          </p>
        </li>
        <li class="steps__item reveal">
          <div class="steps__shot">
            <img
              src="/screens/step-4-dashboard.webp"
              width="655"
              height="698"
              loading="lazy"
              alt="Дашборд прогресса ucimo: слова на повторении, дни подряд и следующий урок"
            />
          </div>
          <span class="steps__num">4</span>
          <h3>Дашборд прогресса</h3>
          <p>
            Видно пройденный путь, слова на очереди и сколько ещё до
            следующего уровня.
          </p>
        </li>
      </ol>
    </section>

    <section id="features" class="section" aria-labelledby="features-title">
      <h2 id="features-title" class="reveal">Что внутри</h2>
      <div class="features-grid">
        <article class="feature-card feature-card--media reveal">
          <img
            src="/mascot/08-achievement.webp"
            width="640"
            height="585"
            loading="lazy"
            alt="Пиксельная ласточка с кубком — иллюстрация курса ucimo из 60 уроков"
          />
          <h3>60 уроков с нуля</h3>
          <p>
            От письма и произношения до падежей и свободной беседы. Первые три уровня бесплатны, следующие открываются за зёрнышки — внутреннюю валюту, которую зарабатываешь учёбой.
          </p>
        </article>
        <article class="feature-card feature-card--media reveal">
          <img
            src="/mascot/02-success.webp"
            width="639"
            height="640"
            loading="lazy"
            alt="Пиксельная ласточка радуется — иллюстрация автопроверки ответов в ucimo"
          />
          <h3>Автопроверка ответов</h3>
          <p>
            Пять типов упражнений: выбор, пропуски, сопоставление, сборка из
            слов, перевод. Ответ проверяется мгновенно.
          </p>
        </article>
        <article class="feature-card feature-card--media reveal">
          <img
            src="/mascot/09-reading.webp"
            width="519"
            height="524"
            loading="lazy"
            alt="Пиксельная ласточка читает книгу — иллюстрация тренажёра слов ucimo"
          />
          <h3>SRS-тренажёр слов</h3>
          <p>
            392+ слова с интервальным повторением: система сама решает, что
            и когда повторить.
          </p>
        </article>
        <article class="feature-card feature-card--media reveal">
          <img
            src="/mascot/04-thinking.webp"
            width="380"
            height="640"
            loading="lazy"
            alt="Пиксельная ласточка задумалась — иллюстрация двух алфавитов сербского языка"
          />
          <h3>Кириллица и латиница</h3>
          <p>
            Сербский язык официально пишут обоими алфавитами — оба вводятся
            с первого урока, без путаницы.
          </p>
        </article>
        <article class="feature-card feature-card--media reveal">
          <img
            src="/mascot/11-sitting.webp"
            width="640"
            height="421"
            loading="lazy"
            alt="Пиксельная ласточка внимательно слушает — иллюстрация озвучки произношения в ucimo"
          />
          <h3>Озвучка произношения</h3>
          <p>
            У слов и упражнений на слух — аудио произношение, чтобы с
            первого урока привыкать к звучанию языка.
          </p>
        </article>
        <article class="feature-card feature-card--media reveal">
          <img
            src="/mascot/03-error-sad.webp"
            width="531"
            height="524"
            loading="lazy"
            alt="Расстроенная пиксельная ласточка — иллюстрация словаря ложных друзей в ucimo"
          />
          <h3>Ложные друзья и ловушки</h3>
          <p>
            Отдельный словарь слов-обманщиков: например, «stolica» — это
            стул, а не столица, а «kuća» — дом, а не кухня.
          </p>
        </article>
      </div>
    </section>

    <section
      id="roadmap"
      class="section section--soft"
      aria-labelledby="roadmap-title"
    >
      <h2 id="roadmap-title" class="reveal">Программа курса</h2>
      <p class="section__lead reveal">
        60 уроков в четырёх готовых уровнях — от первого «здравствуйте» до падежей и уверенного разговора. Пятый уровень — чувства, работа, документы на боравак — на подходе.
      </p>
      <div class="roadmap">
        <img
          class="roadmap__art reveal"
          src="/mascot/10-traveling.webp"
          width="595"
          height="556"
          loading="lazy"
          alt="Пиксельная ласточка с чемоданом на карте маршрута — уровни курса сербского языка ucimo"
        />
        <ol class="levels">
          <li class="levels__item reveal">
            <span class="levels__badge">1</span>
            <div>
              <h3>Первый контакт</h3>
              <p class="levels__meta">16 уроков</p>
              <p>
                Алфавит и произношение, приветствия, рассказ о себе, семья,
                числа и покупки.
              </p>
            </div>
          </li>
          <li class="levels__item reveal">
            <span class="levels__badge">2</span>
            <div>
              <h3>Быт</h3>
              <p class="levels__meta">14 уроков</p>
              <p>
                Распорядок дня, дом, еда и кафе, город, погода и времена
                года.
              </p>
            </div>
          </li>
          <li class="levels__item reveal">
            <span class="levels__badge">3</span>
            <div>
              <h3>Связная речь</h3>
              <p class="levels__meta">12 уроков</p>
              <p>
                Прошедшее и будущее время, транспорт и направления,
                договорённости, тело и здоровье.
              </p>
            </div>
          </li>
          <li class="levels__item reveal">
            <span class="levels__badge">4</span>
            <div>
              <h3>Падежи в жизни</h3>
              <p class="levels__meta">18 уроков</p>
              <p>
                Род и число, прилагательные, все падежи и местоимения: заказ в
                кафе и магазине, аренда жилья, просьбы, договорённости с
                друзьями, переписка, даты.
              </p>
            </div>
          </li>
          <li class="levels__item reveal">
            <span class="levels__badge">5</span>
            <div>
              <h3>Жизнь на сербском</h3>
              <p class="levels__meta">18 уроков · скоро</p>
              <p>
                Чувства и мнения, звонки, бытовые проблемы, работа и
                собеседования, МУП и пријава адресе, региональные особенности
                языка.
              </p>
            </div>
          </li>
        </ol>
      </div>
    </section>

    <section id="why-easy" class="section" aria-labelledby="why-title">
      <h2 id="why-title" class="reveal">
        Почему сербский легко учить со знанием русского
      </h2>
      <p class="reveal">
        Сербский — южнославянский язык, русский — восточнославянский, но у
        них общие корни: падежи, виды глагола, много узнаваемой лексики.
        Многие слова понятны без перевода с первого дня.
      </p>
      <p class="reveal">
        Сложность — в «ложных друзьях»: словах, которые звучат знакомо, но
        значат другое. «Stolica» — это стул, а не столица; «kuća» — дом, а
        не кухня; «pravo» — прямо, а не право. В ucimo такие слова собраны в
        отдельный словарь ловушек, чтобы путаница не превращалась в
        привычку.
      </p>
      <p class="reveal">
        Ещё одна особенность — двойной алфавит: сербский одинаково законно
        пишут кириллицей (вуковицей, почти 1:1 с русской фонетикой) и
        латиницей. В курсе оба вводятся с первого урока, поэтому чтение
        вывесок, меню и документов не превращается в отдельную задачу.
      </p>
    </section>

    <section id="faq" class="section section--soft" aria-labelledby="faq-title">
      <h2 id="faq-title" class="reveal">Частые вопросы</h2>
      <div class="faq">
        <details
          v-for="item in faq"
          :key="item.q"
          class="faq__item reveal"
        >
          <summary>{{ item.q }}</summary>
          <p>{{ item.a }}</p>
        </details>
      </div>
    </section>

    <section id="reviews" class="section" aria-labelledby="reviews-title">
      <h2 id="reviews-title" class="reveal">Что говорят ученики</h2>
      <p class="section__lead reveal">
        Настоящие комментарии из Instagram и TikTok — без редактуры,
        орфография авторов сохранена.
      </p>

      <div class="reviews__slider reveal">
        <div class="reviews__track" role="list">
          <div
            v-for="r in reviews"
            :key="r.src"
            class="review-card"
            role="listitem"
          >
            <img
              :src="r.src"
              :width="r.width"
              :height="r.height"
              loading="lazy"
              :alt="r.alt"
            />
          </div>
        </div>

        <div class="reviews__controls">
          <button type="button" class="reviews__nav reviews__nav--prev" aria-label="Предыдущий отзыв">
            ‹
          </button>
          <div class="reviews__dots"></div>
          <button type="button" class="reviews__nav reviews__nav--next" aria-label="Следующий отзыв">
            ›
          </button>
        </div>
      </div>
    </section>

    <section class="final-cta">
      <img
        class="reveal"
        src="/mascot/12-pointing.webp"
        width="524"
        height="491"
        loading="lazy"
        alt=""
        aria-hidden="true"
      />
      <div>
        <h2>Начните учить сербский сегодня</h2>
        <p>Первый урок — алфавит и произношение. Бесплатно, без рекламы.</p>
        <a class="btn btn--primary" href="/register" data-goal="register_final">Изучить сербский</a>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
// Real screenshots of comments from Instagram/TikTok, as sent — shown as
// images rather than retyped text/avatars. Add a new entry (photo under
// public/reviews/, converted to webp with `cwebp -q 90`, matching every
// other image in public/) as more come in.
const reviews = [
  {
    src: '/reviews/review-notchristin.webp',
    width: 1179,
    height: 525,
    alt: 'Отзыв в Instagram от notchristin: «чудесное приложение! спасибо вам за работу»',
  },
  {
    src: '/reviews/review-p1gl1n.webp',
    width: 1179,
    height: 330,
    alt: 'Отзыв в Instagram от p1gl1n: «кайф, развивайте проект»',
  },
  {
    src: '/reviews/review-tombu.webp',
    width: 1179,
    height: 506,
    alt: 'Отзывы в Instagram от Насти Томбу и «по лицу бытовыми ударами»: «я вас искала, спасибо» и «спасибо большое!!! давно хотела, но не знала с чего начать»',
  },
  {
    src: '/reviews/review-helenabrus.webp',
    width: 1179,
    height: 472,
    alt: 'Отзыв в Instagram от helenabrus: «Спасибо Вам огромное!!»',
  },
  {
    src: '/reviews/review-kv.webp',
    width: 1179,
    height: 685,
    alt: 'Отзыв в Instagram: «спасибо вам, только за сегодня прошла 4 ваших урока, также конспектирую их»',
  },
]

const faq = [
  {
    q: 'С чего начать изучение сербского языка?',
    a: 'С алфавита и произношения — в ucimo это первый урок. Дальше уроки идут по нарастающей: приветствия, рассказ о себе, семья, числа и покупки — к концу уровня 1 вы уже можете вести простой бытовой разговор.',
  },
  {
    q: 'Сербский язык похож на русский?',
    a: 'Да, оба языка славянские и делят общие корни, падежи и виды глагола — часть лексики понятна интуитивно. Но есть и «ложные друзья»: слова вроде stolica (стул, не столица) или kuća (дом, не кухня) звучат знакомо, но значат другое — этому в курсе посвящён отдельный словарь ловушек.',
  },
  {
    q: 'Сколько времени нужно, чтобы выучить сербский с нуля?',
    a: 'Зависит от темпа, но курс рассчитан на 5–10 минут в день. В нём 60 уроков в четырёх готовых уровнях (пятый на подходе) — при регулярных занятиях уровня уверенного бытового общения реально достичь за несколько месяцев.',
  },
  {
    q: 'Кириллица или латиница — на чём писать по-сербски?',
    a: 'В Сербии официально используют оба алфавита, и на практике латиница встречается даже чаще. В ucimo оба вводятся с первого урока, чтобы одинаково свободно читать и вывески, и документы.',
  },
  {
    q: 'ucimo — платный сервис?',
    a: 'Учиться можно бесплатно. В отличие от большинства курсов сербского, где бесплатны только первые уроки или несколько слов в день, в ucimo три полных уровня (42 урока) и весь словарь для SRS-тренажёра открыты без подписки и лимитов. Следующие уровни открываются за зёрнышки — внутреннюю валюту, которую зарабатываешь заданиями и регулярными занятиями; платить деньгами не нужно.',
  },
  {
    q: 'Почему в Duolingo нет сербского языка?',
    a: 'Duolingo действительно не добавил сербский, хотя пользователи просят об этом годами. Для тех, кто не хочет ждать, есть отдельные сервисы для сербского — в том числе ucimo, где курс с нуля построен специально для русскоязычных.',
  },
  {
    q: 'Что такое интервальное повторение (SRS) и зачем оно нужно?',
    a: 'Это способ учить слова с наименьшими усилиями: алгоритм (в ucimo — SM-2) подкидывает слово на повторение именно тогда, когда оно вот-вот забудется, а не по случайному расписанию. Так слова закрепляются в долгосрочной памяти быстрее, чем при зубрёжке списками.',
  },
  {
    q: 'Подходит ли ucimo для переезда в Сербию?',
    a: 'Да — уровень 5 отдельно разбирает бытовые ситуации переезда: МУП и пријаву адресе, папирологию, банк, собеседования на работу, а также разницу между стандартным и новисадским сербским.',
  },
  {
    q: 'Нужен ли репетитор, или можно учить сербский самостоятельно?',
    a: 'Можно самостоятельно: каждое упражнение в ucimo проверяется автоматически, поэтому сразу видно, где ошибка, без участия преподавателя. Уроки построены так, чтобы новая грамматика и лексика вводились постепенно, от простого к сложному.',
  },
]

const title = 'Учи сербский язык с нуля — ucimo'
const description =
  'Веб-приложение для изучения сербского языка с нуля: 60 уроков с автопроверкой, три уровня бесплатно, SRS-тренажёр слов, кириллица и латиница, дашборд прогресса.'
const url = 'https://ucimo.ru/'
const ogImage = 'https://ucimo.ru/og-image.png'

useSeoMeta({
  title,
  description,
  ogType: 'website',
  ogTitle: title,
  ogDescription: description,
  ogUrl: url,
  ogSiteName: 'ucimo',
  ogLocale: 'ru_RU',
  ogImage,
  twitterCard: 'summary_large_image',
  twitterTitle: title,
  twitterDescription: description,
  twitterImage: ogImage,
})

useHead({
  link: [{ rel: 'canonical', href: url }],
  script: [
    {
      key: 'ld-json-index',
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@graph': [
          {
            '@type': 'WebSite',
            name: 'ucimo',
            url,
            description,
            inLanguage: 'ru',
          },
          {
            '@type': 'SoftwareApplication',
            name: 'ucimo',
            url,
            applicationCategory: 'EducationalApplication',
            operatingSystem: 'Web',
            description,
            offers: {
              '@type': 'Offer',
              price: '0',
              priceCurrency: 'RUB',
            },
            inLanguage: 'ru',
          },
          {
            '@type': 'FAQPage',
            mainEntity: faq.map((item) => ({
              '@type': 'Question',
              name: item.q,
              acceptedAnswer: {
                '@type': 'Answer',
                text: item.a,
              },
            })),
          },
        ],
      }),
    },
    {
      key: 'scroll-reveal',
      src: '/scroll-reveal.js',
      defer: true,
    },
    {
      key: 'accordion',
      src: '/accordion.js',
      defer: true,
    },
    {
      key: 'reviews-slider',
      src: '/reviews-slider.js',
      defer: true,
    },
  ],
})
</script>

<style scoped>
/* Scroll-reveal: enhanced by /scroll-reveal.js, which adds .is-hidden to
   .reveal elements before observing them. Without JS (or with
   prefers-reduced-motion) nothing gets .is-hidden, so content is visible
   by default — the animation is a progressive enhancement, never a gate. */
.reveal {
  transition:
    opacity 0.6s ease-out,
    transform 0.6s ease-out;
}

.reveal.is-hidden {
  opacity: 0;
  transform: translateY(24px);
}

.cards-row .reveal:nth-child(2) {
  transition-delay: 0.08s;
}
.cards-row .reveal:nth-child(3) {
  transition-delay: 0.16s;
}

.steps .reveal:nth-child(2) {
  transition-delay: 0.08s;
}
.steps .reveal:nth-child(3) {
  transition-delay: 0.16s;
}
.steps .reveal:nth-child(4) {
  transition-delay: 0.24s;
}

.features-grid .reveal:nth-child(2) {
  transition-delay: 0.06s;
}
.features-grid .reveal:nth-child(3) {
  transition-delay: 0.12s;
}
.features-grid .reveal:nth-child(4) {
  transition-delay: 0.18s;
}
.features-grid .reveal:nth-child(5) {
  transition-delay: 0.24s;
}
.features-grid .reveal:nth-child(6) {
  transition-delay: 0.3s;
}

.levels .reveal:nth-child(2) {
  transition-delay: 0.06s;
}
.levels .reveal:nth-child(3) {
  transition-delay: 0.12s;
}
.levels .reveal:nth-child(4) {
  transition-delay: 0.18s;
}
.levels .reveal:nth-child(5) {
  transition-delay: 0.24s;
}

.faq .reveal:nth-child(2) {
  transition-delay: 0.05s;
}
.faq .reveal:nth-child(3) {
  transition-delay: 0.1s;
}
.faq .reveal:nth-child(4) {
  transition-delay: 0.15s;
}
.faq .reveal:nth-child(5) {
  transition-delay: 0.2s;
}
.faq .reveal:nth-child(6) {
  transition-delay: 0.25s;
}
.faq .reveal:nth-child(7) {
  transition-delay: 0.3s;
}
.faq .reveal:nth-child(8) {
  transition-delay: 0.35s;
}
.faq .reveal:nth-child(9) {
  transition-delay: 0.4s;
}

/* Hero: animates immediately on load, no JS/scroll needed. */
@keyframes hero-in {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

.hero__text {
  animation: hero-in 0.7s ease-out both;
}

.hero__art {
  animation: hero-in 0.7s ease-out 0.12s both;
}

/* Gentle ambient float on every mascot illustration. */
@keyframes mascot-float {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-10px);
  }
}

/* Only elements that don't also carry .reveal get the continuous float —
   an infinite `animation` and the .reveal transition both drive `transform`,
   and would fight on the same element. */
.hero__art img,
.feature-card--media img {
  animation: mascot-float 4.5s ease-in-out infinite;
}

@media (prefers-reduced-motion: reduce) {
  .reveal,
  .reveal.is-hidden {
    transition: none;
    opacity: 1;
    transform: none;
  }

  .hero__text,
  .hero__art,
  .hero__art img,
  .feature-card--media img {
    animation: none;
  }
}

.eyebrow {
  color: var(--color-primary);
  font-weight: 700;
  font-size: 14px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin: 0 0 12px;
}

.hero {
  max-width: 1080px;
  margin: 0 auto;
  padding: 72px 20px 48px;
  display: grid;
  grid-template-columns: 1.1fr 0.9fr;
  gap: 32px;
  align-items: center;
}

.hero__title {
  font-size: clamp(32px, 5vw, 52px);
  font-weight: 800;
  letter-spacing: -0.02em;
  margin: 0 0 16px;
  color: var(--color-text);
}

.hero__subtitle {
  font-size: 18px;
  color: var(--color-muted);
  margin: 0 0 24px;
  max-width: 480px;
}

.hero__actions {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
}

.hero__note {
  color: var(--color-muted);
  font-size: 14px;
  margin: 0;
}

.hero__art {
  display: flex;
  justify-content: center;
}

.hero__art img {
  width: 100%;
  max-width: 320px;
  height: auto;
}

.section {
  max-width: 1080px;
  margin: 0 auto;
  padding: 64px 20px;
}

.section--soft {
  background: var(--color-primary-soft);
  max-width: none;
}

.section--soft > * {
  max-width: 1080px;
  margin-left: auto;
  margin-right: auto;
}

.section h2 {
  font-size: clamp(26px, 3.4vw, 34px);
  font-weight: 800;
  letter-spacing: -0.02em;
  margin: 0 0 16px;
  color: var(--color-text);
}

.section__lead {
  color: var(--color-muted);
  font-size: 17px;
  max-width: 640px;
  margin: 0 0 32px;
}

.section p {
  color: var(--color-text);
  line-height: 1.7;
}

.cards-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.mini-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  padding: 24px;
}

.mini-card h3 {
  margin: 0 0 8px;
  font-size: 17px;
  line-height: 1.3;
  min-height: 2.6em;
  color: var(--color-primary-dark);
}

.mini-card p {
  margin: 0;
  font-size: 15px;
  color: var(--color-muted);
}

.how__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 40px;
}

.how__art {
  width: 100px;
  height: auto;
  flex-shrink: 0;
}

.steps {
  list-style: none;
  margin: 0 auto;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}

.steps__item {
  position: relative;
}

.steps__shot {
  border-radius: 14px;
  overflow: hidden;
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  margin-bottom: 16px;
}

.steps__shot img {
  display: block;
  width: 100%;
  height: auto;
  aspect-ratio: 1 / 1;
  object-fit: cover;
  object-position: top;
}

.steps__num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--color-primary);
  color: var(--color-surface);
  font-weight: 700;
  margin-bottom: 12px;
}

.steps__item h3 {
  margin: 0 0 8px;
  font-size: 17px;
  line-height: 1.3;
  min-height: 2.6em;
  color: var(--color-text);
}

.steps__item p {
  margin: 0;
  font-size: 15px;
  color: var(--color-muted);
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.feature-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  padding: 24px;
}

.feature-card h3 {
  margin: 0 0 8px;
  font-size: 17px;
  line-height: 1.3;
  min-height: 2.6em;
  color: var(--color-primary-dark);
}

.feature-card p {
  margin: 0;
  font-size: 15px;
  color: var(--color-muted);
}

.feature-card--media {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 4px;
}

.feature-card--media img {
  width: 96px;
  height: 96px;
  object-fit: contain;
  margin-bottom: 8px;
}

.roadmap {
  display: grid;
  grid-template-columns: 240px 1fr;
  gap: 32px;
  align-items: start;
}

.roadmap__art {
  width: 100%;
  height: auto;
  position: sticky;
  top: 24px;
}

.levels {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.levels__item {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  padding: 20px 24px;
  display: grid;
  grid-template-columns: 56px 1fr;
  gap: 16px;
  align-items: start;
}

.levels__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 32px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--color-gold);
  color: var(--color-primary-dark);
  font-weight: 800;
  font-size: 13px;
}

.levels__item h3 {
  margin: 0 0 2px;
  font-size: 18px;
  color: var(--color-text);
}

.levels__meta {
  margin: 0 0 8px;
  font-size: 13px;
  color: var(--color-primary);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.levels__item p:last-child {
  margin: 0;
  font-size: 15px;
  color: var(--color-muted);
}

.faq {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.faq__item {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 18px 20px;
}

.faq__item summary {
  cursor: pointer;
  font-weight: 700;
  color: var(--color-text);
  list-style: none;
}

.faq__item summary::-webkit-details-marker {
  display: none;
}

.faq__item summary::before {
  content: '+';
  display: inline-block;
  width: 20px;
  color: var(--color-primary);
  font-weight: 800;
}

.faq__item[open] summary::before {
  content: '−';
}

.faq__item p {
  margin: 12px 0 0 20px;
  color: var(--color-muted);
  font-size: 15px;
}

.reviews__slider {
  position: relative;
}

.reviews__track {
  display: flex;
  gap: 20px;
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  scroll-padding-left: 4px;
  padding: 4px 4px 12px;
  margin: 0 -4px;
  /* Scrollbar stays hidden but the strip is still a native, fully
     keyboard/touch/trackpad-scrollable list without any JS at all —
     reviews-slider.js only adds the arrow buttons and dots on top. */
  scrollbar-width: none;
}

.reviews__track::-webkit-scrollbar {
  display: none;
}

.review-card {
  scroll-snap-align: start;
  flex: 0 0 auto;
  width: min(340px, 82vw);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  overflow: hidden;
}

.review-card img {
  display: block;
  width: 100%;
  height: auto;
}

.reviews__controls {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 20px;
}

.reviews__nav {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-primary-dark);
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  transition: background 0.15s ease, opacity 0.15s ease;
}

.reviews__nav:hover:not(:disabled) {
  background: var(--color-primary-soft);
}

.reviews__nav:disabled {
  opacity: 0.35;
  cursor: default;
}

/* No JS: the buttons render (harmlessly inert) but the track above is
   still fully swipeable/scrollable on its own, so nothing is lost. */
.reviews__dots {
  display: flex;
  align-items: center;
  gap: 8px;
}

.reviews__dot {
  width: 8px;
  height: 8px;
  padding: 0;
  border-radius: 50%;
  border: none;
  background: var(--color-border);
  cursor: pointer;
  transition: background 0.15s ease, transform 0.15s ease;
}

.reviews__dot.is-active {
  background: var(--color-primary);
  transform: scale(1.3);
}

.final-cta {
  max-width: 720px;
  margin: 0 auto;
  padding: 72px 20px 96px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 12px;
}

.final-cta img {
  width: 140px;
  height: auto;
  margin-bottom: 8px;
}

.final-cta h2 {
  font-size: clamp(26px, 3.4vw, 34px);
  font-weight: 800;
  letter-spacing: -0.02em;
  margin: 0;
  color: var(--color-text);
}

.final-cta p {
  color: var(--color-muted);
  margin: 0 0 8px;
}

@media (max-width: 860px) {
  .hero {
    grid-template-columns: 1fr;
    text-align: center;
    padding-top: 48px;
  }

  .hero__text {
    order: 2;
  }

  .hero__actions {
    align-items: center;
  }

  .hero__subtitle {
    margin-left: auto;
    margin-right: auto;
  }

  .hero__art {
    order: 1;
  }

  .cards-row,
  .steps,
  .features-grid {
    grid-template-columns: 1fr;
  }

  .how__header {
    flex-direction: column;
    text-align: center;
  }

  .how__art {
    width: 84px;
  }

  .roadmap {
    grid-template-columns: 1fr;
  }

  .roadmap__art {
    position: static;
    max-width: 200px;
    margin: 0 auto;
  }

  .levels__item {
    grid-template-columns: 1fr;
  }
}
</style>
