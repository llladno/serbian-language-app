<script setup lang="ts">
// Тарифы: two ways to pay rubles for the paid levels. Spec:
// docs/superpowers/specs/2026-10-14-tariffs-design.md.
//
// Design note: the currency screens use gold for money (зёрнышки). Rubles must
// not borrow it, or the two would read as the same thing - so prices here are
// plain ink, and the one thing allowed to be loud is the accent border on the
// year, the plan the spec wants people to look at first.
import { computed, onMounted, ref } from 'vue'
import { Check, ExternalLink } from 'lucide-vue-next'
import { useCourseStore } from '../stores/course'
import { useTariffsStore, payErrorMessage } from '../stores/tariffs'
import { useWalletStore } from '../stores/wallet'
import { useToasts } from '../lib/toasts'
import { isTelegram, initData } from '../telegram'
import { daysLeft, longDate, nextLockedLevel, perMonth, rub } from '../lib/tariffs'
import { pluralRu } from '../lib/plural'
import BotPrefsSection from '../components/BotPrefsSection.vue'
import type { TariffCode } from '../types'

const courseStore = useCourseStore()
const tariffs = useTariffsStore()
const wallet = useWalletStore()
const toasts = useToasts()

const next = computed(() => nextLockedLevel(courseStore.course))
const levelStruck = computed(() => (tariffs.levelNow < tariffs.levelRegular ? tariffs.levelRegular : 0))

// How long a running year has left, for the "my tariff" card.
const yearDays = computed(() => (tariffs.yearUntil ? daysLeft(tariffs.yearUntil) : 0))

// Telegram sells digital goods in Stars only, so inside the Mini App the same
// button hands off to the phone's browser instead of paying in place (spec,
// "Inside Telegram"). Same check as DonateCard: the script is loaded everywhere,
// but initData is only non-empty in a real Telegram session.
const inMiniApp = isTelegram() && !!initData()

const canPay = computed(() => tariffs.paymentsReady && !tariffs.paying)

async function pay(tariff: TariffCode) {
  if (!tariffs.paymentsReady) {
    toasts.push({ title: 'Оплата ещё не подключена', text: 'Пока уровни можно открыть за зёрнышки.' })
    return
  }
  try {
    await tariffs.pay(tariff)
  } catch (e) {
    toasts.push({ title: 'Оплата не началась', text: payErrorMessage(e) })
  }
}

// Coming back from ЮKassa (?order=ID): ask where the payment stands, say so, and
// refresh what the payment may have opened.
const returning = ref(false)

