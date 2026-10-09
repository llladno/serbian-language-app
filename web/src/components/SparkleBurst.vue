<script setup lang="ts">
// A layer of pixel sparkles. Positions, sizes and timings are randomised once
// per mount, so two rewards never look identical.
//
// Sparkles never leave by shrinking to nothing: the last stretch of every
// animation is a fade of `opacity`, with the sprite held at a whole-pixel size.
// (A smooth scale would blur the pixels, and shrinking to zero reads as a
// cut-off rather than a departure.) Between appearing and fading the twinkle
// steps between whole-pixel sizes instead of interpolating.
//
//   twinkle  stay where they are and glitter, over and over (the default)
//   rise     drift upwards once and fade out
import { SPARKLE_PX, sparkleUrl, type SparkleColor, type SparkleSize } from '../lib/seeds'

const props = withDefaults(
  defineProps<{
    count?: number
    // Pins the sparkles to given places (percent of the layer) instead of
    // scattering them, for a layer that has something in it to keep clear of.
    spots?: { left: number; top: number }[]
    mode?: 'twinkle' | 'rise'
    // On-screen pixels per sprite pixel. Whole numbers keep the pixels square.
    base?: number
    // How far a rising sparkle travels, in px.
    lift?: number
    sizes?: SparkleSize[]
    // The pale and cream ones vanish on a light background, so a layer sitting
    // on one asks for gold only.
    colors?: SparkleColor[]
  }>(),
  { count: 18, mode: 'twinkle', base: 4, lift: 50 },
)

const COLORS: SparkleColor[] = props.colors ?? ['pale', 'gold', 'amber', 'cream']
// Small sparkles are the common case, so the big ones land as accents.
const SIZE_BAG: SparkleSize[] = props.sizes ?? ['s', 's', 'm', 'm', 'm', 'l', 'l', 'l', 'xl']

const pick = <T,>(a: T[]): T => a[Math.floor(Math.random() * a.length)]
const rnd = (a: number, b: number) => a + Math.random() * (b - a)
const rising = props.mode === 'rise'

// Unpinned sparkles fill a band across the whole width of the layer and as high
// as the picture in its middle, one to a column of it, rather than being dropped
// at random: a random scatter regularly piles most of them on one side. Columns
// alternate between the upper and the lower half of the band so neighbours do
// not sit level, and the order is shuffled so the twinkling does not sweep from
// one side to the other.
function bandSpots(n: number) {
  const spots = Array.from({ length: n }, (_, i) => ({
    // First column at the left edge, last at the right edge, the rest evenly
    // between. 8–92% across is as far out as a sprite can sit: it is centred on
    // its point and the biggest is 52px, so one nearer the edge would be sliced
    // by the layer's overflow.
    left: Math.min(92, Math.max(8, 8 + ((i + (Math.random() - 0.5) * 0.4) / Math.max(1, n - 1)) * 84)),
    top: 50 + (i % 2 ? 1 : -1) * rnd(0.25, 1) * 26,
  }))
  for (let i = spots.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[spots[i], spots[j]] = [spots[j], spots[i]]
  }
  return spots
}
const placed = props.spots ?? bandSpots(props.count)

const sparks = Array.from({ length: placed.length }, (_, i) => {
  const size = pick(SIZE_BAG)
  const px = SPARKLE_PX[size] * props.base
  return {
    key: i,
    url: sparkleUrl(size, pick(COLORS)),
    px,
    left: placed[i].left,
    top: placed[i].top,
    dur: rising ? rnd(1.9, 2.7) : rnd(1.6, 3.2),
    // A rising sparkle is a one-off, so they leave one after another.
    delay: rising ? i * 0.22 + rnd(0, 0.25) : rnd(0, 2.2),
  }
})
</script>

<template>
  <div class="burst" :style="{ '--base': base, '--lift': -lift + 'px' }" aria-hidden="true">
    <img
      v-for="s in sparks"
      :key="s.key"
      :class="mode"
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
  opacity: 0;
}

/* Whole-pixel sizes only: 1, 2 and `base` screen pixels per sprite pixel. The
   sizes are written as fractions of the full size so one set of keyframes
   serves any base (at base 2 the middle step is simply the full size). */
.burst img.twinkle {
  transform: scale(calc(1 / var(--base)));
  animation: twinkle var(--dur) step-end var(--delay) infinite both;
}
@keyframes twinkle {
  0% { opacity: 0; transform: scale(calc(1 / var(--base))); }
  12% { opacity: 1; transform: scale(calc(1 / var(--base))); }
  28% { transform: scale(calc(2 / var(--base))); }
  45% { transform: scale(1); }
  62% { transform: scale(calc(2 / var(--base))); }
  /* the way out: the size is held and the sparkle fades */
  75% { opacity: 1; transform: scale(calc(1 / var(--base))); animation-timing-function: linear; }
  100% { opacity: 0; transform: scale(calc(1 / var(--base))); }
}

.burst img.rise {
  animation: rise var(--dur) ease-out var(--delay) 1 both;
}
@keyframes rise {
  0% { opacity: 0; transform: translateY(0); }
  20% { opacity: 1; }
  100% { opacity: 0; transform: translateY(var(--lift)); }
}

/* Nobody needs a flashing screen to understand they were paid. */
@media (prefers-reduced-motion: reduce) {
  .burst img.twinkle {
    animation: none;
    transform: scale(calc(2 / var(--base)));
    opacity: 0.7;
  }
  .burst img.rise {
    display: none;
  }
}
</style>
