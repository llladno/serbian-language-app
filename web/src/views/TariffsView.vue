<script setup lang="ts">
// Тарифы: two ways to pay rubles for the paid levels. Spec:
// docs/superpowers/specs/2026-10-14-tariffs-design.md.
//
// Design note: the currency screens use gold for money (зёрнышки). Rubles must
// not borrow it, or the two would read as the same thing - so prices here are
// plain ink. The one loud thing is on the year: the bar that sets its price
// against twelve months of levels, which is the whole argument for the year.
// Everything else stays quiet, and each plan's advantages are set in bold with
// a small icon so they can be read in a glance.
import { computed, onMounted, ref } from 'vue'
import { Check, ExternalLink, Infinity as InfinityIcon, CreditCard, CalendarClock, LockOpen, Sparkles } from 'lucide-vue-next'
import { ApiError } from '../api'
import { useCourseStore } from '../stores/course'
import { useTariffsStore, payErrorMessage } from '../stores/tariffs'
import { useWalletStore } from '../stores/wallet'
import { useToasts } from '../lib/toasts'
import { isTelegram, initData } from '../telegram'
import { daysLeft, grouped, longDate, nextLockedLevel, perMonth, rub, shortDate } from '../lib/tariffs'
import { pluralRu } from '../lib/plural'
import BotPrefsSection from '../components/BotPrefsSection.vue'
import PaymentThanksModal from '../components/PaymentThanksModal.vue'
import SeedIcon from '../components/SeedIcon.vue'
import type { Order, TariffCode } from '../types'

const courseStore = useCourseStore()
const tariffs = useTariffsStore()
const wallet = useWalletStore()
const toasts = useToasts()

const next = computed(() => nextLockedLevel(courseStore.course))
const levelStruck = computed(() => (tariffs.levelNow < tariffs.levelRegular ? tariffs.levelRegular : 0))

// What the next level is about, from its shop entry ("Открывает уровень 4:
// падежи, прилагательные, местоимения"), and how many lessons are written.
const inside = computed(() => {
  const text = next.value ? wallet.productForPhase(next.value.id)?.description ?? '' : ''
  const topics = text.replace(/^Открывает уровень \d+:\s*/i, '').trim()
  const lessons = next.value && courseStore.course
    ? next.value.lessons.filter((id) => !courseStore.course!.lessons.find((l) => l.id === id)?.planned).length
    : 0
  const parts = []
  if (topics) parts.push(topics.charAt(0).toUpperCase() + topics.slice(1))
  // a no-break space, so the number never ends a line without its word
  if (lessons) parts.push(`${lessons}\u00a0${pluralRu(lessons, 'урок', 'урока', 'уроков')}`)
  return parts.join(' · ')
})

// The year against twelve months of levels, as two bars: the level yardstick is
// the full width, the year is its share of it (never so thin that it vanishes).
const yearBar = computed(() => Math.max(6, Math.round((tariffs.yearPrice / (tariffs.monthlyCost || 1)) * 100)))
// "1 000 зёрнышек": the bonus with the currency's own name and plural.
function seeds(n: number): string {
  return `${grouped(n)} ${wallet.currencyWord(n)}`.trim()
}
const levelWaitDays = computed(() => (tariffs.levelFrom ? daysLeft(tariffs.levelFrom) : 0))

// How long a running year has left, for the "my tariff" card.
const yearDays = computed(() => (tariffs.yearUntil ? daysLeft(tariffs.yearUntil) : 0))

// Telegram sells digital goods in Stars only, so inside the Mini App the same
// button hands off to the phone's browser instead of paying in place (spec,
// "Inside Telegram"). Same check as DonateCard: the script is loaded everywhere,
// but initData is only non-empty in a real Telegram session.
const inMiniApp = isTelegram() && !!initData()

const canPay = computed(() => tariffs.paymentsReady && !tariffs.paying)

