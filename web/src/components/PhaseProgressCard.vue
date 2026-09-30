<script setup lang="ts">
import { computed } from 'vue'
import type { Progress } from '../types'
import ProgressRing from './ProgressRing.vue'

// Fetched once by ProfileView and shared with ProgressDashboard — this used
// to fetch the same /api/progress a second time on its own, which was both
// wasteful and one of the causes of the staggered-skeleton flicker.
const props = defineProps<{ progress: Progress }>()

const totalDone = computed(() => props.progress.phases.reduce((a, p) => a + p.done, 0))
const totalLessons = computed(() => props.progress.phases.reduce((a, p) => a + p.total, 0))
</script>

<template>
  <div class="card p-5">
    <p class="mb-3 font-bold">
      Прогресс <span class="text-[var(--muted)]">· {{ totalDone }} / {{ totalLessons }} уроков</span>
    </p>
    <div class="flex flex-wrap justify-around gap-4">
      <div v-for="ph in progress.phases" :key="ph.id" class="flex flex-col items-center gap-1.5">
        <div class="relative">
          <ProgressRing :value="ph.done" :max="ph.total" :size="72" :stroke="7" />
          <div class="absolute inset-0 flex items-center justify-center text-sm font-bold">
            {{ ph.done }}/{{ ph.total }}
          </div>
        </div>
        <span class="text-xs text-[var(--muted)]">Фаза {{ ph.id }}</span>
      </div>
    </div>
  </div>
</template>
