<script setup lang="ts">
// The frame every modal in the app sits in: a dimmed backdrop, one panel in the
// middle of the screen — centred both ways on every screen size — and the
// open/close transition. A modal brings only its content, so a new one cannot
// end up pinned to the top of the phone like the first four did.
//
// Centring uses a scrolling layer with a flex box of at least the viewport's
// height inside it. A panel shorter than the screen is centred; one taller than
// the screen starts at the top and scrolls — plain `items-center` on the
// scrolling layer itself would cut the top of a tall panel off, out of reach.
defineProps<{
  open: boolean
  size?: 'sm' | 'md'
  panelClass?: string
}>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-overlay">
      <div v-if="open" class="fixed inset-0 z-50 overflow-y-auto">
        <div class="fixed inset-0 bg-black/40" aria-hidden="true" />
        <div class="relative flex min-h-full items-center justify-center p-4" @click.self="emit('close')">
          <div class="relative w-full" :class="size === 'sm' ? 'max-w-sm' : 'max-w-md'">
            <div class="card modal-panel relative" :class="panelClass">
              <slot />
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-overlay-enter-active,
.modal-overlay-leave-active {
  transition: opacity 0.18s ease;
}
.modal-overlay-enter-from,
.modal-overlay-leave-to {
  opacity: 0;
}
.modal-overlay-enter-active .modal-panel,
.modal-overlay-leave-active .modal-panel {
  transition: transform 0.18s ease, opacity 0.18s ease;
}
.modal-overlay-enter-from .modal-panel,
.modal-overlay-leave-to .modal-panel {
  opacity: 0;
  transform: scale(0.95) translateY(6px);
}
</style>