// The receipt contact (email or phone) of an account that has no email (it came
// in through Telegram): asked once, above the plans, and checked before any
// payment starts.
const receiptField = ref<HTMLInputElement | null>(null)

async function pay(tariff: TariffCode) {
  if (!tariffs.paymentsReady) {
    toasts.push({ title: 'Оплата ещё не подключена', text: 'Пока уровни можно открыть за зёрнышки.' })
    return
  }
  if (!tariffs.receiptContactValid) {
    toasts.push({ title: 'Нужна почта или телефон для чека', text: 'Впиши их в поле над тарифами.' })
    receiptField.value?.focus()
    return
  }
  try {
    await tariffs.pay(tariff)
  } catch (e) {
    // The limit may have started since the screen loaded (a payment from another
    // device): show the date instead of letting the button promise what it cannot do.
    if (e instanceof ApiError && e.message === 'level_cooldown') await tariffs.load(true)
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
      thanks.value = order
    } else if (order.status === 'canceled') {
      toasts.push({ title: 'Оплата не прошла', text: 'Деньги не списались. Можно попробовать ещё раз.' })
    } else if (attempt < 4) {
      // The bank can take a few seconds to confirm; ask again before giving up.
      setTimeout(() => settleReturn(id, attempt + 1), 3000)
      return
    } else {
      toasts.push({ title: 'Платёж обрабатывается', text: 'Уровень откроется, как только придёт подтверждение.' })
    }
  } catch {
    toasts.push({ title: 'Не удалось проверить оплату', text: 'Обнови страницу через минуту.' })
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
})
</script>

