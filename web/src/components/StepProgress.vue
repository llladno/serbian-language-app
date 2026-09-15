<script setup lang="ts">
// A single continuous bar for the whole lesson: one segment per step, no
// gaps. The current step's segment fills in from the left as you clear its
// exercises (currentFraction), so this one bar carries both "which step"
// and "how far into it" — no second bar layered underneath it.
import type { Step } from '../types'

const props = defineProps<{ steps: Step[]; current: number; currentFraction?: number }>()

function segmentStyle(i: number) {
  if (i < props.current || props.steps[i].status === 'done') {
    return { background: 'var(--accent)' }
  }
  if (i > props.current) {
    return {}
  }
  const pct = Math.max(0, Math.min(1, props.currentFraction ?? 0)) * 100
  const soft = 'color-mix(in srgb, var(--accent) 45%, transparent)'
  return { background: `linear-gradient(to right, var(--accent) ${pct}%, ${soft} ${pct}%)` }
}
</script>

<template>
  <div class="flex items-center overflow-hidden rounded-full bg-[var(--ring-track)]">
    <span
      v-for="(s, i) in steps"
      :key="s.id"
      class="h-1.5 flex-1 transition-[background] duration-300"
      :style="segmentStyle(i)"
    />
  </div>
</template>
