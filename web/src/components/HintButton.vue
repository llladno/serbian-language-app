<script setup lang="ts">
// A "?" the learner taps before answering to reveal a nudge (the same text
// shown automatically after checking — see Exercise.explain / CheckResult.explain).
// Opt-in on purpose: unlike the answer itself, this text ships to the client
// upfront (see exerciseDTO.Explain), but nothing forces it on screen unless asked.
// Sized/styled like SpeakButton (inline next to a prompt), not the toolbar-
// sized shared .icon-btn.
import { ref } from 'vue'
import { CircleQuestionMark } from 'lucide-vue-next'

const props = withDefaults(defineProps<{ text: string; size?: number }>(), { size: 22 })
const shown = ref(false)
</script>

<template>
  <button
    type="button"
    class="inline-grid shrink-0 place-items-center rounded-full text-[var(--muted)] transition hover:bg-[var(--accent-soft)] hover:text-[var(--accent)] active:scale-90"
    :class="{ 'bg-[var(--accent-soft)] text-[var(--accent)]': shown }"
    :style="{ width: props.size + 'px', height: props.size + 'px' }"
    title="Подсказка"
    aria-label="Подсказка"
    @click="shown = !shown"
  >
    <CircleQuestionMark :size="Math.round(props.size * 0.7)" :stroke-width="2.25" />
  </button>
  <Transition name="fade">
    <!-- span, not p: callers embed this inside their own <p> prompts, and a
         block-level child would silently close that outer <p> in the DOM. -->
    <span v-if="shown" class="mt-1.5 block w-full text-sm text-[var(--muted)]">{{ text }}</span>
  </Transition>
</template>
