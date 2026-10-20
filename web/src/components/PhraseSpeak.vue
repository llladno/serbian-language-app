<script setup lang="ts">
// Speaker button for a Serbian phrase; renders nothing when no clip exists
// (non-Serbian strings, phrases not yet voiced).
//
// `autoplay` is for the right sentence shown after an answer: it is read out as
// the verdict appears (and only if the learner has not switched the sound off,
// see lib/sound.ts), and the button stays to hear it again. Only at that moment:
// a lesson keeps every answered exercise of a step on the page, so a clip list
// arriving late must not read them all out, and a changed text does not speak.
import { computed, onMounted } from 'vue'
import { loadPhraseIndex, phraseClip } from '../lib/phraseAudio'
import { speakClip } from '../lib/sound'
import SpeakButton from './SpeakButton.vue'

const props = withDefaults(defineProps<{ text?: string; size?: number; autoplay?: boolean }>(), {
  size: 26,
  autoplay: false,
})
void loadPhraseIndex()
const src = computed(() => phraseClip(props.text))

onMounted(() => {
  if (props.autoplay && src.value) speakClip(src.value)
})
</script>

<template>
  <SpeakButton v-if="src" :src="src" :size="size" />
</template>
