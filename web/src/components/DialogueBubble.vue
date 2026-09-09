<script setup lang="ts">
// One line of a dialogue. The Serbian text is word-clickable (GlossedText);
// the whole-line translation lives behind its own button, so a tap for the
// line never competes with a tap for a single word.
import { computed, ref } from 'vue'
import { Languages } from 'lucide-vue-next'
import GlossedText from './GlossedText.vue'
import SpeakButton from './SpeakButton.vue'

const props = defineProps<{
  who: 'npc' | 'me'
  sr: string
  ru?: string
  audio?: string
  showTranslation?: boolean
  // a "me" line the learner got wrong: the correct line still stands in the
  // chat, but it has to be visibly marked as a miss, with the explanation.
  wrong?: boolean
  note?: string
}>()

const open = ref(false)
const translated = computed(() => props.showTranslation || open.value)
</script>

<template>
  <div class="flex" :class="who === 'me' ? 'justify-end' : 'justify-start'">
    <div
      class="max-w-[85%] rounded-2xl px-3.5 py-2.5"
      :class="[
        who === 'me' ? 'rounded-br-sm bg-[var(--accent-soft)]' : 'rounded-bl-sm bg-[var(--bg-soft)]',
        wrong ? 'ring-1 ring-[var(--bad)]' : '',
      ]"
    >
      <div class="flex items-start gap-2">
        <p class="serbian leading-7"><GlossedText :text="sr" /></p>
        <SpeakButton v-if="audio" :src="audio" :size="26" class="mt-0.5" />
        <button
          v-if="ru"
          type="button"
          data-test="translate"
          class="mt-0.5 inline-grid h-[26px] w-[26px] shrink-0 place-items-center rounded-full border border-[var(--border)] text-[var(--muted)] transition hover:border-[var(--accent)] hover:text-[var(--accent)]"
          :class="{ 'border-[var(--accent)] text-[var(--accent)]': translated }"
          :aria-label="translated ? 'скрыть перевод' : 'перевести реплику'"
          @click="open = !open"
        >
          <Languages :size="13" :stroke-width="2.25" />
        </button>
      </div>

      <p v-if="translated && ru" class="mt-1.5 text-sm text-[var(--muted)]">{{ ru }}</p>

      <p v-if="wrong" class="mt-1.5 text-sm font-semibold text-[var(--bad)]">✗ ты ответил иначе</p>
      <p v-if="wrong && note" class="mt-0.5 text-sm text-[var(--muted)]">{{ note }}</p>
    </div>
  </div>
</template>
