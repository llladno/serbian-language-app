import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TariffsView from './TariffsView.vue'
import { api, ApiError } from '../api'
import { useCourseStore } from '../stores/course'
import { useToasts } from '../lib/toasts'
import { isTelegram, initData, openExternalLink } from '../telegram'
import { enableTariffs, tariffsInfo } from '../test/tariffsFixture'
import type { Course, TariffsInfo } from '../types'

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}))
vi.mock('../telegram', () => ({ isTelegram: vi.fn(), initData: vi.fn(), openExternalLink: vi.fn() }))

enableAutoUnmount(afterEach)

const assign = vi.fn()

function course(locked: { '4': boolean; '5': boolean } = { '4': true, '5': true }): Course {
  return {
    title: 'Курс',
    phases: [
      { id: '3', title: 'Уровень 3 — Связная речь', lessons: ['30'] },
      { id: '4', title: 'Уровень 4 — Падежи в жизни', lessons: ['42'], locked: locked['4'] },
      { id: '5', title: 'Уровень 5 — Жизнь на сербском', lessons: ['60'], locked: locked['5'] },
    ],
    lessons: ['30', '42', '60'].map((id) => ({ id, title: id, subtitle: '', planned: false, status: 'not_started' as const })),
  }
}

// What the learner sees once the prices have loaded. `search` is the address
// bar's query, for the visit back from ЮKassa.
async function mountView(over: Partial<TariffsInfo> = {}, c: Course = course(), search = '') {
  const info = tariffsInfo(over)
  vi.spyOn(api, 'tariffs').mockResolvedValue(info)
  setActivePinia(createPinia())
  useCourseStore().course = c
  enableTariffs(over)
  Object.defineProperty(window, 'location', {
    configurable: true,
    value: { href: 'http://localhost/tariffs' + search, assign },
  })
  const w = mount(TariffsView)
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.mocked(isTelegram).mockReturnValue(false)
  vi.mocked(initData).mockReturnValue('')
  vi.mocked(openExternalLink).mockReset()
  assign.mockReset()
  useToasts().items.value = []
  vi.spyOn(api, 'course').mockResolvedValue(course())
  vi.spyOn(api, 'shop').mockResolvedValue({ items: [] })
})
afterEach(() => {
  vi.restoreAllMocks()
  vi.useRealTimers()
})

describe('TariffsView: the two plans', () => {
  it('offers the next locked level at the first-purchase price with the regular price struck through', async () => {
    const w = await mountView()
    expect(w.get('[data-test="level-title"]').text()).toBe('Уровень 4 — Падежи в жизни')
    const price = w.get('[data-test="level-price"]').text()
    expect(price).toContain('249')
    expect(price).toContain('299')
    expect(w.get('[data-test="level-price"] .was').text()).toContain('299')
    expect(w.get('[data-test="buy-level"]').text()).toContain('249')
  })

  it('moves to the next level once level 4 is open', async () => {
    const w = await mountView({}, course({ '4': false, '5': true }))
    expect(w.get('[data-test="level-title"]').text()).toBe('Уровень 5 — Жизнь на сербском')
  })

  it('keeps the 249 tariff on show when every level is open, with the button switched off', async () => {
    const w = await mountView({}, course({ '4': false, '5': false }))
    expect(w.get('[data-test="level-price"]').text()).toContain('249')
    expect(w.find('[data-test="buy-level"]').exists()).toBe(false)
    const off = w.get('[data-test="level-none"]')
    expect(off.text()).toContain('Все платные уровни открыты')
    expect(off.attributes('disabled')).toBeDefined()
  })

  it('shows the year at 1 000 with 2 000 struck through, and the per-month figure', async () => {
    const w = await mountView()
    const year = w.get('[data-test="tariff-year"]')
    expect(year.get('[data-test="year-price"] .price').text()).toBe('1 000 ₽')
    expect(year.get('[data-test="year-price"] .was').text()).toBe('2 000 ₽')
    expect(year.text()).toContain('83')
  })
})

