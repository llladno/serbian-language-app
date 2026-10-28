<script setup lang="ts">
// Тарифы: two ways to pay rubles for the paid levels. Specs:
// docs/superpowers/specs/2026-10-14-tariffs-design.md (the plans),
// docs/superpowers/specs/2026-10-26-level-choice-design.md (the learner picks the level after paying).
//
// Voice: that of the Telegram channel and the banner in the profile: a warm "мы / вы", a Serbian hello and
// thank-you with no gloss, a pixel heart. The facts (prices, the discount, the monthly limit, the offer) are
// the server's and unchanged; only the words and the pictures are new.
//
// Design note: the currency screens use gold for money (зёрнышки). Rubles must
// not borrow it, or the two would read as the same thing - so prices here are
// plain ink. The one loud thing is on the year: the bar that sets its price
// against twelve months of levels, which is the whole argument for the year.
// Everything else stays quiet, and each plan's advantages are set in bold with
// a small icon so they can be read in a glance.
import { computed, onMounted, ref } from 'vue'
import { LockOpen, Sparkles } from 'lucide-vue-next'
import { ApiError } from '../api'
import { useCourseStore } from '../stores/course'
import { useTariffsStore, payErrorMessage } from '../stores/tariffs'
import { useWalletStore } from '../stores/wallet'
import { useToasts } from '../lib/toasts'
import { useLevelChoiceModal } from '../lib/levelChoiceModal'
import { daysLeft, grouped, longDate, perMonth, rub, shortDate } from '../lib/tariffs'
import { pluralRu } from '../lib/plural'
import BotPrefsSection from '../components/BotPrefsSection.vue'
import LevelCreditNotice from '../components/LevelCreditNotice.vue'
import PaymentThanksModal from '../components/PaymentThanksModal.vue'
import ReceiptContactModal from '../components/ReceiptContactModal.vue'
import SeedIcon from '../components/SeedIcon.vue'
import scene from '../assets/tariffs-promo-scene.webp'
import heart from '../assets/donate/heart.png'
import foreverIcon from '../assets/promo/forever.webp'
import repeatIcon from '../assets/promo/repeat.webp'
import type { Order, TariffCode } from '../types'

const courseStore = useCourseStore()
const tariffs = useTariffsStore()
const wallet = useWalletStore()
const toasts = useToasts()
const { openFor: openChoice } = useLevelChoiceModal()

const levelStruck = computed(() => (tariffs.levelNow < tariffs.levelRegular ? tariffs.levelRegular : 0))

// A paid level opens a level of the learner's own choice, so the card does not name one: it lists what
// there is to choose from ("«Связная речь», «Падежи в жизни» и ещё 2"), by name without the number.
const choiceNames = computed(() =>
  tariffs.choices.map((c) => `«${c.title.replace(/^Уровень\s+\d+\s*[—–-]\s*/, '')}»`),
)
const inside = computed(() => {
  const names = choiceNames.value
  if (!names.length) return ''
  const rest = names.length - 3
  return names.slice(0, 3).join(', ') + (rest > 0 ? ` и ещё ${rest}` : '')
})

// The year against twelve months of levels, one a month: that sum is the full width of the bar, the year is
// its share of it (never so thin that it vanishes).
const yearBar = computed(() => Math.max(6, Math.round((tariffs.yearPrice / (tariffs.monthlyCost || 1)) * 100)))
const levelWaitDays = computed(() => (tariffs.levelFrom ? daysLeft(tariffs.levelFrom) : 0))

// How long a running year has left, for the "my tariff" card.
const yearDays = computed(() => (tariffs.yearUntil ? daysLeft(tariffs.yearUntil) : 0))

const canPay = computed(() => tariffs.paymentsReady && !tariffs.paying)

