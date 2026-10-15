// What the learner has bought for rubles, and what it costs. One store so the
// tariffs screen, the "not enough зёрнышки" modal and the quests banner always
// quote the same price and agree on whether a year is running.
//
// Stand-in until GET /api/tariffs exists (spec 2026-10-14-tariffs-design.md):
// the prices are the agreed defaults and nothing has been paid. In dev,
// `/tariffs?demo=year` or `?demo=paid` fakes those two states so the screens
// can be looked at; the branch is dead code in a production build.
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { TARIFF_DEFAULTS, levelPrice } from '../lib/tariffs'

export const useTariffsStore = defineStore('tariffs', () => {
  const prices = ref(TARIFF_DEFAULTS)
  // The learner has paid for a level before: the 249 first-purchase price is gone.
  const hasPaidLevel = ref(false)
  // When the running year ends (ISO date), or null when there is none.
  const yearUntil = ref<string | null>(null)

  const levelNow = computed(() => levelPrice(prices.value.level, hasPaidLevel.value))
  const yearActive = computed(() => yearUntil.value !== null && new Date(yearUntil.value) > new Date())

  function demo(kind: string | null) {
    if (!import.meta.env.DEV) return
    if (kind === 'year') {
      const d = new Date()
      d.setDate(d.getDate() + 300)
      yearUntil.value = d.toISOString()
      hasPaidLevel.value = true
    } else if (kind === 'paid') {
      hasPaidLevel.value = true
    }
  }

  return { prices, hasPaidLevel, yearUntil, levelNow, yearActive, demo }
})
