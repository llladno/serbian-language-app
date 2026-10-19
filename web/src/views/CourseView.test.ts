import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CourseView from './CourseView.vue'
import { api, ApiError } from '../api'
import { useUnlockModal } from '../lib/unlockModal'
import { useSessionStore } from '../stores/session'
import { enableTariffs, sessionUser, tariffsInfo } from '../test/tariffsFixture'
import { useCourseStore } from '../stores/course'
import type { Course, LessonStatus } from '../types'

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}))

enableAutoUnmount(afterEach)

const PRODUCT = { id: 7, kind: 'phase_unlock', ref: '2', title: 'Уровень 2', description: '', price: 230, price_effective: 230, owned: 0 }

function course(over: { locked?: boolean; planned?: boolean } = {}): Course {
  return {
    title: 'Курс',
    phases: [
      { id: '1', title: 'Уровень 1', lessons: ['00'], locked: false },
      { id: '2', title: 'Уровень 2', lessons: ['16'], locked: over.locked ?? true, price: 230, price_effective: 230 },
    ],
    lessons: [
      { id: '00', title: 'Первый', subtitle: '', planned: false, status: 'not_started' },
      { id: '16', title: 'Второй', subtitle: '', planned: over.planned ?? false, status: 'not_started' },
    ],
  }
}

function wallet(balance: number) {
  return { balance, currency_one: 'зёрнышко', currency_few: 'зёрнышка', currency_many: 'зёрнышек', streak_days: 0 }
}

function mountView() {
  return mount(CourseView, {
    global: { stubs: { teleport: true, RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } } },
    attachTo: document.body,
  })
}

beforeEach(() => {
  localStorage.clear() // the folded levels are remembered there
  setActivePinia(createPinia())
  useUnlockModal().closeModal()
  vi.spyOn(api, 'course').mockResolvedValue(course())
  vi.spyOn(api, 'wallet').mockResolvedValue(wallet(100))
  vi.spyOn(api, 'shop').mockResolvedValue({ items: [PRODUCT] })
})
afterEach(() => {
  vi.restoreAllMocks()
})

