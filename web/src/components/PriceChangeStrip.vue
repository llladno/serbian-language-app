<script setup lang="ts">
// The strip at the top of the profile while a price change is announced (migration 041), in the place of the advert for
// the next level (TariffsPromo hides itself meanwhile). A black strip like a shop's sale bar: a white sticker with the
// campaign's name, one line, the countdown and the one gold button. The words come from `campaignCopy`
// (lib/priceChange.ts). Delete it with the announcement.
//
// Honest by construction: the prices are not cut, the old ones are simply the last. The countdown is the server's
// (lib/priceChange.ts) and the change is made by the server at zero. A learner who has a tariff already (a running year or a paid
// level) has nothing to hurry for and sees no banner of the campaign (`tariffs.campaign`). It has no cross: it is one thin line, and it ends on its own (the owner's choice).
//
// Behind the text, small and quiet, stickers and the two current prices float up and fade away (PriceFloaters).
//
// The funnel: view and click under the source `price_banner` (store/tariff_events.go).
import { computed, watch } from 'vue'
import { useTariffsStore } from '../stores/tariffs'
import { trackTariff } from '../lib/tariffEvents'
import { usePriceChange } from '../lib/priceChange'
import CountdownText from './CountdownText.vue'
import PriceFloaters from './PriceFloaters.vue'
import '../styles/priceChange.css'

const tariffs = useTariffsStore()
const { change, remaining, copy } = usePriceChange()

const props = defineProps<{ spaced?: boolean; place?: 'profile' | 'course' }>()

// The owner's own account (`preview`) sees it whatever else is true: a running year.
const show = computed(
  () => tariffs.ready && !!change.value,
)

// Counted once for each time the strip appears, not on every tick.
watch(show, (shown) => shown && trackTariff('view', 'price_banner', props.place ?? ''), { immediate: true })
</script>

<template>
  <div v-if="show" class="wrap" :class="{ spaced }" data-test="price-strip">
    <section class="strip">
      <PriceFloaters with-prices rise="-3.8rem" test-id="price-strip-floaters" />
      <span class="pc-sticker sticker" data-test="price-strip-sticker">{{ copy?.sticker }}</span>
      <p class="msg" data-test="price-strip-msg">{{ copy?.strip }}</p>
      <p class="tm" role="timer" aria-live="off" data-test="price-strip-timer">
        <span class="long"><CountdownText :ms="remaining" /></span>
        <span class="short">ещё <CountdownText :ms="remaining" /></span>
      </p>
      <RouterLink to="/tariffs" class="pc-btn go" data-test="price-strip-go" @click="trackTariff('click', 'price_banner', place ?? '')">
        {{ copy?.stripButton }}
      </RouterLink>
    </section>
  </div>
</template>

<style scoped>
/* The container, not the viewport, decides the layout: the strip sits in a column that is narrow on a wide screen too. */
.wrap {
  container-type: inline-size;
}
.spaced {
  margin-bottom: 1rem;
}
.strip {
  position: relative;
  overflow: hidden;
  display: grid;
  align-items: center;
  gap: 0.2rem 0.75rem;
  padding: 0.55rem 0.5rem 0.55rem 0.9rem;
  color: #fff;
  background: #0b1211;
  border-radius: 16px;
  /* On a dark page the strip would sink into it: a hairline keeps its edge. */
  box-shadow: 0 0 0 1px rgb(255 255 255 / 0.08);
  /* Phone: the sticker, the line and the time one under another, the button on the right. */
  grid-template-columns: 1fr auto;
  grid-template-areas:
    'sticker go'
    'msg go'
    'tm go';
}
/* The text and the button stay above what floats behind them. */
.strip > :not(.floaters) {
  position: relative;
  z-index: 1;
}

.sticker {
  grid-area: sticker;
  justify-self: start;
  margin-bottom: 0.2rem;
  font-size: 0.8rem;
}
.msg {
  grid-area: msg;
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1.2;
}
.tm {
  grid-area: tm;
  font-size: 0.8125rem;
  font-weight: 600;
  color: #9bb1ae;
  font-variant-numeric: tabular-nums;
}
.long {
  display: none;
}
.go {
  grid-area: go;
  padding: 0.7rem 0.85rem;
}
/* Wide: one line. */
@container (min-width: 740px) {
  .strip {
    grid-template-columns: auto 1fr auto auto;
    grid-template-areas: 'sticker msg tm go';
    gap: 0 0.9rem;
    padding: 0.55rem 0.5rem 0.55rem 1rem;
    min-height: 62px;
  }
  .sticker {
    margin-bottom: 0;
    font-size: 1.05rem;
  }
  .msg {
    font-size: 1.05rem;
  }
  .tm {
    font-size: 1.05rem;
    color: #fff;
    white-space: nowrap;
  }
  .long {
    display: inline;
  }
  .short {
    display: none;
  }
  .go {
    padding: 0.65rem 1rem;
  }
}
</style>
