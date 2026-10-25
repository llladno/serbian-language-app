<script lang="ts">
// Shared by every button on the page: the one that is playing right now.
let stopCurrent: (() => void) | null = null
function stopOthers(self: () => void) {
  if (stopCurrent && stopCurrent !== self) stopCurrent()
  stopCurrent = self
}
</script>

<script setup lang="ts">
// Plays the Serbian pronunciation for a word (content/audio/<id>.mp3,
// served at /audio/). Renders nothing when the clip is missing.
//
// The same button instance is reused when the word under it changes (the
// dictionary drill, review cards), so the <audio> is bound to the clip it was
// made for: a new src drops the old one, and the next tap plays the new word.
// One clip at a time across all buttons; whatever gets cut off goes back to idle.
import { onBeforeUnmount, ref, watch } from 'vue'
import { Volume2 } from 'lucide-vue-next'

const props = withDefaults(defineProps<{ src?: string; size?: number }>(), { size: 30 })

const playing = ref(false)
let el: HTMLAudioElement | null = null
let elSrc: string | undefined

function drop() {
  const old = el
  el = null
  elSrc = undefined
  playing.value = false
  old?.pause()
}

function play(e: Event) {
  e.stopPropagation()
  if (!props.src) return
  if (!el || elSrc !== props.src) {
    drop()
    const a = new Audio(`/audio/${props.src}`)
    a.preload = 'auto'
    const idle = () => {
      if (el === a) playing.value = false
    }
    a.addEventListener('ended', idle)
    a.addEventListener('pause', idle)
    a.addEventListener('error', idle)
    el = a
    elSrc = props.src
  }
  const a = el
  stopOthers(drop)
  a.currentTime = 0
  playing.value = true
  a.play().catch(() => {
    if (el === a) playing.value = false
  })
}

watch(() => props.src, drop)
onBeforeUnmount(() => {
  drop()
  if (stopCurrent === drop) stopCurrent = null
})
</script>

<template>
  <button
    v-if="src"
    type="button"
    class="inline-grid shrink-0 place-items-center rounded-full text-[var(--muted)] transition hover:bg-[var(--accent-soft)] hover:text-[var(--accent)] active:scale-90"
    :class="{ 'bg-[var(--accent-soft)] text-[var(--accent)]': playing }"
    :style="{ width: size + 'px', height: size + 'px' }"
    :aria-label="playing ? 'играет' : 'озвучить'"
    :title="playing ? 'играет…' : 'озвучить'"
    @click="play"
  >
    <Volume2 :size="Math.round(size * 0.5)" :stroke-width="2.25" />
  </button>
</template>
