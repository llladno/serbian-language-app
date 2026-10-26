<script setup lang="ts">
// Shown once, when the learner comes back from the bank with a paid order that needs no choice (a
// year; a level is spent in LevelChoiceModal): a thank-you, what the payment opened, and one way
// forward, the course. It sits in the app's usual modal frame; the two swallows are the same picture
// the invite screen and the quests banner use for "a gift", so it reads as one family. The tone is
// that of the tariffs screen and the Telegram channel: «вы», and the pixel heart.
import { computed } from 'vue'
import { X, LockOpen } from 'lucide-vue-next'
import ModalShell from './ModalShell.vue'
import SeedIcon from './SeedIcon.vue'
import { useWalletStore } from '../stores/wallet'
import { grouped, longDate } from '../lib/tariffs'
import type { Order } from '../types'
import scene from '../assets/invite-scene.webp'
import heart from '../assets/donate/heart.png'

const props = defineProps<{ order: Order | null; yearUntil: string | null }>()
const emit = defineEmits<{ close: [] }>()

const wallet = useWalletStore()

// "1 000 зёрнышек", in the currency's own name and plural.
const bonus = computed(() => {
  const n = props.order?.bonus ?? 0
  return n > 0 ? `${grouped(n)} ${wallet.currencyWord(n)}`.trim() : ''
})

// What the payment opened, in words.
const opened = computed(() => {
  const o = props.order
  if (!o) return ''
  if (o.tariff === 'year') {
    return props.yearUntil ? `Открыты все уровни до ${longDate(props.yearUntil)}` : 'Открыты все уровни на год'
  }
  return o.phase_title ? `Открыт «${o.phase_title}»` : 'Уровень открыт'
})
</script>

<template>
  <ModalShell :open="order !== null" size="sm" panel-class="overflow-hidden text-center" @close="emit('close')">
    <div v-if="order" role="dialog" aria-modal="true" aria-labelledby="thanks-title" data-test="thanks-modal">
      <button class="icon-btn absolute right-3 top-3 z-10" title="Закрыть" aria-label="Закрыть" data-test="thanks-close" @click="emit('close')">
        <X :size="19" :stroke-width="2.25" />
      </button>

      <div class="art" aria-hidden="true"><img :src="scene" alt="" /></div>

      <div class="space-y-4 px-5 pb-5 pt-1">
        <h2 id="thanks-title" class="text-xl font-extrabold leading-snug">
          Спасибо, что вы с нами и поддерживаете проект
          <!-- The pixel heart of the support card after the thanks; decorative, the words carry the meaning. -->
          <img :src="heart" alt="" aria-hidden="true" class="heart" data-test="thanks-heart" />
        </h2>
        <p class="text-sm text-[var(--muted)]">Оплата прошла. Вот что теперь у вас есть.</p>

        <ul class="got">
          <li data-test="thanks-opened">
            <span class="ico"><LockOpen :size="16" :stroke-width="2.5" /></span>
            <span>{{ opened }}</span>
          </li>
          <li v-if="bonus" data-test="thanks-bonus">
            <span class="ico ico-seed"><SeedIcon :size="15" /></span>
            <span>
              <b>+{{ bonus }}</b> на счёте
            </span>
          </li>
        </ul>

        <RouterLink to="/course" class="btn btn-primary w-full" data-test="thanks-course" @click="emit('close')">
          Перейти к курсу
        </RouterLink>
      </div>
    </div>
  </ModalShell>
</template>

<style scoped>
.art {
  height: 8.5rem;
  overflow: hidden;
  background: linear-gradient(135deg, var(--accent-soft), var(--card));
}
.art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 50% 30%;
  image-rendering: pixelated;
  /* The picture melts into the card instead of ending in an edge. */
  mask-image: linear-gradient(to bottom, #000 60%, transparent);
  -webkit-mask-image: linear-gradient(to bottom, #000 60%, transparent);
}
/* The pixel heart sits on the text line like a letter would. */
.heart {
  display: inline-block;
  height: 0.95em;
  width: auto;
  vertical-align: -0.14em;
  image-rendering: pixelated;
}
.got {
  display: grid;
  gap: 0.6rem;
  text-align: left;
  border-radius: 0.9rem;
  background: var(--bg-soft);
  padding: 0.85rem 0.95rem;
  font-size: 0.9rem;
}
.got li {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}
.ico {
  display: grid;
  flex: none;
  place-items: center;
  width: 1.8rem;
  height: 1.8rem;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
}
/* Gold is the currency's colour everywhere in the app, so the bonus wears it. */
.ico-seed {
  background: color-mix(in srgb, var(--gold) 22%, var(--card));
}
</style>
