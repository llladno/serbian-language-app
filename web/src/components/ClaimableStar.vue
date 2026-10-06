<script setup lang="ts">
// The "something is waiting for you" mark on the balance chip: one pixel star
// that pops in and then keeps winking. Decorative — the chip it sits on is
// already a link to the quests, and the screen reader gets the label there.
import { sparkleUrl } from '../lib/feathers'
</script>

<template>
  <span class="pop" aria-hidden="true">
    <img :src="sparkleUrl('l', 'gold')" alt="" class="star" />
  </span>
</template>

<style scoped>
.pop {
  display: block;
  animation: pop 0.4s steps(3) both;
}
.star {
  display: block;
  width: 18px;
  height: 18px;
  image-rendering: pixelated;
  animation: wink 2.4s step-end 0.4s infinite;
}
/* Stepped, and only on whole-pixel multiples, so the star never blurs. */
@keyframes pop {
  0% { transform: scale(0); }
  50% { transform: scale(0.5); }
  75% { transform: scale(1.5); }
  100% { transform: scale(1); }
}
@keyframes wink {
  0% { transform: scale(1); }
  8% { transform: scale(0.5); }
  16% { transform: scale(1); }
  100% { transform: scale(1); }
}
@media (prefers-reduced-motion: reduce) {
  .pop,
  .star {
    animation: none;
  }
}
</style>
