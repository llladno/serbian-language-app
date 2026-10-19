<script setup lang="ts">
// Latin / Cyrillic, one control for the whole app. The choice is an account
// setting (stores/session.setScript), so it is the same in the browser and in the
// Telegram Mini App. `compact` is the short «Лат | Кир» pill for screen headers;
// the full-word form is for the settings.
//
// Two equal halves; one highlight (the thumb) slides between them and the buttons only
// carry the labels, so switching script moves the highlight instead of re-colouring.
// No comment above the root element in the template: it would make the component
// render two roots, and the classes a parent passes (ml-auto ...) would be dropped.
import { computed } from 'vue'
import { useSessionStore } from '../stores/session'

defineProps<{ compact?: boolean }>()
const session = useSessionStore()
// The account's choice, not the effective script: inside a lesson about the
// letters the text stays Latin, but the toggle still shows what the learner chose.
const chosen = computed(() => session.user?.script ?? 'latin')

const OPTIONS = [
  { value: 'latin' as const, short: 'Лат', full: 'Латиница' },
  { value: 'cyrillic' as const, short: 'Кир', full: 'Кириллица' },
]

async function pick(value: 'latin' | 'cyrillic') {
  if (value === chosen.value) return
  try {
    await session.setScript(value)
  } catch {
    /* the store has already put the old choice back */
  }
}
</script>

<template>
  <div
    class="relative shrink-0 grid-cols-2 rounded-full bg-[var(--bg-soft)] p-0.5"
    :class="compact ? 'inline-grid' : 'grid'"
    role="group"
    aria-label="Письмо: латиница или кириллица"
  >
    <span
      class="pointer-events-none absolute inset-y-0.5 left-0.5 w-[calc(50%-0.125rem)] rounded-full bg-[var(--accent)] transition-transform duration-200 ease-out motion-reduce:transition-none"
      :class="{ 'translate-x-full': chosen === 'cyrillic' }"
      data-test="script-thumb"
      aria-hidden="true"
    />
    <button
      v-for="o in OPTIONS"
      :key="o.value"
      type="button"
      class="relative rounded-full text-center font-semibold transition-colors duration-200"
      :class="[
        compact ? 'px-2.5 py-1 text-xs' : 'px-4 py-2 text-sm',
        chosen === o.value ? 'text-white' : 'text-[var(--muted)] hover:text-[var(--fg)]',
      ]"
      :aria-pressed="chosen === o.value"
      :data-test="'script-' + o.value"
      @click="pick(o.value)"
    >
      {{ compact ? o.short : o.full }}
    </button>
  </div>
</template>
