<script setup lang="ts">
// The tick shown next to «Верно»: a green disc pops in, the check mark draws
// itself, and a ring spreads outwards once. With reduced motion it just sits
// there, finished.
defineProps<{ size?: number }>()
</script>

<template>
  <svg class="success-check" :width="size ?? 28" :height="size ?? 28" viewBox="0 0 28 28" aria-hidden="true">
    <circle class="ring" cx="14" cy="14" r="12" />
    <circle class="disc" cx="14" cy="14" r="12" />
    <path class="tick" d="M8.5 14.5l3.8 3.8 7.2-8" />
  </svg>
</template>

<style scoped>
.success-check {
  display: inline-block;
  vertical-align: middle;
  overflow: visible;
}
.ring {
  fill: none;
  stroke: var(--good);
  stroke-width: 2;
  transform-origin: center;
  opacity: 0;
  animation: sc-ring 0.7s 0.15s ease-out forwards;
}
.disc {
  fill: var(--good);
  transform-origin: center;
  animation: sc-pop 0.45s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}
.tick {
  fill: none;
  stroke: #fff;
  stroke-width: 3;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-dasharray: 20;
  stroke-dashoffset: 20;
  animation: sc-draw 0.35s 0.25s ease-out forwards;
}
@keyframes sc-pop {
  from {
    transform: scale(0);
  }
  to {
    transform: scale(1);
  }
}
@keyframes sc-draw {
  to {
    stroke-dashoffset: 0;
  }
}
@keyframes sc-ring {
  from {
    opacity: 0.7;
    transform: scale(1);
  }
  to {
    opacity: 0;
    transform: scale(1.9);
  }
}
@media (prefers-reduced-motion: reduce) {
  .ring {
    display: none;
  }
  .disc,
  .tick {
    animation: none;
  }
  .tick {
    stroke-dashoffset: 0;
  }
}
</style>
