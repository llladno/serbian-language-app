<script setup lang="ts">
// The swallow's tour over the real interface: the screen is dimmed, the buttons she is talking about stay lit in a
// blinking pixel frame, and her line comes in a pixel speech bubble. A tap anywhere moves on (the buttons underneath
// are not pressed), but only once the line has been typed out and a short beat has passed, so a double tap cannot skip
// a line unread. Tour state lives in lib/onboarding.ts.
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useOnboarding } from '../lib/onboarding'
import { useScript } from '../lib/useScript'
import { useWalletStore } from '../stores/wallet'
import { SEED, pileFor } from '../lib/seeds'
import SparkleBurst from './SparkleBurst.vue'
import guideImg from '../assets/support-mascot.webp'
import happyImg from '../assets/mascot-verify-ok.webp'

const tour = useOnboarding()
const { show } = useScript()
const route = useRoute()
const router = useRouter()
const wallet = useWalletStore()

interface Box {
  x: number
  y: number
  w: number
  h: number
  // a thing on the screen she is talking about, as opposed to a button of the menu
  focus?: boolean
}

// The pause after a line has finished typing before «Дальше» works.
const BEAT_MS = 600
// A line takes about a second and a half to type, however long it is.
const TYPE_MS = 1500
const PAD = 6

const reduced = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches

const text = computed(() => show(tour.step.value?.text ?? ''))
const gloss = computed(() => show(tour.step.value?.tr ?? ''))
const shown = ref('')
const typing = ref(false)
const unlocked = ref(false)

let typeTimer: ReturnType<typeof setTimeout> | undefined
let beatTimer: ReturnType<typeof setTimeout> | undefined

function clearTimers() {
  clearTimeout(typeTimer)
  clearTimeout(beatTimer)
}

function lockAfterTyping() {
  typing.value = false
  beatTimer = setTimeout(() => (unlocked.value = true), BEAT_MS)
}

function startTyping() {
  clearTimers()
  unlocked.value = false
  const full = text.value
  if (reduced) {
    shown.value = full
    lockAfterTyping()
    return
  }
  shown.value = ''
  typing.value = true
  const chars = Array.from(full)
  const per = Math.max(10, Math.min(28, TYPE_MS / Math.max(chars.length, 1)))
  let i = 0
  const tick = () => {
    i++
    shown.value = chars.slice(0, i).join('')
    if (i >= chars.length) lockAfterTyping()
    else typeTimer = setTimeout(tick, per)
  }
  typeTimer = setTimeout(tick, per)
}

function finishTyping() {
  clearTimers()
  shown.value = text.value
  lockAfterTyping()
}

watch(
  () => [tour.open.value, tour.step.value?.id, tour.mode.value] as const,
  ([open]) => {
    if (open) startTyping()
    else clearTimers()
  },
  { immediate: true },
)

// --- the lit buttons -------------------------------------------------------

// `goal` is where the lit buttons are; `boxes` is what is drawn, which glides there so the light slides from one
// button to the next instead of jumping.
const goal = ref<Box[]>([])
const boxes = ref<Box[]>([])
const vw = ref(window.innerWidth)
const vh = ref(window.innerHeight)

// The nav exists twice (top bar on wide screens, tab bar on phones) and CSS hides one; the visible one has a size.
function visibleEl(key: string): HTMLElement | null {
  for (const el of document.querySelectorAll<HTMLElement>(`[data-tour="${key}"]`)) {
    const r = el.getBoundingClientRect()
    if (r.width > 0 && r.height > 0) return el
  }
  return null
}

function boxFor(key: string, focus = false): Box | null {
  const el = visibleEl(key)
  if (!el) return null
  const r = el.getBoundingClientRect()
  // A tall thing (the profile's progress) is lit only as far as the top of it, so the swallow still has room to stand.
  const h = focus ? Math.min(r.height, window.innerHeight * 0.4) : r.height
  return { x: r.left - PAD, y: r.top - PAD, w: r.width + PAD * 2, h: h + PAD * 2, focus }
}