// An account that has no email (it came in through Telegram) is asked where the receipt goes when it
// presses a pay button: the window keeps the tariff that was pressed and starts its payment once it
// has an email or a phone. An account with an email goes straight to the bank. (Inside the Telegram
// Mini App the bank's page opens in the phone's browser; the store does that.)
const askingFor = ref<TariffCode | null>(null)
// Dev builds only (`make dev`): the window opens even for an account that has an email, so it can be looked at
// with the admin account; ?receipt=0 on the page switches that off to try the plain path. What is typed is not
// sent then: the server uses the account's own email. A production build has no such branch.
const askAnyway = import.meta.env.DEV && new URL(window.location.href).searchParams.get('receipt') !== '0'

async function pay(tariff: TariffCode) {
  if (!tariffs.paymentsReady) {
    toasts.push({ title: 'Оплата ещё не подключена', text: 'Пока уровни можно открыть за зёрнышки.' })
    return
  }
  if (tariffs.needsReceiptContact || askAnyway) {
    askingFor.value = tariff
    return
  }
  await startPayment(tariff)
}

async function contactGiven(contact: string) {
  const tariff = askingFor.value
  askingFor.value = null
  if (tariffs.needsReceiptContact) tariffs.setReceiptContact(contact)
  if (tariff) await startPayment(tariff)
}

async function startPayment(tariff: TariffCode) {
  try {
    await tariffs.pay(tariff)
  } catch (e) {
    // The limit may have started since the screen loaded (a payment from another
    // device): show the date instead of letting the button promise what it cannot do.
    if (e instanceof ApiError && e.message === 'level_cooldown') await tariffs.load(true)
    // A paid level is waiting to be spent: say so, and let the window do it.
    if (e instanceof ApiError && e.message === 'level_pending_choice') {
      await tariffs.load(true)
      openChoice()
    }
    toasts.push({ title: 'Оплата не началась', text: payErrorMessage(e) })
  }
}

// Coming back from the payment service (?order=ID): ask where the payment
// stands, say so, and refresh what the payment may have opened.
const returning = ref(false)
// A payment that went through is answered with a thank-you modal, not a toast.
const thanks = ref<Order | null>(null)

async function settleReturn(id: number, attempt = 0) {
  returning.value = true
  try {
    const order = await tariffs.checkOrder(id)
    if (order.status === 'succeeded') {
      await Promise.allSettled([courseStore.load(true), wallet.loadShop(), wallet.refresh()])
      // A paid level has no level yet: the learner chooses it in the window. A year (or an old order that
      // fixed its level) is thanked in the usual modal.
      if (order.choose) openChoice(order.id)
      else thanks.value = order
    } else if (order.status === 'canceled') {
      toasts.push({ title: 'Оплата не прошла', text: 'Деньги не списались. Можно попробовать ещё раз.' })
    } else if (attempt < 4) {
      // The bank can take a few seconds to confirm; ask again before giving up.
      setTimeout(() => settleReturn(id, attempt + 1), 3000)
      return
    } else {
      toasts.push({ title: 'Платёж обрабатывается', text: 'Уровень можно будет выбрать, как только придёт подтверждение.' })
    }
  } catch {
    toasts.push({ title: 'Не удалось проверить оплату', text: 'Обновите страницу через минуту.' })
  }
  returning.value = false
}

onMounted(async () => {
  courseStore.load()
  wallet.loadShop()
  wallet.refresh() // the currency's name for the bonus, and the balance it adds to
  const url = new URL(window.location.href)
  const orderId = Number(url.searchParams.get('order'))
  if (orderId > 0) {
    // Drop the parameter so a reload does not announce the payment again.
    window.history.replaceState(null, '', url.pathname)
    await settleReturn(orderId)
  }
  await tariffs.load(true)
  // A paid level that was never spent (the window was closed, or the payment came in while away):
  // ask again, once, when the screen opens.
  if (!(orderId > 0) && tariffs.hasCredit) openChoice()
})
</script>

