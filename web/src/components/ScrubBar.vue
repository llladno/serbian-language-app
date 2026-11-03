<script setup lang="ts">
// A strip that can be pressed or dragged along: the progress bar of a story is one. It reports where the finger is, as
// 0..1 of its width, while it moves (`move`, to show the place) and once more when it lets go (`commit`, to go there).
// A plain tap is a press and a release in the same place, so it needs no code of its own. The strip itself is thin, so
// the area that answers a finger is taller than it looks (the padding in the template). The parent leaves a gap above it
// at least as tall as that padding, so the area never reaches the buttons over it: a press that starts on a button must
// stay a press of the button.
// No comment above the root element in the template: it would make two roots.
import { ref } from 'vue'

const props = defineProps<{ disabled?: boolean; label?: string }>()
const emit = defineEmits<{ move: [p: number]; commit: [p: number] }>()

const el = ref<HTMLElement | null>(null)
let active = false
let last = 0

function pos(e: PointerEvent): number {
  const r = el.value?.getBoundingClientRect()
  if (!r || r.width <= 0) return last
  return Math.min(1, Math.max(0, (e.clientX - r.left) / r.width))
}

function down(e: PointerEvent) {
  if (props.disabled) return
  active = true
  el.value?.setPointerCapture?.(e.pointerId)
  last = pos(e)
  emit('move', last)
}
function move(e: PointerEvent) {
  if (!active) return
  last = pos(e)
  emit('move', last)
}
function up(e: PointerEvent) {
  if (!active) return
  active = false
  el.value?.releasePointerCapture?.(e.pointerId)
  last = pos(e)
  emit('commit', last)
}
function cancel() {
  if (!active) return
  active = false
  emit('commit', last)
}
</script>

<template>
  <div
    ref="el"
    class="scrub py-3"
    :class="{ 'scrub-on': !disabled }"
    :aria-label="label"
    data-test="scrub"
    @pointerdown="down"
    @pointermove="move"
    @pointerup="up"
    @pointercancel="cancel"
  >
    <slot />
  </div>
</template>

<style scoped>
.scrub {
  touch-action: none;
  user-select: none;
}
.scrub-on {
  cursor: pointer;
}
</style>
