<script setup lang="ts">
// The learner's own discount (the owner gives it in the admin panel, docs/PERSONAL_DISCOUNT.md): a quiet card with the
// percent, the time that is left and one button to the tariffs. It is shown only while the server quotes a discount
// (`tariffs.discount`), so it goes away by itself when the discount ends or is spent. The time is the server's clock
// corrected on the phone (stores/tariffs.ts), like the price-change countdown.
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useTariffsStore } from '../stores/tariffs'
import { formatCountdown } from '../lib/priceChange'
import heart from '../assets/donate/heart.png'

// `onTariffs`: the banner sits on the tariffs screen itself, so its button scrolls to the cards instead of navigating.
defineProps<{ spaced?: boolean; onTariffs?: boolean }>()

const tariffs = useTariffsStore()
const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | undefined
const tick = () => (now.value = Date.now())
onMounted(() => {
  timer = setInterval(tick, 1000)
  document.addEventListener('visibilitychange', tick)
})
onBeforeUnmount(() => {
  clearInterval(timer)
  document.removeEventListener('visibilitychange', tick)
})

const show = computed(() => tariffs.ready && !!tariffs.discount && tariffs.discount.until > now.value)
const left = computed(() => (tariffs.discount ? formatCountdown(tariffs.discount.until - now.value) : ''))
</script>

<template>
  <section v-if="show" class="pd card" :class="{ spaced }" data-test="personal-discount">
    <div class="pd-sticker" data-test="personal-discount-percent">−{{ tariffs.discount!.percent }}%</div>
    <div class="pd-body">
      <p class="pd-title">
        Скидка для вас <img :src="heart" alt="" aria-hidden="true" class="pd-heart" />
      </p>
      <p class="pd-text">
        Только для вас, на любой тариф. Осталось <b class="pd-left" role="timer" aria-live="off" data-test="personal-discount-left">{{ left }}</b>
      </p>
    </div>
    <a v-if="onTariffs" href="#tariff-cards" class="btn btn-primary pd-go" data-test="personal-discount-go">Посмотреть</a>
    <RouterLink v-else to="/tariffs" class="btn btn-primary pd-go" data-test="personal-discount-go">Выбрать</RouterLink>
  </section>
</template>

<style scoped>
.pd {
  display: flex;
  align-items: center;
  gap: 0.9rem;
  padding: 0.9rem 1rem;
  background: var(--accent-soft);
}
.spaced {
  margin-bottom: 1rem;
}
.pd-sticker {
  flex: none;
  display: grid;
  place-items: center;
  min-width: 3.4rem;
  height: 3.4rem;
  padding: 0 0.5rem;
  border-radius: 1rem;
  background: var(--accent);
  color: #fff;
  font-weight: 800;
  font-size: 1.15rem;
}
.pd-body {
  flex: 1;
  min-width: 0;
}
.pd-title {
  font-weight: 800;
  line-height: 1.2;
}
.pd-heart {
  display: inline-block;
  width: 0.95em;
  height: 0.95em;
  image-rendering: pixelated;
  vertical-align: -0.08em;
}
.pd-text {
  margin-top: 0.15rem;
  font-size: 0.85rem;
  color: var(--muted);
}
.pd-left {
  color: var(--ink, inherit);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.pd-go {
  flex: none;
}
@media (max-width: 30rem) {
  /* sticker and text share the first row, the button takes the whole second one */
  .pd {
    display: grid;
    grid-template-columns: auto 1fr;
  }
  .pd-go {
    grid-column: 1 / -1;
    text-align: center;
  }
}
</style>
