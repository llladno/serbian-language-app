<script setup lang="ts">
// A single continuous bar for the whole lesson: one segment per step, no
// gaps. The current step's segment fills in from the left as you clear its
// exercises (currentFraction), so this one bar carries both "which step"
// and "how far into it" — no second bar layered underneath it.
//
// The fill animates via `width` on an inner bar rather than a transition on
// `background` (a gradient) — gradients don't interpolate reliably across
// browsers, so that used to snap instead of smoothly filling. The segment's
// own backing tint (solid color, future → reached) is safe to transition
// directly since it's a plain color, not a gradient.
import type { Step } from '../types'

const props = defineProps<{ steps: Step[]; current: number; currentFraction?: number }>()

function fillPct(i: number) {
  if (i < props.current) {
    // A past step always reads as fully done, even if it technically isn't
    // (planned/unreachable segments never get here) — no fraction to show.
    return 100
  }
  if (i > props.current) return 0
  // The current segment always reflects currentFraction, even when this
  // step's own status is already "done" (reviewing a finished lesson) — so
  // paging back within it visibly un-fills the bar instead of staying
  // pinned at 100% just because the step was completed before.
  return Math.max(0, Math.min(1, props.currentFraction ?? 0)) * 100
}
</script>

<template>
  <div class="flex items-center overflow-hidden rounded-full bg-[var(--ring-track)]">
    <span
      v-for="(s, i) in steps"
      :key="s.id"
      class="relative h-1.5 flex-1 overflow-hidden rounded-full transition-colors duration-300 ease-out"
      :style="{ background: i <= current ? 'color-mix(in srgb, var(--accent) 45%, transparent)' : 'transparent' }"
    >
      <span
        class="absolute inset-y-0 left-0 h-full rounded-full bg-[var(--accent)] transition-[width] duration-300 ease-out"
        :style="{ width: fillPct(i) + '%' }"
      />
    </span>
  </div>
</template>
