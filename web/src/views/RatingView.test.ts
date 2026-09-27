import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import RatingView from './RatingView.vue'
import { api } from '../api'
import type { LeaderboardMe, LeaderboardPage, LeaderRow } from '../types'

function makeRows(count: number, offset = 0): LeaderRow[] {
  return Array.from({ length: count }, (_, i) => ({
    name: `user-${offset + i}`,
    lessons_done: 0,
    lessons_total: 59,
    cards_known: 0,
    total_cards: 0,
    streak_days: 0,
    last_active: '',
    reviewed_today: 0,
  }))
}

// jsdom has no IntersectionObserver - stub one that tests can trigger by
// hand to simulate the sentinel scrolling into view.
class FakeIntersectionObserver {
  static instances: FakeIntersectionObserver[] = []
  callback: IntersectionObserverCallback
  constructor(callback: IntersectionObserverCallback) {
    this.callback = callback
    FakeIntersectionObserver.instances.push(this)
  }
  observe() {}
  unobserve() {}
  disconnect() {}
  trigger() {
    this.callback([{ isIntersecting: true } as IntersectionObserverEntry], this as unknown as IntersectionObserver)
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
  FakeIntersectionObserver.instances = []
  vi.stubGlobal('IntersectionObserver', FakeIntersectionObserver)
  // "your rank" is fetched independently of the list - default it to a
  // failure so it silently stays hidden and doesn't add an extra .card to
  // the counts the list tests below assert on. Tests that care about it
  // override this mock explicitly.
  vi.spyOn(api, 'leaderboardMe').mockRejectedValue(new Error('no me'))
})
afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('RatingView', () => {
  it('shows an error message when the leaderboard fails to load', async () => {
    vi.spyOn(api, 'leaderboard').mockRejectedValue(new Error('boom'))
    const w = mount(RatingView)
    await flushPromises()
    expect(w.text()).toContain('boom')
  })

  it('shows the empty state when nobody is on the leaderboard', async () => {
    vi.spyOn(api, 'leaderboard').mockResolvedValue({ rows: [], has_more: false })
    const w = mount(RatingView)
    await flushPromises()
    expect(w.text()).toContain('Пока никого')
  })

  it('requests only the first page on mount, not the whole leaderboard', async () => {
    const spy = vi.spyOn(api, 'leaderboard').mockResolvedValue({ rows: makeRows(30), has_more: true })
    const w = mount(RatingView)
    await flushPromises()

    expect(spy).toHaveBeenCalledTimes(1)
    expect(spy).toHaveBeenCalledWith({ limit: 30, offset: 0 })
    expect(w.findAll('.card').length).toBe(30)
    expect(w.text()).toContain('user-0')
    expect(w.text()).toContain('user-29')
  })

  it('loads the next page when the sentinel scrolls into view', async () => {
    const spy = vi
      .spyOn(api, 'leaderboard')
      .mockImplementationOnce(async () => ({ rows: makeRows(30, 0), has_more: true }) satisfies LeaderboardPage)
      .mockImplementationOnce(async () => ({ rows: makeRows(15, 30), has_more: false }) satisfies LeaderboardPage)
    const w = mount(RatingView)
    await flushPromises()

    expect(FakeIntersectionObserver.instances).toHaveLength(1)
    FakeIntersectionObserver.instances[0].trigger()
    await flushPromises()

    expect(spy).toHaveBeenCalledTimes(2)
    expect(spy).toHaveBeenNthCalledWith(2, { limit: 30, offset: 30 })
    expect(w.findAll('.card').length).toBe(45)
    expect(w.text()).toContain('user-44')
  })

  it('keeps loading consecutive pages while the sentinel stays in view', async () => {
    // Regression test: IntersectionObserver only fires on a visibility
    // *change*. If the user jumps straight to the bottom (or the viewport
    // fits several pages at once), the sentinel can stay continuously
    // intersecting without ever re-firing - a naive "load on intersect"
    // handler would silently stop after one page even though more remain.
    const spy = vi
      .spyOn(api, 'leaderboard')
      .mockImplementationOnce(async () => ({ rows: makeRows(30, 0), has_more: true }) satisfies LeaderboardPage)
      .mockImplementationOnce(async () => ({ rows: makeRows(30, 30), has_more: true }) satisfies LeaderboardPage)
      .mockImplementationOnce(async () => ({ rows: makeRows(10, 60), has_more: false }) satisfies LeaderboardPage)
    const w = mount(RatingView)
    await flushPromises()

    FakeIntersectionObserver.instances[0].trigger() // fires exactly once
    await flushPromises()

    expect(spy).toHaveBeenCalledTimes(3)
    expect(w.findAll('.card').length).toBe(70)
    expect(w.text()).toContain('user-69')
  })

  it('stops requesting once the last page is loaded', async () => {
    const spy = vi.spyOn(api, 'leaderboard').mockResolvedValue({ rows: makeRows(5), has_more: false })
    const w = mount(RatingView)
    await flushPromises()

    FakeIntersectionObserver.instances[0].trigger()
    await flushPromises()
    FakeIntersectionObserver.instances[0].trigger()
    await flushPromises()

    expect(spy).toHaveBeenCalledTimes(1) // has_more was false, no further calls
    expect(w.findAll('.card').length).toBe(5)
  })

  it('pins the current user above the list, independently of the paginated page', async () => {
    // The account might be hundreds of rows down - far past the first page
    // - so /leaderboard/me must not depend on /leaderboard's result at all.
    vi.spyOn(api, 'leaderboard').mockResolvedValue({ rows: makeRows(30), has_more: true })
    vi.mocked(api.leaderboardMe).mockResolvedValue({
      rank: 605,
      row: {
        name: 'RatingTester',
        lessons_done: 3,
        lessons_total: 59,
        cards_known: 12,
        total_cards: 12,
        streak_days: 2,
        last_active: '',
        reviewed_today: 0,
      },
    } satisfies LeaderboardMe)

    const w = mount(RatingView)
    await flushPromises()

    expect(w.text()).toContain('Твой результат')
    expect(w.text()).toContain('605')
    expect(w.text()).toContain('RatingTester')
    // still shows the full paginated list below, unaffected
    expect(w.text()).toContain('Все участники')
    expect(w.text()).toContain('user-0')
    expect(w.findAll('.card').length).toBe(31) // 30 list rows + the pinned "you" card
  })

  it('hides the pinned section entirely if /leaderboard/me fails', async () => {
    vi.spyOn(api, 'leaderboard').mockResolvedValue({ rows: makeRows(5), has_more: false })
    // beforeEach already mocks leaderboardMe to reject
    const w = mount(RatingView)
    await flushPromises()

    expect(w.text()).not.toContain('Твой результат')
    expect(w.findAll('.card').length).toBe(5)
  })
})
