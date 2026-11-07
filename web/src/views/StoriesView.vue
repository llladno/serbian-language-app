<script setup lang="ts">
// Истории: the list of listening stories. The first one is free for everyone; the rest are bought for зёрнышки.
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Check, X } from 'lucide-vue-next'
import { api, ApiError } from '../api'
import { useWalletStore } from '../stores/wallet'
import { useTariffsStore } from '../stores/tariffs'
import { useCourseStore } from '../stores/course'
import { authErrorMessage } from '../lib/authErrors'
import { durationLabel } from '../lib/storyPlayer'
import { pluralRu } from '../lib/plural'
import { grouped, rub } from '../lib/tariffs'
import { trackTariff } from '../lib/tariffEvents'
import { useScript } from '../lib/useScript'
import ModalShell from '../components/ModalShell.vue'
import SeedIcon from '../components/SeedIcon.vue'
import scene from '../assets/tariffs-promo-scene.webp'
import heart from '../assets/donate/heart.png'
import foreverIcon from '../assets/promo/forever.webp'
import type { LessonRef, StoryListItem } from '../types'

const router = useRouter()
const wallet = useWalletStore()
const tariffs = useTariffsStore()
const course = useCourseStore()
const { show } = useScript()

const items = ref<StoryListItem[] | null>(null)
const loadError = ref<string | null>(null)
const buying = ref<StoryListItem | null>(null)
const buyError = ref<string | null>(null)
const busy = ref(false)

async function load() {
  loadError.value = null
  try {
    items.value = (await api.stories()).items
  } catch (e) {
    loadError.value = authErrorMessage(e)
  }
}
onMounted(() => {
  load()
  wallet.refresh()
  course.refresh() // which lessons are finished, for the warning before a story
})

// The order of what a learner meets after pressing a card: the зёрнышки first (a story that cannot be paid for yet
// opens the purchase window with the tariff, nothing else), then, once they can open it, the warning that they have
// not gone through the lessons the story is written on, then the purchase or the story itself.
function open(s: StoryListItem) {
  if (!s.owned && s.price > wallet.balance) showBuy(s)
  else if (needsWarning(s)) showWarn(s)
  else proceed(s)
}

function proceed(s: StoryListItem) {
  if (s.owned) router.push(`/stories/${s.slug}`)
  else showBuy(s)
}

function showBuy(s: StoryListItem) {
  buyError.value = null
  buying.value = s
  tariffs.load()
  api.storyEvent(s.slug, 'buy_view').catch(() => {})
}

// The lessons the learner has not finished, from the start of the course to the story's recommended lesson, in order.
function lessonsLeft(s: StoryListItem): LessonRef[] {
  const lessons = course.course?.lessons
  if (!s.recommended_lesson || !lessons) return []
  const end = lessons.findIndex((l) => l.id === s.recommended_lesson)
  if (end < 0) return []
  return lessons.slice(0, end + 1).filter((l) => l.status !== 'done')
}

// «Продолжить» is remembered per story on this device: the warning is a nudge, not a gate, and is not repeated.
const WARN_KEY = 'srpski.story.warned.'
function warnDismissed(slug: string): boolean {
  try {
    return localStorage.getItem(WARN_KEY + slug) === '1'
  } catch {
    return false
  }
}
function rememberWarned(slug: string) {
  try {
    localStorage.setItem(WARN_KEY + slug, '1')
  } catch {
    /* it just asks again next time */
  }
}
function needsWarning(s: StoryListItem): boolean {
  return !s.completed && !warnDismissed(s.slug) && lessonsLeft(s).length > 0
}

const warn = ref<StoryListItem | null>(null)
const left = computed(() => (warn.value ? lessonsLeft(warn.value) : []))
const nextLesson = computed(() => left.value[0] ?? null)

function showWarn(s: StoryListItem) {
  warn.value = s
  api.storyEvent(s.slug, 'lessons_warn_view').catch(() => {})
}

