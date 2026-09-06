<script setup lang="ts">
import { computed } from 'vue'
import type { LessonStatus } from '../types'

const props = defineProps<{ status: LessonStatus; planned?: boolean }>()

const label = computed(() => {
  if (props.planned) return 'скоро'
  return { not_started: '', in_progress: 'в процессе', done: 'пройден' }[props.status]
})

const style = computed(() => {
  if (props.planned) return { background: 'var(--bg-soft)', color: 'var(--muted)' }
  return {
    not_started: {},
    in_progress: { background: 'var(--accent-soft)', color: 'var(--accent)' },
    done: { background: 'color-mix(in srgb, var(--good) 16%, transparent)', color: 'var(--good)' },
  }[props.status]
})
</script>

<template>
  <span
    v-if="label"
    class="shrink-0 rounded-full px-2 py-0.5 text-xs font-semibold"
    :style="style"
    >{{ label }}</span
  >
</template>
