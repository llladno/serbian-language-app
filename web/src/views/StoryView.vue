<script setup lang="ts">
// A story: three passes in a row (Serbian; a Serbian sentence and its Russian one; fast Serbian with quick
// questions), then the questions of the end. Built like a lesson: the same header, the same progress bar with the
// pass named under it, the main button pinned to the bottom, the same end screen. The order of things is
// lib/storyPlayer.ts, the sound is lib/storyEngine.ts, this file is the screen around them.
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, ArrowRight, Pause, Play, X } from 'lucide-vue-next'
import { api, ApiError } from '../api'
import { useWalletStore } from '../stores/wallet'
import { authErrorMessage } from '../lib/authErrors'
import { answerFinal, currentIndex, isRetry, startFinal, type FinalQueue } from '../lib/finalQueue'
import { useStoryEngine } from '../lib/storyEngine'
import { buildTimeline, maskAnswer, passFraction, PASS_TITLES, PROGRESS_STEPS, seekTarget, subtitleFor, allSentences } from '../lib/storyPlayer'
import { useScript } from '../lib/useScript'
import BottomBar from '../components/BottomBar.vue'
import LessonSummary from '../components/LessonSummary.vue'
import ScriptToggle from '../components/ScriptToggle.vue'
import ScrubBar from '../components/ScrubBar.vue'
import StepProgress from '../components/StepProgress.vue'
import StoryBank from '../components/StoryBank.vue'
import StoryFrames from '../components/StoryFrames.vue'
import StoryIntroSlide from '../components/StoryIntroSlide.vue'
import StoryMatch from '../components/StoryMatch.vue'
import StoryQuiz from '../components/StoryQuiz.vue'
import StorySubtitles from '../components/StorySubtitles.vue'
import type { Story } from '../types'

const route = useRoute()
const router = useRouter()
const wallet = useWalletStore()
const { show } = useScript()
const slug = String(route.params.slug)

const story = ref<Story | null>(null)
const error = ref<string | null>(null)
// slides: the first time, three little explanations. ready: the learner has seen them, one tap to start.
const phase = ref<'slides' | 'ready' | 'play' | 'final' | 'done'>('slides')
const timeline = computed(() => (story.value ? buildTimeline(story.value.content) : []))

const SEEN_KEY = 'srpski.story.seen'
function seenBefore(): boolean {
  try {
    return localStorage.getItem(SEEN_KEY) === '1'
  } catch {
    return false
  }
}
function markSeen() {
  try {
    localStorage.setItem(SEEN_KEY, '1')
  } catch {
    /* the slides simply come again */
  }
}

function ev(kind: string, detail = '') {
  api.storyEvent(slug, kind, detail).catch(() => {})
}

let quizScore = 0
let quizTotal = 0

const engine = useStoryEngine(() => timeline.value, {
  onPassStart: (p) => ev('pass_start', String(p)),
  onPassEnd: (p) => {
    ev('pass_end', String(p))
    api.storyProgress(slug, { pass: p, completed: false, score: 0, total: 0 }).catch(() => {})
  },
  onEnd: () => {
    if (story.value?.content.quiz_final.length) {
      phase.value = 'final'
      resetFinal()
    } else {
      void finish()
    }
  },
})

onMounted(async () => {
  try {
    story.value = await api.story(slug)
    if (seenBefore()) phase.value = 'ready'
    ev('open')
  } catch (e) {
    if (e instanceof ApiError && (e.status === 403 || e.status === 404)) router.replace('/stories')
    else error.value = authErrorMessage(e)
  }
})

// ---- the slides --------------------------------------------------------------------------------------------------
const slideIdx = ref(0)
const SLIDES = [
  { kind: 'listen', title: 'Сначала слушаем', text: 'Пока без перевода: слушайте и следите за словами. Подсвеченное звучит прямо сейчас.' },
  { kind: 'translate', title: 'Потом с переводом', text: 'Каждое предложение звучит дважды: по-сербски и по-русски.' },
  { kind: 'fast', title: 'И с вопросами', text: 'Русский рядом. Когда звук остановится, выберите пропущенное слово. В конце ещё несколько вопросов.' },
  { kind: 'scrub', title: 'Можно перематывать', text: 'Нажмите на полоску вверху или проведите по ней: история начнётся с начала предложения, на которое попали.' },
] as const
const lastSlide = computed(() => slideIdx.value === SLIDES.length - 1)

