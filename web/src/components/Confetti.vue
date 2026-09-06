<script setup lang="ts">
import { onMounted, ref } from 'vue'

const pieces = ref<{ x: number; delay: number; rot: number; color: string; dur: number }[]>([])
const COLORS = ['#d6457d', '#f472b6', '#f9a8d4', '#fbbf24', '#8b6ff0', '#34d399']

onMounted(() => {
  pieces.value = Array.from({ length: 90 }, () => ({
    x: Math.random() * 100,
    delay: Math.random() * 0.4,
    rot: Math.random() * 360,
    color: COLORS[Math.floor(Math.random() * COLORS.length)],
    dur: 1.6 + Math.random() * 1.4,
  }))
})
</script>

<template>
  <div class="pointer-events-none fixed inset-0 z-50 overflow-hidden">
    <span
      v-for="(p, i) in pieces"
      :key="i"
      class="confetti-piece"
      :style="{
        left: p.x + '%',
        background: p.color,
        animationDelay: p.delay + 's',
        animationDuration: p.dur + 's',
        transform: `rotate(${p.rot}deg)`,
      }"
    />
  </div>
</template>

<style scoped>
.confetti-piece {
  position: absolute;
  top: -12px;
  width: 8px;
  height: 12px;
  border-radius: 2px;
  animation: fall linear forwards;
}
@keyframes fall {
  to {
    transform: translateY(110vh) rotate(720deg);
    opacity: 0.9;
  }
}
</style>