<template>
  <div class="space-y-6">
    <!-- The top of the page: the swallow of the banner in the profile, and a hello. -->
    <header class="hero card" data-test="tariffs-hero">
      <div class="hero-art" aria-hidden="true"><img :src="scene" alt="" /></div>
      <div class="hero-body">
        <p class="hello" data-test="tariffs-hello">Ćao! 👋 Рады, что вы учите сербский с нами</p>
        <h1 class="text-2xl font-extrabold">Ласточка летит дальше</h1>
        <p class="mt-1 text-sm text-[var(--muted)]">
          Первый уровень бесплатен навсегда. Следующие открываются зёрнышками, которые вы получаете за учёбу, или сразу
          за рубли. Как удобнее, так и летите.
        </p>
      </div>
    </header>

    <LevelCreditNotice />

    <section v-if="tariffs.ready && !tariffs.paymentsReady" class="card p-4 text-sm" data-test="not-connected">
      <p class="font-bold">Оплата ещё не подключена</p>
      <p class="mt-1 text-[var(--muted)]">Цены можно посмотреть, а кнопки заработают, когда подключим платёжную систему.</p>
    </section>

    <!-- A running year. There is no auto-renew, so there is nothing to cancel:
         the card says so, and the one thing a person can still switch off is the
         bot's reminder that the year is ending. -->
    <section v-if="tariffs.yearActive && tariffs.yearUntil" class="card p-5" data-test="my-tariff">
      <div class="head">
        <p class="plan">Мой тариф</p>
        <span class="badge">Активен</span>
      </div>
      <p class="mt-1 text-lg font-extrabold">Год — все уровни</p>
      <p class="mt-0.5 text-sm text-[var(--muted)]" data-test="my-tariff-until">
        До {{ longDate(tariffs.yearUntil) }} · осталось {{ yearDays }} {{ pluralRu(yearDays, 'день', 'дня', 'дней') }}
      </p>
      <p class="mt-3 text-sm" data-test="no-autorenew">
        Автопродления нет: ничего не спишется, отписываться не нужно. Когда год закончится, вы сами решите, продлевать ли.
      </p>
      <p class="thanks mt-2" data-test="my-tariff-thanks">Hvala lepo <img :src="heart" alt="" aria-hidden="true" class="heart" /></p>
      <div class="mt-4"><BotPrefsSection :only="['billing']" /></div>
      <button class="btn btn-ghost mt-4 w-full" :disabled="!canPay" data-test="extend-year" @click="pay('year')">
        Продлить на год — {{ rub(tariffs.yearPrice) }}
      </button>
    </section>

    <div v-else-if="!tariffs.ready" class="grid gap-4 sm:grid-cols-2">
      <div v-for="i in 2" :key="i" class="card space-y-3 p-5">
        <div class="skel h-4 w-24"></div>
        <div class="skel h-8 w-32"></div>
        <div class="skel h-3.5 w-full"></div>
        <div class="skel h-11 w-full"></div>
      </div>
    </div>

    <!-- The year is the taller card on purpose: it carries the saving. The level card
         stays as short as its content instead of stretching to match it. -->
    <div v-else class="grid gap-4 sm:grid-cols-2 sm:items-start">
      <!-- Уровень -->
      <section class="card flex flex-col p-5" data-test="tariff-level">
        <p class="plan">Уровень</p>
        <p class="mt-0.5 text-lg font-extrabold leading-snug" data-test="level-title">Уровень на ваш выбор</p>
        <p v-if="inside" class="mt-1 text-sm text-[var(--muted)]" data-test="level-inside">
          После оплаты выберете сами: {{ inside }}
        </p>

        <p class="price-row" data-test="level-price">
          <span class="price">{{ rub(tariffs.levelNow) }}</span>
          <span v-if="levelStruck" class="was">{{ rub(levelStruck) }}</span>
          <span v-if="tariffs.levelDiscount" class="badge" data-test="level-discount">−{{ tariffs.levelDiscount }}%</span>
        </p>
        <p v-if="tariffs.levelDiscount" class="mt-1 text-sm font-semibold text-[var(--accent)]" data-test="level-discount-note">
          Скидка {{ tariffs.levelDiscount }}% на первую покупку
        </p>

        <ul class="perks">
          <li data-test="level-forever">
            <span class="ico"><img :src="foreverIcon" alt="" aria-hidden="true" class="px forever" /></span>
            <span><b>Навсегда, без подписки.</b> Уровень остаётся с вами, срока нет. Один платёж, само ничего не спишется.</span>
          </li>
          <li data-test="level-repeat">
            <span class="ico"><img :src="repeatIcon" alt="" aria-hidden="true" class="px repeat" /></span>
            <span><b>Повторения без лимита.</b> Сколько серий «Повторить» в день захотите, столько и будет.</span>
          </li>
          <li v-if="tariffs.levelBonus" class="centered" data-test="level-bonus">
            <span class="ico ico-seed"><SeedIcon :size="14" /></span>
            <span>
              <b>
                +{{ grouped(tariffs.levelBonus) }} <SeedIcon :size="13" class="inline-seed" />
                <span class="sr-only">{{ wallet.currencyWord(tariffs.levelBonus) }}</span>
                в подарок к уровню.
              </b>
            </span>
          </li>
        </ul>
        <p class="monthly" data-test="level-monthly">Новый уровень за рубли можно взять раз в месяц.</p>

        <!-- The monthly limit: say when the next level opens and what to do
             meanwhile, instead of leaving a dead button. -->
        <p v-if="tariffs.levelFrom && tariffs.choices.length" class="note" data-test="level-cooldown">
          Следующий уровень можно взять {{ shortDate(tariffs.levelFrom) }} (через {{ levelWaitDays }}
          {{ pluralRu(levelWaitDays, 'день', 'дня', 'дней') }}). Нужно раньше —
          <a href="#tariff-year" class="font-semibold underline underline-offset-2">откройте все уровни на год</a>.
        </p>

        <!-- A paid level that has not been spent comes first. Otherwise the tariff is always on show, but
             there is nothing to sell once every paid level is open: the button stays, switched off, rather
             than taking money for a level that does not exist. -->
        <button
          v-if="tariffs.hasCredit"
          class="btn btn-primary mt-auto w-full"
          data-test="level-choose"
          @click="openChoice()"
        >
          Выбрать оплаченный уровень
        </button>
        <button
          v-else-if="tariffs.choices.length && !tariffs.levelFrom"
          class="btn btn-primary mt-auto w-full"
          :disabled="!canPay"
          data-test="buy-level"
          @click="pay('level')"
        >
          <template v-if="tariffs.paying === 'level'">Переходим к оплате…</template>
          <template v-else>Хочу уровень — {{ rub(tariffs.levelNow) }}</template>
        </button>
        <button v-else-if="tariffs.choices.length && tariffs.levelFrom" class="btn btn-ghost mt-auto w-full" disabled data-test="level-wait">
          Следующий уровень — с {{ shortDate(tariffs.levelFrom) }}
        </button>
        <button v-else class="btn btn-ghost mt-auto w-full" disabled data-test="level-none">
          Все платные уровни открыты
        </button>
      </section>

      <!-- Год -->
      <section id="tariff-year" class="card year order-first flex flex-col p-5 sm:order-none" data-test="tariff-year">
        <div class="head">
          <p class="plan">Год</p>
          <span v-if="tariffs.yearSaves" class="badge badge-solid" data-test="year-badge">
            Выгода {{ tariffs.yearSaves.percent }}%
          </span>
        </div>
        <p class="mt-0.5 text-lg font-extrabold leading-snug">Все уровни и всё новое</p>

        <p class="price-row" data-test="year-price">
          <span class="price">{{ rub(tariffs.yearPrice) }}</span>
          <span class="was">{{ rub(tariffs.yearOld) }}</span>
        </p>
        <p class="mt-1 text-sm text-[var(--muted)]">≈ {{ rub(perMonth(tariffs.yearPrice)) }} в месяц · ранняя цена</p>

        <!-- The argument for the year, as one picture: a thin bar under the price, the year's share of what a level
             every month for twelve months would cost (the fastest pace the tariff allows); under it the year's
             price at the left and "вместо 3 488 ₽" at the right. -->
        <div v-if="tariffs.yearSaves" class="thin" data-test="year-saving">
          <span class="bar" aria-hidden="true"><i class="bar-fill" :style="{ width: yearBar + '%' }"></i></span>
          <p class="thin-row">
            <b data-test="saving-year">{{ rub(tariffs.yearPrice) }}</b>
            <span data-test="saving-monthly">вместо {{ rub(tariffs.monthlyCost) }}</span>
          </p>
        </div>

        <ul class="perks">
          <li class="centered">
            <span class="ico"><LockOpen :size="15" :stroke-width="2.5" /></span>
            <span><b>Все уровни сразу.</b></span>
          </li>
          <li data-test="year-repeat">
            <span class="ico"><img :src="repeatIcon" alt="" aria-hidden="true" class="px repeat" /></span>
            <span><b>Повторения без лимита.</b> Сколько хотите серий «Повторить» в день.</span>
          </li>
          <li v-if="tariffs.yearBonus" class="centered" data-test="year-bonus">
            <span class="ico ico-seed"><SeedIcon :size="14" /></span>
            <span>
              <b>
                +{{ grouped(tariffs.yearBonus) }} <SeedIcon :size="13" class="inline-seed" />
                <span class="sr-only">{{ wallet.currencyWord(tariffs.yearBonus) }}</span>
                в подарок.
              </b>
            </span>
          </li>
          <li>
            <span class="ico"><Sparkles :size="15" :stroke-width="2.5" /></span>
            <span><b>Новые уровни включены,</b> пока идёт год. Один платёж, автопродления нет, само ничего не спишется.</span>
          </li>
        </ul>
        <button class="btn btn-primary mt-auto w-full" :disabled="!canPay" data-test="buy-year" @click="pay('year')">
          <template v-if="tariffs.paying === 'year'">Переходим к оплате…</template>
          <template v-else>Хочу всё на год — {{ rub(tariffs.yearPrice) }}</template>
        </button>
      </section>
    </div>

    <p class="px-1 text-center text-xs text-[var(--muted)]" data-test="offer-note">
      Оплата проходит на странице банка, данные карты мы не получаем. Чек придёт на почту. Оплачивая, вы принимаете
      условия
      <a href="/oferta" target="_blank" rel="noopener" class="font-semibold underline underline-offset-2" data-test="offer-link"
        >публичной оферты</a
      >.
      <span class="thanks" data-test="thanks-line">Hvala lepo <img :src="heart" alt="" aria-hidden="true" class="heart" /></span>
    </p>

    <p class="px-1 text-center text-sm text-[var(--muted)]">
      Платить не обязательно: зёрнышки за уроки и задания тоже открывают уровни —
      <RouterLink to="/quests" class="font-semibold text-[var(--accent)] underline-offset-2 hover:underline"
        >как заработать</RouterLink
      >.
    </p>

    <PaymentThanksModal :order="thanks" :year-until="tariffs.yearUntil" @close="thanks = null" />
    <ReceiptContactModal
      :open="askingFor !== null"
      :initial="tariffs.receiptContact"
      @close="askingFor = null"
      @submit="contactGiven"
    />
  </div>
