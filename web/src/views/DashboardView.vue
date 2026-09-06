<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import { useCourseStore } from '../stores/course'
import type { Progress, Vocab } from '../types'
import ProgressRing from '../components/ProgressRing.vue'
import ActivityHeatmap from '../components/ActivityHeatmap.vue'

const progress = ref<Progress | null>(null)
const wotd = ref<Vocab | null>(null)
const error = ref<string | null>(null)
const course = useCourseStore()

onMounted(async () => {
  course.load()
  try {
    progress.value = await api.progress()
    const vocab = await api.vocab()
    if (vocab.length) {
      const now = new Date()
      const doy = Math.floor((+now - +new Date(now.getFullYear(), 0, 0)) / 86400000)
      wotd.value = vocab[doy % vocab.length]
    }
  } catch (e) {
    error.value = (e as Error).message
  }
})

const continueLesson = computed(() => {
  const p = progress.value
  if (!p) return null
  const inProgress = p.recent_lessons.find((l) => l.status === 'in_progress')
  if (inProgress) return { id: inProgress.lesson, title: inProgress.title, label: 'Продолжить урок' }
  const done = new Set(p.recent_lessons.filter((l) => l.status === 'done').map((l) => l.lesson))
  const next = course.course?.lessons?.find((l) => !l.planned && !done.has(l.id))
  return next ? { id: next.id, title: next.title, label: 'Следующий урок' } : null
})

const totalDone = computed(() =>
  progress.value ? progress.value.phases.reduce((a, p) => a + p.done, 0) : 0,
)
const todayCount = computed(() => {
  const p = progress.value
  if (!p) return 0
  const today = new Date().toISOString().slice(0, 10)
  return p.activity.find((a) => a.date === today)?.count ?? 0
})
const dueTotal = computed(() =>
  progress.value ? progress.value.srs.due_today + progress.value.srs.new_available : 0,
)
</script>

<template>
  <p v-if="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

  <div v-else-if="progress" class="space-y-4">
    <!-- hero: review + daily goal -->
    <RouterLink to="/review" class="card block p-5 transition hover:-translate-y-0.5">
      <div class="flex items-center gap-4">
        <div class="relative shrink-0">
          <ProgressRing :value="todayCount" :max="progress.daily_goal" :size="76" />
          <div class="absolute inset-0 flex flex-col items-center justify-center leading-none">
            <span class="text-lg font-extrabold">{{ todayCount }}</span>
            <span class="text-[10px] text-[var(--muted)]">/ {{ progress.daily_goal }}</span>
          </div>
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-lg font-extrabold">
            {{ dueTotal > 0 ? 'Повторить слова' : 'Слова на сегодня — всё' }}
          </p>
          <p class="text-sm text-[var(--muted)]">
            <template v-if="dueTotal > 0">
              к повторению <b class="text-[var(--fg)]">{{ progress.srs.due_today }}</b> ·
              новых <b class="text-[var(--fg)]">{{ progress.srs.new_available }}</b>
            </template>
            <template v-else>сделано {{ progress.srs.reviewed_today }} — возвращайся завтра</template>
          </p>
        </div>
        <span class="text-2xl text-[var(--accent)]">→</span>
      </div>
    </RouterLink>

    <!-- streak + heatmap -->
    <div class="card p-5">
      <div class="mb-3 flex items-baseline justify-between">
        <p class="font-bold">
          <span class="text-xl">{{ progress.streak_days }}</span>
          <span class="text-[var(--muted)]">&nbsp;{{ progress.streak_days === 1 ? 'день' : 'дней' }} подряд</span>
          <span v-if="progress.streak_days > 0">&nbsp;🔥</span>
        </p>
        <p class="text-sm text-[var(--muted)]">{{ progress.srs.known }} / {{ progress.srs.total_cards }} закреплено</p>
      </div>
      <ActivityHeatmap :activity="progress.activity" />
    </div>

    <!-- continue -->
    <RouterLink
      v-if="continueLesson"
      :to="`/lesson/${continueLesson.id}`"
      class="card block p-4 transition hover:-translate-y-0.5"
    >
      <p class="text-xs uppercase tracking-wide text-[var(--muted)]">{{ continueLesson.label }}</p>
      <p class="text-lg font-bold">{{ continueLesson.id }}. {{ continueLesson.title }}</p>
    </RouterLink>

    <!-- word of the day -->
    <div v-if="wotd" class="card p-4">
      <p class="mb-1 text-xs uppercase tracking-wide text-[var(--muted)]">Слово дня</p>
      <p class="serbian text-2xl font-semibold">{{ wotd.latin }}</p>
      <p class="text-[var(--muted)]">{{ wotd.cyrillic }} — {{ wotd.ru }}</p>
      <p v-if="wotd.note" class="mt-0.5 text-sm text-[var(--muted)]">{{ wotd.note }}</p>
    </div>

    <!-- progress rings per phase -->
    <div class="card p-5">
      <p class="mb-3 font-bold">Прогресс <span class="text-[var(--muted)]">· {{ totalDone }} / 30 уроков</span></p>
      <div class="flex justify-around">
        <div v-for="ph in progress.phases" :key="ph.id" class="flex flex-col items-center gap-1">
          <div class="relative">
            <ProgressRing :value="ph.done" :max="ph.total" :size="64" :stroke="7" />
            <div class="absolute inset-0 flex items-center justify-center text-sm font-bold">
              {{ ph.done }}/{{ ph.total }}
            </div>
          </div>
          <span class="text-xs text-[var(--muted)]">Фаза {{ ph.id }}</span>
        </div>
      </div>
    </div>

    <!-- weak spots -->
    <div v-if="progress.weak_exercises.length" class="card p-5">
      <p class="mb-2 font-bold">Стоит повторить</p>
      <ul class="space-y-1.5 text-sm">
        <li v-for="w in progress.weak_exercises" :key="w.exercise_id">
          <RouterLink :to="`/lesson/${w.lesson}`" class="flex gap-2 hover:underline">
            <span class="shrink-0 font-mono text-[var(--bad)]">{{ w.wrong }}/{{ w.total }}</span>
            <span class="text-[var(--muted)]">{{ w.prompt || w.exercise_id }}</span>
          </RouterLink>
        </li>
      </ul>
    </div>

    <div
      v-if="totalDone === 0 && progress.srs.reviewed_today === 0"
      class="card p-6 text-center text-[var(--muted)]"
    >
      Начни с <RouterLink to="/lesson/01" class="font-semibold text-[var(--accent)]">урока 01</RouterLink>.
    </div>
  </div>
</template>
