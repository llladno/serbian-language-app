import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '../api'
import type { GramCheckResult, ReviewCard } from '../types'

export const useReviewStore = defineStore('review', () => {
  const queue = ref<ReviewCard[]>([])
  const index = ref(0)
  const sessionCount = ref(0)
  const tally = ref<[number, number, number, number]>([0, 0, 0, 0])
  // Starts true so the very first render (before onMounted's load() call
  // lands) shows the skeleton instead of a flash of "nothing due today" —
  // `current` is undefined until the first fetch resolves either way.
  const loading = ref(true)
  const error = ref<string | null>(null)

  const current = computed<ReviewCard | undefined>(() => queue.value[index.value])
  const remaining = computed(() => Math.max(0, queue.value.length - index.value))
  const total = computed(() => queue.value.length)

  // load() re-runs on every ReviewView mount (a fresh review session each
  // visit). The store is a singleton, so a revisit after the first one
  // already has a card on screen — only show the full skeleton when there's
  // truly nothing to show yet; otherwise keep the stale card visible and
  // swap it for the new queue once it lands (the card's own :key/`.pop`
  // handles the transition, see ReviewView).
  async function load() {
    if (queue.value.length === 0) loading.value = true
    error.value = null
    index.value = 0
    sessionCount.value = 0
    tally.value = [0, 0, 0, 0]
    try {
      queue.value = await api.reviewQueue()
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      loading.value = false
    }
  }

  async function grade(g: number) {
    const card = current.value
    if (!card) return
    await api.grade(card.card_id, g)
    sessionCount.value++
    tally.value[g]++
    if (g === 0) queue.value.push({ ...card })
    index.value++
  }

  // checkGram sends the typed answer for the current grammar card's picked
  // item and returns the check result — it does NOT advance the queue
  // itself, so the view can show correct/incorrect feedback first. Call
  // advanceGram once the learner taps past that feedback.
  async function checkGram(answer: string): Promise<GramCheckResult | null> {
    const card = current.value
    if (!card || card.kind !== 'gram' || card.item_index === undefined) return null
    return api.gradeGram(card.card_id, card.item_index, answer)
  }

  function advanceGram(ok: boolean) {
    const card = current.value
    if (!card) return
    const g = ok ? 2 : 0
    sessionCount.value++
    tally.value[g]++
    if (!ok) queue.value.push({ ...card })
    index.value++
  }

  return {
    queue,
    index,
    sessionCount,
    tally,
    loading,
    error,
    current,
    remaining,
    total,
    load,
    grade,
    checkGram,
    advanceGram,
  }
})
