<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useReviewStore } from '../stores/review'

const store = useReviewStore()
const { current, remaining, sessionCount, loading, error } = storeToRefs(store)
const revealed = ref(false)

onMounted(() => store.load())

const GRADES = [
  { g: 0, label: 'Опять', key: '1', cls: 'bg-red-600' },
  { g: 1, label: 'Трудно', key: '2', cls: 'bg-orange-500' },
  { g: 2, label: 'Хорошо', key: '3', cls: 'bg-emerald-600' },
  { g: 3, label: 'Легко', key: '4', cls: 'bg-sky-600' },
]

async function grade(g: number) {
  await store.grade(g)
  revealed.value = false
}

function onKey(e: KeyboardEvent) {
  if (!current.value) return
  if (e.code === 'Space') {
    e.preventDefault()
    revealed.value = true
    return
  }
  if (revealed.value && ['1', '2', '3', '4'].includes(e.key)) {
    grade(Number(e.key) - 1)
  }
}

onMounted(() => window.addEventListener('keydown', onKey))
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <h1 class="mb-4 text-2xl font-bold">Повторение</h1>

  <p v-if="loading" class="text-stone-500">Загрузка…</p>
  <p v-else-if="error" class="rounded bg-red-100 p-3 text-red-800 dark:bg-red-950 dark:text-red-200">{{ error }}</p>

  <div v-else-if="!current" class="rounded-lg border border-stone-200 p-8 text-center dark:border-stone-700">
    <p class="text-lg">На сегодня всё 🎉</p>
    <p class="mt-1 text-sm text-stone-500">Повторено карточек: {{ sessionCount }}</p>
    <RouterLink to="/" class="mt-4 inline-block text-amber-600 underline">На дашборд</RouterLink>
  </div>

  <div v-else class="space-y-6">
    <p class="text-sm text-stone-500">осталось: {{ remaining }} · сделано: {{ sessionCount }}</p>

    <div
      class="flex min-h-[10rem] cursor-pointer flex-col items-center justify-center rounded-xl border border-stone-200 p-8 text-center dark:border-stone-700"
      @click="revealed = true"
    >
      <p class="text-3xl font-semibold">{{ current.front }}</p>
      <p v-if="current.cyrillic" class="mt-1 text-sm text-stone-400">{{ current.cyrillic }}</p>

      <template v-if="revealed">
        <hr class="my-4 w-24 border-stone-300 dark:border-stone-600" />
        <p class="text-xl">{{ current.back }}</p>
        <p v-if="current.note" class="mt-1 text-sm text-stone-500">{{ current.note }}</p>
      </template>
      <p v-else class="mt-4 text-xs text-stone-400">нажми или пробел, чтобы показать</p>
    </div>

    <div v-if="revealed" class="grid grid-cols-4 gap-2">
      <button
        v-for="b in GRADES"
        :key="b.g"
        class="rounded px-2 py-2 text-sm text-white"
        :class="b.cls"
        @click="grade(b.g)"
      >
        {{ b.label }}<span class="ml-1 opacity-60">{{ b.key }}</span>
      </button>
    </div>
  </div>
</template>
