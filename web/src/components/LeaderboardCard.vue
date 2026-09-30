<script setup lang="ts">
import type { LeaderRow } from '../types'

// Fetched once by ProfileView alongside everything else on the page — see
// ProfileView's loadAll().
const props = defineProps<{ name: string; leaders: LeaderRow[] }>()
</script>

<template>
  <RouterLink v-if="leaders.length > 1" to="/rating" class="card block p-5 transition hover:-translate-y-0.5">
    <div class="mb-2 flex items-baseline justify-between">
      <p class="font-bold">Рейтинг</p>
      <span class="text-sm text-[var(--accent)]">все →</span>
    </div>
    <ul class="space-y-1.5 text-sm">
      <li v-for="(r, i) in leaders.slice(0, 3)" :key="r.name" class="flex items-baseline gap-2">
        <span class="w-4 font-mono text-[var(--muted)]">{{ i + 1 }}</span>
        <span class="flex-1 truncate font-semibold" :class="r.name === props.name ? 'text-[var(--accent)]' : ''">{{
          r.name
        }}</span>
        <span class="shrink-0 text-[var(--muted)]">{{ r.lessons_done }}/{{ r.lessons_total }} · {{ r.cards_known }} сл. · {{ r.streak_days }}🔥</span>
      </li>
    </ul>
  </RouterLink>
</template>