// Buttons that touch (the words and the dictionary side by side) are lit as one.
function mergeTouching(list: Box[]): Box[] {
  const out = [...list]
  const GAP = 8
  for (let again = true; again; ) {
    again = false
    for (let i = 0; i < out.length && !again; i++) {
      for (let j = i + 1; j < out.length && !again; j++) {
        const a = out[i]
        const b = out[j]
        if (a.focus || b.focus) continue
        if (a.x < b.x + b.w + GAP && b.x < a.x + a.w + GAP && a.y < b.y + b.h + GAP && b.y < a.y + a.h + GAP) {
          const x = Math.min(a.x, b.x)
          const y = Math.min(a.y, b.y)
          out.splice(j, 1)
          out[i] = { x, y, w: Math.max(a.x + a.w, b.x + b.w) - x, h: Math.max(a.y + a.h, b.y + b.h) - y }
          again = true
        }
      }
    }
  }
  return out
}

function sameBoxes(a: Box[], b: Box[]) {
  return a.length === b.length && a.every((p, i) => p.x === b[i].x && p.y === b[i].y && p.w === b[i].w && p.h === b[i].h)
}

// A box with no size, in the middle of `b`: where a light grows from and shrinks to.
function pinpoint(b: Box): Box {
  return { x: b.x + b.w / 2, y: b.y + b.h / 2, w: 0, h: 0, focus: b.focus }
}

// Eases `from` towards `to`; `f` is the share of the way covered in this frame.
function glide(from: Box, to: Box, f: number): Box {
  const k = (a: number, b: number) => (Math.abs(b - a) < 0.4 ? b : a + (b - a) * f)
  return { x: k(from.x, to.x), y: k(from.y, to.y), w: k(from.w, to.w), h: k(from.h, to.h), focus: to.focus }
}

// The thing she is talking about may be further down the page than the screen shows: bring it up once per line.
let scrolledFor = ''
function revealFocus() {
  const step = tour.step.value
  if (!step?.focus?.length || scrolledFor === step.id) return
  const el = visibleEl(step.focus[0])
  if (!el) return
  scrolledFor = step.id
  const r = el.getBoundingClientRect()
  if (r.top < 80 || r.top > window.innerHeight * 0.4) {
    // At once, not smoothly: the light is drawn from where things are, and a slow scroll would leave it trailing behind.
    window.scrollTo({ top: window.scrollY + r.top - 96 })
  }
}

let raf = 0
let lastTs = 0
function measure(ts = 0) {
  revealFocus()
  // The same speed on a 60 and a 120 Hz screen: the light covers most of the distance in about a fifth of a second.
  const f = 1 - Math.exp(-Math.min(ts - lastTs || 16, 50) / 70)
  lastTs = ts
  const nav = (tour.step.value?.targets ?? []).map((k) => boxFor(k)).filter((b): b is Box => !!b)
  const things = (tour.step.value?.focus ?? []).map((k) => boxFor(k, true)).filter((b): b is Box => !!b)
  // The light never runs under her bubble: a tall thing is lit down to where the bubble starts.
  const stand = panelEl.value?.getBoundingClientRect()
  if (stand && stand.top > 0) {
    for (const t of things) if (stand.top > t.y + 100) t.h = Math.min(t.h, stand.top - 14 - t.y)
  }
  const want = [...things, ...mergeTouching(nav)]
  if (!sameBoxes(goal.value, want)) goal.value = want
  const cur = boxes.value
  const next: Box[] = []
  for (let i = 0; i < Math.max(want.length, cur.length); i++) {
    const to = want[i]
    const from = cur[i]
    if (to) next.push(reduced ? to : glide(from ?? pinpoint(to), to, f))
    else if (from && !reduced) {
      // A light with nothing to light any more shrinks away.
      const b = glide(from, pinpoint(from), f)
      if (b.w > 8) next.push(b)
    }
  }
  if (!sameBoxes(cur, next)) boxes.value = next
  vw.value = window.innerWidth
  vh.value = window.innerHeight
  if (laterEl.value) later.value = { w: laterEl.value.offsetWidth, h: laterEl.value.offsetHeight }
  if (tour.open.value) raf = requestAnimationFrame(measure)
}

