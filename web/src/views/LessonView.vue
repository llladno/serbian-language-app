<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, ApiError } from '../api'
import { useCourseStore } from '../stores/course'
import { useWalletStore } from '../stores/wallet'
import { useRewardModal } from '../lib/rewardModal'
import { provideWrongGoesToEnd } from '../lib/lessonRules'
import { nextLesson } from '../lib/nextLesson'
import type { Lesson, ExerciseBlock, LessonAttempts, LessonStats, Step } from '../types'
import MarkdownView from '../components/MarkdownView.vue'
import ReadingText from '../components/ReadingText.vue'
import DialogueStep from '../components/DialogueStep.vue'
import StepProgress from '../components/StepProgress.vue'
import BottomBar from '../components/BottomBar.vue'
import ExerciseBlockView from '../components/exercises/ExerciseBlock.vue'
import LessonSummary from '../components/LessonSummary.vue'
import { ArrowLeft, ArrowRight, CircleCheckBig, RotateCcw, X } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const store = useCourseStore()
const wallet = useWalletStore()

const lesson = ref<Lesson | null>(null)
const blocks = ref<ExerciseBlock[]>([])
const priors = ref<LessonAttempts>({})
const error = ref<string | null>(null)
// Set when the lesson has just been finished for the first time: the screen
// that closes it replaces the lesson until the learner leaves.
const summary = ref<{ reward: number; stats?: LessonStats } | null>(null)
const upNext = computed(() => (lesson.value ? nextLesson(store.course, lesson.value.id) : null))

const idx = ref(0)
const exIdx = ref(0)
const graded = reactive<Record<string, boolean>>({})

const steps = computed<Step[]>(() => lesson.value?.steps ?? [])
const cur = computed<Step | undefined>(() => steps.value[idx.value])
const curBlock = computed(() => blocks.value.find((b) => b.id === cur.value?.id))
const atLast = computed(() => idx.value >= steps.value.length - 1)

// One exercise per screen: exIdx walks the current step's queue of exercises.
// Dialogue steps manage their own turn-by-turn flow.
//
// An exercise answered wrongly is not skipped and not forgiven: it goes to the
// end of the queue and has to be answered again, so a step is only over once
// every exercise in it has been answered right. A dialogue is the exception —
// there a wrong line is just wrong and the conversation moves on. A lesson that
// is already finished is only being looked through, so none of this applies.
const reviewing = ref(false)
const enforce = computed(() => !reviewing.value)
provideWrongGoesToEnd(enforce)

// "Не могу прослушать": nothing is skipped. From here to the end of the lesson
// every dictation in it is a sentence to translate into Russian instead, and
// the choice is remembered until the lesson is finished (so leaving and coming
// back does not bring the audio back).
const silentKey = (id: string) => `lesson.silent.${id}`
function readSilent(id: string): boolean {
  try {
    return localStorage.getItem(silentKey(id)) === '1'
  } catch {
    return false
  }
}
function writeSilent(id: string, on: boolean) {
  try {
    if (on) localStorage.setItem(silentKey(id), '1')
    else localStorage.removeItem(silentKey(id))
  } catch {
    /* private mode: the choice just doesn't outlive the page */
  }
}
const silent = ref(false)

// Per step: the order its exercises are asked in. Untouched steps use the
// order they were written in.
const queues = reactive<Record<string, string[]>>({})
// How many times an exercise was sent to the end (see ExerciseBlock).
const rounds = reactive<Record<string, number>>({})

const blockExercises = computed(() => curBlock.value?.exercises ?? [])
const queue = computed<string[]>(() => {
  const ids = blockExercises.value.map((e) => e.id)
  const saved = cur.value ? queues[cur.value.id] : undefined
  return saved && saved.length === ids.length ? saved : ids
})
const curExerciseId = computed(() => queue.value[exIdx.value])
const curExercise = computed(() => blockExercises.value.find((e) => e.id === curExerciseId.value))
const blockAtLast = computed(() => exIdx.value >= queue.value.length - 1)
const paginatesExercises = computed(() => {
  const k = cur.value?.kind
  return !!k && k !== 'teach' && k !== 'dialogue' && blockExercises.value.length > 0
})

