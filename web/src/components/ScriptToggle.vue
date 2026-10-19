<script setup lang="ts">
// Latin / Cyrillic, one control for the whole app. The choice is an account
// setting (stores/session.setScript), so it is the same in the browser and in the
// Telegram Mini App. `compact` is the short «Лат | Ћир» pill for screen headers;
// the full-word form is for the settings.
import { computed } from 'vue'
import { useSessionStore } from '../stores/session'

defineProps<{ compact?: boolean }>()
const session = useSessionStore()
// The account's choice, not the effective script: inside a lesson about the
// letters the text stays Latin, but the toggle still shows what the learner chose.
const chosen = computed(() => session.user?.script ?? 'latin')

const OPTIONS = [
  { value: 'latin' as const, short: 'Лат', full: 'Латиница' },
  { value: 'cyrillic' as const, short: 'Ћир', full: 'Кириллица' },
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
    class="inline-flex shrink-0 gap-0.5 rounded-full bg-[var(--bg-soft)] p-0.5"
    role="group"
    aria-label="Письмо: латиница или кириллица"
  >
    <button
      v-for="o in OPTIONS"
      :key="o.value"
      type="button"
      class="rounded-full font-semibold transition"
      :class="[
        compact ? 'px-2.5 py-1 text-xs' : 'flex-1 px-4 py-2 text-sm',
        chosen === o.value ? 'bg-[var(--accent)] text-white' : 'text-[var(--muted)] hover:text-[var(--fg)]',
      ]"
      :aria-pressed="chosen === o.value"
      :data-test="'script-' + o.value"
      @click="pick(o.value)"
    >
      {{ compact ? o.short : o.full }}
    </button>
  </div>
</template>
