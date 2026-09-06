import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '../api'
import type { ReviewCard } from '../types'

export const useReviewStore = defineStore('review', () => {
  const queue = ref<ReviewCard[]>([])
  const index = ref(0)
  const sessionCount = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const current = computed<ReviewCard | undefined>(() => queue.value[index.value])
  const remaining = computed(() => Math.max(0, queue.value.length - index.value))

  async function load() {
    loading.value = true
    error.value = null
    index.value = 0
    sessionCount.value = 0
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
    if (g === 0) queue.value.push({ ...card })
    index.value++
  }

  return { queue, index, sessionCount, loading, error, current, remaining, load, grade }
})
