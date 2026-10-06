<script setup lang="ts">
// Shown right after a quest pays out. Mounted once, in AppNav, and opened
// through lib/rewardModal.ts from wherever the payment happened.
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { X } from 'lucide-vue-next'
import { useRewardModal } from '../lib/rewardModal'
import { useWalletStore } from '../stores/wallet'
import { pileFor } from '../lib/feathers'
import SparkleBurst from './SparkleBurst.vue'

const { open, reward, title, questsLink, closeModal } = useRewardModal()
const wallet = useWalletStore()

// The heap grows with what the learner now has, so the picture itself is a
// progress bar: one feather, a few, a handful, a mountain.
const pile = computed(() => pileFor(wallet.balance))
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-overlay">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto p-4 sm:items-center"
      >
        <div class="fixed inset-0 bg-black/40" @click="closeModal" />
        <div class="card modal-panel relative z-10 w-full max-w-sm p-5 text-center">
          <button class="icon-btn absolute right-3 top-3" title="Закрыть" aria-label="Закрыть" @click="closeModal">
            <X :size="19" :stroke-width="2.25" />
          </button>

          <div class="relative -mx-2 h-40">
            <SparkleBurst :count="20" />
            <img :src="pile" alt="" aria-hidden="true" class="pile" />
          </div>

          <p class="text-sm font-semibold text-[var(--muted)]">Задание выполнено</p>
          <p class="mt-0.5 text-lg font-extrabold" data-test="reward-quest">{{ title }}</p>
          <p class="mt-3 text-2xl font-extrabold text-[var(--accent)]" data-test="reward-amount">
            +{{ wallet.amount(reward) }}
          </p>
          <p class="mt-1 text-sm text-[var(--muted)]">Всего: {{ wallet.amount(wallet.balance) }}</p>

          <RouterLink
            v-if="questsLink"
            to="/quests"
            class="btn btn-primary mt-4 w-full"
            data-test="reward-to-quests"
            @click="closeModal"
          >
            К заданиям
          </RouterLink>
          <button
            class="btn mt-2 w-full"
            :class="questsLink ? 'btn-ghost' : 'btn-primary mt-4'"
            data-test="reward-ok"
            @click="closeModal"
          >
            Отлично
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.pile {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 8rem;
  image-rendering: pixelated;
}
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
