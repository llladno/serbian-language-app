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
  // A hidden tab does not run animation frames, so a count started there would
  // sit frozen on a number that was never the balance until the tab is looked
  // at again. Nobody is watching anyway.
  if (typeof document !== 'undefined' && document.hidden) return true
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
    // The clock comes from the first frame, not from performance.now():
    // the two are the same origin in a browser but not everywhere, and a
    // mismatch sends the easing off in the wrong direction.
    let started = 0
    const tick = (now: number) => {
      if (!started) started = now
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
