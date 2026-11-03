<script setup lang="ts">
// The gap of a sentence with a missing word: an underline that is the width of a short word. The right answer grows into
// it (green, a small overshoot) and the underline turns into a soft green plate. Used by the final «missing word» question
// and by the subtitle of a quick question, so both look the same.
import { useScript } from '../lib/useScript'

defineProps<{
  /** The right answer was given: write `word` into the gap. */
  filled?: boolean
  word?: string
}>()
const { show } = useScript()
</script>

<template>
  <span class="gap" :class="{ 'gap-filled': filled }">
    <span v-if="filled && word" class="gap-word">{{ show(word) }}</span>
  </span>
</template>

<style scoped>
.gap {
  display: inline-block;
  min-width: 3.2em;
  border-bottom: 3px solid var(--ring-track);
  border-radius: 0.2rem;
  text-align: center;
  vertical-align: baseline;
  line-height: 1.15;
  transition: border-color 0.25s ease, background-color 0.25s ease;
}
.gap-filled {
  min-width: 0;
  padding: 0 0.3em;
  border-color: var(--good);
  background: color-mix(in srgb, var(--good) 14%, transparent);
}
.gap-word {
  display: inline-block;
  color: var(--good);
  font-weight: 800;
  animation: gap-fill 0.5s cubic-bezier(0.2, 1.2, 0.3, 1) both;
}
@keyframes gap-fill {
  from {
    opacity: 0;
    transform: translateY(0.5em) scale(0.7);
  }
  60% {
    opacity: 1;
    transform: translateY(-0.08em) scale(1.12);
  }
}
@media (prefers-reduced-motion: reduce) {
  .gap-word {
    animation: none;
  }
}
</style>