describe('TariffsView: the offer', () => {
  // The terms are shown where the money is asked for, and open in a new tab so
  // the learner does not lose the screen.
  it('says paying means accepting the public offer, and links to it', async () => {
    const w = await mountView()
    expect(w.get('[data-test="offer-note"]').text()).toContain('Оплачивая')
    const link = w.get('[data-test="offer-link"]')
    expect(link.attributes('href')).toBe('/oferta')
    expect(link.attributes('target')).toBe('_blank')
    expect(link.attributes('rel')).toContain('noopener')
  })
})

describe('TariffsView: paying', () => {
  it('starts an order and takes the buyer to ЮKassa', async () => {
    const create = vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 7, tariff: 'level', status: 'pending', confirmation_url: 'https://pay.test/confirm/7',
    })
    const w = await mountView()
    await w.get('[data-test="buy-level"]').trigger('click')
    await flushPromises()
    expect(create).toHaveBeenCalledWith('level')
    expect(assign).toHaveBeenCalledWith('https://pay.test/confirm/7')
  })

  it('the year button orders the year', async () => {
    const create = vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 8, tariff: 'year', status: 'pending', confirmation_url: 'https://pay.test/confirm/8',
    })
    const w = await mountView()
    await w.get('[data-test="buy-year"]').trigger('click')
    await flushPromises()
    expect(create).toHaveBeenCalledWith('year')
    expect(assign).toHaveBeenCalledWith('https://pay.test/confirm/8')
  })

  it('inside a real Telegram session the payment page opens in the phone\'s browser', async () => {
    vi.mocked(isTelegram).mockReturnValue(true)
    vi.mocked(initData).mockReturnValue('query_id=AA&hash=abc')
    vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 9, tariff: 'year', status: 'pending', confirmation_url: 'https://pay.test/confirm/9',
    })
    const w = await mountView()
    expect(w.get('[data-test="buy-year"]').text()).toContain('Оплатить на сайте')
    await w.get('[data-test="buy-year"]').trigger('click')
    await flushPromises()
    expect(openExternalLink).toHaveBeenCalledWith('https://pay.test/confirm/9')
    expect(assign).not.toHaveBeenCalled() // the Mini App itself stays where it is
  })

  it('treats Telegram\'s WebApp object without initData as a plain browser', async () => {
    vi.mocked(isTelegram).mockReturnValue(true)
    vi.mocked(initData).mockReturnValue('')
    const w = await mountView()
    expect(w.get('[data-test="buy-year"]').text()).toContain('Открыть всё за')
  })

  it('says so, and sends nothing, while payments are not connected', async () => {
    const create = vi.spyOn(api, 'createOrder')
    const w = await mountView({ payments_ready: false })
    expect(w.find('[data-test="not-connected"]').exists()).toBe(true)
    expect(w.get('[data-test="buy-year"]').attributes('disabled')).toBeDefined()
    await w.get('[data-test="buy-year"]').trigger('click')
    expect(create).not.toHaveBeenCalled()
  })

  it('tells the buyer when the order could not be started', async () => {
    vi.spyOn(api, 'createOrder').mockRejectedValue(new ApiError(502, 'payment_failed'))
    const w = await mountView()
    await w.get('[data-test="buy-level"]').trigger('click')
    await flushPromises()
    expect(useToasts().items.value.map((t) => t.title)).toEqual(['Оплата не началась'])
    expect(useToasts().items.value[0].text).toContain('Платёжная система не ответила')
    expect(assign).not.toHaveBeenCalled()
  })
})

