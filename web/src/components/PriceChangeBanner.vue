<script setup lang="ts">
// The banner at the top of the tariffs screen while a price change is announced (migration 041): the small one. It takes
// the place of the swallow's hello and is as tall as it, because the screen is where the learner chooses and the offer
// must not push the cards out of sight. The big poster is for the window after a lesson (PriceChangePoster).
//
// A black card: the sticker with the discount and the countdown (with seconds) on one line, the headline, one short
// line, the two prices set against what follows as stickers, a small swallow and the gold button, which takes the
// learner down to the cards. The words are `campaignCopy` (lib/priceChange.ts). Wide: the words on the left, the
// prices on the right.
import { computed, watch } from 'vue'
import { useTariffsStore } from '../stores/tariffs'
import { rub } from '../lib/tariffs'
import { trackTariff } from '../lib/tariffEvents'
import { LEVEL_LABEL, YEAR_LABEL, usePriceChange } from '../lib/priceChange'
import CountdownText from './CountdownText.vue'
import PriceFloaters from './PriceFloaters.vue'
import bird from '../assets/support-mascot.webp'
import '../styles/priceChange.css'

const tariffs = useTariffsStore()
const { change, remaining, copy } = usePriceChange()

// `change` is already empty for a learner who has a tariff (the owner's own account excepted).
const show = computed(() => tariffs.ready && !!change.value)
const showLevel = computed(() => tariffs.choices.length > 0 || tariffs.hasCredit)

// The funnel (source `price_page`): a showing each time the banner appears. The banner has no button: the cards are right below it.
watch(show, (shown) => shown && trackTariff('view', 'price_page'), { immediate: true })
</script>

<template>
  <div v-if="show && change && copy" class="wrap" data-test="price-banner">
    <header class="banner">
      <PriceFloaters with-prices rise="-15rem" test-id="price-banner-floaters" />
      <div class="row">
        <span class="pc-sticker label" data-test="price-banner-sticker">{{ copy.sticker }}</span>
        <p class="tmr" role="timer" aria-live="off" data-test="price-banner-timer"><CountdownText :ms="remaining" /></p>
      </div>
      <h1 class="title" data-test="price-banner-title">{{ copy.title }}</h1>
      <p class="sub" data-test="price-banner-sub">{{ copy.short }}</p>
      <div class="prices">
        <p v-if="showLevel" class="st st-level" data-test="price-banner-level">
          <small>{{ LEVEL_LABEL.toUpperCase() }}</small
          ><span class="a">{{ rub(tariffs.levelNow) }}</span><span class="arr">→</span
          ><span class="b">{{ rub(change.levelPrice) }}</span>
        </p>
        <p class="st st-year" data-test="price-banner-year">
          <small>{{ YEAR_LABEL.toUpperCase() }}</small
          ><span class="a">{{ rub(tariffs.yearPrice) }}</span><span class="arr">→</span
          ><span class="b">{{ rub(change.yearPrice) }}</span>
        </p>
      </div>
      <img :src="bird" alt="" aria-hidden="true" class="bird" />
    </header>
  </div>
</template>

<style scoped>
.wrap {
  container-type: inline-size;
}
.banner {
  position: relative;
  display: grid;
  gap: 0.5rem;
  padding: 1rem 1.1rem 3.4rem;
  overflow: hidden;
  color: #fff;
  background: #0b1211;
  border-radius: 20px;
  box-shadow: 0 0 0 1px rgb(255 255 255 / 0.08);
  grid-template-columns: 1fr;
  grid-template-areas:
    'row'
    'title'
    'sub'
    'prices';
}
.banner > :not(.floaters):not(.bird) {
  position: relative;
  z-index: 1;
}
.row {
  grid-area: row;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}
.label {
  font-size: 0.95rem;
}
.tmr {
  padding: 5px 9px 6px;
  font-family: Oswald, 'Manrope', sans-serif;
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1;
  background: #111;
  border: 2px solid #fff;
  border-radius: 12px;
  transform: rotate(3deg);
}
.title {
  grid-area: title;
  margin-top: 0.15rem;
  font-size: 1.4rem;
  font-weight: 800;
  line-height: 1.15;
}
.sub {
  grid-area: sub;
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.3;
  color: #9bb1ae;
}
.prices {
  grid-area: prices;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 0.6rem;
  margin-top: 0.1rem;
}
.st {
  padding: 6px 10px 8px;
  font-family: Oswald, 'Manrope', sans-serif;
  font-weight: 700;
  line-height: 1;
  white-space: nowrap;
  color: #111;
  background: #fff;
  border-radius: 11px;
  box-shadow: 0 3px 0 rgb(255 255 255 / 0.18);
}
.st small {
  display: block;
  margin-bottom: 2px;
  font-family: Manrope, sans-serif;
  font-size: 0.625rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  opacity: 0.6;
}
.st .a,
.st .b {
  font-size: 1.3rem;
}
.st .arr {
  margin: 0 2px;
  font-size: 1rem;
  opacity: 0.5;
}
.st .b {
  color: #9aa3a2;
}
.st-level {
  transform: rotate(-2deg);
}
.st-year {
  transform: rotate(1.5deg);
}
.bird {
  position: absolute;
  right: 1rem;
  bottom: 0.5rem;
  width: 3.5rem;
  transform: rotate(6deg);
  image-rendering: pixelated;
  filter: drop-shadow(1.5px 0 0 #fff) drop-shadow(-1.5px 0 0 #fff) drop-shadow(0 1.5px 0 #fff) drop-shadow(0 -1.5px 0 #fff)
    drop-shadow(0 3px 5px rgb(0 0 0 / 0.35));
}

/* Wide: the words on the left, the two prices stacked on the right. */
@container (min-width: 640px) {
  .banner {
    grid-template-columns: 1fr auto;
    grid-template-areas:
      'row prices'
      'title prices'
      'sub prices';
    column-gap: 2rem;
    padding: 1.25rem 7rem 1.25rem 1.5rem;
  }
  .row {
    justify-content: flex-start;
    gap: 1.25rem;
  }
  .prices {
    flex-direction: column;
    align-self: center;
    flex-wrap: nowrap;
  }
  .title {
    font-size: 1.6rem;
  }
  .bird {
    right: 1.25rem;
    bottom: 50%;
    transform: translateY(50%) rotate(6deg);
    width: 5rem;
  }
}
</style>
