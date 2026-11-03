<script setup lang="ts">
// «Верно», big and in the middle, in place of the buttons that were answered: the tick draws itself (SuccessCheck),
// the word rises under it, and the right answer stays a moment as a reminder. Held still with reduced motion.
import { useScript } from '../lib/useScript'
import SuccessCheck from './SuccessCheck.vue'

defineProps<{ answer?: string; serbian?: boolean }>()
const { show } = useScript()
</script>

<template>
  <div class="verdict" data-test="story-verdict">
    <SuccessCheck :size="84" />
    <p class="word text-2xl font-extrabold text-[var(--good)]">Верно</p>
    <p v-if="answer" class="hint text-lg" :class="{ serbian }">{{ serbian ? show(answer) : answer }}</p>
  </div>
</template>

<style scoped>
.verdict {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  text-align: center;
  animation: verdict-in 0.35s cubic-bezier(0.2, 0.8, 0.2, 1) both;
}
.word {
  animation: verdict-rise 0.45s 0.15s cubic-bezier(0.2, 0.8, 0.2, 1) both;
}
.hint {
  color: var(--muted);
  animation: verdict-rise 0.45s 0.3s cubic-bezier(0.2, 0.8, 0.2, 1) both;
}
@keyframes verdict-in {
  from {
    opacity: 0;
    transform: scale(0.8);
  }
}
@keyframes verdict-rise {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
}
@media (prefers-reduced-motion: reduce) {
  .verdict,
  .word,
  .hint {
    animation: none;
  }
}
</style>
