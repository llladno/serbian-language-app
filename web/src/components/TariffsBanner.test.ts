import { describe, it, expect, afterEach, beforeEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TariffsBanner from './TariffsBanner.vue'
import { useWalletStore } from '../stores/wallet'
import { useTariffsStore } from '../stores/tariffs'
import type { ShopItem } from '../types'

enableAutoUnmount(afterEach)

const stubs = { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } }

function level(over: Partial<ShopItem> = {}): ShopItem {
  return { id: 1, kind: 'phase_unlock', ref: '4', title: 'Уровень 4 — Падежи в жизни', description: '', price: 230, price_effective: 230, owned: 0, ...over }
}

beforeEach(() => setActivePinia(createPinia()))

describe('TariffsBanner', () => {
  it('names the level the learner is saving for and the price of opening it now', () => {
    useWalletStore().products = [level()]
    const w = mount(TariffsBanner, { global: { stubs } })
    expect(w.find('[data-test="tariffs-banner"]').text()).toContain('Уровень 4 — Падежи в жизни')
    expect(w.find('[data-test="banner-level"]').text()).toContain('249')
    expect(w.find('[data-test="banner-level"]').attributes('href')).toBe('/tariffs')
    expect(w.find('[data-test="banner-year"]').text()).toContain('1 000')
  })

  it('quotes the regular price once the learner has paid for a level before', () => {
    useWalletStore().products = [level()]
    useTariffsStore().hasPaidLevel = true
    const w = mount(TariffsBanner, { global: { stubs } })
    expect(w.find('[data-test="banner-level"]').text()).toContain('299')
  })

  it('is not there when every level is open', () => {
    useWalletStore().products = [level({ owned: 1 })]
    expect(mount(TariffsBanner, { global: { stubs } }).find('[data-test="tariffs-banner"]').exists()).toBe(false)
  })

  it('is not there while a year is running: nothing is left to sell', () => {
    useWalletStore().products = [level()]
    useTariffsStore().yearUntil = new Date(Date.now() + 86_400_000 * 100).toISOString()
    expect(mount(TariffsBanner, { global: { stubs } }).find('[data-test="tariffs-banner"]').exists()).toBe(false)
  })
})