// Answered right — or, in a finished lesson, answered at all.
function passed(id: string) {
  return enforce.value ? graded[id] === true : id in graded
}
const curWrong = computed(
  () => enforce.value && paginatesExercises.value && !!curExerciseId.value && graded[curExerciseId.value] === false,
)
// "Далее" only actually finishes the lesson once there's no further exercise
// left to walk through on this last step — and the one on screen isn't going
// back to the end.
const isFinalAction = computed(
  () => atLast.value && (!paginatesExercises.value || (blockAtLast.value && !curWrong.value)),
)

// How far into the current step's block we are — feeds the single unified
// progress bar (StepProgress fills the current segment by this fraction). It
// counts answers that were right, so a mistake doesn't move it; in a finished
// lesson, where nothing is being answered, it follows the page instead.
const currentFraction = computed(() => {
  if (!paginatesExercises.value) return cur.value?.status === 'done' ? 1 : 0
  const len = queue.value.length
  if (!len) return 0
  if (!enforce.value) return exIdx.value / len
  return queue.value.filter(passed).length / len
})

function firstOpenExIdx() {
  const i = queue.value.findIndex((id) => !passed(id))
  return i === -1 ? 0 : i
}

const canAdvance = computed(() => {
  const s = cur.value
  if (!s) return false
  if (s.kind === 'teach') return true
  if (s.kind === 'dialogue') return (s.exercise_ids ?? []).every((eid) => eid in graded)
  if (!paginatesExercises.value) return true
  // Answered, right or wrong: a wrong one is what "Дальше" sends to the end.
  const id = curExerciseId.value
  return !!id && id in graded
})

// These exercise types render their own "Проверить" pinned to the bottom of
// the screen (see the answer components) instead of the lesson's Далее bar,
// so only one bottom action is ever visible at once. Choice grades itself on
// tap (no button), so it never needs the lesson's bar hidden.
const TYPES_WITH_OWN_ACTION = new Set([
  'translate',
  'fill_blank',
  'fix_error',
  'listen',
  'word_bank',
  'conjugate',
  'match',
])
// A dialogue step never paginates (it grows in place, see paginatesExercises
// below), so it falls outside the block/exIdx machinery above — but its
// active turn can still be a translate/fill_blank exercise with its own
// pinned "Проверить", which needs the same one-bar-at-a-time treatment.
const activeDialogueExercise = computed(() => {
  const s = cur.value
  if (!s || s.kind !== 'dialogue') return undefined
  const turn = (s.turns ?? []).find((t) => t.who === 'me' && t.exercise_id && !(t.exercise_id in graded))
  return blockExercises.value.find((e) => e.id === turn?.exercise_id)
})
const showOwnBottomButton = computed(() => {
  const s = cur.value
  if (s?.kind === 'dialogue') {
    const ex = activeDialogueExercise.value
    return !!ex && TYPES_WITH_OWN_ACTION.has(ex.type)
  }
  const ex = curExercise.value
  if (!s || !ex || !paginatesExercises.value) return false
  if (canAdvance.value) return false
  return TYPES_WITH_OWN_ACTION.has(ex.type)
})

// The open dialogue step, which knows when its conversation is over.
const dialogueRef = ref<InstanceType<typeof DialogueStep> | null>(null)
const dialogueFinished = computed(() => !!dialogueRef.value?.finished)

// One BottomBar instance for the whole view (loading skeleton included)
// instead of one per branch, so it's already pinned to the bottom before
// the lesson even loads and never pops in/jumps once it does - only its
// inner content swaps (skeleton -> Далее/Завершить), never the bar itself.
const showBottomBar = computed(() => {
  if (!lesson.value) return true
  if (lesson.value.planned || summary.value) return false
  // A dialogue has nothing to press until it is over: no disabled button
  // waiting at the bottom while the conversation is still going.
  if (cur.value?.kind === 'dialogue' && !dialogueFinished.value) return false
  return !!cur.value && !showOwnBottomButton.value
})

