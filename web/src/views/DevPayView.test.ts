import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DevPayView from './DevPayView.vue'
import { api, ApiError } from '../api'
import type { DevOrder } from '../types'

enableAutoUnmount(afterEach)

const assign = vi.fn()
const pending: DevOrder = { id: 7, tariff: 'level', title: 'Уровень', amount: 199, status: 'pending' }

async function mountPage(search = '?order=7') {
  setActivePinia(createPinia())
  Object.defineProperty(window, 'location', {
    configurable: true,
    value: { href: 'http://localhost:5173/dev/pay' + search, assign },
  })
  const w = mount(DevPayView)
  await flushPromises()
  return w
}

beforeEach(() => assign.mockReset())
afterEach(() => vi.restoreAllMocks())

describe('DevPayView: the stand-in cash desk', () => {
  it('says what is being bought and for how much, and that no money moves', async () => {
    const info = vi.spyOn(api, 'devOrder').mockResolvedValue(pending)
    const w = await mountPage()
    expect(info).toHaveBeenCalledWith(7)
    const text = w.text().replace(/\s+/g, ' ')
    expect(text).toContain('Уровень')
    expect(text).toContain('199 ₽')
    expect(text).toMatch(/деньги не списываются/i)
    expect(w.find('[data-test="dev-pay"]').exists()).toBe(true)
    expect(w.find('[data-test="dev-fail"]').exists()).toBe(true)
  })

  it('"Оплатить" pays the order and returns to the tariffs screen the way a bank would', async () => {
    vi.spyOn(api, 'devOrder').mockResolvedValue(pending)
    const pay = vi.spyOn(api, 'devPay').mockResolvedValue({ id: 7, tariff: 'level', status: 'succeeded', choose: true })
    const w = await mountPage()
    await w.get('[data-test="dev-pay"]').trigger('click')
    await flushPromises()
    expect(pay).toHaveBeenCalledWith(7, 'paid')
    expect(assign).toHaveBeenCalledWith('/tariffs?order=7')
  })

  it('"Отказаться" fails the payment and returns the same way', async () => {
    vi.spyOn(api, 'devOrder').mockResolvedValue(pending)
    const pay = vi.spyOn(api, 'devPay').mockResolvedValue({ id: 7, tariff: 'level', status: 'canceled' })
    const w = await mountPage()
    await w.get('[data-test="dev-fail"]').trigger('click')
    await flushPromises()
    expect(pay).toHaveBeenCalledWith(7, 'failed')
    expect(assign).toHaveBeenCalledWith('/tariffs?order=7')
  })

  it('does not press twice while the first press is on its way', async () => {
    vi.spyOn(api, 'devOrder').mockResolvedValue(pending)
    const pay = vi.spyOn(api, 'devPay').mockImplementation(() => new Promise(() => {}))
    const w = await mountPage()
    await w.get('[data-test="dev-pay"]').trigger('click')
    await w.get('[data-test="dev-pay"]').trigger('click')
    await w.get('[data-test="dev-fail"]').trigger('click')
    expect(pay).toHaveBeenCalledTimes(1)
  })

  it('an order that is already settled offers only the way back', async () => {
    vi.spyOn(api, 'devOrder').mockResolvedValue({ ...pending, status: 'succeeded' })
    const w = await mountPage()
    expect(w.find('[data-test="dev-pay"]').exists()).toBe(false)
    expect(w.find('[data-test="dev-fail"]').exists()).toBe(false)
    await w.get('[data-test="dev-back"]').trigger('click')
    expect(assign).toHaveBeenCalledWith('/tariffs?order=7')
  })

  it('shows an error and keeps the buttons when the desk refuses', async () => {
    vi.spyOn(api, 'devOrder').mockResolvedValue(pending)
    vi.spyOn(api, 'devPay').mockRejectedValue(new ApiError(500, 'internal error'))
    const w = await mountPage()
    await w.get('[data-test="dev-pay"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="dev-error"]').exists()).toBe(true)
    expect(w.find('[data-test="dev-pay"]').attributes('disabled')).toBeUndefined()
    expect(assign).not.toHaveBeenCalled()
  })

  it('where the desk is off (or the order is not ours) it explains instead of showing buttons', async () => {
    vi.spyOn(api, 'devOrder').mockRejectedValue(new ApiError(404, 'not found'))
    const w = await mountPage()
    expect(w.find('[data-test="dev-unavailable"]').text()).toContain('PAYMENTS_DEV')
    expect(w.find('[data-test="dev-pay"]').exists()).toBe(false)
  })

  it('without an order number there is nothing to pay', async () => {
    const info = vi.spyOn(api, 'devOrder')
    const w = await mountPage('')
    expect(info).not.toHaveBeenCalled()
    expect(w.find('[data-test="dev-unavailable"]').exists()).toBe(true)
  })
})
