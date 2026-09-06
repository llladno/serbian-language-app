<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { useCourseStore } from '../stores/course'
import type { Lesson, ExerciseBlock } from '../types'
import MarkdownView from '../components/MarkdownView.vue'
import ExerciseBlockView from '../components/exercises/ExerciseBlock.vue'

const route = useRoute()
const store = useCourseStore()

const lesson = ref<Lesson | null>(null)
const blocks = ref<ExerciseBlock[]>([])
const error = ref<string | null>(null)
const savedDone = ref(false)

watch(
  () => route.params.id as string,
  async (id) => {
    lesson.value = null
    blocks.value = []
    error.value = null
    savedDone.value = false
    try {
      lesson.value = await api.lesson(id)
      if (!lesson.value.planned) blocks.value = await api.exercises(id)
    } catch (e) {
      error.value = (e as Error).message
    }
  },
  { immediate: true },
)

async function markDone() {
  if (!lesson.value) return
  await store.markDone(lesson.value.id)
  lesson.value.status = 'done'
  savedDone.value = true
}
</script>

<template>
  <p v-if="error" class="rounded bg-red-100 p-3 text-red-800 dark:bg-red-950 dark:text-red-200">{{ error }}</p>

  <template v-else-if="lesson">
    <RouterLink to="/course" class="text-sm text-stone-500 hover:underline">← к курсу</RouterLink>

    <div v-if="lesson.planned" class="mt-4 rounded-lg border border-dashed border-stone-300 p-6 text-center dark:border-stone-600">
      <h1 class="text-xl font-bold">{{ lesson.title }}</h1>
      <p class="mt-1 text-stone-500">{{ lesson.subtitle }}</p>
      <p class="mt-4 text-sm text-stone-400">Урок ещё не готов — скоро появится.</p>
    </div>

    <template v-else>
      <MarkdownView :source="lesson.markdown" class="mt-2" />

      <div v-if="blocks.length" class="mt-10 space-y-10 border-t border-stone-200 pt-6 dark:border-stone-700">
        <h2 class="text-xl font-bold">Упражнения</h2>
        <ExerciseBlockView
          v-for="b in blocks"
          :key="b.id"
          :lesson="lesson.id"
          :block="b"
        />
      </div>

      <div class="sticky bottom-0 mt-10 border-t border-stone-200 bg-[var(--bg)] py-3 dark:border-stone-700">
        <button
          class="rounded bg-emerald-600 px-4 py-2 text-white disabled:opacity-50"
          :disabled="lesson.status === 'done'"
          @click="markDone"
        >
          {{ lesson.status === 'done' ? '✓ Пройден' : 'Отметить пройденным' }}
        </button>
        <span v-if="savedDone" class="ml-3 text-sm text-emerald-600">сохранено</span>
      </div>
    </template>
  </template>
</template>
