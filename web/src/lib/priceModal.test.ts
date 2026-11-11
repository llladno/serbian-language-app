import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePriceModal, CLOSABLE_AFTER_MS } from './priceModal'
import { useCourseStore } from '../stores/course'
import { useSurveyModal } from './surveyModal'
import { enableTariffs } from '../test/tariffsFixture'
import { api } from '../api'
import type { Course, PriceChange, TariffsInfo } from '../types'

const DAY_KEY = 'srpski.pricemodal.day'
const day = 86_400_000

function announced(over: Partial<PriceChange> = {}): Partial<TariffsInfo> {
  const at = Date.parse('2026-11-27T23:00:00Z')
  return {
    price_change: {
      at: new Date(at).toISOString(),
      now: new Date(at - 10 * day).toISOString(),
      level_price: 399,
      year_price: 2000,
      ...over,
    },
  }
}

// A course in which `done` lessons are finished.
function courseWith(done: number): Course {
  return {
    title: 'Курс',
    phases: [{ id: '1', title: 'Уровень 1', lessons: ['a', 'b', 'c'] }],
    lessons: ['a', 'b', 'c'].map((id, i) => ({
      id,
      title: id,
      subtitle: '',
      planned: false,
      status: i < done ? ('done' as const) : ('not_started' as const),
    })),
  }
}

const SHOW_AFTER = 3200

function setUp(over: Partial<TariffsInfo> = announced(), done = 3) {
  setActivePinia(createPinia())
  useCourseStore().course = courseWith(done)
  return enableTariffs(over)
}

beforeEach(() => {
  vi.useFakeTimers()
  localStorage.removeItem(DAY_KEY)
  vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
})
afterEach(() => {
  const m = usePriceModal()
  vi.advanceTimersByTime(CLOSABLE_AFTER_MS)
  m.close()
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('the price window at the end of a lesson', () => {
  it('opens a few seconds after the lesson and cannot be closed for the first second', async () => {
    setUp()
    const m = usePriceModal()
    await m.offerAfterLesson()
    expect(m.open.value).toBe(false)
    vi.advanceTimersByTime(SHOW_AFTER)
    expect(m.open.value).toBe(true)
    expect(m.closable.value).toBe(false)
    m.close()
    expect(m.open.value).toBe(true)
    vi.advanceTimersByTime(CLOSABLE_AFTER_MS - 1)
    m.close()
    expect(m.open.value).toBe(true)
    vi.advanceTimersByTime(1)
    expect(m.closable.value).toBe(true)
    m.close()
    expect(m.open.value).toBe(false)
  })

  it('reports the showing, the cross and the button', async () => {
    setUp()
    const m = usePriceModal()
    await m.offerAfterLesson()
    vi.advanceTimersByTime(SHOW_AFTER)
    expect(api.trackTariffEvent).toHaveBeenCalledWith('view', 'price_modal')
    vi.advanceTimersByTime(CLOSABLE_AFTER_MS)
    m.close()
    expect(api.trackTariffEvent).toHaveBeenCalledWith('dismiss', 'price_modal')

    localStorage.removeItem(DAY_KEY)
    await m.offerAfterLesson()
    vi.advanceTimersByTime(SHOW_AFTER)
    m.go()
    expect(api.trackTariffEvent).toHaveBeenCalledWith('click', 'price_modal')
    expect(m.open.value).toBe(false)
  })

  it('comes once a day', async () => {
    setUp()
    const m = usePriceModal()
    await m.offerAfterLesson()
    vi.advanceTimersByTime(SHOW_AFTER + CLOSABLE_AFTER_MS)
    m.close()
    await m.offerAfterLesson()
    vi.advanceTimersByTime(SHOW_AFTER)
    expect(m.open.value).toBe(false)
  })

  it.each([
    ['no announcement', {}, 3],
    ['a running year', { ...announced(), year: { price: 1000, old_price: 2000, until: new Date(Date.now() + 100 * day).toISOString() } }, 3],
    ['a paid level', { ...announced(), has_paid_level: true }, 3],
    ['a paid level and the first lesson done only', { ...announced(), has_paid_level: true }, 1],
    ['only the first lesson done', announced(), 1],
  ])('does not open with %s', async (_name, over, done) => {
    setUp(over as Partial<TariffsInfo>, done as number)
    const m = usePriceModal()
    await m.offerAfterLesson()
    vi.advanceTimersByTime(SHOW_AFTER)
    expect(m.open.value).toBe(false)
  })

  it('leaves the quiet moment to the survey', async () => {
    setUp()
    const m = usePriceModal()
    await m.offerAfterLesson()
    useSurveyModal().open.value = true
    vi.advanceTimersByTime(SHOW_AFTER)
    expect(m.open.value).toBe(false)
    // and the day is not spent on a window that did not open
    expect(localStorage.getItem(DAY_KEY)).toBeNull()
    useSurveyModal().open.value = false
  })

  it("is shown to the owner's own account whatever else is true", async () => {
    setUp({ ...announced({ preview: true }), has_paid_level: true }, 1)
    localStorage.setItem(DAY_KEY, new Date().toISOString().slice(0, 10))
    const m = usePriceModal()
    await m.offerAfterLesson()
    vi.advanceTimersByTime(SHOW_AFTER)
    expect(m.open.value).toBe(true)
  })
})