<template>
  <div class="space-y-6">
    <header class="px-1">
      <h1 class="text-2xl font-extrabold">Тарифы</h1>
      <p class="mt-1 text-sm text-[var(--muted)]">
        Три первых уровня бесплатны навсегда. Дальше уровни можно заработать зёрнышками за учёбу или открыть сразу.
      </p>
    </header>

    <section v-if="tariffs.ready && !tariffs.paymentsReady" class="card p-4 text-sm" data-test="not-connected">
      <p class="font-bold">Оплата ещё не подключена</p>
      <p class="mt-1 text-[var(--muted)]">Цены можно посмотреть, а кнопки заработают, когда подключим платёжную систему.</p>
    </section>

    <!-- An account that came in through Telegram has no email: the receipt (54-ФЗ)
         has to go somewhere, an email or a phone, so it is asked for here, once. -->
    <section v-if="tariffs.ready && tariffs.needsReceiptContact" class="card p-4" data-test="receipt-contact">
      <label for="receipt-contact" class="block text-sm font-bold">Почта или телефон для чека</label>
      <input
        id="receipt-contact"
        ref="receiptField"
        :value="tariffs.receiptContact"
        type="text"
        inputmode="email"
        autocomplete="email"
        autocapitalize="off"
        placeholder="почта или телефон"
        class="field mt-2 w-full"
        data-test="receipt-contact-input"
        @input="tariffs.setReceiptContact(($event.target as HTMLInputElement).value)"
      />
      <p class="mt-1.5 text-xs text-[var(--muted)]">Туда придёт чек об оплате. Доступ откроется на этом аккаунте.</p>
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
        Автопродления нет: ничего не спишется, отписываться не нужно. Когда год закончится, ты сам решишь, продлевать ли.
      </p>
      <div class="mt-4"><BotPrefsSection :only="['billing']" /></div>
      <button class="btn btn-ghost mt-4 w-full" :disabled="!canPay" data-test="extend-year" @click="pay('year')">
        <template v-if="inMiniApp">Продлить на сайте <ExternalLink :size="16" :stroke-width="2.25" /></template>
        <template v-else>Продлить на год — {{ rub(tariffs.yearPrice) }}</template>
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
        <p class="mt-0.5 text-lg font-extrabold leading-snug" data-test="level-title">
          {{ next ? next.title : 'Следующий уровень' }}
        </p>
        <p v-if="inside" class="mt-1 text-sm text-[var(--muted)]" data-test="level-inside">{{ inside }}</p>

        <p class="price-row" data-test="level-price">
          <span class="price">{{ rub(tariffs.levelNow) }}</span>
          <span v-if="levelStruck" class="was">{{ rub(levelStruck) }}</span>
          <span v-if="tariffs.levelDiscount" class="badge" data-test="level-discount">−{{ tariffs.levelDiscount }}%</span>
        </p>
        <p v-if="tariffs.levelDiscount" class="mt-1 text-sm font-semibold text-[var(--accent)]" data-test="level-discount-note">
          Скидка {{ tariffs.levelDiscount }}% на первую покупку: {{ rub(tariffs.levelNow) }} вместо
          {{ rub(tariffs.levelRegular) }}
        </p>

        <ul class="perks">
          <li>
            <span class="ico"><InfinityIcon :size="15" :stroke-width="2.5" /></span>
            <span><b>Навсегда.</b> Уровень остаётся у тебя, срока нет.</span>
          </li>
          <li v-if="tariffs.levelBonus" data-test="level-bonus">
            <span class="ico ico-seed"><SeedIcon :size="14" /></span>
            <span><b>+{{ seeds(tariffs.levelBonus) }}.</b> В подарок к уровню, тратятся в магазине.</span>
          </li>
          <li>
            <span class="ico"><CreditCard :size="15" :stroke-width="2.5" /></span>
            <span><b>Один платёж.</b> Подписки и автосписаний нет.</span>
          </li>
          <li>
            <span class="ico"><CalendarClock :size="15" :stroke-width="2.5" /></span>
            <span><b>Раз в месяц.</b> Новый уровень за рубли открывается не чаще.</span>
          </li>
        </ul>

        <!-- The monthly limit: say when the next level opens and what to do
             meanwhile, instead of leaving a dead button. -->
        <p v-if="tariffs.levelFrom && next" class="note" data-test="level-cooldown">
          Следующий уровень можно взять {{ shortDate(tariffs.levelFrom) }} (через {{ levelWaitDays }}
          {{ pluralRu(levelWaitDays, 'день', 'дня', 'дней') }}). Нужно раньше —
          <a href="#tariff-year" class="font-semibold underline underline-offset-2">открой все уровни на год</a>.
        </p>

        <!-- The tariff is always on show, but there is nothing to sell once every
             paid level is open: the button stays, switched off, rather than
             taking money for a level that does not exist. -->
        <button
          v-if="next && !tariffs.levelFrom"
          class="btn btn-ghost mt-auto w-full"
          :disabled="!canPay"
          data-test="buy-level"
          @click="pay('level')"
        >
          <template v-if="tariffs.paying === 'level'">Переходим к оплате…</template>
          <template v-else-if="inMiniApp">Оплатить на сайте <ExternalLink :size="16" :stroke-width="2.25" /></template>
          <template v-else>Открыть за {{ rub(tariffs.levelNow) }}</template>
        </button>
        <button v-else-if="next && tariffs.levelFrom" class="btn btn-ghost mt-auto w-full" disabled data-test="level-wait">
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

        <!-- The argument for the year, as one picture: what twelve months of
             levels cost at the fastest pace the tariff allows, against the year. -->
        <div v-if="tariffs.yearSaves" class="saving" data-test="year-saving">
          <p class="font-extrabold">Выгоднее на {{ tariffs.yearSaves.percent }}%, чем брать уровень каждый месяц</p>
          <div class="bars">
            <span class="bar-label">Уровень в месяц, 12 месяцев</span>
            <span class="bar-amount" data-test="saving-monthly">{{ rub(tariffs.monthlyCost) }}</span>
            <span class="bar" aria-hidden="true"><i class="bar-fill bar-muted" style="width: 100%"></i></span>
            <span class="bar-label">Год</span>
            <span class="bar-amount" data-test="saving-year">{{ rub(tariffs.yearPrice) }}</span>
            <span class="bar" aria-hidden="true"><i class="bar-fill" :style="{ width: yearBar + '%' }"></i></span>
          </div>
          <p class="mt-2 text-sm" data-test="saving-rub">
            Экономия <b>{{ rub(tariffs.yearSaves.rub) }}</b>
          </p>
        </div>

        <ul class="perks">
          <li>
            <span class="ico"><LockOpen :size="15" :stroke-width="2.5" /></span>
            <span><b>Все платные уровни сразу.</b> Без очереди и ожидания.</span>
          </li>
          <li v-if="tariffs.yearBonus" data-test="year-bonus">
            <span class="ico ico-seed"><SeedIcon :size="14" /></span>
            <span>
              <b>+{{ seeds(tariffs.yearBonus) }}.</b> В подарок<template v-if="tariffs.yearBonusExtra"
                >, на {{ grouped(tariffs.yearBonusExtra) }} больше, чем за двенадцать месяцев по
                {{ grouped(tariffs.levelBonus) }}</template
              >.
            </span>
          </li>
          <li>
            <span class="ico"><Sparkles :size="15" :stroke-width="2.5" /></span>
            <span><b>Новые уровни включены,</b> пока идёт год.</span>
          </li>
          <li>
            <span class="ico"><Check :size="15" :stroke-width="2.5" /></span>
            <span><b>Один платёж.</b> Автопродления нет, само ничего не спишется.</span>
          </li>
        </ul>
        <button class="btn btn-primary mt-auto w-full" :disabled="!canPay" data-test="buy-year" @click="pay('year')">
          <template v-if="tariffs.paying === 'year'">Переходим к оплате…</template>
          <template v-else-if="inMiniApp">Оплатить на сайте <ExternalLink :size="16" :stroke-width="2.25" /></template>
          <template v-else>Открыть всё за {{ rub(tariffs.yearPrice) }}</template>
        </button>
      </section>
    </div>

    <p class="px-1 text-center text-xs text-[var(--muted)]" data-test="offer-note">
      Оплата проходит на странице банка, данные карты мы не получаем. Чек придёт на почту. Оплачивая, ты принимаешь
      условия
      <a href="/oferta" target="_blank" rel="noopener" class="font-semibold underline underline-offset-2" data-test="offer-link"
        >публичной оферты</a
      >.
    </p>

    <p class="px-1 text-center text-sm text-[var(--muted)]">
      Не хочется платить? Зёрнышки за уроки и задания тоже открывают уровни —
      <RouterLink to="/quests" class="font-semibold text-[var(--accent)] underline-offset-2 hover:underline"
        >как заработать</RouterLink
      >.
    </p>

    <PaymentThanksModal :order="thanks" :year-until="tariffs.yearUntil" @close="thanks = null" />
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
.perks b {
  font-weight: 800;
}
.ico {
  display: grid;
  flex: none;
  place-items: center;
  width: 1.65rem;
  height: 1.65rem;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
}
/* Gold is the currency's colour everywhere in the app, so the bonus wears it. */
.ico-seed {
  background: color-mix(in srgb, var(--gold) 22%, var(--card));
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
.saving {
  margin-top: 1rem;
  border-radius: 0.9rem;
  background: var(--card);
  padding: 0.85rem 0.95rem;
  font-size: 0.875rem;
  box-shadow: inset 0 0 0 1px var(--accent-soft);
}
.bars {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 0.15rem 0.75rem;
  margin-top: 0.7rem;
}
.bars > .bar {
  grid-column: 1 / -1;
  margin-bottom: 0.5rem;
}
.bar-label {
  font-size: 0.8125rem;
  color: var(--muted);
}
.bar-amount {
  font-size: 0.875rem;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}
.bar {
  display: block;
  height: 0.55rem;
  overflow: hidden;
  border-radius: 999px;
  background: var(--bg-soft);
}
.bar-fill {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--accent);
}
.bar-muted {
  background: var(--muted);
  opacity: 0.4;
}
</style>
