<script setup lang="ts">
// Plays the Serbian pronunciation for a word (content/audio/<id>.mp3,
// served at /audio/). Renders nothing when the clip is missing.
import { ref } from 'vue'

const props = withDefaults(defineProps<{ src?: string; size?: number }>(), { size: 30 })

const playing = ref(false)
let el: HTMLAudioElement | null = null

function play(e: Event) {
  e.stopPropagation()
  if (!props.src) return
  if (!el) {
    el = new Audio(`/audio/${props.src}`)
    el.addEventListener('ended', () => (playing.value = false))
    el.addEventListener('error', () => (playing.value = false))
  }
  el.currentTime = 0
  playing.value = true
  el.play().catch(() => (playing.value = false))
}
</script>

<template>
  <button
    v-if="src"
    type="button"
    class="inline-grid shrink-0 place-items-center rounded-full text-[var(--muted)] transition hover:bg-[var(--ring-track)] hover:text-[var(--accent)] active:scale-90"
    :class="{ 'text-[var(--accent)]': playing }"
    :style="{ width: size + 'px', height: size + 'px', fontSize: Math.round(size * 0.6) + 'px' }"
    :aria-label="playing ? 'играет' : 'озвучить'"
    :title="playing ? 'играет…' : 'озвучить'"
    @click="play"
  >
    {{ playing ? '🔊' : '🔈' }}
  </button>
</template>
