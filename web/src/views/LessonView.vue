<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { useCourseStore } from '../stores/course'
import type { Lesson, ExerciseBlock, LessonAttempts, Step } from '../types'
import MarkdownView from '../components/MarkdownView.vue'
import ReadingText from '../components/ReadingText.vue'
import DialogueStep from '../components/DialogueStep.vue'
import StepProgress from '../components/StepProgress.vue'
import ExerciseBlockView from '../components/exercises/ExerciseBlock.vue'
import Confetti from '../components/Confetti.vue'
import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
  CircleCheckBig,
  Dumbbell,
  Lightbulb,
  RotateCcw,
} from 'lucide-vue-next'

const route = useRoute()
const store = useCourseStore()

const lesson = ref<Lesson | null>(null)
const blocks = ref<ExerciseBlock[]>([])
const priors = ref<LessonAttempts>({})
const error = ref<string | null>(null)
const celebrate = ref(false)

const idx = ref(0)
const graded = reactive<Record<string, boolean>>({})

const KIND_META: Record<string, { icon: unknown; label: string }> = {
  teach: { icon: Lightbulb, label: 'Теория' },
  practice: { icon: Dumbbell, label: 'Практика' },
  reading: { icon: BookOpen, label: 'Чтение' },
  checkpoint: { icon: CircleCheckBig, label: 'Проверка' },
}
const kindMeta = (k: string) => KIND_META[k] ?? { icon: Lightbulb, label: k }

const steps = computed<Step[]>(() => lesson.value?.steps ?? [])
const cur = computed<Step | undefined>(() => steps.value[idx.value])
const curBlock = computed(() => blocks.value.find((b) => b.id === cur.value?.id))
const atLast = computed(() => idx.value >= steps.value.length - 1)

const canAdvance = computed(() => {
  const s = cur.value
  if (!s) return false
  if (s.kind === 'teach' || s.kind === 'reading') return true
  return (s.exercise_ids ?? []).every((eid) => eid in graded)
})

async function loadLesson(id: string) {
  lesson.value = null
  blocks.value = []
  priors.value = {}
  error.value = null
  celebrate.value = false
  idx.value = 0
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
    markSeen()
    window.scrollTo(0, 0)
  } else {
    await finish()
  }
}

function back() {
  if (idx.value > 0) {
    idx.value--
    window.scrollTo(0, 0)
  }
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
      <header class="mb-6">
        <div class="mb-2.5 flex items-center justify-between gap-3 text-xs">
          <RouterLink
            to="/course"
            class="inline-flex items-center gap-1 text-[var(--muted)] transition hover:text-[var(--fg)]"
          >
            <ArrowLeft :size="13" :stroke-width="2.5" />{{ lesson.title }}
          </RouterLink>
          <button
            v-if="Object.keys(priors).length"
            class="inline-flex items-center gap-1 font-medium text-[var(--muted)] transition hover:text-[var(--accent)]"
            @click="resetLesson"
          >
            <RotateCcw :size="12" :stroke-width="2.5" />заново
          </button>
        </div>

        <div class="flex items-center gap-2.5">
          <span
            class="inline-flex items-center gap-1.5 rounded-full bg-[var(--accent-soft)] px-2.5 py-1 text-[11px] font-bold uppercase tracking-wide text-[var(--accent)]"
          >
            <component :is="kindMeta(cur.kind).icon" :size="13" :stroke-width="2.5" />
            {{ kindMeta(cur.kind).label }}
          </span>
          <h1 class="min-w-0 flex-1 truncate text-lg font-extrabold tracking-tight">{{ cur.title }}</h1>
          <span class="shrink-0 text-xs font-semibold tabular-nums text-[var(--muted)]">
            {{ idx + 1 }}<span class="opacity-50">/{{ steps.length }}</span>
          </span>
        </div>

        <StepProgress class="mt-3" :steps="steps" :current="idx" />
      </header>

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
              <ExerciseBlockView :lesson="lesson.id" :block="curBlock" :priors="priors" @graded="onGraded" />
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
            <ExerciseBlockView :lesson="lesson.id" :block="curBlock" :priors="priors" @graded="onGraded" />
          </div>
        </div>
      </Transition>

      <div
        class="sticky bottom-0 mt-10 -mx-4 flex items-center justify-between gap-3 border-t border-[var(--border)] bg-[var(--bg)]/90 px-4 py-3 backdrop-blur sm:-mx-6 sm:px-6"
        style="padding-bottom: calc(0.75rem + env(safe-area-inset-bottom))"
      >
        <button
          class="btn btn-ghost disabled:invisible"
          :disabled="idx === 0"
          @click="back"
        >
          <ArrowLeft :size="16" :stroke-width="2.5" />Назад
        </button>
        <span v-if="!canAdvance" class="text-center text-xs text-[var(--muted)]">
          ответь на все задания
        </span>
        <button class="btn btn-primary disabled:opacity-40" :disabled="!canAdvance" @click="next">
          <template v-if="atLast">
            <CircleCheckBig :size="16" :stroke-width="2.5" />
            {{ lesson.status === 'done' ? 'Урок пройден' : 'Завершить урок' }}
          </template>
          <template v-else> Дальше<ArrowRight :size="16" :stroke-width="2.5" /> </template>
        </button>
      </div>
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
