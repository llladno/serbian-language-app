// What the learner can buy for rubles, what it costs, and what they already
// have. One store so the tariffs screen, the "not enough зёрнышки" modal and the
// quests banner always quote the same price and agree on whether a year runs.
//
// Closed to most accounts: `enabled` follows the server's features flag, and
// until it is on nothing here talks to the server or shows anything.
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, ApiError } from '../api'
import { initData, isTelegram, openExternalLink } from '../telegram'
import { useSessionStore } from './session'
import { discountPercent, monthlyYearCost, yearSaving } from '../lib/tariffs'
import type { Order, TariffCode, TariffsInfo } from '../types'

export const useTariffsStore = defineStore('tariffs', () => {
  const session = useSessionStore()
  const info = ref<TariffsInfo | null>(null)
  const loading = ref(false)
  const paying = ref<TariffCode | null>(null)

  const enabled = computed(() => session.user?.features?.tariffs === true)
  const ready = computed(() => enabled.value && info.value !== null)

  // 249 for the first level, 299 after; the server decides which.
  const levelNow = computed(() => info.value?.level.price ?? 0)
  const levelRegular = computed(() => info.value?.level.regular_price ?? 0)
  const yearPrice = computed(() => info.value?.year.price ?? 0)
  const yearOld = computed(() => info.value?.year.old_price ?? 0)
  // "Скидка 17%": the first-purchase price against the regular one. 0 once the
  // learner has paid for a level (the price is then the regular one).
  const levelDiscount = computed(() => discountPercent(levelNow.value, levelRegular.value))
  // When the one-level-a-month limit lets the learner buy a level again, or null
  // when nothing holds them back.
  const levelFrom = computed(() => {
    const at = info.value?.level.available_from
    return at && new Date(at) > new Date() ? at : null
  })
  // What the year saves against buying a level every month for twelve months.
  const monthlyCost = computed(() => monthlyYearCost(levelNow.value, levelRegular.value))
  const yearSaves = computed(() => yearSaving(yearPrice.value, monthlyCost.value))
  // The зёрнышки that come with each tariff, once, when it is paid for.
  const levelBonus = computed(() => info.value?.level.bonus ?? 0)
  const yearBonus = computed(() => info.value?.year.bonus ?? 0)
  // How many more the year brings than twelve months of levels would (0 when it
  // brings no more): the extra the year's зёрнышки are worth on top of the saving.
  const yearBonusExtra = computed(() => Math.max(0, yearBonus.value - levelBonus.value * 12))
  const hasPaidLevel = computed(() => info.value?.has_paid_level ?? false)
  const yearUntil = computed(() => info.value?.year.until ?? null)
  const yearActive = computed(() => yearUntil.value !== null && new Date(yearUntil.value) > new Date())
  const paymentsReady = computed(() => info.value?.payments_ready ?? false)

  async function load(force = false) {
    if (!enabled.value || loading.value) return
    if (info.value && !force) return
    loading.value = true
    try {
      info.value = await api.tariffs()
    } catch {
      // The screens that depend on this simply stay hidden: a failed price
      // lookup must not break the quests or the course.
    } finally {
      loading.value = false
    }
  }

  // Starts a payment and takes the learner to the payment service's page for it.
  // Inside the Telegram Mini App that page opens in the phone's browser: digital
  // goods are not paid for inside Telegram (spec, "Inside Telegram").
  async function pay(tariff: TariffCode): Promise<void> {
    if (paying.value) return
    paying.value = tariff
    try {
      const order = await api.createOrder(tariff)
      if (!order.confirmation_url) throw new Error('no confirmation url')
      if (isTelegram() && initData()) {
        openExternalLink(order.confirmation_url)
      } else {
        window.location.assign(order.confirmation_url)
      }
    } finally {
      paying.value = null
    }
  }

  // Where a payment stands, after the learner comes back from the payment
  // service. The server asks the service itself, and settles the order if it was
  // paid.
  async function checkOrder(id: number): Promise<Order> {
    const order = await api.order(id)
    if (order.status === 'succeeded') await load(true)
    return order
  }

  return {
    info, loading, paying, enabled, ready, levelNow, levelRegular, yearPrice, yearOld,
    levelDiscount, levelFrom, monthlyCost, yearSaves, levelBonus, yearBonus, yearBonusExtra, hasPaidLevel, yearUntil, yearActive, paymentsReady, load, pay, checkOrder,
  }
})

export function payErrorMessage(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.message === 'payments_unavailable') return 'Оплата ещё не подключена'
    if (e.message === 'nothing_to_buy') return 'Все платные уровни уже открыты'
    if (e.message === 'level_cooldown') return 'Уровень за рубли можно открывать раз в месяц. Дата — на странице тарифов'
    if (e.message === 'payment_failed') return 'Платёжная система не ответила. Попробуй ещё раз'
  }
  return 'Не получилось начать оплату. Попробуй ещё раз'
}
