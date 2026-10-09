import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CourseView from './CourseView.vue'
import { api, ApiError } from '../api'
import { useUnlockModal } from '../lib/unlockModal'
import type { Course } from '../types'

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
  return mount(CourseView, { global: { stubs: { teleport: true } }, attachTo: document.body })
}

beforeEach(() => {
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