async function settleReturn(id: number, attempt = 0) {
  returning.value = true
  try {
    const order = await tariffs.checkOrder(id)
    if (order.status === 'succeeded') {
      await Promise.allSettled([courseStore.load(true), wallet.loadShop()])
      toasts.push({
        title: 'Оплата прошла',
        text: order.tariff === 'year' ? 'Все уровни открыты на год.' : 'Уровень открыт.',
      })
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

    <!-- The shop is ЮKassa's test shop: say so, and say how to pay in it. -->
    <section v-if="tariffs.testMode" class="card p-4 text-sm" data-test="test-mode">
      <p class="font-bold">Тестовая оплата</p>
      <p class="mt-1 text-[var(--muted)]">
        Деньги не списываются. На странице оплаты введи карту <span class="font-semibold text-[var(--fg)]">5555 5555 5555 4444</span>,
        любой срок в будущем и любой CVC.
      </p>
    </section>

    <section v-if="tariffs.ready && !tariffs.paymentsReady" class="card p-4 text-sm" data-test="not-connected">
      <p class="font-bold">Оплата ещё не подключена</p>
      <p class="mt-1 text-[var(--muted)]">Цены можно посмотреть, а кнопки заработают, когда подключим платёжную систему.</p>
    </section>

    <!-- A running year. There is no auto-renew, so there is nothing to cancel:
         the card says so, and the one thing a person can still switch off is the
         bot's reminder that the year is ending. -->
    <section v-if="tariffs.yearActive && tariffs.yearUntil" class="card p-5" data-test="my-tariff">
      <div class="head">
        <p class="eyebrow">Мой тариф</p>
        <span class="chip">Активен</span>
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

    <div v-else class="grid gap-4 sm:grid-cols-2">
      <!-- Уровень -->
      <section class="card flex flex-col p-5" data-test="tariff-level">
        <div class="head"><p class="eyebrow">Уровень</p></div>
        <p class="mt-1 font-extrabold" data-test="level-title">{{ next ? next.title : 'Следующий уровень' }}</p>
        <p class="price-row" data-test="level-price">
          <span class="price">{{ rub(tariffs.levelNow) }}</span>
          <span v-if="levelStruck" class="was">{{ rub(levelStruck) }}</span>
        </p>
        <p v-if="levelStruck" class="mt-0.5 text-xs font-semibold text-[var(--accent)]">Первая покупка</p>
        <ul class="perks">
          <li><Check :size="16" :stroke-width="2.5" /> Открывается навсегда</li>
          <li><Check :size="16" :stroke-width="2.5" /> Один платёж, без подписки</li>
          <li><Check :size="16" :stroke-width="2.5" /> Следующий уровень — когда захочешь</li>
        </ul>
        <!-- The tariff is always on show, but there is nothing to sell once every
             paid level is open: the button stays, switched off, rather than
             taking money for a level that does not exist. -->
        <button
          v-if="next"
          class="btn btn-ghost mt-auto w-full"
          :disabled="!canPay"
          data-test="buy-level"
          @click="pay('level')"
        >
          <template v-if="tariffs.paying === 'level'">Переходим к оплате…</template>
          <template v-else-if="inMiniApp">Оплатить на сайте <ExternalLink :size="16" :stroke-width="2.25" /></template>
          <template v-else>Открыть за {{ rub(tariffs.levelNow) }}</template>
        </button>
        <button v-else class="btn btn-ghost mt-auto w-full" disabled data-test="level-none">
          Все платные уровни открыты
        </button>
      </section>

      <!-- Год -->
      <section class="card year flex flex-col p-5" data-test="tariff-year">
        <div class="head">
          <p class="eyebrow">Год</p>
          <span class="chip">Ранняя цена</span>
        </div>
        <p class="mt-1 font-extrabold">Все уровни и всё новое</p>
        <p class="price-row" data-test="year-price">
          <span class="price">{{ rub(tariffs.yearPrice) }}</span>
          <span class="was">{{ rub(tariffs.yearOld) }}</span>
        </p>
        <p class="mt-0.5 text-xs text-[var(--muted)]">≈ {{ rub(perMonth(tariffs.yearPrice)) }} в месяц</p>
        <ul class="perks">
          <li><Check :size="16" :stroke-width="2.5" /> Все платные уровни сразу</li>
          <li><Check :size="16" :stroke-width="2.5" /> Новые уровни, которые выйдут за год</li>
          <li><Check :size="16" :stroke-width="2.5" /> Один платёж, без автопродления</li>
        </ul>
        <button class="btn btn-primary mt-auto w-full" :disabled="!canPay" data-test="buy-year" @click="pay('year')">
          <template v-if="tariffs.paying === 'year'">Переходим к оплате…</template>
          <template v-else-if="inMiniApp">Оплатить на сайте <ExternalLink :size="16" :stroke-width="2.25" /></template>
          <template v-else>Открыть всё за {{ rub(tariffs.yearPrice) }}</template>
        </button>
      </section>
    </div>

    <p class="px-1 text-center text-sm text-[var(--muted)]">
      Не хочется платить? Зёрнышки за уроки и задания тоже открывают уровни —
      <RouterLink to="/quests" class="font-semibold text-[var(--accent)] underline-offset-2 hover:underline"
        >как заработать</RouterLink
      >.
    </p>
  </div>
</template>

<style scoped>
/* One height for both cards' top row, so the plan names and prices line up
   even though only the year carries a chip. */
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
.eyebrow {
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--muted);
}
.price-row {
  display: flex;
  align-items: baseline;
  gap: 0.6rem;
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
.perks {
  margin: 1rem 0 1.25rem;
  display: grid;
  gap: 0.5rem;
  font-size: 0.875rem;
}
.perks li {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
}
.perks svg {
  flex: none;
  margin-top: 0.15rem;
  color: var(--accent);
}
.year {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}
.chip {
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
  padding: 0.1rem 0.6rem;
  font-size: 0.75rem;
  font-weight: 800;
}
</style>
