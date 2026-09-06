<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { getAccount } from '../account'
import type { LeaderRow } from '../types'

const rows = ref<LeaderRow[]>([])
const error = ref<string | null>(null)
const me = getAccount()

onMounted(async () => {
  try {
    rows.value = await api.leaderboard()
  } catch (e) {
    error.value = (e as Error).message
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
  <h1 class="mb-1 text-2xl font-extrabold">Люди</h1>
  <p class="mb-4 text-sm text-[var(--muted)]">Прогресс всех, кто занимается по этому курсу.</p>

  <p v-if="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

  <div v-else class="space-y-2">
    <div
      v-for="(r, i) in rows"
      :key="r.name"
      class="card flex items-center gap-3 p-4"
      :class="r.name === me ? 'ring-2 ring-[var(--accent)]' : ''"
    >
      <span class="w-5 shrink-0 text-center font-mono text-sm text-[var(--muted)]">{{ i + 1 }}</span>
      <div class="min-w-0 flex-1">
        <p class="font-bold">
          {{ r.name }}
          <span v-if="r.name === me" class="text-xs font-normal text-[var(--accent)]">— это ты</span>
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
