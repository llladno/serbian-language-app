<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { useSessionStore } from '../stores/session'
import type { LeaderRow } from '../types'

const rows = ref<LeaderRow[]>([])
const error = ref<string | null>(null)
const loading = ref(true)
const session = useSessionStore()

onMounted(async () => {
  try {
    rows.value = await api.leaderboard()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
})

function activeLabel(d: string) {
  if (!d) return 'ещё не занимался'
  const today = new Date().toISOString().slice(0, 10)
  const y = new Date(Date.now() - 86400000).toISOString().slice(0, 10)
  if (d === today) return 'сегодня'
  if (d === y) return 'вчера'
  return d
}
</script>

<template>
  <h1 class="mb-1 text-2xl font-extrabold">Рейтинг</h1>
  <p class="mb-4 text-sm text-[var(--muted)]">Прогресс всех, кто занимается по этому курсу.</p>

  <p v-if="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

  <div v-else-if="loading" class="space-y-2">
    <div v-for="i in 6" :key="i" class="card flex items-center gap-3 p-4">
      <div class="skel h-5 w-5 shrink-0 rounded"></div>
      <div class="min-w-0 flex-1 space-y-1.5">
        <div class="skel h-4 w-2/5"></div>
        <div class="skel h-3.5 w-3/5"></div>
      </div>
      <div class="shrink-0 space-y-1.5 text-right">
        <div class="skel ml-auto h-4 w-8"></div>
        <div class="skel h-3 w-16"></div>
      </div>
    </div>
  </div>

  <div v-else class="space-y-2">
    <div
      v-for="(r, i) in rows"
      :key="r.name"
      class="card flex items-center gap-3 p-4"
      :class="r.name === session.user?.name ? 'ring-2 ring-[var(--accent)]' : ''"
    >
      <span class="w-5 shrink-0 text-center font-mono text-sm text-[var(--muted)]">{{ i + 1 }}</span>
      <div class="min-w-0 flex-1">
        <p class="font-bold">
          {{ r.name }}
          <span v-if="r.name === session.user?.name" class="text-xs font-normal text-[var(--accent)]">— это ты</span>
        </p>
        <p class="text-sm text-[var(--muted)]">
          {{ r.lessons_done }}/{{ r.lessons_total }} уроков · {{ r.cards_known }} слов ·
          {{ activeLabel(r.last_active) }}
        </p>
      </div>
      <div class="shrink-0 text-right">
        <p class="font-bold">{{ r.streak_days }}<span v-if="r.streak_days" class="ml-0.5">🔥</span></p>
        <p class="text-[10px] text-[var(--muted)]">дней подряд</p>
      </div>
    </div>

    <p v-if="!rows.length" class="card p-6 text-center text-[var(--muted)]">Пока никого.</p>
  </div>
</template>