const firstImage = computed(() => story.value?.content.chapters[0]?.scenes[0]?.image ?? '')
const sample = computed(() => {
  const c = story.value?.content
  const sentences = c ? allSentences(c) : []
  const q = c?.quiz_inline[0]
  const quizSentence = q ? sentences.find((s) => s.n === q.after) : undefined
  const basic = sentences[1] ?? sentences[0]
  return {
    sr: basic?.sr ?? '',
    ru: basic?.ru ?? '',
    quizSr: quizSentence?.sr ?? basic?.sr ?? '',
    quiz: { mask: q?.mask ?? '', options: q?.options ?? [] },
    scrubLines: sentences.length ? [sentences[0].sr, sentences[Math.floor(sentences.length / 2)].sr, sentences[sentences.length - 1].sr] : [],
  }
})

function nextSlide() {
  if (lastSlide.value) listen()
  else slideIdx.value++
}

// ---- listening ---------------------------------------------------------------------------------------------------
function listen() {
  markSeen()
  phase.value = 'play'
  engine.start()
}

// The picture follows the sentence on the air and keeps the last scene across a card.
const lastScene = ref<number | null>(null)
const scenes = computed(() => story.value?.content.chapters.flatMap((c) => c.scenes) ?? [])
const sceneId = computed(() => {
  const s = engine.step.value
  if (s && s.type !== 'card') lastScene.value = s.sentence.scene
  return lastScene.value ?? scenes.value[0]?.id ?? 0
})

const subtitle = computed(() => {
  const s = engine.step.value
  return s ? subtitleFor(s) : { sr: null, ru: null }
})
const quizStep = computed(() => {
  const s = engine.step.value
  return s && s.type === 'quiz' ? s : null
})
const cardStep = computed(() => {
  const s = engine.step.value
  return s && s.type === 'card' ? s : null
})
const wordIdx = computed(() => (engine.step.value?.type === 'clip' && engine.step.value.lang === 'sr' ? engine.wordIdx.value : -1))

// A right answer to a quick question is written back into the gap of the subtitle; a new question starts blank again.
const inlineRight = ref(false)
// The answer has its gap in the subtitle (the mask was found in the sentence): then «Верно» does not repeat the word.
const gapOnScreen = computed(() => {
  const q = quizStep.value
  if (!q) return false
  const plain = q.sentence.sr.split(/\s+/).filter(Boolean)
  return maskAnswer(q.sentence.sr, q.quiz.mask).some((w, i) => w !== plain[i])
})
watch(
  () => quizStep.value?.index,
  () => (inlineRight.value = false),
)
async function checkInline(choice: string) {
  const q = quizStep.value
  if (!q) return { correct: false, answer: '' }
  const r = await api.answerStory(slug, 'inline', q.index, choice)
  ev('quiz_answer', `inline:${q.index}:${r.correct ? 1 : 0}`)
  if (r.correct) inlineRight.value = true
  return r
}

// ---- the questions at the end ------------------------------------------------------------------------------------
// A question answered wrongly comes again at the end, as the exercises of a lesson do (lib/finalQueue.ts); only the first
// try of each question counts for the score.
const finalState = ref<FinalQueue>(startFinal(0))
function resetFinal() {
  finalState.value = startFinal(story.value?.content.quiz_final.length ?? 0)
  quizScore = 0
  quizTotal = 0
}
const finalIdx = computed(() => currentIndex(finalState.value))
const finalQuiz = computed(() => story.value?.content.quiz_final[finalIdx.value] ?? null)
// Over a question: its number, and for a Serbian word or phrase what is asked about it. One that has come back says so.
const finalEyebrow = computed(() => {
  const n = story.value?.content.quiz_final.length ?? 0
  const head = isRetry(finalState.value) ? `Ещё раз · вопрос ${finalIdx.value + 1}` : `Вопрос ${finalIdx.value + 1} из ${n}`
  return finalQuiz.value?.kind === 'word' ? `${head} · что значит?` : head
})
async function checkFinal(choice: string, pairs?: Record<string, string>) {
  const r = await api.answerStory(slug, 'final', finalIdx.value, choice, pairs)
  ev('quiz_answer', `final:${finalIdx.value}:${finalQuiz.value?.kind ?? ''}:${r.correct ? 1 : 0}`)
  return r
}
function finalDone(correct: boolean) {
  if (phase.value !== 'final') return // the learner dragged back to the story meanwhile
  const r = answerFinal(finalState.value, correct)
  finalState.value = r.queue
  quizScore = r.queue.score
  quizTotal = r.queue.asked
  if (r.finished) void finish()
}

