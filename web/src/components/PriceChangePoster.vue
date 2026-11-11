<script setup lang="ts">
// The poster of the price change (migration 041): a black poster with the last day of the old prices as a giant date,
// the two prices as stickers (now → after), the countdown and the one gold button. It is on the tariffs screen, in the
// place of the swallow's hello, and in a window that opens at the end of a lesson (PriceChangeModal), which asks for
// `animate`: the pieces arrive one after another, the sticker first, the button, the date, the prices, the swallow.
//
// The words are `campaignCopy` (lib/priceChange.ts) and say what is true: the prices that follow the discount really
// take over on the day. The prices are the server's (stores/tariffs.ts), the learner's own: a learner who has paid for
// a level already sees the regular one now and after.
//
// The button goes to `to` when one is given (the window sends the learner to the tariffs screen), otherwise it takes the
// learner down to the cards below the poster. Either way `go` is emitted first.
import { computed } from 'vue'
import { useTariffsStore } from '../stores/tariffs'
import { rub } from '../lib/tariffs'
import { LEVEL_LABEL, YEAR_LABEL, lastDay, usePriceChange } from '../lib/priceChange'
import CountdownText from './CountdownText.vue'
import PriceFloaters from './PriceFloaters.vue'
import bird from '../assets/support-mascot.webp'
import heart from '../assets/donate/heart.png'
import ruble from '../assets/promo/stickers/ruble.webp'
import bolt from '../assets/promo/stickers/bolt.webp'
import cross from '../assets/promo/stickers/cross.webp'
import '../styles/priceChange.css'

defineProps<{ animate?: boolean; to?: string }>()
const emit = defineEmits<{ go: [] }>()

const tariffs = useTariffsStore()
const { change, remaining, copy } = usePriceChange()

// `change` is already empty for a learner who has a tariff (the owner's own account excepted), and there is nothing to show a
// level for once every level is open.
const show = computed(() => tariffs.ready && !!change.value)
const showLevel = computed(() => tariffs.choices.length > 0 || tariffs.hasCredit)
const until = computed(() => (change.value ? lastDay(change.value.at) : { day: '', month: '' }))
</script>

<template>
  <div v-if="show && change" class="wrap" :class="{ enter: animate }" data-test="price-poster">
    <header class="poster">
      <PriceFloaters rise="-34rem" start-delay="2.4s" size="2rem" test-id="price-poster-floaters" />
      <div class="top">
        <span class="pc-sticker label" data-test="price-poster-sticker">{{ copy?.sticker }}</span>
        <h1 class="title" data-test="price-poster-title">{{ copy?.title }}</h1>
        <p class="sub" data-test="price-poster-sub">{{ copy?.sub }}</p>
        <div class="act">
          <RouterLink v-if="to" :to="to" class="pc-btn" data-test="price-poster-go" @click="emit('go')">
            {{ copy?.posterButton }}
          </RouterLink>
          <a v-else href="#tariff-cards" class="pc-btn" data-test="price-poster-go" @click="emit('go')">
            {{ copy?.posterButton }}
          </a>
          <span class="fine"
            >Купленное остаётся навсегда. Hvala lepo
            <img :src="heart" alt="" aria-hidden="true" class="heart"
          /></span>
        </div>
      </div>

      <div class="big">
        <p class="date" data-test="price-poster-date">{{ until.day }}<span>{{ until.month.toUpperCase() }}</span></p>
        <p class="tmr" role="timer" aria-live="off" data-test="price-poster-timer"><CountdownText :ms="remaining" /></p>
        <p v-if="showLevel" class="st st-level" data-test="price-poster-level">
          <small>{{ LEVEL_LABEL.toUpperCase() }}</small><span class="a">{{ rub(tariffs.levelNow) }}</span
          ><span class="arr">→</span><span class="b">{{ rub(change.levelPrice) }}</span>
        </p>
        <p class="st st-year" data-test="price-poster-year">
          <small>{{ YEAR_LABEL.toUpperCase() }}</small><span class="a">{{ rub(tariffs.yearPrice) }}</span
          ><span class="arr">→</span><span class="b">{{ rub(change.yearPrice) }}</span>
        </p>
        <img :src="bolt" alt="" aria-hidden="true" class="deco deco-bolt" />
        <img :src="cross" alt="" aria-hidden="true" class="deco deco-cross" />
        <img :src="ruble" alt="" aria-hidden="true" class="deco deco-ruble" />
        <img :src="bird" alt="" aria-hidden="true" class="bird" />
      </div>
    </header>
  </div>
</template>

<style scoped>
.wrap {
  container-type: inline-size;
}
.poster {
  position: relative;
  overflow: hidden;
  color: #fff;
  background: #0b1211;
  border-radius: 20px;
  box-shadow: 0 0 0 1px rgb(255 255 255 / 0.08);
}
.top {
  position: relative;
  z-index: 1;
  padding: 1.25rem 1.25rem 0;
}

