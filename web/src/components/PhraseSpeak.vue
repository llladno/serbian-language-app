<script setup lang="ts">
// Speaker button for a Serbian phrase; renders nothing when no clip exists
// (non-Serbian strings, phrases not yet voiced).
import { computed } from 'vue'
import { loadPhraseIndex, phraseClip } from '../lib/phraseAudio'
import SpeakButton from './SpeakButton.vue'

const props = withDefaults(defineProps<{ text?: string; size?: number }>(), { size: 26 })
void loadPhraseIndex()
const src = computed(() => phraseClip(props.text))
</script>

<template>
  <SpeakButton v-if="src" :src="src" :size="size" />
</template>
