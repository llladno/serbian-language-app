<script setup lang="ts">
// The poster of the price change in a window, opened at the end of a lesson (lib/priceModal.ts, which says who sees
// it). The window brings the poster in piece by piece (PriceChangePoster `animate`) and cannot be closed for the first
// second: the cross arrives after it, and until then a tap beside the window or Escape does nothing.
import { onBeforeUnmount, onMounted, watch } from 'vue'
import { X } from 'lucide-vue-next'
import ModalShell from './ModalShell.vue'
import PriceChangePoster from './PriceChangePoster.vue'
import { usePriceModal } from '../lib/priceModal'
import { useTariffsStore } from '../stores/tariffs'
import '../styles/priceChange.css'

const m = usePriceModal()
const tariffs = useTariffsStore()

// The countdown ends while the window is open: the prices have changed and the poster has nothing true to say, so it goes.
watch(
  () => tariffs.campaign,
  (change) => {
    if (!change) m.endNow()
  },
)

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') m.close()
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <ModalShell :open="m.open.value" panel-class="pc-modal-panel" @close="m.close">
    <div data-test="price-modal">
      <PriceChangePoster animate to="/tariffs" @go="m.go" />
      <Transition name="x">
        <button
          v-if="m.closable.value"
          type="button"
          class="x"
          aria-label="Закрыть"
          title="Закрыть"
          data-test="price-modal-close"
          @click="m.close"
        >
          <X :size="18" :stroke-width="2.5" />
        </button>
      </Transition>
    </div>
  </ModalShell>
</template>

<style scoped>
.x {
  position: absolute;
  top: 0.65rem;
  right: 0.65rem;
  z-index: 5;
  display: grid;
  place-items: center;
  width: 2rem;
  height: 2rem;
  color: #fff;
  background: rgb(255 255 255 / 0.16);
  border-radius: 999px;
  -webkit-backdrop-filter: blur(6px);
  backdrop-filter: blur(6px);
}
.x:hover {
  background: rgb(255 255 255 / 0.28);
}
.x-enter-active {
  transition:
    opacity 0.35s ease,
    transform 0.35s cubic-bezier(0.2, 1.5, 0.4, 1);
}
.x-enter-from {
  opacity: 0;
  transform: scale(0.4) rotate(-90deg);
}
@media (prefers-reduced-motion: reduce) {
  .x-enter-active {
    transition: none;
  }
}
</style>