.label {
  font-size: 1.15rem;
}
.title {
  margin-top: 0.85rem;
  font-size: 1.5rem;
  font-weight: 800;
  line-height: 1.15;
}
.sub {
  margin-top: 0.5rem;
  font-size: 0.95rem;
  font-weight: 600;
  line-height: 1.35;
  color: #9bb1ae;
}
.act {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.6rem 0.9rem;
  padding: 1rem 0 0;
}
.fine {
  font-size: 0.75rem;
  color: #9bb1ae;
}
.heart {
  display: inline-block;
  height: 0.95em;
  width: auto;
  margin-left: 0.1em;
  vertical-align: -0.14em;
  image-rendering: pixelated;
}

/* The lower half: the date, the countdown, the two prices and the swallow, laid out by hand on a fixed stage. */
.big {
  position: relative;
  z-index: 1;
  height: 19rem;
  margin-top: 0.5rem;
}
.date {
  position: absolute;
  left: 1rem;
  top: 0.4rem;
  font-family: Oswald, 'Manrope', sans-serif;
  font-size: 7rem;
  font-weight: 700;
  line-height: 1;
  color: #fff;
}
.date span {
  display: inline-block;
  margin-left: 0.4rem;
  font-size: 1.9rem;
  letter-spacing: 0.04em;
  color: #9bb1ae;
}
.tmr {
  position: absolute;
  right: 0.9rem;
  top: 1.6rem;
  padding: 7px 11px 8px;
  font-family: Oswald, 'Manrope', sans-serif;
  font-size: 1.05rem;
  font-weight: 700;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  background: #111;
  border: 2px solid #fff;
  border-radius: 14px;
  transform: rotate(4deg);
}
.st {
  position: absolute;
  padding: 9px 14px 11px;
  font-family: Oswald, 'Manrope', sans-serif;
  font-weight: 700;
  line-height: 1;
  white-space: nowrap;
  color: #111;
  background: #fff;
  border-radius: 14px;
  box-shadow: 0 5px 0 rgb(255 255 255 / 0.18);
}
.st small {
  display: block;
  margin-bottom: 3px;
  font-family: Manrope, sans-serif;
  font-size: 0.6875rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  opacity: 0.6;
}
.st .a,
.st .b {
  font-size: 1.9rem;
}
.st .arr {
  margin: 0 3px;
  font-size: 1.3rem;
  opacity: 0.5;
}
.st .b {
  color: #9aa3a2;
}
.st-level {
  left: 0.8rem;
  bottom: 5.3rem;
  transform: rotate(-3deg);
}
.st-year {
  left: 1.9rem;
  bottom: 1rem;
  transform: rotate(2deg);
}
.deco {
  position: absolute;
  display: block;
  height: auto;
  filter: drop-shadow(0 3px 4px rgb(0 0 0 / 0.4));
}
.deco-bolt {
  width: 2.4rem;
  left: 10rem;
  top: 0.4rem;
  transform: rotate(-14deg);
}
.deco-cross {
  width: 2.3rem;
  right: 1.1rem;
  bottom: 4.2rem;
  transform: rotate(10deg);
}
.deco-ruble {
  width: 2.8rem;
  right: 0.6rem;
  bottom: 0.9rem;
  transform: rotate(-9deg);
}
.bird {
  position: absolute;
  right: 0.4rem;
  bottom: 6.9rem;
  width: 6rem;
  transform: rotate(6deg);
  image-rendering: pixelated;
  filter: drop-shadow(2px 0 0 #fff) drop-shadow(-2px 0 0 #fff) drop-shadow(0 2px 0 #fff) drop-shadow(0 -2px 0 #fff)
    drop-shadow(0 4px 6px rgb(0 0 0 / 0.35));
}

/* Wide: the words on the left, the stage on the right. */
@container (min-width: 700px) {
  .poster {
    display: grid;
    grid-template-columns: 1fr 22rem;
    align-items: stretch;
  }
  .top {
    padding: 1.75rem 0 1.75rem 1.75rem;
    align-self: center;
  }
  .title {
    font-size: 1.9rem;
  }
  .big {
    height: auto;
    min-height: 19rem;
    margin-top: 0;
  }
  .date {
    font-size: 6rem;
  }
}

/* The entrance (`animate`): about two seconds, in the order the eye should take the poster: the sticker drops in, the
   words rise, the button, the date slides in out of a blur, the countdown pops, the two prices fly in from the sides,
   the swallow swoops down and then floats. Each piece ends in the state set above (`both` holds the first frame
   during its delay), so switching the animation off leaves the finished poster. */
.enter .label {
  animation: pc-drop 0.6s 0.1s cubic-bezier(0.2, 1.5, 0.4, 1) both;
}
.enter .title {
  animation: pc-rise 0.55s 0.28s cubic-bezier(0.2, 0.8, 0.3, 1) both;
}
.enter .sub {
  animation: pc-rise 0.55s 0.4s cubic-bezier(0.2, 0.8, 0.3, 1) both;
}
.enter .act {
  animation: pc-rise 0.55s 0.55s cubic-bezier(0.2, 0.8, 0.3, 1) both;
}
.enter .date {
  animation: pc-date 0.85s 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) both;
}
.enter .date span {
  animation: pc-rise 0.5s 0.95s cubic-bezier(0.2, 0.8, 0.3, 1) both;
}
.enter .tmr {
  animation: pc-pop 0.55s 1.05s cubic-bezier(0.2, 1.6, 0.4, 1) both;
}
.enter .st-level {
  animation: pc-fly-left 0.75s 1.1s cubic-bezier(0.2, 1.25, 0.4, 1) both;
}
.enter .st-year {
  animation: pc-fly-right 0.75s 1.28s cubic-bezier(0.2, 1.25, 0.4, 1) both;
}
.enter .bird {
  animation:
    pc-swoop 0.95s 0.85s cubic-bezier(0.2, 0.9, 0.3, 1.1) both,
    pc-float 3.4s 1.9s ease-in-out infinite;
}
.enter .fine {
  animation: pc-fade 0.7s 1.5s ease both;
}
.enter .deco-bolt {
  animation: pc-deco-bolt 0.55s 1.5s cubic-bezier(0.2, 1.6, 0.4, 1) both;
}
.enter .deco-cross {
  animation: pc-deco-cross 0.55s 1.65s cubic-bezier(0.2, 1.6, 0.4, 1) both;
}
.enter .deco-ruble {
  animation: pc-deco-ruble 0.55s 1.8s cubic-bezier(0.2, 1.6, 0.4, 1) both;
}
@keyframes pc-deco-bolt {
  from {
    opacity: 0;
    transform: scale(0.3) rotate(-60deg);
  }
  to {
    opacity: 1;
    transform: scale(1) rotate(-14deg);
  }
}
@keyframes pc-deco-cross {
  from {
    opacity: 0;
    transform: scale(0.3) rotate(70deg);
  }
  to {
    opacity: 1;
    transform: scale(1) rotate(10deg);
  }
}
@keyframes pc-deco-ruble {
  from {
    opacity: 0;
    transform: scale(0.3) rotate(-70deg);
  }
  to {
    opacity: 1;
    transform: scale(1) rotate(-9deg);
  }
}
@keyframes pc-drop {
  from {
    opacity: 0;
    transform: translateY(-20px) rotate(-16deg) scale(0.7);
  }
  to {
    opacity: 1;
    transform: translateY(0) rotate(-4deg) scale(1);
  }
}
@keyframes pc-rise {
  from {
    opacity: 0;
    transform: translateY(14px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}
@keyframes pc-date {
  from {
    opacity: 0;
    transform: translateX(-46px) scale(1.18);
    filter: blur(10px);
  }
  to {
    opacity: 1;
    transform: none;
    filter: blur(0);
  }
}
@keyframes pc-pop {
  from {
    opacity: 0;
    transform: scale(0.35) rotate(-14deg);
  }
  to {
    opacity: 1;
    transform: scale(1) rotate(4deg);
  }
}
@keyframes pc-fly-left {
  from {
    opacity: 0;
    transform: translateX(-70px) rotate(-14deg);
  }
  to {
    opacity: 1;
    transform: translateX(0) rotate(-3deg);
  }
}
@keyframes pc-fly-right {
  from {
    opacity: 0;
    transform: translateX(80px) rotate(10deg);
  }
  to {
    opacity: 1;
    transform: translateX(0) rotate(2deg);
  }
}
@keyframes pc-swoop {
  from {
    opacity: 0;
    transform: translate(80px, -60px) rotate(34deg) scale(0.5);
  }
  to {
    opacity: 1;
    transform: translate(0, 0) rotate(6deg) scale(1);
  }
}
@keyframes pc-float {
  0%,
  100% {
    transform: translateY(0) rotate(6deg);
  }
  50% {
    transform: translateY(-7px) rotate(9deg);
  }
}
@keyframes pc-fade {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}
@media (prefers-reduced-motion: reduce) {
  .enter .label,
  .enter .title,
  .enter .sub,
  .enter .act,
  .enter .date,
  .enter .date span,
  .enter .tmr,
  .enter .st-level,
  .enter .st-year,
  .enter .bird,
  .enter .deco,
  .enter .fine {
    animation: none;
  }
}
</style>
