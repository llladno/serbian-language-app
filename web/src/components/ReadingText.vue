<script setup lang="ts">
// Short Serbian reading text with a collapsible Russian translation. Word
// lookup is handled by GlossedText.
import GlossedText from './GlossedText.vue'

defineProps<{ serbian: string; translation?: string }>()
</script>

<template>
  <div class="reading">
    <p class="reading-body whitespace-pre-wrap text-[1.05rem] leading-8">
      <GlossedText :text="serbian" />
    </p>

    <details v-if="translation" class="mt-4 text-sm text-[var(--muted)]">
      <summary class="cursor-pointer select-none hover:text-[var(--fg)]">Перевод</summary>
      <p class="mt-2 whitespace-pre-wrap leading-7">{{ translation }}</p>
    </details>
  </div>
</template>