describe('TariffsView: coming back from ЮKassa', () => {
  it('a paid order opens the level, says so and refreshes the course', async () => {
    const order = vi.spyOn(api, 'order').mockResolvedValue({ id: 7, tariff: 'level', status: 'succeeded' })
    const reload = vi.spyOn(api, 'course').mockResolvedValue(course({ '4': false, '5': true }))
    await mountView({}, course(), '?order=7')
    expect(order).toHaveBeenCalledWith(7)
    expect(reload).toHaveBeenCalled() // the course is fetched again: level 4 is open now
    const toast = useToasts().items.value.map((t) => `${t.title}: ${t.text}`)
    expect(toast).toEqual(['Оплата прошла: Уровень открыт.'])
  })

  it('a paid year says that every level is open', async () => {
    vi.spyOn(api, 'order').mockResolvedValue({ id: 8, tariff: 'year', status: 'succeeded' })
    await mountView({}, course(), '?order=8')
    expect(useToasts().items.value[0].text).toContain('Все уровни открыты на год')
  })

  it('a canceled order says nothing was charged', async () => {
    vi.spyOn(api, 'order').mockResolvedValue({ id: 7, tariff: 'level', status: 'canceled' })
    await mountView({}, course(), '?order=7')
    const t = useToasts().items.value[0]
    expect(t.title).toBe('Оплата не прошла')
    expect(t.text).toContain('Деньги не списались')
  })

  it('a payment the bank has not confirmed yet is asked about again', async () => {
    vi.useFakeTimers()
    const order = vi
      .spyOn(api, 'order')
      .mockResolvedValueOnce({ id: 7, tariff: 'level', status: 'pending' })
      .mockResolvedValue({ id: 7, tariff: 'level', status: 'succeeded' })
    vi.spyOn(api, 'tariffs').mockResolvedValue(tariffsInfo())
    setActivePinia(createPinia())
    useCourseStore().course = course()
    enableTariffs()
    Object.defineProperty(window, 'location', { configurable: true, value: { href: 'http://localhost/tariffs?order=7', assign } })
    mount(TariffsView)
    await vi.advanceTimersByTimeAsync(0)
    expect(order).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(3100)
    expect(order).toHaveBeenCalledTimes(2)
    expect(useToasts().items.value.map((t) => t.title)).toContain('Оплата прошла')
  })

  it('without an order in the address nothing is announced', async () => {
    const order = vi.spyOn(api, 'order')
    await mountView()
    expect(order).not.toHaveBeenCalled()
    expect(useToasts().items.value).toEqual([])
  })
})

describe('TariffsView: a running year', () => {
  const running = (days: number): Partial<TariffsInfo> => ({
    has_paid_level: true,
    year: { price: 1000, old_price: 2000, until: new Date(Date.now() + days * 86_400_000).toISOString() },
  })

  it('shows the plan, when it ends and how long is left', async () => {
    const w = await mountView(running(100))
    expect(w.get('[data-test="my-tariff"]').text()).toContain('Год')
    expect(w.get('[data-test="my-tariff-until"]').text()).toMatch(/осталось 10[01] дн/)
  })

  it('says there is no auto-renew, so there is nothing to cancel', async () => {
    const w = await mountView(running(100))
    expect(w.get('[data-test="no-autorenew"]').text()).toContain('Автопродления нет')
    expect(w.get('[data-test="no-autorenew"]').text()).toContain('отписываться не нужно')
  })

  it('stops selling what the learner already has, and offers to extend instead', async () => {
    const w = await mountView(running(100))
    expect(w.find('[data-test="tariff-level"]').exists()).toBe(false)
    expect(w.find('[data-test="tariff-year"]').exists()).toBe(false)
    expect(w.get('[data-test="extend-year"]').text()).toContain('Продлить')
  })

  it('extending orders another year', async () => {
    const create = vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 10, tariff: 'year', status: 'pending', confirmation_url: 'https://pay.test/confirm/10',
    })
    const w = await mountView(running(100))
    await w.get('[data-test="extend-year"]').trigger('click')
    await flushPromises()
    expect(create).toHaveBeenCalledWith('year')
  })

  it('an ended year is just the two plans again', async () => {
    const w = await mountView(running(-3))
    expect(w.find('[data-test="my-tariff"]').exists()).toBe(false)
    expect(w.find('[data-test="tariff-year"]').exists()).toBe(true)
  })
})
