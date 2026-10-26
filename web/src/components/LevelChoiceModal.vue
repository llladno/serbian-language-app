<script setup lang="ts">
// The window that follows a paid level: «Какой уровень открываем?». A payment for «Уровень» is a
// credit, and the learner spends it here on any level they have not opened yet (the server sends
// exactly those: the tariffs quote's `choices`). Raised from the bank's return page, from the notice
// in the profile and from the tariffs screen; rendered once by AppNav (state in lib/levelChoiceModal).
//
// Two steps on purpose: pick a card, then confirm, because the choice cannot be undone. Closing
// without choosing keeps the paid level (the notice reminds). Docs:
// docs/superpowers/specs/2026-10-26-level-choice-design.md.
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Check, X } from 'lucide-vue-next'
import { ApiError } from '../api'
import { useCourseStore } from '../stores/course'
import { useTariffsStore } from '../stores/tariffs'
import { useWalletStore } from '../stores/wallet'
import { useLevelChoiceModal } from '../lib/levelChoiceModal'
import { grouped } from '../lib/tariffs'
import { pluralRu } from '../lib/plural'
import ModalShell from './ModalShell.vue'
import SeedIcon from './SeedIcon.vue'
import scene from '../assets/tariffs-promo-scene.webp'
import heart from '../assets/donate/heart.png'
import type { Order } from '../types'

const router = useRouter()
const tariffs = useTariffsStore()
const wallet = useWalletStore()
const courseStore = useCourseStore()
const { open, orderId, closeModal } = useLevelChoiceModal()

const selected = ref<string | null>(null)

// The list scrolls inside the window; it fades at the edge where there is more to see, because a phone shows
// no scrollbar. Re-read when the list appears, when the levels change and while it is scrolled.
const listEl = ref<HTMLElement | null>(null)
const moreAbove = ref(false)
const moreBelow = ref(false)
function readEdges() {
  const el = listEl.value
  if (!el) {
    moreAbove.value = moreBelow.value = false
    return
  }
  moreAbove.value = el.scrollTop > 1
  moreBelow.value = el.scrollTop + el.clientHeight < el.scrollHeight - 1
}
const busy = ref(false)
const error = ref('')
const done = ref<Order | null>(null)

watch(listEl, () => nextTick(readEdges))

// A fresh start every time the window is raised.
watch(open, (isOpen) => {
  if (isOpen) {
    selected.value = null
    error.value = ''
    done.value = null
  }
})

// The paid order to spend: the one the caller named (just back from the bank), else the oldest waiting.
const target = computed(() => orderId.value ?? tariffs.credits[0] ?? null)

// "Уровень 4 — Падежи в жизни" -> the number, and the name without it.
function split(title: string): { num: string; name: string } {
  const m = /^Уровень\s+(\d+)\s*[—–-]\s*(.+)$/.exec(title)
  return m ? { num: m[1], name: m[2] } : { num: '', name: title }
}

const levels = computed(() =>
  tariffs.choices.map((c, i) => {
    const { num, name } = split(c.title)
    // The shop says what is inside ("Открывает уровень 4: падежи, прилагательные").
    const text = wallet.productForPhase(c.id)?.description ?? ''
    const topics = text.replace(/^Открывает уровень \d+:\s*/i, '').trim()
    return {
      id: c.id,
      num: num || String(i + 1),
      name,
      topics: topics ? topics.charAt(0).toUpperCase() + topics.slice(1) : '',
      // a no-break space, so the number never ends a line without its word
      lessons: `${c.lessons} ${pluralRu(c.lessons, 'урок', 'урока', 'уроков')}`,
      first: i === 0,
      // the level before it, by name without the number: «Связная речь», not «Уровень 4 — Связная речь»
      after: c.after ? split(c.after).name : '',
    }
  }),
)

watch(levels, () => nextTick(readEdges))

const picked = computed(() => levels.value.find((l) => l.id === selected.value) ?? null)
const bonus = computed(() => done.value?.bonus ?? tariffs.levelBonus)
const openedName = computed(() => split(done.value?.phase_title ?? '').name || done.value?.phase_title || '')

function errorText(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.message === 'already_chosen') return 'Этот платёж уже потрачен на другой уровень'
    if (e.message === 'bad_phase') return 'Этот уровень уже нельзя выбрать. Выберите другой'
    if (e.message === 'not_claimable') return 'Для этого платежа выбрать уровень нельзя'
  }
  return 'Не получилось открыть уровень. Попробуйте ещё раз'
}

