<script setup lang="ts">
// "У вас есть оплаченный уровень": shown while a paid level has not been spent on a level yet
// (the learner closed the choice window, or paid while away and the notification came in without
// them). It sits at the top of the profile and of the tariffs screen and raises the window.
// The tone is that of the banner and the Telegram channel.
import { computed } from 'vue'
import { useTariffsStore } from '../stores/tariffs'
import { useLevelChoiceModal } from '../lib/levelChoiceModal'
import foreverIcon from '../assets/promo/forever.webp'

const tariffs = useTariffsStore()
const { openFor } = useLevelChoiceModal()

const show = computed(() => tariffs.ready && tariffs.hasCredit)
</script>

<template>
  <section v-if="show" class="notice card" data-test="level-credit">
    <span class="tile" aria-hidden="true"><img :src="foreverIcon" alt="" class="px" /></span>
    <div class="text">
      <p class="title">Оплаченный уровень ждёт вас</p>
      <p class="sub">Выберите, какой открыть: он останется с вами навсегда.</p>
    </div>
    <button class="btn btn-primary shrink-0" data-test="level-credit-choose" @click="openFor()">Выбрать уровень</button>
  </section>
</template>

<style scoped>
.notice {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  text-align: center;
  background: linear-gradient(135deg, var(--accent-soft), var(--card) 70%);
}
.tile {
  display: flex;
  height: 2.5rem;
  width: 2.5rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 0.75rem;
  background: #e4f5f0;
  border: 1px solid rgb(255 255 255 / 0.85);
  box-shadow: 0 1px 2px rgb(8 46 55 / 0.2);
}
.px {
  display: block;
  width: 1.75rem;
  height: auto;
  image-rendering: pixelated;
}
.text {
  min-width: 0;
  flex: 1;
}
.title {
  font-weight: 800;
}
.sub {
  font-size: 0.875rem;
  color: var(--muted);
}
@media (min-width: 640px) {
  .notice {
    flex-direction: row;
    text-align: left;
  }
}
</style>
