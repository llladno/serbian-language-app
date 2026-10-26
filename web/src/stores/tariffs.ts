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
import { discountPercent, looksLikeContact, monthlyYearCost, yearSaving } from '../lib/tariffs'
import type { LevelChoice, Order, TariffCode, TariffsInfo } from '../types'

export const useTariffsStore = defineStore('tariffs', () => {
  const session = useSessionStore()
  const info = ref<TariffsInfo | null>(null)
  const loading = ref(false)
  const paying = ref<TariffCode | null>(null)

  const enabled = computed(() => session.user?.features?.tariffs === true)
  const ready = computed(() => enabled.value && info.value !== null)

  // An account that came in through Telegram has no email, so the receipt needs
  // an email or phone typed in. It is kept on the device so the next purchase is one tap.
  const RECEIPT_KEY = 'tariffs.receiptContact'
  const needsReceiptContact = computed(() => enabled.value && !session.user?.email)
  const receiptContact = ref(readSaved(RECEIPT_KEY))
  const receiptContactValid = computed(() => !needsReceiptContact.value || looksLikeContact(receiptContact.value))
  function setReceiptContact(value: string) {
    receiptContact.value = value
    try {
      localStorage.setItem(RECEIPT_KEY, value.trim())
    } catch {
      // private mode or blocked storage: the field simply starts empty next time
    }
  }

  // The first level at the first-purchase price, the rest at the regular one; the server decides which.
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
  // Has a tariff: a running year, or a level paid for in rubles. A level bought
  // with зёрнышки is not one, and nor is a year that has run out.
  const hasTariff = computed(() => yearActive.value || hasPaidLevel.value)
  const paymentsReady = computed(() => info.value?.payments_ready ?? false)
  // A paid level is a credit: the learner chooses the level after paying. These are the levels
  // they can still choose and the paid orders waiting to be spent.
  const choices = computed<LevelChoice[]>(() => info.value?.level.choices ?? [])
  const credits = computed<number[]>(() => info.value?.level.credits ?? [])
  const hasCredit = computed(() => credits.value.length > 0)

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
      const order = needsReceiptContact.value
        ? await api.createOrder(tariff, receiptContact.value.trim())
        : await api.createOrder(tariff)
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

  // Spends a paid level on the level picked, then reads the quote again: the credit is gone and
  // the list of levels left to choose has changed.
  async function chooseLevel(orderId: number, phase: string): Promise<Order> {
    const order = await api.chooseLevel(orderId, phase)
    await load(true)
    return order
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
    info, loading, paying, enabled, ready, needsReceiptContact, receiptContact, receiptContactValid, setReceiptContact, levelNow, levelRegular, yearPrice, yearOld,
    levelDiscount, levelFrom, monthlyCost, yearSaves, levelBonus, yearBonus, yearBonusExtra, hasPaidLevel, hasTariff, yearUntil, yearActive, paymentsReady, choices, credits, hasCredit, load, pay, checkOrder, chooseLevel,
  }
})

function readSaved(key: string): string {
  try {
    return localStorage.getItem(key) ?? ''
  } catch {
    return ''
  }
}

export function payErrorMessage(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.message === 'payments_unavailable') return 'Оплата ещё не подключена'
    if (e.message === 'nothing_to_buy') return 'Все платные уровни уже открыты'
    if (e.message === 'level_pending_choice') return 'Сначала выберите уровень, который вы уже оплатили'
    if (e.message === 'level_cooldown') return 'Уровень за рубли можно открывать раз в месяц. Дата — на странице тарифов'
    if (e.message === 'receipt_contact_invalid') return 'Проверь почту или телефон для чека: в них, похоже, опечатка'
    if (e.message === 'payment_failed') return 'Платёжная система не ответила. Попробуйте ещё раз'
  }
  return 'Не получилось начать оплату. Попробуйте ещё раз'
}