async function confirm() {
  if (!selected.value || target.value == null || busy.value) return
  busy.value = true
  error.value = ''
  try {
    done.value = await tariffs.chooseLevel(target.value, selected.value)
    // The course map, the wallet and the shop show the new level as open.
    void Promise.allSettled([courseStore.load(true), wallet.loadShop(), wallet.refresh()])
  } catch (e) {
    error.value = errorText(e)
    // The list of levels may have moved on (opened elsewhere): read it again.
    if (e instanceof ApiError && e.message === 'bad_phase') {
      selected.value = null
      void tariffs.load(true)
    }
  } finally {
    busy.value = false
  }
}

function toCourse() {
  closeModal()
  router.push('/course')
}
</script>

<template>
  <ModalShell :open="open" size="md" panel-class="overflow-hidden" @close="closeModal">
    <button
      class="icon-btn close absolute right-3 top-3 z-10"
      title="Закрыть"
      aria-label="Закрыть"
      data-test="choice-close"
      @click="closeModal"
    >
      <X :size="19" :stroke-width="2.25" />
    </button>

    <div class="art" aria-hidden="true"><img :src="scene" alt="" /></div>

    <div class="px-5 pb-5">
      <!-- Spent: what opened. -->
      <div v-if="done" class="space-y-4 text-center" data-test="choice-done">
        <div class="space-y-1">
          <p class="eyebrow">Оплата прошла</p>
          <h2 class="text-xl font-extrabold leading-snug">Уровень открыт</h2>
          <p class="text-sm font-semibold">{{ openedName }}</p>
        </div>
        <p v-if="bonus" class="got" data-test="choice-bonus">
          <span class="chip"><SeedIcon :size="14" /></span>
          <span>+{{ grouped(bonus) }} {{ wallet.currencyWord(bonus) }} уже у вас</span>
        </p>
        <p class="thanks" data-test="choice-thanks">
          Hvala lepo <img :src="heart" alt="" aria-hidden="true" class="heart" />
        </p>
        <button class="btn btn-primary w-full" data-test="choice-to-course" @click="toCourse">Перейти в курс</button>
      </div>

      <!-- Nothing left to open: the paid level waits for the next one. -->
      <div v-else-if="!levels.length" class="space-y-4 text-center" data-test="choice-waiting">
        <div class="space-y-1">
          <h2 class="text-xl font-extrabold leading-snug">Оплаченный уровень ждёт вас</h2>
          <p class="text-sm text-[var(--muted)]">
            Сейчас все платные уровни уже открыты. Как только выйдет новый, вы сможете выбрать его здесь.
          </p>
        </div>
        <p class="thanks">Hvala lepo <img :src="heart" alt="" aria-hidden="true" class="heart" /></p>
        <button class="btn btn-ghost w-full" @click="closeModal">Понятно</button>
      </div>

      <!-- The choice. -->
      <div v-else class="space-y-4">
        <div class="space-y-1 text-center">
          <p class="eyebrow">Оплата прошла. Hvala lepo <img :src="heart" alt="" aria-hidden="true" class="heart" /></p>
          <h2 class="text-xl font-extrabold leading-snug">Какой уровень открываем?</h2>
          <p class="text-sm text-[var(--muted)]">
            Он останется с вами навсегда<template v-if="tariffs.levelBonus">, а {{ grouped(tariffs.levelBonus) }} <SeedIcon :size="13" class="inline-seed" /> уже у вас на счету</template>.
          </p>
        </div>

        <ul
          ref="listEl"
          class="levels soft-scroll"
          :class="{ 'more-above': moreAbove, 'more-below': moreBelow }"
          role="radiogroup"
          aria-label="Уровень, который откроется"
          data-test="choice-list"
          @scroll.passive="readEdges"
        >
          <li v-for="l in levels" :key="l.id">
            <label class="level" :class="{ on: selected === l.id }" :data-test="`choice-card-${l.id}`">
              <input v-model="selected" type="radio" name="level" :value="l.id" class="sr-only" />
              <span class="num" aria-hidden="true">{{ l.num }}</span>
              <span class="text">
                <span class="name">{{ l.name }}</span>
                <span v-if="l.topics" class="topics">{{ l.topics }}</span>
                <span class="meta">
                  <span class="count">{{ l.lessons }}</span>
                  <span v-if="l.first" class="tag tag-first">Следующий по порядку</span>
                  <span v-else-if="l.after" class="tag">Лучше после «{{ l.after }}»</span>
                </span>
              </span>
              <span class="tick" aria-hidden="true"><Check :size="14" :stroke-width="3" /></span>
            </label>
          </li>
        </ul>

        <p v-if="error" class="error" role="alert" data-test="choice-error">{{ error }}</p>

        <div class="space-y-2">
          <button class="btn btn-primary w-full" :disabled="!picked || busy" data-test="choice-confirm" @click="confirm">
            <template v-if="busy">Открываем…</template>
            <template v-else-if="picked">Открыть «{{ picked.name }}»</template>
            <template v-else>Выберите уровень</template>
          </button>
          <p class="fine">Выбор нельзя изменить, зато уровень останется навсегда.</p>
        </div>
      </div>
    </div>
  </ModalShell>
