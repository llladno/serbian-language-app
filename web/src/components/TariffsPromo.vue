<script setup lang="ts">
// The advert for the next level at the top of the profile. It is for a learner who
// has no tariff yet: once a year runs or a level has been paid for it goes away,
// and the small TariffsCard (the way back to "Мой тариф") takes its place.
//
// Design "Стекло": the picture fills the whole card and stays sharp; the text sits
// on a frosted-glass panel (left on a wide screen, at the bottom on a phone).
// Colours follow the app's accent; gold is kept for the one button and for the
// зёрнышки, which are gold everywhere in the app.
//
// The words are the owner's choice (2026-10-24), headline included: «1 000
// [зёрнышко] и уровень в подарок». It was raised twice that the level itself is
// paid for (the price sits right under the button, small); the owner decided to
// keep the wording. To soften it, say «в подарок к уровню».
//
// The tone is that of the Telegram channel: a warm "мы / вы", a Serbian hello and a
// Serbian thank-you with no gloss (the owner asked for none here), a heart. On a
// phone the hello is a small label in the sky above the bird, so the text panel
// stays short and the bird stays in view; on a wide screen it is the panel's first line.
//
// The price and the зёрнышки bonus are the server's (stores/tariffs.ts); nothing
// here is a number typed in. The profile loads the quote together with the rest
// of the page, so the advert appears with it instead of pushing it down later.
import { computed, ref, watch } from 'vue'
import { useTariffsStore } from '../stores/tariffs'
import { useWalletStore } from '../stores/wallet'
import { grouped, rub } from '../lib/tariffs'
import { trackTariff } from '../lib/tariffEvents'
import SeedIcon from './SeedIcon.vue'
import scene from '../assets/tariffs-promo-scene.webp'
// The same pixel heart as on the "Поддержать проект" card (already on this page, so no extra download).
import heart from '../assets/donate/heart.png'
// Pixel icons for the first two lines (docs/promo-icons/crop-icons.py makes them from the raw renders).
import foreverIcon from '../assets/promo/forever.webp'
import repeatIcon from '../assets/promo/repeat.webp'

const tariffs = useTariffsStore()
const wallet = useWalletStore()

const hello = 'Ćao! 👋 Рады, что вы учите сербский с нами'

// A level is the offer, so there has to be one left to open and nothing holding
// the learner back from buying it (the one-a-month limit only bites after a paid
// level, which already hides the advert, but the server has the last word).
const nextLevel = computed(() => tariffs.info?.level.next ?? null)
// The cross closes the banner for good on this device (the admin's «Воронка тарифов» counts who pressed it).
const DISMISS_KEY = 'srpski.promo.dismissed'
function readDismissed(): boolean {
  try {
    return localStorage.getItem(DISMISS_KEY) === '1'
  } catch {
    return false
  }
}
const dismissed = ref(readDismissed())
function dismiss() {
  dismissed.value = true
  try {
    localStorage.setItem(DISMISS_KEY, '1')
  } catch {
    /* it just comes back on the next visit */
  }
  trackTariff('dismiss', 'profile_banner')
}
const show = computed(
  () => !dismissed.value && tariffs.ready && !tariffs.hasTariff && !!nextLevel.value && !tariffs.levelFrom,
)

// The admin's «Воронка тарифов» counts the showings: once each time the banner appears, not on every re-render.
watch(show, (shown) => shown && trackTariff('view', 'profile_banner'), { immediate: true })
</script>

<template>
  <section v-if="show" class="promo card" data-test="tariffs-promo">
    <img class="scene" :src="scene" alt="" aria-hidden="true" />
    <div class="topbar">
      <p class="hook-pill">{{ hello }}</p>
      <button type="button" class="close" aria-label="Закрыть" title="Закрыть" data-test="promo-close" @click="dismiss">
        <span aria-hidden="true">×</span>
      </button>
    </div>
    <div class="glass">
      <p class="hook">{{ hello }}</p>
      <p class="title" data-test="promo-title">
        <template v-if="tariffs.levelBonus">
          {{ grouped(tariffs.levelBonus) }}
          <span class="seed-inline"><SeedIcon :size="20" /></span>
          <span class="sr-only">{{ wallet.currencyWord(tariffs.levelBonus) }}</span>
          и&nbsp;уровень в&nbsp;подарок
        </template>
        <template v-else>Следующий уровень навсегда</template>
      </p>

      <ul class="perks">
        <li data-test="promo-next">
          <span class="ico"><img :src="foreverIcon" alt="" aria-hidden="true" class="px forever" /></span>
          Навсегда, без подписки
        </li>
        <li data-test="promo-repeats">
          <span class="ico"><img :src="repeatIcon" alt="" aria-hidden="true" class="px repeat" /></span>
          Повторения без лимита
        </li>
        <li v-if="tariffs.levelBonus" data-test="promo-bonus">
          <span class="ico ico-seed"><SeedIcon :size="14" /></span>
          Зёрнышки — на другие уровни
        </li>
      </ul>

      <div class="actions">
        <RouterLink to="/tariffs" class="btn cta" data-test="promo-level" @click="trackTariff('click', 'profile_banner')">
          {{ tariffs.levelBonus ? 'Хочу зёрнышки и уровень' : 'Хочу уровень' }}
        </RouterLink>
        <p class="fine">
          <span data-test="promo-price">{{ rub(tariffs.levelNow) }} один раз, подписки нет</span>.
          <span data-test="promo-thanks">Hvala lepo <img :src="heart" alt="" aria-hidden="true" class="heart" /></span>
        </p>
      </div>
    </div>
  </section>
