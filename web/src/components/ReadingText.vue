<script setup lang="ts">
// Short Serbian reading text with a collapsible Russian translation. Word
// lookup is handled by GlossedText.
import { computed } from 'vue'
import GlossedText from './GlossedText.vue'
import PhraseSpeak from './PhraseSpeak.vue'

const props = defineProps<{ serbian: string; translation?: string }>()

// One block per line so each line of a dialogue gets its own speaker button.
const lines = computed(() => props.serbian.split('\n').filter((l) => l.trim()))
</script>

<template>
  <div class="reading">
    <div class="reading-body text-[1.05rem] leading-8">
      <p v-for="(line, i) in lines" :key="i" class="flex items-start gap-1.5">
        <span class="min-w-0"><GlossedText :text="line" /></span>
        <PhraseSpeak :text="line" :size="26" class="mt-[3px]" />
      </p>
    </div>

    <details v-if="translation" class="mt-4 text-sm text-[var(--muted)]">
      <summary class="cursor-pointer select-none hover:text-[var(--fg)]">Перевод</summary>
      <p class="mt-2 whitespace-pre-wrap leading-7">{{ translation }}</p>
    </details>
  </div>
</template>