function goToLesson() {
  const s = warn.value
  const lesson = nextLesson.value
  if (!s || !lesson) return
  api.storyEvent(s.slug, 'lessons_warn_lesson').catch(() => {})
  warn.value = null
  const phase = course.course?.phases.find((p) => p.lessons.includes(lesson.id))
  router.push(phase?.locked ? '/course' : `/lesson/${lesson.id}`)
}

function continueAnyway() {
  const s = warn.value
  if (!s) return
  rememberWarned(s.slug)
  api.storyEvent(s.slug, 'lessons_warn_continue').catch(() => {})
  warn.value = null
  proceed(s)
}

// «16 фраз», and the best result once the story is listened to: «16 фраз · 6 из 6».
function meta(s: StoryListItem): string {
  const parts: string[] = []
  if (s.phrases > 0) parts.push(`${s.phrases} ${pluralRu(s.phrases, 'фраза', 'фразы', 'фраз')}`)
  if (s.completed && s.quiz_total > 0) parts.push(`${s.quiz_score} из ${s.quiz_total}`)
  return parts.join(' · ')
}

// How hard a story is to follow, a plate on every card and a filter on the right of the title (`difficulty` in the
// story's spec: 1 easy, 2 medium, 3 hard; a story without one counts as medium). Every story written so far stands
// on the words of the first level, so they are all easy; the other filters wait for the stories of the later levels.
const LEVELS: Record<number, { title: string; one: string }> = {
  1: { title: 'Лёгкие', one: 'Лёгкая' },
  2: { title: 'Средние', one: 'Средняя' },
  3: { title: 'Сложные', one: 'Сложная' },
}
const FILTERS = [{ n: 0, title: 'Все' }, ...[1, 2, 3].map((n) => ({ n, title: LEVELS[n].title }))]
const filter = ref(0)
const level = (s: StoryListItem) => s.difficulty || 2
const visible = computed(() => (items.value ?? []).filter((s) => filter.value === 0 || level(s) === filter.value))

const shortBy = computed(() => (buying.value ? Math.max(buying.value.price - wallet.balance, 0) : 0))

function earn() {
  buying.value = null
  router.push('/quests')
}

// Short of зёрнышки, a tariff is the way forward: «Уровень» brings зёрнышки as a gift. The window is something in
// between the one of a paid level (the picture, the price, «Не хватает») and the banner of the profile (the offer
// itself on a tinted panel with the gold button); it is offered while a level is left to open and no monthly
// limit holds the purchase back. The admin's «Воронка тарифов» counts each showing and each press.
const pitch = computed(
  () => !!buying.value && shortBy.value > 0 && tariffs.ready && !tariffs.levelFrom && !!tariffs.info?.level.next,
)
watch(pitch, (selling) => selling && trackTariff('view', 'story_modal'), { immediate: true })

function toTariffs() {
  trackTariff('click', 'story_modal')
  buying.value = null
  router.push('/tariffs')
}

