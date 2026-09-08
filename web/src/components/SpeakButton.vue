<script setup lang="ts">
// Plays the Serbian pronunciation for a word (content/audio/<id>.mp3,
// served at /audio/). Renders nothing when the clip is missing.
import { ref } from 'vue'
import { Volume2 } from 'lucide-vue-next'

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
    class="inline-grid shrink-0 place-items-center rounded-full border border-[var(--border)] text-[var(--muted)] transition hover:border-[var(--accent)] hover:bg-[var(--accent-soft)] hover:text-[var(--accent)] active:scale-90"
    :class="{ 'border-[var(--accent)] bg-[var(--accent-soft)] text-[var(--accent)]': playing }"
    :style="{ width: size + 'px', height: size + 'px' }"
    :aria-label="playing ? 'играет' : 'озвучить'"
    :title="playing ? 'играет…' : 'озвучить'"
    @click="play"
  >
    <Volume2 :size="Math.round(size * 0.5)" :stroke-width="2.25" />
  </button>
</template>