watch(
  () => tour.open.value,
  (open) => {
    cancelAnimationFrame(raf)
    if (open) {
      lastTs = 0
      raf = requestAnimationFrame(measure)
    }
    else {
      goal.value = []
      boxes.value = []
      scrolledFor = ''
    }
  },
  { immediate: true },
)
// While the tour is up the page underneath is switched off: nothing in it can take focus (a field on the words screen
// would otherwise pop the keyboard up and shove the layout), and the tour's own keys are the only ones that count.
function freezePage(on: boolean) {
  const main = document.querySelector('main')
  if (!main) return
  if (on) {
    main.setAttribute('inert', '')
    document.body.dataset.tour = 'open'
    ;(document.activeElement as HTMLElement | null)?.blur?.()
  } else {
    main.removeAttribute('inert')
    delete document.body.dataset.tour
  }
}
watch(() => tour.open.value, freezePage, { immediate: true })
onBeforeUnmount(() => {
  cancelAnimationFrame(raf)
  clearTimers()
  freezePage(false)
})

// --- where the swallow stands ---------------------------------------------

const PANEL_MAX = 440
const panelStyle = computed(() => {
  const w = Math.min(PANEL_MAX, vw.value - 32)
  const thing = goal.value.find((b) => b.focus)
  const first = goal.value[0]
  let left = (vw.value - w) / 2
  const style: Record<string, string> = { width: `${w}px` }
  if (!thing && first && vw.value >= 640) {
    // On a wide screen the panel stands under (or over) the first lit button instead of in the middle of nowhere.
    left = Math.min(Math.max(first.x + first.w / 2 - w / 2, 16), vw.value - w - 16)
  }
  style.left = `${left}px`
  const bottomSlot = vw.value < 640 ? 'calc(6.25rem + env(safe-area-inset-bottom))' : 'calc(2rem + env(safe-area-inset-bottom))'
  if (thing) {
    // She stands on whichever side of the thing she is talking about has more room, so it stays in view.
    if (vh.value - (thing.y + thing.h) >= thing.y) style.bottom = bottomSlot
    else style.top = 'calc(4.5rem + env(safe-area-inset-top))'
  } else if (first && first.y < vh.value / 2) {
    // A button near the top (the wide nav): the swallow stands below it.
    style.top = `${first.y + first.h + 20}px`
  } else if (first) {
    // A button near the bottom (the phone's tab bar): the swallow stands above it.
    style.bottom = `${vh.value - first.y + 20}px`
  } else {
    // Nothing lit (the opening line, the finale): stand clear of the phone's tab bar rather than on top of it.
    style.bottom = bottomSlot
  }
  return style
})

// «Отложить на потом» sits in the top right corner, out of the way of the line. When the swallow is lighting something
// up there (the seed chip), it steps aside to the left of it.
const laterEl = ref<HTMLButtonElement | null>(null)
const panelEl = ref<HTMLDivElement | null>(null)
const later = ref({ w: 190, h: 40 })
const laterStyle = computed(() => {
  const { w, h } = later.value
  const top = 12
  let right = vw.value - 12
  for (let pass = 0; pass < 2; pass++) {
    for (const b of goal.value) {
      const overlapsRow = b.y < top + h + 8 && b.y + b.h > top - 8
      if (overlapsRow && b.x < right && b.x + b.w > right - w) right = Math.max(b.x - 20, w + 8)
    }
  }
  const atTop = 'calc(12px + env(safe-area-inset-top))'
  // Pinned by its right edge, so the link keeps clear of the lit button whatever its width turns out to be (the pixel
  // font arrives late and the measured width lags a frame behind).
  return { top: atTop, right: `${vw.value - right}px` }
})

// --- input ---------------------------------------------------------------

// The tour walks the learner through the app: each line that has a screen takes them there, and when it ends they are
// put back where they started.
let origin = ''
watch(
  () => tour.open.value,
  (open) => {
    if (open) origin = route.fullPath
  },
  { immediate: true },
)
watch(
  () => [tour.open.value, tour.step.value?.id] as const,
  ([open]) => {
    const to = tour.step.value?.route
    if (open && to && route.path !== to) void router.push(to)
  },
)
function backHome() {
  if (origin && route.fullPath !== origin) void router.push(origin)
}
async function postponeTour() {
  await tour.postpone()
  backHome()
}

function advance() {
  if (!tour.open.value) return
  if (typing.value) {
    finishTyping()
    return
  }
  if (!unlocked.value) return
  if (tour.isLast.value) {
    tour.close()
    backHome()
  } else tour.next()
}

// A tap on the empty screen or on the bubble moves on, except at the opening offer, which wants a real answer.
function onTap() {
  if (tour.isOffer.value) {
    if (typing.value) finishTyping()
    return
  }
  advance()
}

