import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { mount, enableAutoUnmount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import PriceChangeModal from './PriceChangeModal.vue'
import { usePriceModal, CLOSABLE_AFTER_MS } from '../lib/priceModal'
import { useCourseStore } from '../stores/course'
import { useTariffsStore } from '../stores/tariffs'
import { enableTariffs } from '../test/tariffsFixture'
import { api } from '../api'

enableAutoUnmount(afterEach)
const stubs = { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } }
const at = Date.parse('2026-11-27T23:00:00Z')

beforeEach(() => {
  vi.useFakeTimers()
  localStorage.clear()
  vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
  setActivePinia(createPinia())
  useCourseStore().course = {
    title: 'Курс',
    phases: [],
    lessons: ['a', 'b'].map((id) => ({ id, title: id, subtitle: '', planned: false, status: 'done' as const })),
  }
  enableTariffs({
    price_change: { at: new Date(at).toISOString(), now: new Date(at - 10 * 86_400_000).toISOString(), level_price: 399, year_price: 2000 },
  })
})
afterEach(() => {
  vi.advanceTimersByTime(CLOSABLE_AFTER_MS)
  usePriceModal().close()
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('PriceChangeModal', () => {
  it('shows the poster, animated, with a button to the tariffs and a cross that comes after a second', async () => {
    const m = usePriceModal()
    mount(PriceChangeModal, { global: { stubs }, attachTo: document.body })
    await m.offerAfterLesson()
    vi.advanceTimersByTime(3200)
    await flushPromises()
    const poster = document.body.querySelector('[data-test="price-poster"]')
    expect(poster).not.toBeNull()
    expect(poster!.classList.contains('enter')).toBe(true)
    expect(document.body.querySelector('[data-test="price-poster-go"]')!.getAttribute('href')).toBe('/tariffs')
    expect(document.body.querySelector('[data-test="price-modal-close"]')).toBeNull()

    vi.advanceTimersByTime(CLOSABLE_AFTER_MS)
    await flushPromises()
    const cross = document.body.querySelector<HTMLButtonElement>('[data-test="price-modal-close"]')
    expect(cross).not.toBeNull()
    cross!.click()
    expect(m.open.value).toBe(false)
  })

  it('does not close on Escape before the second is up, and does after', async () => {
    const m = usePriceModal()
    mount(PriceChangeModal, { global: { stubs }, attachTo: document.body })
    await m.offerAfterLesson()
    vi.advanceTimersByTime(3200)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(m.open.value).toBe(true)
    vi.advanceTimersByTime(CLOSABLE_AFTER_MS)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(m.open.value).toBe(false)
  })
})

describe('PriceChangeModal when the prices change', () => {
  it('goes by itself when the announcement ends while it is open', async () => {
    const m = usePriceModal()
    mount(PriceChangeModal, { global: { stubs }, attachTo: document.body })
    await m.offerAfterLesson()
    vi.advanceTimersByTime(3200)
    await flushPromises()
    expect(m.open.value).toBe(true)
    // the quote is read again after the moment and has no announcement any more
    const tariffs = useTariffsStore()
    tariffs.info = { ...tariffs.info!, price_change: undefined }
    await flushPromises()
    expect(m.open.value).toBe(false)
  })
})