</template>

<style scoped>
/* One height for both cards' top row, so the plan names line up even though
   only the year carries a badge. */
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  min-height: 1.6rem;
}
button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.plan {
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--muted);
}
.price-row {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.35rem 0.6rem;
  margin-top: 0.75rem;
}
.price {
  font-size: 2rem;
  font-weight: 800;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}
.was {
  font-size: 1rem;
  font-weight: 600;
  color: var(--muted);
  text-decoration: line-through;
  font-variant-numeric: tabular-nums;
}
.badge {
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
  padding: 0.1rem 0.6rem;
  font-size: 0.8rem;
  font-weight: 800;
  white-space: nowrap;
}
.badge-solid {
  background: var(--accent);
  color: var(--bg);
}

/* The advantages: a bold lead, then the rest. The icon sits in a soft disc so
   the list reads as a row of small facts, not a column of ticks. */
.perks {
  margin: 1.1rem 0 1.25rem;
  display: grid;
  gap: 0.7rem;
  font-size: 0.875rem;
  line-height: 1.35;
}
.perks li {
  display: flex;
  align-items: flex-start;
  gap: 0.6rem;
}
/* A row that is one short line sits at the middle of its icon tile, not at the top of it. */
.perks li.centered {
  align-items: center;
}
.perks b {
  font-weight: 800;
}
.ico {
  display: grid;
  flex: none;
  place-items: center;
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 0.55rem;
  /* A light tile, like an app icon: the pixel icons have a dark outline, so on a dark tile
     (dark theme) they would sink into it. The same in both themes. */
  background: #e4f5f0;
  border: 1px solid rgb(255 255 255 / 0.85);
  box-shadow: 0 1px 2px rgb(8 46 55 / 0.2);
  color: #176b68;
}
/* Gold is the currency's colour everywhere in the app, so the bonus wears it. */
.ico-seed {
  background: #f4df9f;
}
.px {
  display: block;
  image-rendering: pixelated;
}
.forever {
  width: 1.25rem;
  height: auto;
}
.repeat {
  width: auto;
  height: 1rem;
}
.inline-seed {
  display: inline-block;
  vertical-align: -0.2em;
  margin: 0 0.2em 0 0.1em;
}
.heart {
  display: inline-block;
  height: 0.95em;
  width: auto;
  vertical-align: -0.14em;
  image-rendering: pixelated;
}
.thanks {
  font-weight: 700;
}
.monthly {
  margin: -0.5rem 0 1rem;
  font-size: 0.75rem;
  color: var(--muted);
}