describe('CourseView: levels for sale', () => {
  it('shows the price on a locked level and none on a free one', async () => {
    const w = mountView()
    await flushPromises()
    expect(w.find('[data-test="price-2"]').text()).toBe('230')
    expect(w.find('[data-test="price-1"]').exists()).toBe(false)
  })

  it('a free level still links to its lesson', async () => {
    const w = mountView()
    await flushPromises()
    expect(w.findAll('a').some((a) => a.attributes('href') === '/lesson/00')).toBe(true)
    expect(w.findAll('a').some((a) => a.attributes('href') === '/lesson/16')).toBe(false)
  })

  it('tapping a lesson of a level that is too expensive says how much is missing', async () => {
    const w = mountView()
    await flushPromises()
    await w.find('[data-test="locked-16"]').trigger('click')
    expect(w.find('[data-test="unlock-missing"]').text()).toContain('Не хватает 130')
    expect(w.find('[data-test="unlock-buy"]').exists()).toBe(false)
  })

  it('when short of seeds, the next level can be opened for rubles instead', async () => {
    vi.spyOn(api, 'tariffs').mockResolvedValue(tariffsInfo())
    const w = mountView()
    enableTariffs()
    await flushPromises()
    await w.find('[data-test="locked-16"]').trigger('click')
    await flushPromises()
    const pay = w.find('[data-test="unlock-pay"]')
    expect(pay.text()).toContain('Открыть сразу за 249')
    expect(pay.attributes('href')).toBe('/tariffs')
    expect(w.find('[data-test="unlock-order"]').exists()).toBe(false)
    // earning is still on offer, next to paying
    expect(w.find('[data-test="unlock-quests"]').attributes('href')).toBe('/quests')
  })

  it('says what comes with paying: зёрнышки for a level, more of them for the year', async () => {
    vi.spyOn(api, 'tariffs').mockResolvedValue(tariffsInfo())
    const w = mountView()
    enableTariffs()
    await flushPromises()
    await w.find('[data-test="locked-16"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="unlock-bonus"]').text().replace(/\s+/g, ' ')).toContain('И ещё 1 000')
  })

  it('names the discount where the first purchase is offered', async () => {
    vi.spyOn(api, 'tariffs').mockResolvedValue(tariffsInfo())
    const w = mountView()
    enableTariffs()
    await flushPromises()
    await w.find('[data-test="locked-16"]').trigger('click')
    await flushPromises()
    const text = w.find('[data-test="unlock-discount"]').text().replace(/\s+/g, ' ')
    expect(text).toContain('Скидка 17% на первую покупку')
    expect(text).toContain('249 ₽ вместо 299 ₽')
  })

  it('answers the next level with the year, and the date, while the monthly limit holds a level back', async () => {
    const from = new Date(Date.now() + 9 * 86_400_000).toISOString()
    const info = tariffsInfo({ has_paid_level: true, level: { price: 299, regular_price: 299, available_from: from, next: null } })
    vi.spyOn(api, 'tariffs').mockResolvedValue(info)
    const w = mountView()
    enableTariffs(info)
    await flushPromises()
    await w.find('[data-test="locked-16"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="unlock-cooldown"]').text()).toContain('раз в месяц')
    expect(w.find('[data-test="unlock-pay"]').text()).toContain('Все уровни')
    expect(w.find('[data-test="unlock-saving"]').text()).toContain('Выгода 72%')
    expect(w.find('[data-test="unlock-order"]').exists()).toBe(false)
  })

  // The gate: the same screen, the same shortfall, and no way to pay for an
  // account the server has not opened tariffs for. Earning stays the main way.
  it('offers no payment to an account that does not have tariffs', async () => {
    const tariffs = vi.spyOn(api, 'tariffs')
    const w = mountView()
    useSessionStore().user = sessionUser(false)
    await flushPromises()
    await w.find('[data-test="locked-16"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="unlock-missing"]').exists()).toBe(true)
    expect(w.find('[data-test="unlock-pay"]').exists()).toBe(false)
    expect(w.find('[data-test="unlock-quests"]').classes()).toContain('btn-primary')
    expect(tariffs).not.toHaveBeenCalled()
  })

  it('a level that is not the next one is answered with the year, and says why', async () => {
    const three: Course = {
      title: 'Курс',
      phases: [
        { id: '1', title: 'Уровень 1', lessons: ['00'], locked: false },
        { id: '2', title: 'Уровень 2', lessons: ['16'], locked: true, price: 230, price_effective: 230 },
        { id: '3', title: 'Уровень 3', lessons: ['30'], locked: true, price: 375, price_effective: 375 },
      ],
      lessons: ['00', '16', '30'].map((id) => ({ id, title: id, subtitle: '', planned: false, status: 'not_started' as const })),
    }
    vi.spyOn(api, 'course').mockResolvedValue(three)
    vi.spyOn(api, 'shop').mockResolvedValue({
      items: [PRODUCT, { ...PRODUCT, id: 8, ref: '3', title: 'Уровень 3', price: 375, price_effective: 375 }],
    })
    vi.spyOn(api, 'tariffs').mockResolvedValue(tariffsInfo())
    const w = mountView()
    enableTariffs()
    await flushPromises()
    await w.find('[data-test="locked-30"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="unlock-pay"]').text()).toContain('Все уровни')
    expect(w.find('[data-test="unlock-pay"]').text()).toContain('1\u00a0000')
    expect(w.find('[data-test="unlock-order"]').text()).toContain('Уровень 2')
  })

  it('offers to buy when the balance is enough, then reloads the course and closes', async () => {
    vi.spyOn(api, 'wallet').mockResolvedValue(wallet(500))
    const purchase = vi.spyOn(api, 'purchase').mockResolvedValue({ paid: 230, balance: 270, replayed: false })
    const w = mountView()
    await flushPromises()
    await w.find('[data-test="price-2"]').trigger('click')
    expect(w.find('[data-test="unlock-enough"]').text()).toContain('останется 270')

    vi.spyOn(api, 'course').mockResolvedValue(course({ locked: false }))
    await w.find('[data-test="unlock-buy"]').trigger('click')
    await flushPromises()

    expect(purchase).toHaveBeenCalledTimes(1)
    expect(purchase.mock.calls[0][0]).toBe(7)
    expect(purchase.mock.calls[0][1]).toMatch(/\S+/) // an idempotency key
    expect(useUnlockModal().phaseId.value).toBeNull()
    // the level is open now: its lesson is a link and the price is gone
    expect(w.findAll('a').some((a) => a.attributes('href') === '/lesson/16')).toBe(true)
    expect(w.find('[data-test="price-2"]').exists()).toBe(false)
  })

  it('falls back to "not enough" when the server says the balance moved', async () => {
    vi.spyOn(api, 'wallet').mockResolvedValueOnce(wallet(500)).mockResolvedValue(wallet(50))
    vi.spyOn(api, 'purchase').mockRejectedValue(new ApiError(409, 'insufficient_funds'))
    const w = mountView()
    await flushPromises()
    await w.find('[data-test="price-2"]').trigger('click')
    await w.find('[data-test="unlock-buy"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="unlock-missing"]').text()).toContain('Не хватает 180')
  })

  it('shows an error and keeps the modal open when the purchase fails', async () => {
    vi.spyOn(api, 'wallet').mockResolvedValue(wallet(500))
    vi.spyOn(api, 'purchase').mockRejectedValue(new ApiError(500, 'internal error'))
    const w = mountView()
    await flushPromises()
    await w.find('[data-test="price-2"]').trigger('click')
    await w.find('[data-test="unlock-buy"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="unlock-error"]').exists()).toBe(true)
    expect(useUnlockModal().phaseId.value).toBe('2')
  })

  it('a locked level that is all "скоро" has no price and cannot be tapped', async () => {
    vi.spyOn(api, 'course').mockResolvedValue(course({ planned: true }))
    const w = mountView()
    await flushPromises()
    expect(w.find('[data-test="price-2"]').exists()).toBe(false)
    expect(w.find('[data-test="locked-16"]').exists()).toBe(false)
    expect(w.findAll('a').some((a) => a.attributes('href') === '/lesson/16')).toBe(false)
  })
})

