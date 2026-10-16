import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TariffsBanner from './TariffsBanner.vue'
import { api } from '../api'
import { useWalletStore } from '../stores/wallet'
import { useSessionStore } from '../stores/session'
import { enableTariffs, sessionUser, tariffsInfo } from '../test/tariffsFixture'
import type { ShopItem } from '../types'

enableAutoUnmount(afterEach)

const stubs = { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } }

function level(over: Partial<ShopItem> = {}): ShopItem {
  return { id: 1, kind: 'phase_unlock', ref: '4', title: 'Уровень 4 — Падежи в жизни', description: '', price: 230, price_effective: 230, owned: 0, ...over }
}

function mountBanner() {
  return mount(TariffsBanner, { global: { stubs } })
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.spyOn(api, 'tariffs').mockResolvedValue(tariffsInfo())
})
afterEach(() => vi.restoreAllMocks())

describe('TariffsBanner', () => {
  it('names the level the learner is saving for and the price of opening it now', async () => {
    enableTariffs()
    useWalletStore().products = [level()]
    const w = mountBanner()
    await flushPromises()
    expect(w.find('[data-test="tariffs-banner"]').text()).toContain('Уровень 4 — Падежи в жизни')
    expect(w.find('[data-test="banner-level"]').text()).toContain('249')
    expect(w.find('[data-test="banner-level"]').attributes('href')).toBe('/tariffs')
    expect(w.find('[data-test="banner-year"]').text()).toContain('1 000')
  })

  it('says the first purchase is discounted, and how much cheaper the year is', async () => {
    enableTariffs()
    useWalletStore().products = [level()]
    const w = mountBanner()
    await flushPromises()
    expect(w.find('[data-test="banner-discount"]').text()).toContain('Скидка 17% на первую покупку')
    expect(w.find('[data-test="banner-year"]').text()).toContain('выгода 72%')
  })

  it('promises the зёрнышки that come with the level, in the currency\'s own name', async () => {
    enableTariffs()
    const wallet = useWalletStore()
    wallet.products = [level()]
    wallet.nameOne = 'зёрнышко'
    wallet.nameFew = 'зёрнышка'
    wallet.nameMany = 'зёрнышек'
    const w = mountBanner()
    await flushPromises()
    expect(w.find('[data-test="banner-bonus"]').text()).toBe('1\u00a0000 зёрнышек')
  })

  it('names no discount once the regular price applies', async () => {
    const info = tariffsInfo({ has_paid_level: true, level: { price: 299, regular_price: 299, next: null } })
    vi.spyOn(api, 'tariffs').mockResolvedValue(info)
    enableTariffs(info)
    useWalletStore().products = [level()]
    const w = mountBanner()
    await flushPromises()
    expect(w.find('[data-test="banner-discount"]').exists()).toBe(false)
  })

  // A level was paid for less than a month ago: the banner stops offering one
  // and offers the year, which is not held back.
  it('offers the year, with the date, while the monthly limit holds a level back', async () => {
    const from = new Date(Date.now() + 10 * 86_400_000).toISOString()
    const info = tariffsInfo({ has_paid_level: true, level: { price: 299, regular_price: 299, available_from: from, next: null } })
    vi.spyOn(api, 'tariffs').mockResolvedValue(info)
    enableTariffs(info)
    useWalletStore().products = [level()]
    const w = mountBanner()
    await flushPromises()
    expect(w.find('[data-test="banner-level"]').exists()).toBe(false)
    expect(w.find('[data-test="banner-wait"]').text()).toContain('Следующий уровень за рубли — с ')
    expect(w.find('[data-test="banner-year"]').text()).toContain('1\u00a0000')
    expect(w.find('[data-test="banner-year"]').classes()).toContain('btn-primary')
  })

  it('quotes what the server quotes: the regular price after a first purchase', async () => {
    const info = tariffsInfo({ has_paid_level: true, level: { price: 299, regular_price: 299, next: null } })
    vi.spyOn(api, 'tariffs').mockResolvedValue(info)
    enableTariffs(info)
    useWalletStore().products = [level()]
    const w = mountBanner()
    await flushPromises()
    expect(w.find('[data-test="banner-level"]').text()).toContain('299')
  })

  // The whole point of the gate: an account the server has not opened tariffs for
  // sees no offer to pay, however short of seeds it is.
  it('is not there for an account that does not have tariffs', async () => {
    useSessionStore().user = sessionUser(false)
    useWalletStore().products = [level()]
    const w = mountBanner()
    await flushPromises()
    expect(w.find('[data-test="tariffs-banner"]').exists()).toBe(false)
    expect(api.tariffs).not.toHaveBeenCalled() // and it does not even ask for prices
  })

  it('is not there when every level is open', async () => {
    enableTariffs()
    useWalletStore().products = [level({ owned: 1 })]
    const w = mountBanner()
    await flushPromises()
    expect(w.find('[data-test="tariffs-banner"]').exists()).toBe(false)
  })

  it('is not there while a year is running: nothing is left to sell', async () => {
    const info = tariffsInfo({ year: { price: 1000, old_price: 2000, until: new Date(Date.now() + 86_400_000 * 100).toISOString() } })
    vi.spyOn(api, 'tariffs').mockResolvedValue(info)
    enableTariffs(info)
    useWalletStore().products = [level()]
    const w = mountBanner()
    await flushPromises()
    expect(w.find('[data-test="tariffs-banner"]').exists()).toBe(false)
  })
})
