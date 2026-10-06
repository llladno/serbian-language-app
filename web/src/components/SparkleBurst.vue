<script setup lang="ts">
// A layer of pixel sparkles for the reward modal. Positions, sizes and timings
// are randomised once per mount, so two rewards never look identical, and the
// twinkle keyframes step between whole-pixel scales (0.25 / 0.5 / 1 of a 4x
// render) instead of interpolating — a smooth scale would blur the pixels.
import { SPARKLE_PX, sparkleUrl, type SparkleColor, type SparkleSize } from '../lib/feathers'

const props = withDefaults(defineProps<{ count?: number }>(), { count: 18 })

const COLORS: SparkleColor[] = ['pale', 'gold', 'amber', 'cream']
// Small sparkles are the common case, so the big ones land as accents.
const SIZE_BAG: SparkleSize[] = ['s', 's', 'm', 'm', 'm', 'l', 'l', 'l', 'xl']
// On-screen pixels per sprite pixel at full scale.
const BASE = 4

const pick = <T,>(a: T[]): T => a[Math.floor(Math.random() * a.length)]
const rnd = (a: number, b: number) => a + Math.random() * (b - a)

const sparks = Array.from({ length: props.count }, (_, i) => {
  const size = pick(SIZE_BAG)
  const px = SPARKLE_PX[size] * BASE
  return {
    key: i,
    url: sparkleUrl(size, pick(COLORS)),
    px,
    left: rnd(2, 98),
    top: rnd(2, 96),
    dur: rnd(1.6, 3.2),
    delay: rnd(0, 2.2),
  }
})
</script>

<template>
  <div class="burst" aria-hidden="true">
    <img
      v-for="s in sparks"
      :key="s.key"
      :src="s.url"
      alt=""
      :width="s.px"
      :height="s.px"
      :style="{
        left: s.left + '%',
        top: s.top + '%',
        marginLeft: -s.px / 2 + 'px',
        marginTop: -s.px / 2 + 'px',
        '--dur': s.dur + 's',
        '--delay': s.delay + 's',
      }"
    />
  </div>
</template>

<style scoped>
.burst {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}
.burst img {
  position: absolute;
  image-rendering: pixelated;
  transform: scale(0);
  animation: twinkle var(--dur) step-end var(--delay) infinite both;
}
@keyframes twinkle {
  0% { transform: scale(0); }
  12% { transform: scale(0.25); }
  28% { transform: scale(0.5); }
  45% { transform: scale(1); }
  62% { transform: scale(0.5); }
  80% { transform: scale(0.25); }
  100% { transform: scale(0); }
}
/* Nobody needs a flashing screen to understand they were paid. */
@media (prefers-reduced-motion: reduce) {
  .burst img {
    animation: none;
    transform: scale(0.5);
    opacity: 0.7;
  }
}
</style>
