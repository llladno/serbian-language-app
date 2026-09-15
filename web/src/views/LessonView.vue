<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { useCourseStore } from '../stores/course'
import type { Lesson, ExerciseBlock, LessonAttempts, Step } from '../types'
import MarkdownView from '../components/MarkdownView.vue'
import ReadingText from '../components/ReadingText.vue'
import DialogueStep from '../components/DialogueStep.vue'
import StepProgress from '../components/StepProgress.vue'
import BottomBar from '../components/BottomBar.vue'
import ExerciseBlockView from '../components/exercises/ExerciseBlock.vue'
import Confetti from '../components/Confetti.vue'
import { ArrowLeft, ArrowRight, CircleCheckBig, RotateCcw } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const store = useCourseStore()

const lesson = ref<Lesson | null>(null)
const blocks = ref<ExerciseBlock[]>([])
const priors = ref<LessonAttempts>({})
const error = ref<string | null>(null)
const celebrate = ref(false)

const idx = ref(0)
const exIdx = ref(0)
const graded = reactive<Record<string, boolean>>({})

const steps = computed<Step[]>(() => lesson.value?.steps ?? [])
const cur = computed<Step | undefined>(() => steps.value[idx.value])
const curBlock = computed(() => blocks.value.find((b) => b.id === cur.value?.id))
const atLast = computed(() => idx.value >= steps.value.length - 1)

// One exercise per screen: exIdx walks the current step's block. Reading
// blocks stay optional (canAdvance ignores them, as before) but still page
// one at a time; dialogue steps manage their own turn-by-turn flow.
const blockExercises = computed(() => curBlock.value?.exercises ?? [])
const curExercise = computed(() => blockExercises.value[exIdx.value])
const blockAtLast = computed(() => exIdx.value >= blockExercises.value.length - 1)
const paginatesExercises = computed(() => {
  const k = cur.value?.kind
  return !!k && k !== 'teach' && k !== 'dialogue' && blockExercises.value.length > 0
})
// "Далее" only actually finishes the lesson once there's no further exercise
// page left to walk through on this last step.
const isFinalAction = computed(() => atLast.value && (!paginatesExercises.value || blockAtLast.value))

// How far into the current step's block we are — feeds the single unified
// progress bar (StepProgress fills the current segment by this fraction).
const currentFraction = computed(() => {
  if (!paginatesExercises.value) return 0
  const len = blockExercises.value.length
  return len ? exIdx.value / len : 0
})

function firstOpenExIdx() {
  const i = blockExercises.value.findIndex((e) => !(e.id in graded))
  return i === -1 ? 0 : i
}

const canAdvance = computed(() => {
  const s = cur.value
  if (!s) return false
  if (s.kind === 'teach' || s.kind === 'reading') return true
  if (s.kind === 'dialogue') return (s.exercise_ids ?? []).every((eid) => eid in graded)
  if (!paginatesExercises.value) return true
  const ex = curExercise.value
  return !!ex && ex.id in graded
})

// These exercise types render their own "Проверить" pinned to the bottom of
// the screen (see the answer components) instead of the lesson's Далее bar,
// so only one bottom action is ever visible at once. Choice grades itself on
// tap (no button); free-answer's two-step self-grade flow stays inline in
// the card — neither needs the lesson's bar hidden.
const TYPES_WITH_OWN_ACTION = new Set([
  'translate',
  'fill_blank',
  'fix_error',
  'listen',
  'word_bank',
  'conjugate',
  'match',
])
const showOwnBottomButton = computed(() => {
  const s = cur.value
  const ex = curExercise.value
  if (!s || !ex || !paginatesExercises.value) return false
  if (s.kind === 'reading') return false // optional — keep the shared Далее reachable to skip
  if (canAdvance.value) return false
  return TYPES_WITH_OWN_ACTION.has(ex.type)
})