async function loadLesson(id: string) {
  lesson.value = null
  blocks.value = []
  priors.value = {}
  error.value = null
  summary.value = null
  idx.value = 0
  exIdx.value = 0
  for (const k of Object.keys(graded)) delete graded[k]
  for (const k of Object.keys(queues)) delete queues[k]
  for (const k of Object.keys(rounds)) delete rounds[k]
  try {
    const l = await api.lesson(id)
    // Pin the starting step in the same tick the lesson lands, so the body
    // never renders step 0 for a frame while exercises load.
    const wanted = route.query.step as string | undefined
    const at = wanted ? (l.steps ?? []).findIndex((s) => s.id === wanted) : -1
    const firstOpen = (l.steps ?? []).findIndex((s) => s.status !== 'done')
    idx.value = at >= 0 ? at : firstOpen === -1 ? 0 : firstOpen
    lesson.value = l
    reviewing.value = l.status === 'done'
    // A finished lesson is only looked through, with its audio.
    silent.value = !reviewing.value && readSilent(id)
    if (reviewing.value) writeSilent(id, false)

    if (!l.planned) {
      const [bl, pr] = await Promise.all([
        api.exercises(id, { audioOff: silent.value }),
        api.lessonAttempts(id),
      ])
      blocks.value = bl
      priors.value = pr
      // A wrong answer from an earlier visit means "not done yet" — except in
      // a dialogue, where a wrong line stays wrong, and in a lesson that is
      // already finished.
      const inDialogue = new Set(
        (l.steps ?? []).filter((st) => st.kind === 'dialogue').flatMap((st) => st.exercise_ids ?? []),
      )
      for (const [exId, a] of Object.entries(pr)) {
        if (a.correct || reviewing.value || inDialogue.has(exId)) graded[exId] = a.correct
      }
      exIdx.value = firstOpenExIdx()
    }
    markSeen()
    window.scrollTo(0, 0)
  } catch (e) {
    // A level the learner has not bought: the course screen is where it is
    // explained and bought, so a direct link lands there rather than on an error.
    if (e instanceof ApiError && e.status === 403) {
      router.replace('/course')
      return
    }
    error.value = (e as Error).message
  }
}

// route.fullPath covers ?step= deep links within the same lesson.
watch(() => route.fullPath, () => loadLesson(route.params.id as string), { immediate: true })

async function markSeen() {
  const s = cur.value
  if (!lesson.value || !s || s.status !== 'not_started') return
  s.status = 'in_progress'
  try {
    await api.setStepStatus(lesson.value.id, s.id, 'in_progress')
  } catch {
    /* non-fatal */
  }
}

function onGraded(exId: string, ok: boolean) {
  graded[exId] = ok
}

function onUngraded(exId: string) {
  delete graded[exId]
}


// What the exercise components show for an earlier attempt. Outside a finished
// lesson only a right answer counts as done, so one answered wrongly last time
// is simply asked again; dialogue turns keep their wrong marks, which are part
// of the conversation.
const exercisePriors = computed<LessonAttempts>(() => {
  if (reviewing.value) return priors.value
  return Object.fromEntries(Object.entries(priors.value).filter(([, a]) => a.correct))
})

// Puts an exercise last in its step's queue and asks for it afresh, moving on
// to whatever is still open.
function sendToEnd(id: string) {
  const step = cur.value
  if (!step) return
  const q = queue.value.filter((x) => x !== id)
  q.push(id)
  queues[step.id] = q
  rounds[id] = (rounds[id] ?? 0) + 1
  delete graded[id]
  exIdx.value = firstOpenExIdx()
  window.scrollTo(0, 0)
}

async function cantListen() {
  if (!lesson.value || silent.value) return
  silent.value = true
  writeSilent(lesson.value.id, true)
  try {
    blocks.value = await api.exercises(lesson.value.id, { audioOff: true })
  } catch {
    // Without the text there is nothing to put in the audio's place, so the
    // lesson goes on as it was and the button is still there to try again.
    silent.value = false
    writeSilent(lesson.value.id, false)
  }
}

async function next() {
  const s = cur.value
  if (!lesson.value || !s) return

  if (paginatesExercises.value) {
    const id = curExerciseId.value
    if (enforce.value && id && graded[id] === false) {
      sendToEnd(id)
      return
    }
    if (!blockAtLast.value) {
      exIdx.value++
      window.scrollTo(0, 0)
      return
    }
    // The last one is done — unless something earlier was taken back for a redo.
    const open = enforce.value ? queue.value.findIndex((x) => !passed(x)) : -1
    if (open !== -1) {
      exIdx.value = open
      window.scrollTo(0, 0)
      return
    }
  }

  if (s.status !== 'done') {
    s.status = 'done'
    try {
      await api.setStepStatus(lesson.value.id, s.id, 'done')
    } catch {
      /* non-fatal — progress catches up on next action */
    }
  }
  if (!atLast.value) {
    idx.value++
    exIdx.value = firstOpenExIdx()
    markSeen()
    window.scrollTo(0, 0)
  } else if (lesson.value.status === 'done') {
    // Revisiting an already-completed lesson: the final button reads "Урок
    // пройден" (nothing left to finish), so pressing it just closes out.
    closeLesson()
  } else {
    await finish()
  }
}

