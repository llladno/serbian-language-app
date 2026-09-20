<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import type { Progress } from '../types'
import ProgressRing from './ProgressRing.vue'

const progress = ref<Progress | null>(null)
const loading = ref(true)
onMounted(async () => {
  try {
    progress.value = await api.progress()
  } catch {
    // silent — this is a secondary widget, ProgressDashboard already surfaces load errors
  } finally {
    loading.value = false
  }
})

const totalDone = computed(() => (progress.value ? progress.value.phases.reduce((a, p) => a + p.done, 0) : 0))
const totalLessons = computed(() => (progress.value ? progress.value.phases.reduce((a, p) => a + p.total, 0) : 0))
</script>

<template>
  <div v-if="loading" class="card p-5">
    <div class="skel mb-3 h-4 w-40"></div>
    <div class="flex flex-wrap justify-around gap-4">
      <div v-for="i in 3" :key="i" class="flex flex-col items-center gap-1.5">
        <div class="skel h-[72px] w-[72px] rounded-full"></div>
        <div class="skel h-3 w-14"></div>
      </div>
    </div>
  </div>

  <div v-else-if="progress" class="card p-5">
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