function onKey(e: KeyboardEvent) {
  if (!tour.open.value) return
  if (e.key === 'Escape') {
    e.preventDefault()
    void postponeTour()
  } else if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowRight') {
    // The opening offer wants a real answer: Enter and Space press the focused «Показать» themselves.
    if (tour.isOffer.value) return
    e.preventDefault()
    onTap()
  }
}
window.addEventListener('keydown', onKey)
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

// Opening a lesson (or any screen without the nav) ends the tour without an answer: it comes back next time.
watch(
  () => route.fullPath,
  () => {
    if (tour.open.value && (route.meta?.hideTabBar || route.meta?.fullscreen)) tour.close()
  },
)

const primary = ref<HTMLButtonElement | null>(null)
watch(
  () => [tour.open.value, tour.step.value?.id],
  async () => {
    if (!tour.open.value) return
    await nextTick()
    primary.value?.focus({ preventScroll: true })
  },
)

const mascot = computed(() => (tour.step.value?.mascot === 'happy' ? happyImg : guideImg))
const maskId = 'tour-mask'
const isHappy = computed(() => tour.step.value?.mascot === 'happy')

// The finale: seeds fall over the whole screen, pixel stars twinkle round the heap.
const pile = computed(() => pileFor(wallet.balance))
const prize = computed(() => (tour.reward.value ? `+${wallet.amount(tour.reward.value)}` : `Всего: ${wallet.amount(wallet.balance)}`))
const drops = [
  { x: 6, d: 0, t: 3.4 },
  { x: 18, d: 1.1, t: 4 },
  { x: 29, d: 0.4, t: 3.1 },
  { x: 41, d: 1.7, t: 3.8 },
  { x: 52, d: 0.8, t: 3.3 },
  { x: 63, d: 2, t: 4.1 },
  { x: 74, d: 0.2, t: 3.6 },
  { x: 85, d: 1.4, t: 3.2 },
  { x: 94, d: 0.6, t: 3.9 },
  { x: 35, d: 2.4, t: 3.5 },
]
// Where the stars twinkle: scattered over the upper part of the screen, clear of the swallow's panel at the bottom.
const starSpots = [
  [10, 12], [24, 30], [38, 8], [50, 22], [62, 6], [76, 28], [90, 14], [16, 46], [32, 52],
  [68, 50], [84, 44], [45, 40], [6, 30], [94, 36], [56, 58], [28, 18], [72, 16], [20, 62],
].map(([left, top]) => ({ left, top }))
</script>

