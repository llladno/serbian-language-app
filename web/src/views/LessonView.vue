<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { useCourseStore } from '../stores/course'
import type { Lesson, ExerciseBlock, LessonAttempts } from '../types'
import MarkdownView from '../components/MarkdownView.vue'
import ExerciseBlockView from '../components/exercises/ExerciseBlock.vue'
import Confetti from '../components/Confetti.vue'

const route = useRoute()
const store = useCourseStore()

const lesson = ref<Lesson | null>(null)
const blocks = ref<ExerciseBlock[]>([])
const priors = ref<LessonAttempts>({})
const error = ref<string | null>(null)
const celebrate = ref(false)
const resuming = ref(false)

async function loadLesson(id: string) {
  lesson.value = null
  blocks.value = []
  priors.value = {}
  error.value = null
  celebrate.value = false
  resuming.value = false
  try {
    lesson.value = await api.lesson(id)
    if (!lesson.value.planned) {
      const [bl, pr] = await Promise.all([api.exercises(id), api.lessonAttempts(id)])
      blocks.value = bl
      priors.value = pr
      await nextTick()
      setTimeout(jumpToFirstUnanswered, 120)
    } else {
      window.scrollTo(0, 0)
    }
  } catch (e) {
    error.value = (e as Error).message
  }
}

watch(() => route.params.id as string, loadLesson, { immediate: true })

function jumpToFirstUnanswered() {
  const all = blocks.value.flatMap((b) => b.exercises.map((e) => e.id))
  const answered = new Set(Object.keys(priors.value))
  if (answered.size === 0 || answered.size === all.length) {
    window.scrollTo(0, 0)
    return
  }
  const next = all.find((id) => !answered.has(id))
  if (!next) return
  const el = document.querySelector(`[data-ex="${next}"]`)
  if (el) {
    resuming.value = true
    el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
}

async function markDone() {
  if (!lesson.value || lesson.value.status === 'done') return
  await store.markDone(lesson.value.id)
  lesson.value.status = 'done'
  celebrate.value = true
  setTimeout(() => (celebrate.value = false), 3500)
}

async function resetLesson() {
  if (!lesson.value) return
  if (!confirm('Сбросить все ответы в этом уроке и пройти заново?')) return
  await api.resetLesson(lesson.value.id)
  store.setStatus(lesson.value.id, 'not_started')
  await loadLesson(lesson.value.id)
}
</script>

<template>
  <Confetti v-if="celebrate" />
  <p v-if="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

  <template v-else-if="lesson">
    <RouterLink to="/course" class="text-sm text-[var(--muted)] hover:text-[var(--fg)]">← к курсу</RouterLink>

    <div v-if="lesson.planned" class="card mt-4 border-dashed p-8 text-center">
      <h1 class="text-xl font-bold">{{ lesson.title }}</h1>
      <p class="mt-1 text-[var(--muted)]">{{ lesson.subtitle }}</p>
      <p class="mt-4 text-sm text-[var(--muted)]">Урок ещё не готов — скоро появится.</p>
    </div>

    <template v-else>
      <MarkdownView :source="lesson.markdown" class="mt-3" />

      <div v-if="blocks.length" class="mt-12 space-y-12 border-t border-[var(--border)] pt-8">
        <div class="flex items-center justify-between">
          <h2 class="text-xl font-extrabold">Упражнения</h2>
          <button
            v-if="Object.keys(priors).length"
            class="text-sm font-medium text-[var(--muted)] hover:text-[var(--accent)]"
            @click="resetLesson"
          >
            Пройти заново
          </button>
        </div>
        <p v-if="resuming" class="-mt-8 text-sm text-[var(--accent)]">
          ↓ продолжаешь с того места, где остановился
        </p>
        <ExerciseBlockView
          v-for="b in blocks"
          :key="b.id"
          :lesson="lesson.id"
          :block="b"
          :priors="priors"
        />
      </div>

      <div class="sticky bottom-0 mt-12 -mx-4 border-t border-[var(--border)] bg-[var(--bg)]/90 px-4 py-3 backdrop-blur">
        <button
          class="btn w-full sm:w-auto"
          :class="lesson.status === 'done' ? 'btn-ghost' : 'btn-primary'"
          :disabled="lesson.status === 'done'"
          @click="markDone"
        >
          {{ lesson.status === 'done' ? '✓ Урок пройден' : 'Отметить пройденным' }}
        </button>
      </div>
    </template>
  </template>
</template>
