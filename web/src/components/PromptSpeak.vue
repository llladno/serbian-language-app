<script setup lang="ts">
// Speaker buttons for the Serbian «quoted» phrases inside an exercise prompt
// (one button per voiced phrase; Russian quotes and blanks have no clip).
import { computed } from 'vue'
import PhraseSpeak from './PhraseSpeak.vue'

const props = defineProps<{ prompt?: string }>()
const phrases = computed(() => [...(props.prompt ?? '').matchAll(/«([^»]+)»/g)].map((m) => m[1]))
</script>

<template>
  <PhraseSpeak v-for="(p, i) in phrases" :key="i" :text="p" />
</template>