async function loadLesson(id: string) {
  lesson.value = null
  blocks.value = []
  priors.value = {}
  error.value = null
  celebrate.value = false
  idx.value = 0
  exIdx.value = 0
  for (const k of Object.keys(graded)) delete graded[k]
  try {
    const l = await api.lesson(id)
    // Pin the starting step in the same tick the lesson lands, so the body
    // never renders step 0 for a frame while exercises load.
    const wanted = route.query.step as string | undefined
    const at = wanted ? (l.steps ?? []).findIndex((s) => s.id === wanted) : -1
    const firstOpen = (l.steps ?? []).findIndex((s) => s.status !== 'done')
    idx.value = at >= 0 ? at : firstOpen === -1 ? 0 : firstOpen
    lesson.value = l

    if (!l.planned) {
      const [bl, pr] = await Promise.all([api.exercises(id), api.lessonAttempts(id)])
      blocks.value = bl
      priors.value = pr
      for (const [exId, a] of Object.entries(pr)) graded[exId] = a.correct
      exIdx.value = firstOpenExIdx()
    }
    markSeen()
    window.scrollTo(0, 0)
  } catch (e) {
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

async function next() {
  const s = cur.value
  if (!lesson.value || !s) return

  if (paginatesExercises.value && !blockAtLast.value) {
    exIdx.value++
    window.scrollTo(0, 0)
    return
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
  } else {
    await finish()
  }
}

// The lesson's only back-navigation: one arrow that steps back through
// exercises, then steps, and only leaves for the course list once there's
// nowhere left inside the lesson to go back to.
function goBack() {
  if (paginatesExercises.value && exIdx.value > 0) {
    exIdx.value--
    window.scrollTo(0, 0)
    return
  }
  if (idx.value > 0) {
    idx.value--
    exIdx.value = Math.max(0, blockExercises.value.length - 1)
    window.scrollTo(0, 0)
    return
  }
  router.push('/course')
}

async function finish() {
  if (!lesson.value) return
  if (lesson.value.status !== 'done') {
    await store.markDone(lesson.value.id)
    lesson.value.status = 'done'
  }
  celebrate.value = true
  setTimeout(() => (celebrate.value = false), 3500)
}

async function resetLesson() {
  if (!lesson.value) return
  if (!confirm('Сбросить весь прогресс по уроку и пройти заново?')) return
  await api.resetLesson(lesson.value.id)
  store.setStatus(lesson.value.id, 'not_started')
  await loadLesson(lesson.value.id)
}
</script>

<template>
  <Confetti v-if="celebrate" />
  <p v-if="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

  <template v-else-if="lesson">
    <div v-if="lesson.planned">
      <RouterLink to="/course" class="text-sm text-[var(--muted)] hover:text-[var(--fg)]">← к курсу</RouterLink>
      <div class="card mt-4 border-dashed p-8 text-center">
        <h1 class="text-xl font-bold">{{ lesson.title }}</h1>
        <p class="mt-1 text-[var(--muted)]">{{ lesson.subtitle }}</p>
        <p class="mt-4 text-sm text-[var(--muted)]">Урок ещё не готов — скоро появится.</p>
      </div>
    </div>

    <template v-else-if="cur">
      <div class="flex min-h-[calc(100dvh-11rem)] flex-col">
        <header class="shrink-0 pb-4">
          <div class="flex items-center w-full">
            <button class="icon-btn shrink-0" title="Назад" aria-label="Назад" @click="goBack">
              <ArrowLeft :size="20" :stroke-width="2.5" />
            </button>
            <div>
              <p class="text-[14px]">
                {{ lesson.title }}
              </p>
              <p class="text-[10px] text-[var(--muted)]">
                {{ lesson.subtitle }}
              </p>
            </div>
            <button
                v-if="Object.keys(priors).length"
                class="icon-btn shrink-0 ml-auto"
                title="Сбросить и пройти заново"
                aria-label="заново"
                @click="resetLesson"
            >
              <RotateCcw :size="18" :stroke-width="2.25" />
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
                  <ExerciseBlockView :lesson="lesson.id" :block="curBlock" :ex-idx="exIdx" :priors="priors" @graded="onGraded" />
                </div>
              </template>

              <DialogueStep
                v-else-if="cur.kind === 'dialogue'"
                :lesson="lesson.id"
                :step="cur"
                :exercises="curBlock?.exercises ?? []"
                :priors="priors"
                @graded="onGraded"
              />

              <div v-else-if="curBlock" class="space-y-4">
                <p v-if="cur.markdown" class="rounded-xl bg-[var(--bg-soft)] px-4 py-2.5 text-sm text-[var(--muted)]">
                  {{ cur.markdown }}
                </p>
                <ExerciseBlockView :lesson="lesson.id" :block="curBlock" :ex-idx="exIdx" :priors="priors" @graded="onGraded" />
              </div>
            </div>
          </Transition>
        </div>
      </div>

      <BottomBar v-if="!showOwnBottomButton">
        <span v-if="!canAdvance" class="mb-2 block text-center text-xs text-[var(--muted)]">
          ответь на все задания
        </span>
        <button class="btn btn-primary w-full disabled:opacity-40" :disabled="!canAdvance" @click="next">
          <template v-if="isFinalAction">
            <CircleCheckBig :size="16" :stroke-width="2.5" />
            {{ lesson.status === 'done' ? 'Урок пройден' : 'Завершить урок' }}
          </template>
          <template v-else> Дальше<ArrowRight :size="16" :stroke-width="2.5" /> </template>
        </button>
      </BottomBar>
    </template>
  </template>
</template>

<style scoped>
.step-enter-active,
.step-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}
.step-enter-from {
  opacity: 0;
  transform: translateX(12px);
}
.step-leave-to {
  opacity: 0;
  transform: translateX(-12px);
}
</style>