async function buy() {
  const s = buying.value
  if (!s || busy.value) return
  busy.value = true
  buyError.value = null
  try {
    await api.buyStory(s.slug)
    await wallet.refresh()
    buying.value = null
    router.push(`/stories/${s.slug}`)
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : ''
    buyError.value =
      msg === 'insufficient_funds' ? 'Не хватает зёрнышек' : msg === 'already_owned' ? 'Эта история уже ваша' : authErrorMessage(e)
    if (msg === 'already_owned') await load()
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="mx-auto w-full max-w-2xl space-y-4 p-4">
    <header class="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
      <div>
        <h1 class="text-2xl font-extrabold">Истории</h1>
        <p class="text-sm text-[var(--muted)]">Короткие диалоги из жизни в Сербии: учат понимать живую речь на слух.</p>
      </div>
      <div class="filter" role="group" aria-label="Сложность" data-test="stories-filter">
        <button
          v-for="f in FILTERS"
          :key="f.n"
          type="button"
          class="chip"
          :aria-pressed="filter === f.n"
          :data-test="`stories-filter-${f.n}`"
          @click="filter = f.n"
        >
          {{ f.title }}
        </button>
      </div>
    </header>

    <p v-if="loadError" class="card p-4 text-sm" role="alert">{{ loadError }}</p>
    <p v-else-if="items === null" class="text-sm text-[var(--muted)]">Загружаем…</p>
    <p v-else-if="items.length === 0" class="card p-5 text-sm text-[var(--muted)]" data-test="stories-empty">
      Историй пока нет. Скоро появятся.
    </p>

    <p v-else-if="visible.length === 0" class="card p-5 text-sm text-[var(--muted)]" data-test="stories-filter-empty">
      Таких историй пока нет. Скоро появятся.
    </p>

    <ul v-else class="grid gap-4 sm:grid-cols-2" data-test="stories-list">
      <li v-for="(s, i) in visible" :key="s.slug">
        <button
          type="button"
          class="card poster relative block w-full overflow-hidden bg-[var(--bg-soft)] text-left transition hover:-translate-y-0.5"
          data-test="story-card"
          :data-tour="i === 0 ? 'story' : undefined"
          @click="open(s)"
        >
          <img v-if="s.cover" :src="s.cover" alt="" class="absolute inset-0 h-full w-full object-cover" style="image-rendering: pixelated" />
          <div class="shade absolute inset-0" aria-hidden="true"></div>
          <div class="absolute inset-x-3 top-3 flex items-start justify-between gap-2">
            <span class="pill" :class="{ 'pill-gold': !s.completed && !s.is_demo && !s.owned, 'pill-ok': s.completed }" data-test="story-badge">
              <template v-if="s.completed"><Check :size="14" :stroke-width="3" /> Прослушано</template>
              <template v-else-if="s.is_demo">Бесплатно</template>
              <template v-else-if="s.owned">Открыта</template>
              <template v-else><SeedIcon :size="13" />{{ s.price }}</template>
            </span>
            <span class="pill" data-test="story-time">{{ durationLabel(s.duration_ms) }} с заданиями</span>
          </div>
          <div class="absolute inset-x-4 bottom-3.5 flex items-end justify-between gap-3 text-white">
            <div class="min-w-0 space-y-0.5">
              <p class="text-xl font-extrabold leading-tight">{{ show(s.title_sr) }}</p>
              <p class="text-sm opacity-90">{{ s.title_ru }}</p>
              <p v-if="meta(s)" class="text-xs font-semibold opacity-90" data-test="story-meta">{{ meta(s) }}</p>
            </div>
            <!-- The difficulty sits in the lower right corner of the poster. -->
            <span class="level-plate shrink-0" :class="`level-${level(s)}`" data-test="story-level">{{ LEVELS[level(s)].one }}</span>
          </div>
        </button>
      </li>
    </ul>

    <ModalShell :open="!!warn" panel-class="space-y-3 p-5 text-center" size="sm" @close="warn = null">
      <div v-if="warn" class="space-y-3" data-test="story-warn">
        <p class="text-lg font-extrabold">Многое в истории может быть непонятно</p>
        <p class="text-sm text-[var(--muted)]" data-test="story-warn-text">
          В ней слова из уроков, которые вы ещё не прошли. Рекомендуем сначала пройти
          {{ left.length }} {{ pluralRu(left.length, 'урок', 'урока', 'уроков') }}.
        </p>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button type="button" class="btn btn-primary flex-1" data-test="story-warn-lesson" @click="goToLesson">Пройти урок</button>
          <button type="button" class="btn btn-ghost flex-1" data-test="story-warn-continue" @click="continueAnyway">Продолжить</button>
        </div>
      </div>
    </ModalShell>

    <ModalShell
      :open="!!buying"
      :panel-class="pitch ? 'overflow-hidden text-center' : 'p-5 text-center'"
      size="sm"
      @close="buying = null"
    >
      <template v-if="buying">
        <template v-if="pitch">
          <button type="button" class="icon-btn close absolute right-3 top-3 z-10" title="Закрыть" aria-label="Закрыть" @click="buying = null">
            <X :size="19" :stroke-width="2.25" />
          </button>
          <div class="art" aria-hidden="true"><img :src="scene" alt="" /></div>
        </template>

        <div class="space-y-3" :class="{ 'px-5 pb-5 pt-1': pitch }" data-test="story-modal">

          <template v-if="shortBy === 0">
            <p class="flex items-center justify-center gap-2 text-3xl font-extrabold tabular-nums" data-test="story-price">
              <SeedIcon :size="26" />
              {{ buying.price }}
            </p>
            <p class="text-sm text-[var(--muted)]" data-test="story-enough">
              После покупки останется {{ wallet.amount(wallet.balance - buying.price) }}.
            </p>
          </template>
          <!-- Short of зёрнышки: the shortfall is the one number, and the seed itself stands in for the word. -->
          <p v-else class="lack" data-test="story-short">
            Вам не хватает <b class="lack-num">{{ shortBy }}</b>
            <SeedIcon :size="30" />
            <span class="sr-only">{{ wallet.currencyWord(shortBy) }}</span>
          </p>
          <p v-if="buyError" class="text-sm text-[var(--bad)]" role="alert">{{ buyError }}</p>

          <section v-if="pitch" class="offer" data-test="story-offer">
            <p class="offer-title" data-test="story-offer-title">
              <template v-if="tariffs.levelBonus">
                {{ grouped(tariffs.levelBonus) }}
                <span class="seed-inline"><SeedIcon :size="20" /></span>
                <span class="sr-only">{{ wallet.currencyWord(tariffs.levelBonus) }}</span>
                и&nbsp;уровень в&nbsp;подарок
              </template>
              <template v-else>Следующий уровень навсегда</template>
            </p>
            <ul class="perks">
              <li>
                <span class="ico"><img :src="foreverIcon" alt="" aria-hidden="true" class="px forever" /></span>
                Навсегда, без подписки
              </li>
              <li v-if="tariffs.levelBonus">
                <span class="ico ico-seed"><SeedIcon :size="14" /></span>
                Зёрнышки — на истории и уровни
              </li>
            </ul>
            <!-- The big gold button names what the learner is short of: the зёрнышки. -->
            <button type="button" class="btn btn-seed cta mt-3 w-full" data-test="story-tariff" @click="toTariffs">
              {{ tariffs.levelBonus ? 'Получить зёрнышки' : 'Получить уровень' }}
            </button>
            <p class="fine">
              <span data-test="story-price-fine">
                {{ rub(tariffs.levelNow) }}<s v-if="tariffs.levelDiscount" class="was">{{ rub(tariffs.levelRegular) }}</s>
                один раз, подписки нет</span>.
              Hvala lepo <img :src="heart" alt="" aria-hidden="true" class="heart" />
            </p>
          </section>

          <template v-else>
            <div class="flex flex-col gap-2 sm:flex-row">
              <button
                type="button"
                class="btn btn-primary flex-1"
                :disabled="busy || shortBy > 0"
                data-test="story-buy"
                @click="buy"
              >
                Открыть
              </button>
              <button type="button" class="btn btn-ghost flex-1" @click="buying = null">Не сейчас</button>
            </div>
          </template>
          <button v-if="shortBy > 0" type="button" class="quiet-link" data-test="story-earn" @click="earn">
            Заработать зёрнышки
          </button>
        </div>
      </template>
    </ModalShell>
  </div>
</template>

<style scoped>
/* The filter on the right of the title: a few small chips. */
.filter {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}
.chip {
  border-radius: 999px;
  padding: 0.3rem 0.75rem;
  font-size: 0.8125rem;
  font-weight: 700;
  background: var(--bg-soft);
  color: var(--muted);
  transition: background-color 0.15s ease, color 0.15s ease;
}
.chip[aria-pressed='true'] {
  background: var(--accent);
  color: #fff;
}
/* The plate of the difficulty on a card: a light plate with a dot, green for easy, gold for medium, red for hard. */
.level-plate {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border-radius: 999px;
  padding: 0.1rem 0.55rem 0.1rem 0.45rem;
  font-size: 0.6875rem;
  font-weight: 800;
  color: #172326;
  background: rgba(255, 255, 255, 0.9);
}
.level-plate::before {
  content: '';
  height: 0.5rem;
  width: 0.5rem;
  border-radius: 999px;
  background: var(--good);
}
.level-plate.level-2::before {
  background: var(--warning);
}
.level-plate.level-3::before {
  background: var(--bad);
}
/* A poster: the frame fills the card, the title lies on its dark lower edge, the badges on its top. */
.poster {
  aspect-ratio: 4 / 3.2;
}
.shade {
  background: linear-gradient(to top, rgba(12, 24, 26, 0.82) 0%, rgba(12, 24, 26, 0.35) 42%, transparent 66%);
}
.pill {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  white-space: nowrap;
  border-radius: 999px;
  padding: 0.25rem 0.7rem;
  font-size: 0.75rem;
  font-weight: 800;
  background: color-mix(in srgb, var(--card) 90%, transparent);
  backdrop-filter: blur(6px);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.14);
  color: var(--fg);
}
/* Gold is the colour of the currency: a story to be bought for зёрнышки. */
.pill-gold {
  background: var(--seed);
  color: var(--seed-ink);
}
.pill-ok {
  color: var(--good);
}
/* Earning is a quiet way out under the buttons, not a third button (same as the level modal). */
.quiet-link {
  display: block;
  margin: 0 auto;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--muted);
  text-decoration: underline;
  text-underline-offset: 3px;
}
.quiet-link:hover {
  color: var(--fg);
}

