// The learner's own wallet: balance, the currency's name, and the quest list
// behind /quests. One store because the header chip and the quests screen must
// never show two different balances — claiming a quest updates both at once.
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '../api'
import { pluralRu } from '../lib/plural'
import type { Quest } from '../types'

export const useWalletStore = defineStore('wallet', () => {
  const balance = ref(0)
  const streakDays = ref(0)
  const quests = ref<Quest[]>([])
  // The currency is named in the admin panel, so the app has no name for it
  // until /me/wallet answers. Until then amounts render as bare numbers
  // rather than guessing a word the admin may have changed.
  const nameOne = ref('')
  const nameFew = ref('')
  const nameMany = ref('')
  const loaded = ref(false)
  const questsLoaded = ref(false)

  // How many quests are finished and still unclaimed — the number the profile
  // teaser and the header dot are about.
  const claimable = computed(() => quests.value.filter((q) => q.done && !q.claimed).length)

  function currencyWord(n: number): string {
    return pluralRu(n, nameOne.value, nameFew.value, nameMany.value)
  }

  // "15 пёрышек", or just "15" before the first wallet response.
  function amount(n: number): string {
    const word = currencyWord(n)
    return word ? `${n} ${word}` : String(n)
  }

  async function refresh() {
    try {
      const w = await api.wallet()
      balance.value = w.balance
      streakDays.value = w.streak_days
      nameOne.value = w.currency_one
      nameFew.value = w.currency_few
      nameMany.value = w.currency_many
      loaded.value = true
    } catch {
      // Best-effort: the header chip just keeps the last known balance.
    }
  }

  async function loadQuests() {
    const res = await api.quests()
    quests.value = res.quests
    questsLoaded.value = true
  }

  // Claims one quest and returns the reward, so the caller can celebrate it.
  // The balance comes back from the server rather than being added up here:
  // the daily drip may have credited something since the last refresh.
  async function claim(id: number): Promise<number> {
    const res = await api.claimQuest(id)
    balance.value = res.balance
    const q = quests.value.find((x) => x.id === id)
    if (q) q.claimed = true
    return res.reward
  }

  return {
    balance,
    streakDays,
    quests,
    nameOne,
    nameFew,
    nameMany,
    loaded,
    questsLoaded,
    claimable,
    currencyWord,
    amount,
    refresh,
    loadQuests,
    claim,
  }
})