// Folding: a passed level folds by itself, any level can be folded by hand, and
// the learner's choice is remembered in this browser.
describe('CourseView: folding levels', () => {
  const lessonsOf = (done: boolean): Course['lessons'] => [
    { id: '00', title: 'Первый', subtitle: '', planned: false, status: done ? 'done' : 'in_progress' },
    { id: '01', title: 'Второй', subtitle: '', planned: false, status: done ? 'done' : 'not_started' },
    { id: '16', title: 'Третий', subtitle: '', planned: false, status: 'not_started' },
  ]
  const twoLevels = (level1Done: boolean): Course => ({
    title: 'Курс',
    phases: [
      { id: '1', title: 'Уровень 1', lessons: ['00', '01'], locked: false },
      { id: '2', title: 'Уровень 2', lessons: ['16'], locked: false },
    ],
    lessons: lessonsOf(level1Done),
  })
  const lessonLinks = (w: ReturnType<typeof mountView>) => w.findAll('a').map((a) => a.attributes('href'))

  it('folds a level the learner has passed, and shows it as passed', async () => {
    vi.spyOn(api, 'course').mockResolvedValue(twoLevels(true))
    const w = mountView()
    await flushPromises()
    expect(w.find('[data-test="passed-1"]').exists()).toBe(true)
    expect(w.get('[data-test="toggle-1"]').attributes('aria-expanded')).toBe('false')
    expect(lessonLinks(w)).not.toContain('/lesson/00')
    // the level in progress stays open
    expect(w.get('[data-test="toggle-2"]').attributes('aria-expanded')).toBe('true')
    expect(lessonLinks(w)).toContain('/lesson/16')
  })

  it('leaves a level open until all of it is done, and says how far the learner is when folded', async () => {
    vi.spyOn(api, 'course').mockResolvedValue(twoLevels(false))
    const w = mountView()
    await flushPromises()
    expect(w.get('[data-test="toggle-1"]').attributes('aria-expanded')).toBe('true')
    expect(w.find('[data-test="passed-1"]').exists()).toBe(false)
    await w.get('[data-test="toggle-1"]').trigger('click')
    expect(lessonLinks(w)).not.toContain('/lesson/00')
    expect(w.get('[data-test="progress-1"]').text()).toBe('0 из 2')
  })

  it('folds and unfolds a level by hand', async () => {
    vi.spyOn(api, 'course').mockResolvedValue(twoLevels(false))
    const w = mountView()
    await flushPromises()
    await w.get('[data-test="toggle-2"]').trigger('click')
    expect(w.get('[data-test="toggle-2"]').attributes('aria-expanded')).toBe('false')
    expect(lessonLinks(w)).not.toContain('/lesson/16')
    await w.get('[data-test="toggle-2"]').trigger('click')
    expect(lessonLinks(w)).toContain('/lesson/16')
  })

  it('remembers the choice in localStorage and keeps it on the next visit', async () => {
    vi.spyOn(api, 'course').mockResolvedValue(twoLevels(false))
    const first = mountView()
    await flushPromises()
    await first.get('[data-test="toggle-2"]').trigger('click')
    expect(localStorage.getItem('course.collapsed.2')).toBe('1')
    first.unmount()

    setActivePinia(createPinia())
    const second = mountView()
    await flushPromises()
    expect(second.get('[data-test="toggle-2"]').attributes('aria-expanded')).toBe('false')
  })

  it('a passed level the learner unfolded stays unfolded', async () => {
    vi.spyOn(api, 'course').mockResolvedValue(twoLevels(true))
    const first = mountView()
    await flushPromises()
    await first.get('[data-test="toggle-1"]').trigger('click')
    expect(localStorage.getItem('course.collapsed.1')).toBe('0')
    first.unmount()

    setActivePinia(createPinia())
    const second = mountView()
    await flushPromises()
    expect(second.get('[data-test="toggle-1"]').attributes('aria-expanded')).toBe('true')
    expect(lessonLinks(second)).toContain('/lesson/00')
  })

  it('folds and unfolds the whole course at once', async () => {
    vi.spyOn(api, 'course').mockResolvedValue(twoLevels(false))
    const w = mountView()
    await flushPromises()
    expect(w.get('[data-test="toggle-all"]').text()).toBe('Свернуть всё')
    await w.get('[data-test="toggle-all"]').trigger('click')
    expect(lessonLinks(w).some((h) => h?.startsWith('/lesson/'))).toBe(false)
    expect(w.get('[data-test="toggle-all"]').text()).toBe('Развернуть всё')
    expect(localStorage.getItem('course.collapsed.1')).toBe('1')
    await w.get('[data-test="toggle-all"]').trigger('click')
    expect(lessonLinks(w)).toContain('/lesson/16')
  })

  it('keeps the price of a level for sale in view while it is folded', async () => {
    const w = mountView()
    await flushPromises()
    await w.get('[data-test="toggle-2"]').trigger('click')
    expect(w.get('[data-test="price-2"]').text()).toContain('230')
  })
})

