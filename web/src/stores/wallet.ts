// The learner's own wallet: balance, the currency's name, and the quest list
// behind /quests. One store because the header chip and the quests screen must
// never show two different balances — claiming a quest updates both at once.
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '../api'
import { pluralRu } from '../lib/plural'
import { useToasts } from '../lib/toasts'
import { groupQuests } from '../lib/questGroups'
import type { Quest } from '../types'

// Levels pay out on their own and get the reward modal, so they are the one
// kind the toast stays quiet about.
const PHASE_KIND = 'phase_completed'

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

  // silent suppresses the "задание выполнено" toasts — for the quests screen
  // itself, where the card moving into «Можно забрать» is the notification.
  async function loadQuests(silent = false) {
    const res = await api.quests()
    const before = questsLoaded.value ? quests.value : null
    quests.value = res.quests
    questsLoaded.value = true
    if (before && !silent) announceFinished(before, res.quests)
  }

  // A quest usually finishes mid-lesson or mid-review, nowhere near the quests
  // screen, so finishing one says so where the learner actually is. Only the
  // transition is announced: a quest already waiting before this refresh has
  // been announced once already.
  //
  // What counts as finished is what the quests screen would offer to claim, not
  // every row that crossed its target: a ladder shows one rung, and a toast
  // naming a rung the screen is not showing would send the learner looking for
  // a card that is not there.
  function announceFinished(before: Quest[], after: Quest[]) {
    const waiting = new Set(groupQuests(before).ready.map((q) => q.id))
    const { push } = useToasts()
    for (const q of groupQuests(after).ready) {
      if (q.kind === PHASE_KIND || waiting.has(q.id)) continue
      push({ title: 'Задание выполнено', text: q.title, reward: q.reward, to: '/quests' })
    }
  }

  // Finishing the last lesson of a level finishes its quest, and making the
  // learner go and press a button for a reward they just earned by finishing a
  // level would be absurd — these pay themselves out. Returns what was paid,
  // for the caller to celebrate; null when nothing was due.
  async function claimFinishedPhases(): Promise<{ reward: number; title: string } | null> {
    await loadQuests()
    const due = quests.value.filter((q) => q.kind === PHASE_KIND && q.done && !q.claimed)
    let paid: { reward: number; title: string } | null = null
    for (const q of due) {
      paid = { reward: await claim(q.id), title: q.title }
    }
    return paid
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
    claimFinishedPhases,
  }
})
