<script setup lang="ts">
// Shown right after a quest pays out. Mounted once, in AppNav, and opened
// through lib/rewardModal.ts from wherever the payment happened.
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { X } from 'lucide-vue-next'
import { useRewardModal } from '../lib/rewardModal'
import { useWalletStore } from '../stores/wallet'
import { pileFor } from '../lib/seeds'
import SparkleBurst from './SparkleBurst.vue'
import ModalShell from './ModalShell.vue'

const { open, reward, title, questsLink, closeModal } = useRewardModal()
const wallet = useWalletStore()

// The heap grows with what the learner now has, so the picture itself is a
// progress bar: one seed, a few, a heap, a sack.
const pile = computed(() => pileFor(wallet.balance))
</script>

<template>
  <ModalShell :open="open" size="sm" panel-class="p-5 text-center" @close="closeModal">
    <button class="icon-btn absolute right-3 top-3" title="Закрыть" aria-label="Закрыть" @click="closeModal">
      <X :size="19" :stroke-width="2.25" />
    </button>

    <div class="relative -mx-2 h-40">
      <!-- A soft gold glow right behind the seeds: the money's own colour,
           so the moment you are paid is the one moment the picture lights up. -->
      <div class="glow" aria-hidden="true" />
      <SparkleBurst :count="9" />
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
  </ModalShell>
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

/* Centred on the heap of seeds and a little wider than it, breathing
   slowly. It is the first thing in the box, so the sparkles and the seeds
   are drawn over it. */
.glow {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 13rem;
  aspect-ratio: 1;
  transform: translate(-50%, -50%);
  border-radius: 50%;
  background: radial-gradient(
    closest-side,
    color-mix(in srgb, var(--seed-hi) 85%, transparent),
    color-mix(in srgb, var(--seed) 45%, transparent) 55%,
    transparent 100%
  );
  filter: blur(10px);
  pointer-events: none;
  animation: breathe 3.6s ease-in-out infinite;
}
@keyframes breathe {
  0%,
  100% {
    opacity: 0.7;
    transform: translate(-50%, -50%) scale(0.96);
  }
  50% {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1.04);
  }
}
@media (prefers-reduced-motion: reduce) {
  .glow {
    animation: none;
    opacity: 0.85;
  }
}
</style>