describe('CourseView and the learner’s script', () => {
  it('shows lesson titles and subtitles in Cyrillic for a Cyrillic reader', async () => {
    const c = course()
    c.lessons[0].title = 'Привет! (Zdravo!)'
    c.lessons[0].subtitle = 'особые буквы č ć'
    vi.spyOn(api, 'course').mockResolvedValue(c)
    useSessionStore().user = {
      id: 'u', name: 'Гриша', email: '', email_verified: true,
      telegram: { linked: false, username: '' }, script: 'cyrillic',
    } as never
    const w = mountView()
    await flushPromises()
    expect(w.text()).toContain('Привет! (Здраво!)')
    expect(w.text()).toContain('особые буквы ч ћ')
  })
})

describe('CourseView scrolls to where the learner left off', () => {
  // Two levels, A (10-12) and B (20-21), nothing locked. `statuses` says what is started or done.
  function progress(statuses: Record<string, LessonStatus>): Course {
    const ids = { A: ['10', '11', '12'], B: ['20', '21'] }
    return {
      title: 'Курс',
      phases: [
        { id: 'A', title: 'Уровень A', lessons: ids.A, locked: false },
        { id: 'B', title: 'Уровень B', lessons: ids.B, locked: false },
      ],
      lessons: [...ids.A, ...ids.B].map((id) => ({
        id, title: `Урок ${id}`, subtitle: '', planned: false, status: statuses[id] ?? 'not_started',
      })),
    }
  }

  let scrolled: string[]
  beforeEach(() => {
    scrolled = []
    Element.prototype.scrollIntoView = vi.fn(function (this: Element) {
      scrolled.push(this.getAttribute('data-lesson') ?? 'level ' + this.getAttribute('data-phase'))
    })
  })
  afterEach(() => {
    delete (Element.prototype as { scrollIntoView?: unknown }).scrollIntoView
  })

  async function open(statuses: Record<string, LessonStatus>) {
    vi.spyOn(api, 'course').mockResolvedValue(progress(statuses))
    const w = mountView()
    await flushPromises()
    await new Promise((r) => setTimeout(r, 70)) // the card may need a beat to appear
    return w
  }

  it('scrolls to the lesson that is started', async () => {
    await open({ '10': 'done', '11': 'in_progress' })
    expect(scrolled).toEqual(['11'])
  })

  it('scrolls to the furthest started lesson when several are', async () => {
    await open({ '10': 'done', '11': 'in_progress', '20': 'in_progress' })
    expect(scrolled).toEqual(['20'])
  })

  it('scrolls to the last finished lesson when none is started', async () => {
    await open({ '10': 'done', '11': 'done' })
    expect(scrolled).toEqual(['11'])
  })

  it('stays put for a learner who has not begun', async () => {
    await open({})
    expect(scrolled).toEqual([])
  })

  it('scrolls to the header of a finished level that is folded, and does not open it', async () => {
    // Level A is all done, so it is folded by default; the last lesson studied is in it.
    const w = await open({ '10': 'done', '11': 'done', '12': 'done' })
    expect(scrolled).toEqual(['level A'])
    expect(w.find('[data-test="toggle-A"]').attributes('aria-expanded')).toBe('false')
    expect(w.find('[data-lesson="12"]').exists()).toBe(false)
    expect(localStorage.getItem('course.collapsed.A')).toBeNull()
  })

  it('respects a level the learner folded by hand, and scrolls to its header', async () => {
    localStorage.setItem('course.collapsed.A', '1')
    const w = await open({ '10': 'in_progress' })
    expect(scrolled).toEqual(['level A'])
    expect(w.find('[data-test="toggle-A"]').attributes('aria-expanded')).toBe('false')
    expect(localStorage.getItem('course.collapsed.A')).toBe('1')
  })

  it('goes by what the server says now, not by a course remembered from an earlier visit', async () => {
    // The cache still says nothing is started; the learner has since started lesson 12.
    useCourseStore().course = progress({ '10': 'done' })
    const spy = vi.spyOn(api, 'course').mockResolvedValue(progress({ '10': 'done', '12': 'in_progress' }))
    mountView()
    await flushPromises()
    await new Promise((r) => setTimeout(r, 70))
    expect(spy).toHaveBeenCalledTimes(1)
    expect(scrolled).toEqual(['12'])
  })

  it('scrolls once per visit, not again when the course is read afresh', async () => {
    await open({ '11': 'in_progress' })
    useCourseStore().course = progress({ '11': 'done', '20': 'in_progress' })
    await flushPromises()
    await new Promise((r) => setTimeout(r, 70))
    expect(scrolled).toEqual(['11'])
  })
})
