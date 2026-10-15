import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TariffsView from './TariffsView.vue'
import { useCourseStore } from '../stores/course'
import { useTariffsStore } from '../stores/tariffs'
import { useSessionStore } from '../stores/session'
import { api } from '../api'
import { useToasts } from '../lib/toasts'
import { isTelegram, initData } from '../telegram'
import type { Course } from '../types'

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}))
vi.mock('../telegram', () => ({ isTelegram: vi.fn(), initData: vi.fn() }))

enableAutoUnmount(afterEach)

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

function mountWith(c: Course, opts: { linked?: boolean; yearDays?: number } = {}) {
  setActivePinia(createPinia())
  useCourseStore().course = c
  if (opts.linked) {
    useSessionStore().user = { id: 'u', name: 'Г', email: 'g@e.com', email_verified: true, telegram: { linked: true, username: 'g' } }
  }
  if (opts.yearDays !== undefined) {
    useTariffsStore().yearUntil = new Date(Date.now() + opts.yearDays * 86_400_000).toISOString()
  }
  return mount(TariffsView)
}

beforeEach(() => {
  vi.mocked(isTelegram).mockReturnValue(false)
  vi.mocked(initData).mockReturnValue('')
  useToasts().items.value = []
})

describe('TariffsView', () => {
  it('offers the next locked level at the first-purchase price with the regular price struck through', async () => {
    const w = mountWith(course())
    await flushPromises()

    expect(w.get('[data-test="level-title"]').text()).toBe('Уровень 4 — Падежи в жизни')
    const price = w.get('[data-test="level-price"]').text()
    expect(price).toContain('249')
    expect(price).toContain('299')
    expect(w.get('[data-test="level-price"] .was').text()).toContain('299')
    expect(w.get('[data-test="buy-level"]').text()).toContain('249')
  })

  it('moves to the next level once level 4 is open', async () => {
    const w = mountWith(course({ '4': false, '5': true }))
    await flushPromises()
    expect(w.get('[data-test="level-title"]').text()).toBe('Уровень 5 — Жизнь на сербском')
  })

  it('keeps the 249 tariff on show when every level is open, with the button switched off', async () => {
    const w = mountWith(course({ '4': false, '5': false }))
    await flushPromises()
    expect(w.get('[data-test="level-price"]').text()).toContain('249')
    expect(w.find('[data-test="buy-level"]').exists()).toBe(false)
    const off = w.get('[data-test="level-none"]')
    expect(off.text()).toContain('Все платные уровни открыты')
    expect(off.attributes('disabled')).toBeDefined()
  })

  it('shows the year at 1 000 with 2 000 struck through, and the per-month figure', async () => {
    const w = mountWith(course())
    await flushPromises()
    const year = w.get('[data-test="tariff-year"]')
    expect(year.get('[data-test="year-price"] .price').text()).toBe('1 000 ₽')
    expect(year.get('[data-test="year-price"] .was').text()).toBe('2 000 ₽')
    expect(year.text()).toContain('83')
  })

  it('says payments are not connected yet instead of silently doing nothing', async () => {
    const w = mountWith(course())
    await flushPromises()
    await w.get('[data-test="buy-year"]').trigger('click')
    expect(useToasts().items.value.map((t) => t.title)).toEqual(['Оплата скоро появится'])
  })

  it('inside a real Telegram session the buttons hand off to the website', async () => {
    vi.mocked(isTelegram).mockReturnValue(true)
    vi.mocked(initData).mockReturnValue('query_id=AA&hash=abc')
    const w = mountWith(course())
    await flushPromises()
    expect(w.get('[data-test="buy-level"]').text()).toContain('Оплатить на сайте')
    expect(w.get('[data-test="buy-year"]').text()).toContain('Оплатить на сайте')
  })

  it('treats Telegram\'s WebApp object without initData as a plain browser', async () => {
    vi.mocked(isTelegram).mockReturnValue(true)
    vi.mocked(initData).mockReturnValue('')
    const w = mountWith(course())
    await flushPromises()
    expect(w.get('[data-test="buy-year"]').text()).toContain('Открыть всё за')
  })
})

describe('TariffsView: a running year', () => {
  const withYear = (days: number, linked = false) => mountWith(course(), { yearDays: days, linked })

  it('shows the plan, when it ends and how long is left', async () => {
    const w = withYear(100)
    await flushPromises()
    expect(w.get('[data-test="my-tariff"]').text()).toContain('Год')
    expect(w.get('[data-test="my-tariff-until"]').text()).toMatch(/осталось 10[01] дн/)
  })

  it('says there is no auto-renew, so there is nothing to cancel', async () => {
    const w = withYear(100)
    await flushPromises()
    expect(w.get('[data-test="no-autorenew"]').text()).toContain('Автопродления нет')
    expect(w.get('[data-test="no-autorenew"]').text()).toContain('отписываться не нужно')
  })

  it('stops selling what the learner already has, and offers to extend instead', async () => {
    const w = withYear(100)
    await flushPromises()
    expect(w.find('[data-test="tariff-level"]').exists()).toBe(false)
    expect(w.find('[data-test="tariff-year"]').exists()).toBe(false)
    expect(w.get('[data-test="extend-year"]').text()).toContain('Продлить')
  })

  it('lets a Telegram-linked learner switch the "year is ending" reminder off', async () => {
    vi.spyOn(api, 'botPrefs').mockResolvedValue({ learning: true, billing: true, news: false })
    const save = vi.spyOn(api, 'setBotPrefs').mockResolvedValue({ learning: true, billing: false, news: false })
    const w = withYear(100, true)
    await flushPromises()
    await w.get('[data-test="pref-billing"]').setValue(false)
    await flushPromises()
    expect(save).toHaveBeenCalledWith({ billing: false })
    expect(w.find('[data-test="pref-learning"]').exists()).toBe(false) // only the tariff switch is here
  })

  it('an ended year is just the two plans again', async () => {
    const w = withYear(-3)
    await flushPromises()
    expect(w.find('[data-test="my-tariff"]').exists()).toBe(false)
    expect(w.find('[data-test="tariff-year"]').exists()).toBe(true)
  })
})