</template>

<style scoped>
/* Compact on purpose: this sits at the very top of the profile and must not push
   the page down more than it has to. */

/* Phone: the picture is the background, the bird in its upper part, and the
   glass panel sits at the bottom, under it. */
.promo {
  position: relative;
  display: flex;
  align-items: flex-end;
  overflow: hidden;
  min-height: 25rem;
  padding: 0.625rem;
}
.scene {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 66% 0%;
}
.glass {
  position: relative;
  width: min(26rem, 100%);
  padding: 0.9rem 1rem;
  border-radius: 16px;
  /* Made of the card's own colour, so it is light in a light theme and dark in a
     dark one, and the text on it keeps its contrast either way. */
  background: color-mix(in srgb, var(--card) 72%, transparent);
  -webkit-backdrop-filter: blur(16px) saturate(1.3);
  backdrop-filter: blur(16px) saturate(1.3);
  border: 1px solid color-mix(in srgb, var(--fg) 10%, transparent);
  box-shadow:
    inset 0 1px 0 color-mix(in srgb, #fff 45%, transparent),
    0 12px 32px -8px rgb(30 30 40 / 0.3);
}

.title {
  font-size: 1.2rem;
  font-weight: 800;
  line-height: 1.25;
}
/* The seed glyph sits in the line of text, after the number. */
.seed-inline {
  display: inline-block;
  vertical-align: -0.2em;
  margin: 0 0.1em;
}
.perks {
  margin-top: 0.625rem;
  display: grid;
  gap: 0.3rem;
  font-size: 0.875rem;
  font-weight: 600;
}
.perks li {
  display: flex;
  align-items: center;
  gap: 0.55rem;
}
.ico {
  display: flex;
  height: 1.75rem;
  width: 1.75rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 0.55rem;
  /* A soft light tile, like an app icon: the pixel icons have a dark outline, so on a
     dark tile (dark theme) they would sink into it. The same in both themes. */
  background: #e4f5f0;
  border: 1px solid rgb(255 255 255 / 0.85);
  box-shadow: 0 1px 2px rgb(8 46 55 / 0.2);
}
/* Gold is the currency's colour everywhere in the app, so the line about the
   зёрнышки wears it, with the seed itself in place of an icon. */
.ico-seed {
  background: #f4df9f;
}
.actions {
  margin-top: 0.8rem;
}

/* The one gold button: flat. */
.cta {
  padding: 0.6rem 0.9rem;
  font-size: 0.9rem;
  font-weight: 700;
  white-space: nowrap;
  color: #2b2007;
  background: color-mix(in srgb, var(--gold) 92%, #fff);
}
.cta:hover {
  background: var(--gold);
}
/* The price: small, but right under the button where the decision is made. */
.fine {
  margin-top: 0.35rem;
  font-size: 0.75rem;
  color: var(--muted);
}

/* The pixel icons in the chips. The sprites are drawn at about twice their size in device
   pixels on a phone screen, so the blocks stay whole. */
.px {
  display: block;
  image-rendering: pixelated;
}
.forever {
  width: 1.5rem;
  height: auto;
}
.repeat {
  width: auto;
  height: 1.25rem;
}

/* The heart after the thank-you: a pixel picture, so it is always drawn crisp. */
.heart {
  display: inline-block;
  height: 0.95em;
  width: auto;
  margin-left: 0.1em;
  vertical-align: -0.14em;
  image-rendering: pixelated;
}

/* Wider screens: the picture is wide, the glass panel sits on its left. */
@media (min-width: 640px) {
  .promo {
    align-items: center;
    min-height: 17rem;
    padding: 1rem;
  }
  .scene {
    object-position: 50% 15%;
  }
  .glass {
    padding: 1.1rem 1.25rem;
  }
  .title {
    font-size: 1.35rem;
  }
  .cta {
    padding: 0.6rem 1.25rem;
    font-size: 0.95rem;
  }
}

/* The hello. Phone: a small glass label in the sky, above the bird. Wide screen: the
   first line of the panel (the other one is hidden, so a screen reader hears it once). */
.hook {
  display: none;
  margin-bottom: 0.3rem;
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.3;
  color: var(--muted);
}
.topbar {
  position: absolute;
  top: 0.75rem;
  left: 0.75rem;
  right: 0.75rem;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.close {
  flex: none;
  margin-left: auto;
  display: grid;
  place-items: center;
  width: 1.75rem;
  height: 1.75rem;
  font-size: 1.25rem;
  line-height: 1;
  border-radius: 999px;
  color: var(--fg);
  background: color-mix(in srgb, var(--card) 72%, transparent);
  -webkit-backdrop-filter: blur(12px);
  backdrop-filter: blur(12px);
  border: 1px solid color-mix(in srgb, var(--fg) 10%, transparent);
}
.hook-pill {
  min-width: 0;
  padding: 0.35rem 0.75rem;
  border-radius: 14px;
  font-size: 0.8rem;
  font-weight: 600;
  line-height: 1.3;
  background: color-mix(in srgb, var(--card) 72%, transparent);
  -webkit-backdrop-filter: blur(12px);
  backdrop-filter: blur(12px);
  border: 1px solid color-mix(in srgb, var(--fg) 10%, transparent);
}
@media (min-width: 640px) {
  .hook {
    display: block;
  }
  .hook-pill {
    display: none;
  }
}
</style>