const reward = ref(0)
const saved = ref(true)
async function finish() {
  ev('complete', `${quizScore}/${quizTotal}`)
  try {
    const r = await api.storyProgress(slug, { pass: 3, completed: true, score: quizScore, total: quizTotal })
    reward.value = r.reward
    wallet.refresh()
  } catch {
    saved.value = false
  }
  phase.value = 'done'
}

const stats = computed(() => ({
  answered: quizTotal,
  mistakes: quizTotal - quizScore,
  percent: quizTotal ? Math.round((100 * quizScore) / quizTotal) : 0,
  new_words: 0,
}))

// ---- the header and the progress bar -----------------------------------------------------------------------------
const progressIdx = computed(() => {
  if (phase.value === 'final') return 3
  if (phase.value === 'play') return Math.max(0, (engine.step.value?.pass ?? 1) - 1)
  return 0
})
const progressFraction = computed(() => {
  if (phase.value === 'final') return finalState.value.right / Math.max(1, story.value?.content.quiz_final.length ?? 1)
  if (phase.value === 'play') return passFraction(timeline.value, engine.stepIndex.value)
  if (phase.value === 'slides') return (slideIdx.value + 1) / SLIDES.length
  return 0
})
const stepTitle = computed(() => {
  if (phase.value === 'slides') return 'Как это работает'
  if (phase.value === 'final') return 'Вопросы'
  if (phase.value === 'play') return PASS_TITLES[(engine.step.value?.pass ?? 1) as 1 | 2 | 3]
  return 'Три прохода и вопросы'
})

// A press or a drag along the progress bar goes to the start of the sentence under the finger: the picture and the
// subtitle follow the finger, the sound starts when it lets go. From the questions it goes back into the story.
const scrubbable = computed(() => phase.value === 'play' || phase.value === 'final')
function intoStory() {
  if (phase.value !== 'final') return
  phase.value = 'play'
  resetFinal()
}
function scrubMove(p: number) {
  if (!scrubbable.value) return
  const target = seekTarget(timeline.value, p)
  if (target === null) return // over the questions: nothing to show until the finger lets go
  intoStory()
  engine.preview(target)
}
function scrubCommit(p: number) {
  if (!scrubbable.value) return
  const target = seekTarget(timeline.value, p)
  if (target === null) {
    // The last segment is the questions: straight to them, no sentence first. Already there: nothing to do.
    if (phase.value === 'play') {
      quizScore = 0
      quizTotal = 0
      engine.toEnd()
    }
    return
  }
  intoStory()
  engine.seek(target)
}

function goBack() {
  if (phase.value === 'slides' && slideIdx.value > 0) slideIdx.value--
  else if (phase.value === 'play') engine.back()
  else leave()
}
function leave() {
  engine.destroy()
  router.push('/stories')
}
function again() {
  // The simplest honest restart: a fresh screen with a fresh engine (the slides are skipped: they were seen).
  router.go(0)
}

// ---- fitting the screen: no scrolling of the page -----------------------------------------------------------------
// The page is a header, the app's own top bar and padding, and this screen. The screen is given exactly the height that
// is left (so the page itself never scrolls), the picture gives up height when something else needs it, and the room under
// the content is the real height of the pinned bottom panel, whatever is in it (buttons, a few options, tiles, pairs).
const fit = ref<HTMLElement | null>(null)
const fitHeight = ref<number | null>(null)
const mainPad = ref(0) // the bottom padding the app puts under every page
function measureFit() {
  const el = fit.value
  if (!el) return
  const top = el.getBoundingClientRect().top + window.scrollY
  const main = el.closest('main')
  mainPad.value = main ? parseFloat(getComputedStyle(main).paddingBottom) || 0 : 0
  fitHeight.value = Math.max(240, Math.round(window.innerHeight - top - mainPad.value))
}

