<script setup lang="ts">
// The quiet layer behind a price-change banner: a few stickers (the ruble, the lightning bolt, the cross, picked by the owner
// from the generated sheet, docs/price-campaign) rise from the bottom edge to the top and fade away one after another, each
// tilted its own way. On the small banners (the strip, the tariffs banner) the two prices as they are now rise among them,
// drawn as text stickers, dark letters with a white die-cut border and no card round them (`withPrices`); the big poster has no prices behind it, because it shows them in front.
//
// Put it first inside a box that is `position: relative; overflow: hidden` and lift the content above it (z-index 1). `rise` is
// how far a sticker travels (a negative length: about the height of the box), `startDelay` waits for an entrance to be over.
import { computed } from 'vue'
import { useTariffsStore } from '../stores/tariffs'
import { rub } from '../lib/tariffs'
import ruble from '../assets/promo/stickers/ruble.webp'
import bolt from '../assets/promo/stickers/bolt.webp'
import cross from '../assets/promo/stickers/cross.webp'

const props = withDefaults(
  defineProps<{ withPrices?: boolean; rise: string; startDelay?: string; size?: string; testId?: string }>(),
  { withPrices: false, startDelay: '0s', size: '1.7rem', testId: 'price-floaters' },
)
const tariffs = useTariffsStore()

type Item = { src?: string; text?: string; left: string; tilt: string }
const items = computed<Item[]>(() => {
  const showLevel = tariffs.choices.length > 0 || tariffs.hasCredit
  const list: Item[] = props.withPrices
    ? [
        { src: ruble, left: '4%', tilt: '-8deg' },
        ...(showLevel ? [{ text: rub(tariffs.levelNow), left: '26%', tilt: '5deg' }] : []),
        { src: bolt, left: '50%', tilt: '9deg' },
        { text: rub(tariffs.yearPrice), left: '66%', tilt: '-6deg' },
        { src: cross, left: '88%', tilt: '7deg' },
      ]
    : [
        { src: ruble, left: '9%', tilt: '-8deg' },
        { src: bolt, left: '45%', tilt: '9deg' },
        { src: cross, left: '83%', tilt: '-5deg' },
      ]
  return list
})
// One starts every 2 s with three, every 1.5 s with five; each lives a little over three starts.
const life = computed(() => (items.value.length <= 3 ? 6 : 7.5))
const gap = computed(() => life.value / items.value.length)
</script>

<template>
  <ul class="floaters" aria-hidden="true" :data-test="testId" :style="{ '--rise': rise, '--life': `${life}s`, '--size': size }">
    <li v-for="(f, i) in items" :key="i" :style="{ left: f.left, '--tilt': f.tilt, animationDelay: `calc(${startDelay} + ${i * gap}s)` }">
      <img v-if="f.src" :src="f.src" alt="" />
      <span v-else class="price">{{ f.text }}</span>
    </li>
  </ul>
</template>

<style scoped>
.floaters {
  position: absolute;
  inset: 0;
  z-index: 0;
  margin: 0;
  padding: 0;
  overflow: hidden;
  list-style: none;
  pointer-events: none;
}
.floaters li {
  position: absolute;
  bottom: 0;
  opacity: 0;
  animation: pc-floater var(--life) ease-in-out infinite;
}
.floaters img {
  display: block;
  width: var(--size);
  height: auto;
}
/* A price as a text sticker, with no card round it: dark letters and a thick white die-cut border that follows them (a ring of
   white shadows, so the corners are round, not spiked), and a small flat shadow under it, like the stickers on the sheet. */
.price {
  display: block;
  font-family: Oswald, 'Manrope', sans-serif;
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
  color: #0f3f3d;
  /* the border is measured in em, so it stays a thin ring round the letters and does not fill their counters */
  text-shadow:
    0.070em 0.000em 0 #fff,
    0.065em 0.027em 0 #fff,
    0.049em 0.049em 0 #fff,
    0.027em 0.065em 0 #fff,
    0.000em 0.070em 0 #fff,
    -0.027em 0.065em 0 #fff,
    -0.049em 0.049em 0 #fff,
    -0.065em 0.027em 0 #fff,
    -0.070em 0.000em 0 #fff,
    -0.065em -0.027em 0 #fff,
    -0.049em -0.049em 0 #fff,
    -0.027em -0.065em 0 #fff,
    -0.000em -0.070em 0 #fff,
    0.027em -0.065em 0 #fff,
    0.049em -0.049em 0 #fff,
    0.065em -0.027em 0 #fff,
    0.120em 0.000em 0 #fff,
    0.111em 0.046em 0 #fff,
    0.085em 0.085em 0 #fff,
    0.046em 0.111em 0 #fff,
    0.000em 0.120em 0 #fff,
    -0.046em 0.111em 0 #fff,
    -0.085em 0.085em 0 #fff,
    -0.111em 0.046em 0 #fff,
    -0.120em 0.000em 0 #fff,
    -0.111em -0.046em 0 #fff,
    -0.085em -0.085em 0 #fff,
    -0.046em -0.111em 0 #fff,
    -0.000em -0.120em 0 #fff,
    0.046em -0.111em 0 #fff,
    0.085em -0.085em 0 #fff,
    0.111em -0.046em 0 #fff,
    0 0.2em 0 rgb(0 0 0 / 0.35);
}
/* Rises from the bottom edge, shows quietly in the lower part of its path, and is gone before the top. */
@keyframes pc-floater {
  0% {
    opacity: 0;
    transform: translateY(0.7rem) rotate(var(--tilt));
  }
  18% {
    opacity: 0.55;
  }
  70% {
    opacity: 0.42;
  }
  100% {
    opacity: 0;
    transform: translateY(var(--rise)) rotate(var(--tilt));
  }
}
@media (prefers-reduced-motion: reduce) {
  .floaters {
    display: none;
  }
}
</style>
