<script setup lang="ts">
import { computed } from 'vue'
import type { LessonStatus } from '../types'

const props = defineProps<{ status: LessonStatus; planned?: boolean }>()

const label = computed(() => {
  if (props.planned) return 'запланирован'
  return { not_started: '', in_progress: 'в процессе', done: 'пройден' }[props.status]
})

const cls = computed(() => {
  if (props.planned) return 'bg-stone-200 text-stone-500 dark:bg-stone-700 dark:text-stone-400'
  return {
    not_started: '',
    in_progress: 'bg-amber-200 text-amber-900 dark:bg-amber-900 dark:text-amber-200',
    done: 'bg-emerald-200 text-emerald-900 dark:bg-emerald-900 dark:text-emerald-200',
  }[props.status]
})
</script>

<template>
  <span v-if="label" class="rounded px-1.5 py-0.5 text-xs font-medium" :class="cls">{{ label }}</span>
</template>
