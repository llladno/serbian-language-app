<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{ value: number; max: number; size?: number; stroke?: number; color?: string }>(),
  { size: 72, stroke: 8, color: 'var(--accent)' },
)

const r = computed(() => (props.size - props.stroke) / 2)
const circ = computed(() => 2 * Math.PI * r.value)
const pct = computed(() => (props.max <= 0 ? 0 : Math.min(1, props.value / props.max)))
const dash = computed(() => `${circ.value * pct.value} ${circ.value}`)
</script>

<template>
  <svg :width="size" :height="size" :viewBox="`0 0 ${size} ${size}`" class="-rotate-90">
    <circle
      :cx="size / 2"
      :cy="size / 2"
      :r="r"
      fill="none"
      stroke="var(--ring-track)"
      :stroke-width="stroke"
    />
    <circle
      :cx="size / 2"
      :cy="size / 2"
      :r="r"
      fill="none"
      :stroke="color"
      :stroke-width="stroke"
      stroke-linecap="round"
      :stroke-dasharray="dash"
      style="transition: stroke-dasharray 0.5s ease"
    />
  </svg>
</template>