// The lesson's only back-navigation: one arrow that steps back through
// exercises, then steps, and only leaves for the course list once there's
// nowhere left inside the lesson to go back to.
function goBack() {
  if (cur.value?.kind === 'dialogue' && dialogueRef.value?.stepBack()) {
    window.scrollTo(0, 0)
    return
  }
  if (paginatesExercises.value && exIdx.value > 0) {
    exIdx.value--
    window.scrollTo(0, 0)
    return
  }
  if (idx.value > 0) {
    idx.value--
    exIdx.value = Math.max(0, queue.value.length - 1)
    window.scrollTo(0, 0)
    return
  }
  router.push('/course')
}

async function finish() {
  if (!lesson.value) return
  const wasOpen = lesson.value.status !== 'done'
  let reward = 0
  let stats: LessonStats | undefined
  if (wasOpen) {
    const done = await store.markDone(lesson.value.id)
    reward = done.reward
    stats = done.stats
    lesson.value.status = 'done'
  }
  if (!wasOpen) return

  // The lesson's own reward is told on the screen that closes it (it is never
  // announced beforehand). The course is only needed for the "next lesson"
  // button, which appears last, so the screen does not wait for it.
  summary.value = { reward, stats }
  window.scrollTo(0, 0)
  store.load().catch(() => {})

  // This lesson may also have been the one that finished the level, and it
  // moved the counters behind the lesson quests either way. Rewards are the
  // lesson's reward, not its job: a failure here must not touch the screen the
  // learner is looking at.
  try {
    wallet.refresh()
    const paid = await wallet.claimFinishedPhases()
    if (paid) useRewardModal().celebrate(paid.reward, paid.title, true)
  } catch {
    /* the quests screen will show it */
  }
}

// Progress is already saved step by step as the learner goes (each answered
// exercise and completed step is posted to the API), so closing the lesson
// needs nothing beyond leaving — there is no unsaved state to confirm away.
function closeLesson() {
  router.push('/course')
}
</script>