</template>

<style scoped>
/* The picture melts into the card instead of ending in an edge. */
.art {
  height: 9.5rem;
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
/* The cross sits on the picture, which can be bright in a dark theme: give it its own plate. */
.close {
  background: color-mix(in srgb, var(--card) 72%, transparent);
  backdrop-filter: blur(4px);
}

.eyebrow {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--accent);
}
.heart {
  display: inline-block;
  height: 0.95em;
  width: auto;
  vertical-align: -0.14em;
  image-rendering: pixelated;
}
.inline-seed {
  display: inline-block;
  vertical-align: -0.2em;
}
.thanks {
  font-weight: 700;
}

/* The levels to pick from. Tall lists scroll inside, so the button stays on the screen. */
.levels {
  display: grid;
  gap: 0.5rem;
  max-height: min(46vh, 22rem);
  overflow-y: auto;
  padding: 2px;
  margin: 0 -2px;
}
/* The edge with more behind it melts away, so the list does not just end in a cut card. */
.levels.more-below {
  -webkit-mask-image: linear-gradient(to bottom, #000 calc(100% - 1.75rem), transparent);
  mask-image: linear-gradient(to bottom, #000 calc(100% - 1.75rem), transparent);
}
.levels.more-above {
  -webkit-mask-image: linear-gradient(to top, #000 calc(100% - 1.75rem), transparent);
  mask-image: linear-gradient(to top, #000 calc(100% - 1.75rem), transparent);
}
.levels.more-above.more-below {
  -webkit-mask-image: linear-gradient(to bottom, transparent, #000 1.75rem, #000 calc(100% - 1.75rem), transparent);
  mask-image: linear-gradient(to bottom, transparent, #000 1.75rem, #000 calc(100% - 1.75rem), transparent);
}
.level {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  cursor: pointer;
  border-radius: 1rem;
  padding: 0.7rem 0.8rem;
  background: var(--card);
  border: 1.5px solid var(--border);
  transition: border-color 0.15s ease, background-color 0.15s ease, box-shadow 0.15s ease;
}
.level:hover {
  border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
}
.level:has(input:focus-visible) {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
.level.on {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent-soft) 55%, var(--card));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 18%, transparent);
}
.num {
  display: flex;
  height: 2.5rem;
  width: 2.5rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 0.8rem;
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--accent);
  background: var(--accent-soft);
  transition: background-color 0.15s ease, color 0.15s ease;
}
.level.on .num {
  color: #fff;
  background: var(--accent);
}
.text {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: 0.15rem;
}
.name {
  font-weight: 800;
  line-height: 1.25;
}
.topics {
  font-size: 0.8125rem;
  line-height: 1.35;
  color: var(--muted);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.25rem 0.5rem;
  margin-top: 0.1rem;
  font-size: 0.75rem;
  color: var(--muted);
}
.count {
  font-weight: 700;
}
.tag {
  border-radius: 9999px;
  padding: 0.05rem 0.5rem;
  background: var(--bg-soft);
}
.tag-first {
  font-weight: 700;
  color: var(--accent);
  background: color-mix(in srgb, var(--accent) 14%, transparent);
}
.tick {
  display: flex;
  height: 1.5rem;
  width: 1.5rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  border: 1.5px solid var(--border);
  color: transparent;
  transition: all 0.15s ease;
}
.level.on .tick {
  color: #fff;
  border-color: var(--accent);
  background: var(--accent);
}

.error {
  border-radius: 0.75rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  color: var(--bad);
  background: color-mix(in srgb, var(--bad) 10%, transparent);
}
.fine {
  text-align: center;
  font-size: 0.75rem;
  color: var(--muted);
}

/* After choosing. */
.got {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  font-size: 0.9375rem;
  font-weight: 600;
}
.chip {
  display: flex;
  height: 1.75rem;
  width: 1.75rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.55rem;
  background: #f4df9f;
  border: 1px solid rgb(255 255 255 / 0.85);
  box-shadow: 0 1px 2px rgb(8 46 55 / 0.2);
}
</style>
