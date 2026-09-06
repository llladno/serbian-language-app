<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { useCourseStore } from '../stores/course'
import type { Lesson, ExerciseBlock } from '../types'
import MarkdownView from '../components/MarkdownView.vue'
import ExerciseBlockView from '../components/exercises/ExerciseBlock.vue'
import Confetti from '../components/Confetti.vue'

const route = useRoute()
const store = useCourseStore()

const lesson = ref<Lesson | null>(null)
const blocks = ref<ExerciseBlock[]>([])
const error = ref<string | null>(null)
const celebrate = ref(false)

watch(
  () => route.params.id as string,
  async (id) => {
    lesson.value = null
    blocks.value = []
    error.value = null
    celebrate.value = false
    try {
      lesson.value = await api.lesson(id)
      if (!lesson.value.planned) blocks.value = await api.exercises(id)
      window.scrollTo(0, 0)
    } catch (e) {
      error.value = (e as Error).message
    }
  },
  { immediate: true },
)

async function markDone() {
  if (!lesson.value || lesson.value.status === 'done') return
  await store.markDone(lesson.value.id)
  lesson.value.status = 'done'
  celebrate.value = true
  setTimeout(() => (celebrate.value = false), 3500)
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
        <h2 class="text-xl font-extrabold">Упражнения</h2>
        <ExerciseBlockView v-for="b in blocks" :key="b.id" :lesson="lesson.id" :block="b" />
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
