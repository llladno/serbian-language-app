<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { useCourseStore } from '../stores/course'
import type { Lesson, ExerciseBlock, LessonAttempts, Step } from '../types'
import MarkdownView from '../components/MarkdownView.vue'
import ReadingText from '../components/ReadingText.vue'
import StepProgress from '../components/StepProgress.vue'
import ExerciseBlockView from '../components/exercises/ExerciseBlock.vue'
import Confetti from '../components/Confetti.vue'

const route = useRoute()
const store = useCourseStore()

const lesson = ref<Lesson | null>(null)
const blocks = ref<ExerciseBlock[]>([])
const priors = ref<LessonAttempts>({})
const error = ref<string | null>(null)
const celebrate = ref(false)

const idx = ref(0)
const graded = reactive<Record<string, boolean>>({})

const KIND_META: Record<string, { icon: string; label: string }> = {
  teach: { icon: '▤', label: 'Теория' },
  practice: { icon: '✎', label: 'Практика' },
  reading: { icon: '📖', label: 'Чтение' },
  checkpoint: { icon: '✓', label: 'Проверка' },
}
const kindMeta = (k: string) => KIND_META[k] ?? { icon: '•', label: k }

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

function firstUnfinished(): number {
  const i = steps.value.findIndex((s) => s.status !== 'done')
  return i === -1 ? 0 : i
}

async function loadLesson(id: string) {
  lesson.value = null
  blocks.value = []
  priors.value = {}
  error.value = null
  celebrate.value = false
  for (const k of Object.keys(graded)) delete graded[k]
  try {
    lesson.value = await api.lesson(id)
    if (!lesson.value.planned) {
      const [bl, pr] = await Promise.all([api.exercises(id), api.lessonAttempts(id)])
      blocks.value = bl
      priors.value = pr
      for (const [exId, a] of Object.entries(pr)) graded[exId] = a.correct
    }
    const wanted = route.query.step as string | undefined
    const at = wanted ? steps.value.findIndex((s) => s.id === wanted) : -1
    idx.value = at >= 0 ? at : firstUnfinished()
    markSeen()
    window.scrollTo(0, 0)
  } catch (e) {
    error.value = (e as Error).message
  }
}

watch(() => route.params.id as string, loadLesson, { immediate: true })

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
      <header class="card mt-1 p-4">
        <div class="flex items-center justify-between gap-3 text-xs text-[var(--muted)]">
          <RouterLink to="/course" class="hover:text-[var(--fg)]">← {{ lesson.title }}</RouterLink>
          <button
            v-if="Object.keys(priors).length"
            class="font-medium hover:text-[var(--accent)]"
            @click="resetLesson"
          >
            ↺ заново
          </button>
        </div>

        <div class="mt-2 flex items-center gap-2">
          <span
            class="rounded-full bg-[var(--accent-soft)] px-2 py-0.5 text-[11px] font-semibold text-[var(--accent)]"
          >
            {{ kindMeta(cur.kind).icon }} {{ kindMeta(cur.kind).label }}
          </span>
          <h1 class="truncate text-base font-extrabold">{{ cur.title }}</h1>
          <span class="ml-auto shrink-0 text-xs tabular-nums text-[var(--muted)]">
            {{ idx + 1 }}/{{ steps.length }}
          </span>
        </div>

        <StepProgress class="mt-3" :steps="steps" :current="idx" />
      </header>

      <div class="mt-5">
        <MarkdownView v-if="cur.kind === 'teach'" :source="cur.markdown ?? ''" />

        <template v-else-if="cur.kind === 'reading'">
          <MarkdownView v-if="cur.markdown && !cur.markdown_ru" :source="cur.markdown" />
          <ReadingText v-else :serbian="cur.markdown ?? ''" :translation="cur.markdown_ru" />
          <div v-if="curBlock" class="mt-8 space-y-8">
            <ExerciseBlockView :lesson="lesson.id" :block="curBlock" :priors="priors" @graded="onGraded" />
          </div>
        </template>

        <div v-else-if="curBlock" class="space-y-8">
          <p v-if="cur.markdown" class="text-sm text-[var(--muted)]">{{ cur.markdown }}</p>
          <ExerciseBlockView :lesson="lesson.id" :block="curBlock" :priors="priors" @graded="onGraded" />
        </div>
      </div>

      <div
        class="sticky bottom-0 mt-10 -mx-4 flex items-center justify-between gap-3 border-t border-[var(--border)] bg-[var(--bg)]/90 px-4 py-3 backdrop-blur"
      >
        <button class="btn btn-ghost" :disabled="idx === 0" @click="back">← Назад</button>
        <span v-if="!canAdvance" class="text-xs text-[var(--muted)]">ответь на все задания шага</span>
        <button class="btn btn-primary disabled:opacity-40" :disabled="!canAdvance" @click="next">
          {{ atLast ? (lesson.status === 'done' ? '✓ Урок пройден' : 'Завершить урок') : 'Дальше →' }}
        </button>
      </div>
    </template>
  </template>
</template>
