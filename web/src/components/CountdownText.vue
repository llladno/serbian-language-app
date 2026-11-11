<script setup lang="ts">
// A countdown with seconds, "5 дн 12:34:07". The seconds are what lets the learner see time running out, so each
// tick gets a small movement (the digits drop in and light up): the element is replaced every second, which restarts the
// animation. Reduced motion: plain digits.
import { computed } from 'vue'
import { countdownParts } from '../lib/priceChange'

const props = defineProps<{ ms: number }>()
const parts = computed(() => countdownParts(props.ms))
</script>

<template>
  <span class="cd">{{ parts.head }}:<span :key="parts.sec" class="sec">{{ parts.sec }}</span></span>
</template>

<style scoped>
.cd {
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.sec {
  display: inline-block;
  animation: pc-tick 0.5s ease-out;
}
@keyframes pc-tick {
  from {
    opacity: 0.3;
    transform: translateY(-5px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}
@media (prefers-reduced-motion: reduce) {
  .sec {
    animation: none;
  }
}
</style>