/* The top of the page: the swallow fades into the card, the words sit below it. */
.hero {
  overflow: hidden;
}
.hero-art {
  height: 8.5rem;
  overflow: hidden;
}
.hero-art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 50% 24%;
  mask-image: linear-gradient(to bottom, #000 58%, transparent);
  -webkit-mask-image: linear-gradient(to bottom, #000 58%, transparent);
}
.hero-body {
  padding: 0.25rem 1.25rem 1.25rem;
}
.hello {
  margin-bottom: 0.25rem;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--accent);
}
@media (min-width: 640px) {
  .hero-art {
    height: 11rem;
  }
  .hero-art img {
    object-position: 50% 22%;
  }
}
.note {
  margin: -0.25rem 0 1rem;
  border-radius: 0.75rem;
  background: var(--bg-soft);
  padding: 0.65rem 0.8rem;
  font-size: 0.8125rem;
  line-height: 1.4;
}

.year {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
  background: linear-gradient(165deg, var(--accent-soft), var(--card) 42%);
}
/* The year's saving: a thin bar under the price, the amounts under its two ends. */
.thin {
  margin-top: 0.75rem;
}
.thin-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 0.5rem;
  margin-top: 0.35rem;
  font-size: 0.8125rem;
  color: var(--muted);
  font-variant-numeric: tabular-nums;
}
.thin-row b {
  font-weight: 800;
  color: var(--fg);
}
.bar {
  display: block;
  height: 0.4rem;
  overflow: hidden;
  border-radius: 999px;
  background: color-mix(in srgb, var(--muted) 30%, transparent);
}
.bar-fill {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--accent);
}
</style>
