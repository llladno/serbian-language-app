<script setup lang="ts">
// The quests screen is where a learner stares at "not enough зёрнышки", so it is
// also where the other way in is offered: pay and open the level now. Hidden
// when there is nothing left to open, or a year already covers everything.
//
// Gold stays the currency's colour (see TariffsView): the banner is accent
// teal, and the swallow handing over a seed is the same picture the invite
// screen uses for "a gift", so it reads as one family.
import { computed } from 'vue'
import { useWalletStore } from '../stores/wallet'
import { useTariffsStore } from '../stores/tariffs'
import { rub } from '../lib/tariffs'
import scene from '../assets/invite-scene.webp'

const wallet = useWalletStore()
const tariffs = useTariffsStore()

const show = computed(() => !!wallet.goal && !tariffs.yearActive)
</script>

<template>
  <section v-if="show && wallet.goal" class="banner card" data-test="tariffs-banner">
    <div class="art" aria-hidden="true"><img :src="scene" alt="" /></div>
    <div class="body">
      <p class="eyebrow">Не хочется копить?</p>
      <p class="title">Открой «{{ wallet.goal.title }}» сразу</p>
      <p class="sub">Один платёж — и уровень твой навсегда. Подписки нет.</p>
      <div class="actions">
        <RouterLink to="/tariffs" class="btn btn-primary" data-test="banner-level">
          Открыть за {{ rub(tariffs.levelNow) }}
        </RouterLink>
        <RouterLink to="/tariffs" class="all" data-test="banner-year">
          Все уровни на год — {{ rub(tariffs.prices.year.price) }}
        </RouterLink>
      </div>
    </div>
  </section>
</template>

<style scoped>
.banner {
  position: relative;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: linear-gradient(135deg, var(--accent-soft), var(--card) 70%);
}
.art {
  height: 7.5rem;
  overflow: hidden;
}
.art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 50% 30%;
  image-rendering: pixelated;
  /* The picture melts into the card instead of ending in an edge. */
  mask-image: linear-gradient(to bottom, #000 55%, transparent);
  -webkit-mask-image: linear-gradient(to bottom, #000 55%, transparent);
}
.body {
  padding: 0.25rem 1.25rem 1.25rem;
}
.eyebrow {
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--accent);
}
.title {
  margin-top: 0.25rem;
  font-size: 1.125rem;
  font-weight: 800;
  line-height: 1.3;
}
.sub {
  margin-top: 0.25rem;
  font-size: 0.875rem;
  color: var(--muted);
}
.actions {
  margin-top: 1rem;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem 1rem;
}
.all {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--accent);
  text-underline-offset: 3px;
}
.all:hover {
  text-decoration: underline;
}

@media (min-width: 640px) {
  .banner {
    flex-direction: row-reverse;
    align-items: stretch;
  }
  .art {
    width: 44%;
    height: auto;
    flex: none;
  }
  .art img {
    mask-image: linear-gradient(to left, #000 50%, transparent);
    -webkit-mask-image: linear-gradient(to left, #000 50%, transparent);
  }
  .body {
    flex: 1;
    padding: 1.5rem 0.5rem 1.5rem 1.5rem;
    align-self: center;
  }
}
</style>