<template>
  <Transition name="tour" appear>
    <div
      v-if="tour.open.value && tour.step.value"
      class="tour"
      role="dialog"
      aria-modal="true"
      aria-label="Знакомство с приложением"
      data-test="tour"
      @click="onTap"
    >
      <svg class="dim" :width="vw" :height="vh" aria-hidden="true">
        <defs>
          <mask :id="maskId">
            <rect :width="vw" :height="vh" fill="#fff" />
            <rect v-for="(b, i) in boxes" :key="i" :x="b.x" :y="b.y" :width="b.w" :height="b.h" rx="16" fill="#000" />
          </mask>
        </defs>
        <rect :width="vw" :height="vh" fill="rgba(7, 28, 27, 0.7)" :mask="`url(#${maskId})`" />
      </svg>
      <div
        v-for="(b, i) in boxes"
        :key="'f' + i"
        class="frame"
        data-test="tour-frame"
        :style="{ left: b.x + 'px', top: b.y + 'px', width: b.w + 'px', height: b.h + 'px' }"
      ></div>

      <button
        v-if="!tour.isLast.value"
        ref="laterEl"
        type="button"
        class="later"
        :style="laterStyle"
        data-test="tour-later"
        @click.stop="postponeTour"
      >
        Отложить <br />на потом
      </button>

      <!-- The last line: the whole screen celebrates the seeds. -->
      <div v-if="tour.step.value.finale" class="finale" aria-hidden="true">
        <SparkleBurst :spots="starSpots" :base="6" />
        <img v-for="(d, i) in drops" :key="i" :src="SEED" alt="" class="drop" :style="{ left: d.x + '%', animationDelay: d.d + 's', animationDuration: d.t + 's' }" />
        <div class="prize">
          <div class="glow"></div>
          <img :src="pile" alt="" class="pile" />
          <p class="plus">{{ prize }}</p>
        </div>
      </div>

      <!-- Keyed by the line, so every new line brings a fresh panel that pops up. -->
      <div :key="tour.step.value.id" ref="panelEl" class="panel" :style="panelStyle">
        <div class="bubble" data-test="tour-bubble" aria-live="polite">
          <!-- The whole line sits invisibly under the typed part, so the bubble has its final size from the first letter. -->
          <p class="line">
            <span class="ghost" aria-hidden="true">{{ text }}</span>
            <span class="typed">{{ shown }}<span v-if="typing" class="caret"></span></span>
          </p>
          <p v-if="gloss && !typing" class="gloss">{{ gloss }}</p>
          <svg class="tail" viewBox="0 0 6 3" shape-rendering="crispEdges" aria-hidden="true">
            <rect x="0" y="0" width="1" height="1" class="ink" />
            <rect x="1" y="0" width="4" height="1" class="paper" />
            <rect x="5" y="0" width="1" height="1" class="ink" />
            <rect x="1" y="1" width="1" height="1" class="ink" />
            <rect x="2" y="1" width="2" height="1" class="paper" />
            <rect x="4" y="1" width="1" height="1" class="ink" />
            <rect x="2" y="2" width="2" height="1" class="ink" />
          </svg>
          <svg v-if="unlocked && !tour.isOffer.value" class="more" viewBox="0 0 3 2" shape-rendering="crispEdges" aria-hidden="true">
            <rect width="3" height="1" class="ink" />
            <rect x="1" y="1" width="1" height="1" class="ink" />
          </svg>
        </div>

        <div class="row">
          <div class="mascot-wrap">
            <img :src="mascot" alt="" class="mascot" :class="{ talking: typing, happy: isHappy }" />
          </div>
          <div class="side">
            <div class="pips" aria-hidden="true">
              <i v-for="(s, i) in tour.steps.value" :key="s.id" class="pip" :class="{ on: i <= tour.index.value, cur: i === tour.index.value }"></i>
            </div>

            <button
              v-if="tour.isOffer.value"
              ref="primary"
              type="button"
              class="pbtn"
              :class="{ ready: unlocked }"
              data-test="tour-start"
              :disabled="!unlocked"
              @click.stop="advance"
            >
              Показать
            </button>
            <button
              v-else
              ref="primary"
              type="button"
              class="pbtn"
              :class="{ wait: !unlocked, ready: unlocked }"
              :aria-disabled="!unlocked"
              data-test="tour-next"
              @click.stop="advance"
            >
              {{ tour.isLast.value ? 'Спасибо!' : 'Дальше' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.tour {
  position: fixed;
  inset: 0;
  z-index: 60;
  -webkit-tap-highlight-color: transparent;
  cursor: pointer;
}
.tour-enter-active {
  animation: veil-in 0.35s ease-out;
}
.tour-leave-active {
  animation: veil-in 0.3s ease-in reverse forwards;
  pointer-events: none;
}
@keyframes veil-in {
  from {
    opacity: 0;
  }
}
.dim {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

/* The frame round a lit button: a rounded ring that breathes between white and mint, settles in when it first appears
   and sends a soft ring outwards now and then. */
.frame {
  position: absolute;
  pointer-events: none;
  border-radius: 16px;
  box-shadow: 0 0 0 4px #fff;
  animation:
    frame-in 0.45s cubic-bezier(0.2, 0.9, 0.3, 1.1) both,
    glow 1.8s ease-in-out infinite;
}
.frame::after {
  content: '';
  position: absolute;
  inset: -4px;
  border: 3px solid rgba(255, 255, 255, 0.7);
  border-radius: 20px;
  opacity: 0;
  animation: ping 2.2s ease-out 0.6s infinite;
}
@keyframes glow {
  50% {
    box-shadow: 0 0 0 4px #8ccbc6;
  }
}
@keyframes frame-in {
  from {
    transform: scale(1.12);
    opacity: 0;
  }
}
@keyframes ping {
  0% {
    inset: -4px;
    opacity: 0.8;
  }
  80%,
  100% {
    inset: -18px;
    opacity: 0;
  }
}

/* The finale: a darker veil, falling seeds, twinkling stars and the heap of seeds with the sum under it. */
.finale {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
  background: rgba(7, 28, 27, 0.78);
  animation: veil-in 0.6s ease-out;
}
.drop {
  position: absolute;
  top: 0;
  width: 28px;
  height: auto;
  image-rendering: pixelated;
  opacity: 0.9;
  animation: fall 3.5s linear infinite backwards;
}
@keyframes fall {
  from {
    transform: translateY(-60px) rotate(-20deg);
  }
  to {
    transform: translateY(105vh) rotate(25deg);
  }
}
.prize {
  position: absolute;
  left: 50%;
  top: max(14%, calc(3rem + env(safe-area-inset-top)));
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
  transform: translateX(-50%);
  animation: prize-in 0.8s cubic-bezier(0.2, 0.9, 0.3, 1.15) 0.2s both;
}
.glow {
  position: absolute;
  left: 50%;
  top: 0;
  width: 17rem;
  height: 17rem;
  transform: translate(-50%, -12%);
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255, 215, 94, 0.45), rgba(255, 215, 94, 0) 66%);
  animation: breathe 2.4s ease-in-out infinite;
}
.pile {
  position: relative;
  width: min(12rem, 52vw);
  image-rendering: pixelated;
  animation: pile-bob 2.2s ease-in-out infinite;
}
.plus {
  position: relative;
  font-family: 'Press Start 2P', monospace;
  font-size: 1.15rem;
  color: #ffd75e;
  text-shadow:
    3px 3px 0 #7a4b00,
    0 0 18px rgba(255, 215, 94, 0.55);
  white-space: nowrap;
  animation: plus-pop 0.8s cubic-bezier(0.2, 0.9, 0.3, 1.4) 0.8s both;
}
@keyframes prize-in {
  from {
    transform: translateX(-50%) translateY(50px) scale(0.5);
    opacity: 0;
  }
}
@keyframes breathe {
  50% {
    opacity: 0.6;
    transform: translate(-50%, -12%) scale(1.08);
  }
}
@keyframes pile-bob {
  50% {
    transform: translateY(-7px);
  }
}
@keyframes plus-pop {
  from {
    transform: scale(0.2);
    opacity: 0;
  }
}

/* «Отложить на потом»: just a link in the top right corner, underlined. */
.later {
  position: absolute;
  padding: 6px 4px;
  background: none;
  border: 0;
  color: #fff;
  font-family: 'Press Start 2P', monospace;
  font-size: 0.6rem;
  line-height: 1.4;
  text-align: right;
  text-decoration: underline;
  text-underline-offset: 4px;
  text-shadow: 0 1px 8px rgba(0, 0, 0, 0.6);
  cursor: pointer;
  transition:
    right 0.3s ease,
    opacity 0.2s;
  animation: fade-in 0.5s ease-out 0.5s both;
}
.later:hover {
  opacity: 0.8;
}
.later:focus-visible {
  outline: 2px solid rgba(255, 255, 255, 0.7);
  outline-offset: 3px;
  border-radius: 6px;
}
@keyframes fade-in {
  from {
    opacity: 0;
  }
}

.panel {
  position: absolute;
  cursor: default;
}

.bubble {
  --ink: var(--accent);
  position: relative;
  background: var(--card);
  color: var(--fg);
  padding: 18px 20px 20px;
  border-radius: 16px;
  cursor: pointer;
  transform-origin: 60px 100%;
  animation: bubble-in 0.45s cubic-bezier(0.2, 0.9, 0.3, 1.15) both;
  box-shadow:
    0 0 0 4px var(--ink),
    0 12px 28px rgba(0, 0, 0, 0.35);
}
@keyframes bubble-in {
  from {
    transform: scale(0.8) translateY(14px);
    opacity: 0;
  }
}
/* The line is set in a real pixel font. */
.line {
  display: grid;
  font-family: 'Press Start 2P', 'Manrope', monospace;
  font-size: 0.8rem;
  font-weight: 400;
  line-height: 1.8;
  word-break: break-word;
}
.line > span {
  grid-area: 1 / 1;
}
.ghost {
  visibility: hidden;
}
.caret {
  display: inline-block;
  width: 0.5em;
  height: 1em;
  margin-left: 2px;
  vertical-align: -0.15em;
  background: var(--ink);
  animation: caret 0.9s ease-in-out infinite;
}
@keyframes caret {
  50% {
    opacity: 0;
  }
}
.gloss {
  margin-top: 8px;
  font-family: 'Manrope', system-ui, sans-serif;
  font-size: 0.82rem;
  line-height: 1.35;
  color: var(--muted);
  animation: rise 0.35s ease-out both;
}
@keyframes rise {
  from {
    transform: translateY(6px);
    opacity: 0;
  }
}
.tail {
  position: absolute;
  left: 52px;
  bottom: -15px;
  width: 30px;
  height: 15px;
}
.tail .ink {
  fill: var(--ink);
}
.tail .paper {
  fill: var(--card);
}
.more {
  position: absolute;
  right: 14px;
  bottom: 10px;
  width: 12px;
  height: 8px;
  animation: bob 1.1s ease-in-out infinite;
}
.more .ink {
  fill: var(--ink);
}
@keyframes bob {
  50% {
    transform: translateY(3px);
  }
}

.row {
  display: flex;
  align-items: flex-end;
  gap: 14px;
  margin-top: 14px;
  padding-left: 8px;
}
/* The swallow hops in with the line, floats a little while she waits and bounces gently while she talks. */
.mascot-wrap {
  position: relative;
  flex: none;
  animation: hop 0.8s cubic-bezier(0.3, 1.3, 0.5, 1) 0.15s both;
}
.mascot {
  display: block;
  height: 132px;
  width: auto;
  image-rendering: pixelated;
  object-fit: contain;
  pointer-events: none;
  animation: idle 3s ease-in-out 0.9s infinite;
}
.mascot.talking {
  animation: talk 0.5s ease-in-out infinite;
}
.mascot.happy {
  animation: wiggle 1.4s ease-in-out infinite;
}
@keyframes hop {
  from {
    transform: translateY(36px);
    opacity: 0;
  }
}
@keyframes idle {
  50% {
    transform: translateY(-4px);
  }
}
@keyframes talk {
  50% {
    transform: translateY(-5px);
  }
}
@keyframes wiggle {
  25% {
    transform: rotate(-4deg) translateY(-6px);
  }
  75% {
    transform: rotate(4deg) translateY(-6px);
  }
}

.side {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  align-items: stretch;
  gap: 12px;
  padding-bottom: 6px;
}
/* Where she is in the tour: round dots, the current one stretched. */
.pips {
  display: flex;
  gap: 6px;
  justify-content: center;
  align-items: center;
}
.pip {
  width: 8px;
  height: 8px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.4);
  transition:
    width 0.35s cubic-bezier(0.3, 1.2, 0.5, 1),
    background 0.3s;
}
.pip.on {
  background: #fff;
}
.pip.cur {
  width: 22px;
}