/* The window of a story the learner is short of зёрнышки for: the same look as the level window (the picture melts into the
   card, the cross has its own plate) with the offer of a tariff on a tinted panel under the shortfall. */
.art {
  height: 8.5rem;
  overflow: hidden;
  background: linear-gradient(135deg, var(--accent-soft), var(--card));
}
.art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 50% 20%;
  mask-image: linear-gradient(to bottom, #000 62%, transparent);
  -webkit-mask-image: linear-gradient(to bottom, #000 62%, transparent);
}
.close {
  background: color-mix(in srgb, var(--card) 72%, transparent);
  backdrop-filter: blur(4px);
}
/* «Вам не хватает 9 [зёрнышко]»: one line, the seed stands where the word would be. */
.lack {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  font-size: 1.125rem;
  font-weight: 700;
}
.lack-num {
  font-size: 1.75rem;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}
.offer {
  border-radius: 16px;
  padding: 0.9rem 1rem;
  text-align: left;
  background: color-mix(in srgb, var(--accent-soft) 70%, var(--card));
}
.offer-title {
  font-size: 1.15rem;
  font-weight: 800;
  line-height: 1.3;
  text-align: center;
}
/* The seed glyph sits in the line of text, after the number. */
.seed-inline {
  display: inline-block;
  vertical-align: -0.2em;
  margin: 0 0.1em;
}
.perks {
  margin-top: 0.625rem;
  display: grid;
  gap: 0.3rem;
  font-size: 0.875rem;
  font-weight: 600;
}
.perks li {
  display: flex;
  align-items: center;
  gap: 0.55rem;
}
/* A soft light tile, like an app icon (the same in both themes: the pixel icons have a dark outline). */
.ico {
  display: flex;
  height: 1.75rem;
  width: 1.75rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 0.55rem;
  background: #e4f5f0;
  border: 1px solid rgb(255 255 255 / 0.85);
  box-shadow: 0 1px 2px rgb(8 46 55 / 0.2);
}
.ico-seed {
  background: #f4df9f;
}
.px {
  display: block;
  image-rendering: pixelated;
}
.forever {
  width: 1.5rem;
  height: auto;
}
.cta {
  padding: 0.6rem 0.9rem;
  font-size: 0.9rem;
  font-weight: 700;
}
.fine {
  margin-top: 0.35rem;
  font-size: 0.75rem;
  color: var(--muted);
  text-align: center;
}
/* The regular price, struck through next to the first-purchase price. */
.was {
  margin-left: 0.4rem;
  font-size: 0.8em;
  font-weight: 600;
  opacity: 0.65;
}
/* The pixel heart sits on the text line like a letter would. */
.heart {
  display: inline-block;
  height: 0.95em;
  width: auto;
  margin-left: 0.1em;
  vertical-align: -0.14em;
  image-rendering: pixelated;
}
</style>
