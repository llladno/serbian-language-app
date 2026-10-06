<script setup lang="ts">
// A number that counts to its new value instead of jumping to it. The whole
// point of a reward is watching the balance climb.
import { onBeforeUnmount, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{ value: number; format?: (n: number) => string; duration?: number }>(),
  { duration: 700 },
)

const shown = ref(props.value)
let frame = 0

function stop() {
  if (frame) cancelAnimationFrame(frame)
  frame = 0
}
onBeforeUnmount(stop)

// Counting is decoration. Someone who asked for less motion gets the number.
function instant(): boolean {
  if (props.duration <= 0) return true
  if (typeof requestAnimationFrame !== 'function') return true
  return !!window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
}

watch(
  () => props.value,
  (to) => {
    const from = shown.value
    stop()
    if (from === to || instant()) {
      shown.value = to
      return
    }
    const started = performance.now()
    const tick = (now: number) => {
      const p = Math.min(1, (now - started) / props.duration)
      // easeOutCubic — most of the distance early, so the number spends its
      // time near the value that matters rather than racing past it.
      shown.value = Math.round(from + (to - from) * (1 - Math.pow(1 - p, 3)))
      if (p < 1) {
        frame = requestAnimationFrame(tick)
      } else {
        shown.value = to
        frame = 0
      }
    }
    frame = requestAnimationFrame(tick)
  },
)
</script>

<template>
  <span class="tabular-nums">{{ format ? format(shown) : shown }}</span>
</template>
