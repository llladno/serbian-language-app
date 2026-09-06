import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useReviewStore } from './review'
import { api } from '../api'
import type { ReviewCard } from '../types'

function card(id: string): ReviewCard {
  return { card_id: id, kind: 'vocab', front: id, back: id, state: 'new' }
}

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('useReviewStore', () => {
  it('re-queues an "again" card to the end and advances otherwise', async () => {
    const store = useReviewStore()
    store.$patch({ queue: [card('a'), card('b')], index: 0 })
    vi.spyOn(api, 'grade').mockResolvedValue({ due: '', interval_days: 1, state: 'learning' })

    await store.grade(0) // Again on 'a' — re-queued to the end, index advances
    expect(store.current?.card_id).toBe('b')
    expect(store.queue.map((c) => c.card_id)).toEqual(['a', 'b', 'a'])
    expect(store.remaining).toBe(2)

    await store.grade(2) // Good on 'b'
    expect(store.current?.card_id).toBe('a')
    expect(store.sessionCount).toBe(2)
  })

  it('remaining counts down to zero', async () => {
    const store = useReviewStore()
    store.$patch({ queue: [card('a')], index: 0 })
    vi.spyOn(api, 'grade').mockResolvedValue({ due: '', interval_days: 1, state: 'review' })
    expect(store.remaining).toBe(1)
    await store.grade(3)
    expect(store.remaining).toBe(0)
    expect(store.current).toBeUndefined()
  })
})