// Below this the end screen is drawn tighter so that it fits (it is taller than the others).
const shortScreen = computed(() => (fitHeight.value ?? 1000) < 620)

const bar = ref<{ $el: HTMLElement } | null>(null)
const panelHeight = ref(0)
let panelObserver: ResizeObserver | null = null
watch(bar, (b) => {
  panelObserver?.disconnect()
  panelHeight.value = 0
  if (b?.$el && typeof ResizeObserver !== 'undefined') {
    panelObserver = new ResizeObserver(([e]) => (panelHeight.value = Math.round(e.contentRect.height)))
    panelObserver.observe(b.$el)
  }
})
// Under the content: what the panel takes beyond the padding the page already has, and a breath.
const underPad = computed(() => (panelHeight.value ? Math.max(0, panelHeight.value - mainPad.value) + 8 : 0))

onMounted(() => {
  measureFit()
  window.addEventListener('resize', measureFit)
  window.visualViewport?.addEventListener('resize', measureFit)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', measureFit)
  window.visualViewport?.removeEventListener('resize', measureFit)
  panelObserver?.disconnect()
})
</script>

<template>
  <div ref="fit" class="story-fit" :style="{ height: fitHeight ? fitHeight + 'px' : undefined }">
  <p v-if="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

  <div v-else-if="!story" class="flex h-full flex-col">
    <header class="shrink-0 pb-4">
      <div class="flex w-full items-center gap-3">
        <div class="skel h-11 w-11 shrink-0 rounded-full"></div>
        <div class="flex-1 space-y-1.5">
          <div class="skel h-3.5 w-2/5"></div>
          <div class="skel h-2.5 w-1/4"></div>
        </div>
      </div>
      <div class="skel mt-3 h-1.5 w-full rounded-full"></div>
    </header>
  </div>

  <div v-else-if="phase === 'done'" class="flex min-h-full flex-col justify-center" :class="shortScreen ? 'py-0' : 'py-4'">
    <LessonSummary
      :title="show(story.title_sr)"
      :reward="reward"
      :stats="stats"
      :next="null"
      :compact="shortScreen"
      heading="История прослушана"
      back-to="/stories"
      back-label="К историям"
    >
      <template #extra>
        <p v-if="!saved" class="text-sm text-[var(--muted)]">Результат не сохранился. Проверьте сеть и пройдите ещё раз.</p>
        <button type="button" class="btn btn-ghost w-full" data-test="story-again" @click="again">Послушать ещё раз</button>
      </template>
    </LessonSummary>
  </div>

  <div
    v-else
    class="mx-auto flex h-full w-full max-w-md flex-col"
    :style="{ paddingBottom: underPad + 'px', transition: 'padding-bottom 0.3s ease' }"
  >
    <header class="shrink-0 pb-4">
      <div class="flex w-full items-center">
        <button class="icon-btn shrink-0" title="Назад" aria-label="Назад" data-test="story-back" @click="goBack">
          <ArrowLeft :size="20" :stroke-width="2.5" />
        </button>
        <div class="min-w-0 flex-1">
          <p class="serbian truncate text-[12px] sm:text-[14px]">{{ show(story.title_sr) }}</p>
          <p class="truncate text-[9px] text-[var(--muted)] sm:text-[10px]">{{ story.title_ru }}</p>
        </div>
        <ScriptToggle compact class="ml-auto mr-1.5" />
        <button class="icon-btn shrink-0" title="Закрыть" aria-label="Закрыть" data-test="story-close" @click="leave">
          <X :size="20" :stroke-width="2.25" />
        </button>
      </div>
      <ScrubBar class="mt-3 -mb-2" :disabled="!scrubbable" label="Перемотка" @move="scrubMove" @commit="scrubCommit">
        <StepProgress :steps="PROGRESS_STEPS" :current="progressIdx" :current-fraction="progressFraction" />
      </ScrubBar>
      <p class="mt-2 truncate pl-1 text-sm font-semibold text-[var(--muted)]" data-test="story-step-title">{{ stepTitle }}</p>
    </header>

    <div class="flex min-h-0 flex-1 flex-col justify-center">
      <!-- The slides before the first listening -->
      <Transition name="story-fade" mode="out-in">
        <div v-if="phase === 'slides'" :key="slideIdx" class="flex min-h-0 flex-col gap-4" data-test="story-intro">
          <StoryIntroSlide
            :kind="SLIDES[slideIdx].kind"
            :image="firstImage"
            :sr="slideIdx === 2 ? sample.quizSr : sample.sr"
            :ru="sample.ru"
            :quiz="sample.quiz"
            :lines="sample.scrubLines"
          />
          <div class="shrink-0 text-center">
            <p class="text-xl font-extrabold">{{ SLIDES[slideIdx].title }}</p>
            <p class="mt-1 text-sm text-[var(--muted)]">{{ SLIDES[slideIdx].text }}</p>
          </div>
          <div class="flex shrink-0 justify-center gap-1.5" aria-hidden="true">
            <span
              v-for="(_, i) in SLIDES"
              :key="i"
              class="h-1.5 rounded-full transition-all duration-300"
              :class="i === slideIdx ? 'w-5 bg-[var(--accent)]' : 'w-1.5 bg-[var(--ring-track)]'"
            />
          </div>
        </div>

        <div v-else-if="phase === 'ready'" key="ready" class="flex min-h-0 flex-col gap-4 text-center" data-test="story-ready">
          <div class="story-pic"><StoryFrames :scenes="scenes" :active="scenes[0]?.id ?? 0" /></div>
          <p class="shrink-0 text-sm text-[var(--muted)]">Три прохода: по-сербски, с переводом, с вопросами. В конце ещё несколько вопросов.</p>
        </div>

        <div v-else key="play" class="flex min-h-0 flex-col gap-4" data-test="story-play">
          <div class="story-pic">
            <StoryFrames :scenes="scenes" :active="sceneId" />
            <Transition name="story-fade">
              <div v-if="cardStep" class="story-glass" data-test="story-card">
                <p class="text-2xl font-extrabold">{{ cardStep.title }}</p>
                <p class="text-sm text-[var(--muted)]">{{ cardStep.subtitle }}</p>
              </div>
            </Transition>
          </div>
          <StorySubtitles
            v-if="phase === 'play'"
            :subtitle="subtitle"
            :word-idx="wordIdx"
            :mask="quizStep ? quizStep.quiz.mask : null"
            :reveal="inlineRight"
          />
        </div>
      </Transition>
    </div>
  </div>

  </div>

  <BottomBar v-if="story && phase !== 'done'" ref="bar">
    <!-- As wide as the story above it, on a big screen too; two buttons share that width equally. -->
    <div class="mx-auto w-full max-w-md">
      <!-- One thing at a time: the buttons, a question, the next question fade into each other instead of snapping. -->
      <Transition name="bar-swap" mode="out-in">
        <!-- slides -->
        <div v-if="phase === 'slides'" key="slides" class="grid gap-2" :class="lastSlide ? 'grid-cols-1' : 'grid-cols-2'">
          <button v-if="!lastSlide" type="button" class="btn btn-ghost" data-test="story-skip" @click="listen">Пропустить</button>
          <button type="button" class="btn btn-primary" data-test="story-next" @click="nextSlide">
            <template v-if="lastSlide"><Play :size="16" :stroke-width="2.5" />Слушать</template>
            <template v-else>Дальше<ArrowRight :size="16" :stroke-width="2.5" /></template>
          </button>
        </div>

        <!-- seen before -->
        <div v-else-if="phase === 'ready'" key="ready" class="grid grid-cols-2 gap-2">
          <button type="button" class="btn btn-ghost" data-test="story-howto" @click="phase = 'slides'">Как это работает</button>
          <button type="button" class="btn btn-primary" data-test="story-listen" @click="listen">
            <Play :size="16" :stroke-width="2.5" />Слушать
          </button>
        </div>

        <!-- a quick question: the options sit under the thumb -->
        <div v-else-if="quizStep" :key="`i${quizStep.index}`">
          <StoryQuiz
            eyebrow="Какое слово прозвучало?"
            :prompt="`«${quizStep.quiz.hint}»`"
            :options="quizStep.quiz.options"
            options-serbian
            :answer-in-place="gapOnScreen"
            :say="quizStep.sentence.sr_audio"
            :say-ms="quizStep.sentence.sr_ms"
            :play="engine.playOnce"
            :check="checkInline"
            @done="engine.answerDone()"
          />
        </div>

        <!-- the questions at the end -->
        <div v-else-if="phase === 'final' && finalQuiz" :key="`f${finalState.pos}`">
          <StoryBank
            v-if="finalQuiz.kind === 'bank'"
            :eyebrow="finalEyebrow"
            :prompt="finalQuiz.prompt"
            :bank="finalQuiz.bank ?? []"
            :say="finalQuiz.say"
            :say-ms="finalQuiz.say_ms"
            :play="engine.playOnce"
            :check="(c) => checkFinal(c)"
            @done="finalDone"
          />
          <StoryMatch
            v-else-if="finalQuiz.kind === 'match'"
            :eyebrow="finalEyebrow"
            :prompt="finalQuiz.prompt"
            :left="finalQuiz.left ?? []"
            :right="finalQuiz.right ?? []"
            :check="(pairs) => checkFinal('', pairs)"
            @done="finalDone"
          />
          <StoryQuiz
            v-else
            :eyebrow="finalEyebrow"
            :prompt="finalQuiz.prompt"
            :prompt-serbian="finalQuiz.kind === 'word' || finalQuiz.kind === 'blank'"
            :options="finalQuiz.options ?? []"
            :options-serbian="finalQuiz.kind === 'blank'"
            :fill-gap="finalQuiz.kind === 'blank'"
            :audio="finalQuiz.audio"
            :say="finalQuiz.say"
            :say-ms="finalQuiz.say_ms"
            :play="engine.playOnce"
            :check="(c) => checkFinal(c)"
            @done="finalDone"
          />
        </div>

        <!-- listening -->
        <button v-else key="toggle" type="button" class="btn btn-primary w-full" data-test="story-toggle" @click="engine.toggle()">
          <template v-if="engine.playing.value"><Pause :size="16" :stroke-width="2.5" />Пауза</template>
          <template v-else><Play :size="16" :stroke-width="2.5" />Продолжить</template>
        </button>
      </Transition>
    </div>
  </BottomBar>