<template>
  <Transition name="fade" mode="out-in">
    <p v-if="error" key="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

    <div v-else-if="!lesson" key="skel" class="flex min-h-[calc(100dvh-11rem)] flex-col">
      <header class="shrink-0 pb-4">
        <div class="flex items-center w-full gap-3">
          <div class="skel h-11 w-11 shrink-0 rounded-full"></div>
          <div class="flex-1 space-y-1.5">
            <div class="skel h-3.5 w-2/5"></div>
            <div class="skel h-2.5 w-1/4"></div>
          </div>
        </div>
        <div class="skel mt-3 h-1.5 w-full rounded-full"></div>
      </header>
      <div class="flex flex-1 flex-col justify-center">
        <div class="card space-y-3 p-5 sm:p-6">
          <div class="skel h-5 w-1/3"></div>
          <div class="skel h-3.5 w-full"></div>
          <div class="skel h-3.5 w-full"></div>
          <div class="skel h-3.5 w-4/5"></div>
        </div>
      </div>
    </div>

    <div v-else-if="lesson.planned" key="planned">
      <RouterLink to="/course" class="text-sm text-[var(--muted)] hover:text-[var(--fg)]">← к курсу</RouterLink>
      <div class="card mt-4 border-dashed p-8 text-center">
        <h1 class="text-lg font-bold sm:text-xl">{{ lesson.title }}</h1>
        <p class="mt-1 text-sm text-[var(--muted)] sm:text-base">{{ lesson.subtitle }}</p>
        <p class="mt-4 text-sm text-[var(--muted)]">Урок ещё не готов — скоро появится.</p>
      </div>
    </div>

    <div v-else-if="summary" key="summary" class="flex min-h-[calc(100dvh-11rem)] flex-col justify-center py-4">
      <LessonSummary
        :title="lesson.title"
        :reward="summary.reward"
        :stats="summary.stats"
        :next="upNext"
      />
    </div>

    <div v-else-if="cur" key="lesson" class="flex min-h-[calc(100dvh-11rem)] flex-col">
      <header class="shrink-0 pb-4">
        <div class="flex items-center w-full">
          <button class="icon-btn shrink-0" title="Назад" aria-label="Назад" @click="goBack">
            <ArrowLeft :size="20" :stroke-width="2.5" />
          </button>
          <div class="min-w-0 flex-1">
            <p class="truncate text-[12px] sm:text-[14px]">
              {{ lesson.title }}
            </p>
            <p class="truncate text-[9px] text-[var(--muted)] sm:text-[10px]">
              {{ lesson.subtitle }}
            </p>
          </div>
          <button
              class="icon-btn shrink-0 ml-auto"
              title="Закрыть урок"
              aria-label="Закрыть урок"
              @click="closeLesson"
          >
            <X :size="20" :stroke-width="2.25" />
          </button>
        </div>
        <div class="flex items-center gap-3">


          <StepProgress class="flex-1" :steps="steps" :current="idx" :current-fraction="currentFraction" />
        </div>
        <p class="mt-2 truncate pl-1 text-sm font-semibold text-[var(--muted)]">{{ cur.title }}</p>
      </header>

      <div class="flex flex-1 flex-col justify-center">
        <Transition name="step" mode="out-in">
          <div :key="cur.id" class="space-y-6">
            <article v-if="cur.kind === 'teach'" class="card p-5 sm:p-6">
              <MarkdownView :source="cur.markdown ?? ''" />
            </article>

            <template v-else-if="cur.kind === 'reading'">
              <section class="card p-5 sm:p-6">
                <MarkdownView v-if="cur.markdown && !cur.markdown_ru" :source="cur.markdown" />
                <ReadingText v-else :serbian="cur.markdown ?? ''" :translation="cur.markdown_ru" />
              </section>
              <div v-if="curBlock" class="space-y-4">
                <ExerciseBlockView
                  :lesson="lesson.id"
                  :block="curBlock"
                  :active-id="curExerciseId"
                  :priors="exercisePriors"
                  :rounds="rounds"
                  @graded="onGraded"
                  @ungraded="onUngraded"
                  @cant-listen="cantListen"
                />
              </div>
            </template>

            <DialogueStep
              v-else-if="cur.kind === 'dialogue'"
              ref="dialogueRef"
              :lesson="lesson.id"
              :step="cur"
              :exercises="curBlock?.exercises ?? []"
              :priors="priors"
              @graded="onGraded"
              @ungraded="onUngraded"
            />

            <div v-else-if="curBlock" class="space-y-4">
              <p v-if="cur.markdown" class="rounded-xl bg-[var(--bg-soft)] px-4 py-2.5 text-sm text-[var(--muted)]">
                {{ cur.markdown }}
              </p>
              <ExerciseBlockView
                :lesson="lesson.id"
                :block="curBlock"
                :active-id="curExerciseId"
                :priors="exercisePriors"
                :rounds="rounds"
                @graded="onGraded"
                @ungraded="onUngraded"
                @cant-listen="cantListen"
              />
            </div>
          </div>
        </Transition>
      </div>
    </div>
  </Transition>

  <BottomBar v-if="showBottomBar">
    <div v-if="!lesson" class="skel h-12 w-full rounded-2xl"></div>
    <button v-else class="btn btn-primary w-full" :disabled="!canAdvance" @click="next">
      <template v-if="isFinalAction">
        <CircleCheckBig :size="16" :stroke-width="2.5" />
        {{ lesson.status === 'done' ? 'Урок пройден' : 'Завершить урок' }}
      </template>
      <template v-else-if="curWrong">
        <template v-if="blockAtLast"><RotateCcw :size="16" :stroke-width="2.5" />Ещё раз</template>
        <template v-else>Вернёмся позже<ArrowRight :size="16" :stroke-width="2.5" /></template>
      </template>
      <template v-else> Дальше<ArrowRight :size="16" :stroke-width="2.5" /> </template>
    </button>
  </BottomBar>
</template>

<style scoped>
/* Opacity only, deliberately no transform: a `transform` here — even a
   brief one during enter/leave — gives any `position: fixed` descendant
   (the own/shared BottomBar rendered inside this transition) a new
   containing block, so it would pin to this wrapper instead of the
   viewport and visibly float mid-page for the transition's duration. */
.step-enter-active,
.step-leave-active {
  transition: opacity 0.18s ease;
}
.step-enter-from,
.step-leave-to {
  opacity: 0;
}
</style>
