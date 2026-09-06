<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import { useCourseStore } from '../stores/course'
import type { Progress } from '../types'

const progress = ref<Progress | null>(null)
const error = ref<string | null>(null)
const course = useCourseStore()

onMounted(async () => {
  course.load()
  try {
    progress.value = await api.progress()
  } catch (e) {
    error.value = (e as Error).message
  }
})

const continueLesson = computed(() => {
  const p = progress.value
  if (!p) return null
  const inProgress = p.recent_lessons.find((l) => l.status === 'in_progress')
  if (inProgress) return { id: inProgress.lesson, title: inProgress.title, label: 'Продолжить' }
  const done = new Set(p.recent_lessons.filter((l) => l.status === 'done').map((l) => l.lesson))
  const next = course.course?.lessons.find((l) => !l.planned && !done.has(l.id))
  return next ? { id: next.id, title: next.title, label: 'Начать' } : null
})

const totalDone = computed(() =>
  progress.value ? progress.value.phases.reduce((a, p) => a + p.done, 0) : 0,
)
</script>

<template>
  <h1 class="mb-4 text-2xl font-bold">{{ course.course?.title || 'Српски' }}</h1>

  <p v-if="error" class="rounded bg-red-100 p-3 text-red-800 dark:bg-red-950 dark:text-red-200">{{ error }}</p>

  <div v-else-if="progress" class="space-y-4">
    <!-- review card -->
    <RouterLink
      to="/review"
      class="block rounded-xl border border-stone-200 p-4 hover:bg-stone-100 dark:border-stone-700 dark:hover:bg-stone-800"
    >
      <p class="text-lg font-semibold">Повторить слова</p>
      <p class="text-stone-500">
        к повторению: <b>{{ progress.srs.due_today }}</b> ·
        новых: <b>{{ progress.srs.new_available }}</b>
        <span v-if="progress.srs.reviewed_today"> · сегодня уже {{ progress.srs.reviewed_today }}</span>
      </p>
    </RouterLink>

    <!-- continue -->
    <RouterLink
      v-if="continueLesson"
      :to="`/lesson/${continueLesson.id}`"
      class="block rounded-xl border border-stone-200 p-4 hover:bg-stone-100 dark:border-stone-700 dark:hover:bg-stone-800"
    >
      <p class="text-sm text-stone-500">{{ continueLesson.label }}</p>
      <p class="text-lg font-semibold">{{ continueLesson.id }}. {{ continueLesson.title }}</p>
    </RouterLink>

    <!-- progress bars -->
    <div class="rounded-xl border border-stone-200 p-4 dark:border-stone-700">
      <p class="mb-2 font-semibold">Прогресс <span class="text-stone-400">({{ totalDone }} / 30)</span></p>
      <div v-for="ph in progress.phases" :key="ph.id" class="mb-2">
        <div class="mb-0.5 flex justify-between text-sm text-stone-500">
          <span>{{ ph.title }}</span><span>{{ ph.done }} / {{ ph.total }}</span>
        </div>
        <div class="h-2 overflow-hidden rounded bg-stone-200 dark:bg-stone-700">
          <div class="h-full bg-emerald-500" :style="{ width: (ph.total ? (ph.done / ph.total) * 100 : 0) + '%' }" />
        </div>
      </div>
    </div>

    <!-- streak + known -->
    <div class="flex gap-4">
      <div class="flex-1 rounded-xl border border-stone-200 p-4 text-center dark:border-stone-700">
        <p class="text-2xl font-bold">{{ progress.streak_days }} 🔥</p>
        <p class="text-sm text-stone-500">дней подряд</p>
      </div>
      <div class="flex-1 rounded-xl border border-stone-200 p-4 text-center dark:border-stone-700">
        <p class="text-2xl font-bold">{{ progress.srs.known }} / {{ progress.srs.total_cards }}</p>
        <p class="text-sm text-stone-500">слов закреплено</p>
      </div>
    </div>

    <!-- weak spots -->
    <div v-if="progress.weak_exercises.length" class="rounded-xl border border-stone-200 p-4 dark:border-stone-700">
      <p class="mb-2 font-semibold">Слабые места</p>
      <ul class="space-y-1 text-sm">
        <li v-for="w in progress.weak_exercises" :key="w.exercise_id">
          <RouterLink :to="`/lesson/${w.lesson}`" class="hover:underline">
            <span class="text-red-600">{{ w.wrong }}/{{ w.total }}</span>
            — {{ w.prompt || w.exercise_id }}
          </RouterLink>
        </li>
      </ul>
    </div>

    <div v-if="totalDone === 0 && progress.srs.reviewed_today === 0" class="rounded-xl border border-dashed border-stone-300 p-6 text-center text-stone-500 dark:border-stone-600">
      Начни с <RouterLink to="/lesson/01" class="text-amber-600 underline">урока 01</RouterLink>.
    </div>
  </div>
</template>