</template>

<style scoped>
/* Opacity only: a transform on an ancestor would hold the pinned bottom bar to it (see BottomBar.vue). */
.story-fade-enter-active,
.story-fade-leave-active {
  transition: opacity 0.2s ease;
}
.story-fade-enter-from,
.story-fade-leave-to {
  opacity: 0;
}

/* The fitted box: exactly the height left on the screen. It only scrolls inside itself if something truly cannot fit
   (a very short screen), never the page. */
.story-fit {
  overflow-y: auto;
  overscroll-behavior: contain;
}

/* The picture: a 4:3 frame that gives up height, cropped, when the screen is short, but never below a thumbnail. */
.story-pic {
  position: relative;
  flex: 0 1 auto;
  width: 100%;
  min-height: 5rem;
  aspect-ratio: 4 / 3;
}

/* The pinned panel's content changes: the old one fades, the new one rises a little. The bar itself stays put, the
   transform is on its content only. */
.bar-swap-enter-active {
  transition: opacity 0.28s ease, transform 0.28s ease;
}
.bar-swap-leave-active {
  transition: opacity 0.16s ease;
}
.bar-swap-enter-from {
  opacity: 0;
  transform: translateY(10px);
}
.bar-swap-leave-to {
  opacity: 0;
}
@media (prefers-reduced-motion: reduce) {
  .bar-swap-enter-active,
  .bar-swap-leave-active {
    transition: none;
  }
}

/* The card between passes: frosted, in the card's own colour, not a black veil. */
.story-glass {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  border-radius: 1rem;
  padding: 1.5rem;
  text-align: center;
  background: color-mix(in srgb, var(--card) 72%, transparent);
  backdrop-filter: blur(10px);
}
</style>