.pbtn {
  height: 52px;
  padding: 0 20px;
  background: var(--accent);
  color: #fff;
  font-family: 'Press Start 2P', monospace;
  font-size: 0.8rem;
  line-height: 1;
  cursor: pointer;
  border: 0;
  /* Pixel corners: no radius, the dark outline is four offset shadows, so each corner is left notched by one pixel. */
  border-radius: 0;
  margin: 4px;
  box-shadow:
    0 -4px 0 0 #0b3f3d,
    0 4px 0 0 #0b3f3d,
    -4px 0 0 0 #0b3f3d,
    4px 0 0 0 #0b3f3d,
    inset 0 4px 0 0 rgba(255, 255, 255, 0.22),
    inset 0 -4px 0 0 rgba(0, 0, 0, 0.25);
  transition:
    opacity 0.3s,
    transform 0.12s,
    filter 0.15s;
}
.pbtn.wait,
.pbtn:disabled {
  opacity: 0.55;
  cursor: default;
}
/* When the line is done the button wakes up. */
.pbtn.ready {
  animation: wake 0.5s cubic-bezier(0.2, 0.9, 0.3, 1.2);
}
.pbtn:not(:disabled):not(.wait):hover {
  filter: brightness(1.12);
}
.pbtn:not(:disabled):not(.wait):active {
  transform: translateY(3px);
}
.pbtn:focus-visible {
  /* No outline box round the button: it read as a stray rectangle, and a tap is what moves the tour on anyway. */
  outline: none;
}
@keyframes wake {
  from {
    transform: scale(0.94);
    filter: brightness(1.5);
  }
}

@media (prefers-reduced-motion: reduce) {
  .tour,
  .tour *,
  .tour *::after {
    animation: none !important;
    transition: none !important;
  }
}
</style>
