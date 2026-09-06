<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useReviewStore } from '../stores/review'
import WordMedia from '../components/WordMedia.vue'
import SpeakButton from '../components/SpeakButton.vue'

const store = useReviewStore()
const { current, remaining, total, sessionCount, tally, loading, error } = storeToRefs(store)
const revealed = ref(false)

onMounted(() => store.load())

const GRADES = [
  { g: 0, label: 'Опять', key: 'again', cls: 'bg-[var(--bad)]' },
  { g: 1, label: 'Трудно', key: 'hard', cls: 'bg-[#e08a1e]' },
  { g: 2, label: 'Хорошо', key: 'good', cls: 'bg-[var(--good)]' },
  { g: 3, label: 'Легко', key: 'easy', cls: 'bg-[#8b6ff0]' },
]

function fmtInterval(days: number) {
  if (days <= 0) return '<10м'
  if (days === 1) return '1 дн'
  if (days < 30) return `${days} дн`
  if (days < 365) return `${Math.round(days / 30)} мес`
  return `${(days / 365).toFixed(days % 365 ? 1 : 0)} г`
}

const done = computed(() => total.value > 0 && !current.value)
const progressPct = computed(() =>
  total.value ? Math.round((sessionCount.value / total.value) * 100) : 0,
)

async function grade(g: number) {
  await store.grade(g)
  revealed.value = false
}

function onKey(e: KeyboardEvent) {
  if (!current.value) return
  if (e.code === 'Space' || e.code === 'Enter') {
    e.preventDefault()
    if (!revealed.value) revealed.value = true
    return
  }
  if (revealed.value && ['1', '2', '3', '4'].includes(e.key)) grade(Number(e.key) - 1)
}
onMounted(() => window.addEventListener('keydown', onKey))
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <p v-if="loading" class="text-[var(--muted)]">Загрузка…</p>
  <p v-else-if="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

  <!-- session summary -->
  <div v-else-if="done" class="card p-8 text-center">
    <p class="text-4xl">🎉</p>
    <p class="mt-2 text-lg font-bold">Сессия закончена</p>
    <p class="mt-1 text-[var(--muted)]">{{ sessionCount }} карточек</p>
    <div class="mx-auto mt-4 grid max-w-xs grid-cols-4 gap-2 text-sm">
      <div v-for="(b, i) in GRADES" :key="i" class="rounded-lg bg-[var(--bg-soft)] py-2">
        <p class="font-bold">{{ tally[i] }}</p>
        <p class="text-xs text-[var(--muted)]">{{ b.label }}</p>
      </div>
    </div>
    <div class="mt-6 flex justify-center gap-2">
      <button class="btn btn-ghost" @click="store.load()">Ещё раз</button>
      <RouterLink to="/" class="btn btn-primary">На главную</RouterLink>
    </div>
  </div>

  <!-- empty -->
  <div v-else-if="!current" class="card p-8 text-center">
    <p class="text-3xl">✨</p>
    <p class="mt-2 text-lg font-bold">На сегодня всё</p>
    <p class="mt-1 text-[var(--muted)]">Новые карточки и повторения появятся завтра.</p>
    <RouterLink to="/" class="btn btn-primary mt-5">На главную</RouterLink>
  </div>

  <!-- card -->
  <div v-else class="space-y-5">
    <div class="h-1.5 overflow-hidden rounded-full bg-[var(--ring-track)]">
      <div class="h-full rounded-full bg-[var(--accent)] transition-all duration-300" :style="{ width: progressPct + '%' }" />
    </div>
    <p class="text-center text-xs text-[var(--muted)]">осталось {{ remaining }}</p>

    <div
      class="card flex min-h-[13rem] cursor-pointer flex-col items-center justify-center p-8 text-center pop"
      :key="current.card_id"
      @click="revealed = true"
    >
      <p v-if="current.kind === 'ff'" class="mb-2 text-[10px] uppercase tracking-widest text-[var(--accent)]">ложный друг</p>
      <div class="flex items-center gap-2">
        <p class="serbian text-4xl font-semibold">{{ current.front }}</p>
        <SpeakButton :src="current.audio" :size="36" />
      </div>
      <p v-if="current.cyrillic" class="mt-1 text-sm text-[var(--muted)]">{{ current.cyrillic }}</p>

      <Transition name="fade">
        <div v-if="revealed" class="mt-4 flex flex-col items-center border-t border-[var(--border)] pt-4">
          <WordMedia :image="current.image" :emoji="current.emoji" :alt="current.back" :size="112" />
          <p class="mt-3 text-xl">{{ current.back }}</p>
          <p v-if="current.note" class="mt-1 text-sm text-[var(--muted)]">{{ current.note }}</p>
        </div>
      </Transition>
      <p v-if="!revealed" class="mt-5 text-xs text-[var(--muted)]">нажми или пробел</p>
    </div>

    <div v-if="revealed" class="grid grid-cols-4 gap-2 pop">
      <button
        v-for="b in GRADES"
        :key="b.g"
        class="flex flex-col items-center rounded-xl py-2 text-white transition active:scale-95"
        :class="b.cls"
        @click="grade(b.g)"
      >
        <span class="text-sm font-semibold">{{ b.label }}</span>
        <span class="text-[11px] opacity-80">{{ fmtInterval(current.preview[b.key as 'again']) }}</span>
      </button>
    </div>
    <button v-else class="btn btn-primary w-full" @click="revealed = true">Показать</button>
  </div>
</template>
