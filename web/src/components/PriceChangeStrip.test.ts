import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import PriceChangeStrip from './PriceChangeStrip.vue'
import PriceChangePoster from './PriceChangePoster.vue'
import PriceChangeBanner from './PriceChangeBanner.vue'
import TariffsPromo from './TariffsPromo.vue'
import { enableTariffs, tariffsInfo } from '../test/tariffsFixture'
import { api } from '../api'
import type { PriceChange, TariffsInfo } from '../types'

enableAutoUnmount(afterEach)
const stubs = { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } }

const day = 86_400_000
// the words keep no-break spaces so a line does not break after a short word
const flat = (t: string) => t.replace(/\u00a0/g, ' ')

// The server's clock is the phone's here: the change is 17 days and a bit ahead, at the end of a Friday in Belgrade.
function announced(over: Partial<PriceChange> = {}): Partial<TariffsInfo> {
  const at = Date.parse('2026-11-27T23:00:00Z')
  return {
    price_change: { at: new Date(at).toISOString(), now: new Date(at - 17 * day - 12 * 3_600_000).toISOString(), level_price: 399, year_price: 2000, ...over },
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('PriceChangeStrip', () => {
  it('shows the last day of the old prices, the countdown and one button to the tariffs', () => {
    enableTariffs(announced())
    const w = mount(PriceChangeStrip, { global: { stubs } })
    expect(w.find('[data-test="price-strip"]').exists()).toBe(true)
    expect(flat(w.get('[data-test="price-strip-msg"]').text())).toContain('27 ноября цены прежние')
    // 17 days and half a day ahead on the server's clock, a moment of it already gone
    // (the seconds run in both layouts: the long line and the short one)
    expect(w.get('[data-test="price-strip-timer"]').text()).toMatch(/17 дн 1[12]:\d\d:\d\dещё 17 дн 1[12]:\d\d:\d\d/)
    const go = w.get('[data-test="price-strip-go"]')
    expect(go.text()).toBe('Успеть')
    expect(go.attributes('href')).toBe('/tariffs')
  })

  it('is not there without an announcement, and not for a running year', () => {
    enableTariffs()
    expect(mount(PriceChangeStrip, { global: { stubs } }).find('[data-test="price-strip"]').exists()).toBe(false)
    const until = new Date(Date.now() + 100 * day).toISOString()
    enableTariffs({ ...announced(), year: { price: 1000, old_price: 2000, until } })
    expect(mount(PriceChangeStrip, { global: { stubs } }).find('[data-test="price-strip"]').exists()).toBe(false)
  })

  it("is always shown to the owner's own account, with a running year too", () => {
    const until = new Date(Date.now() + 100 * day).toISOString()
    enableTariffs({ ...announced({ preview: true }), year: { price: 1000, old_price: 2000, until } })
    expect(mount(PriceChangeStrip, { global: { stubs } }).find('[data-test="price-strip"]').exists()).toBe(true)
    expect(mount(PriceChangePoster).find('[data-test="price-poster"]').exists()).toBe(true)
  })

  it('sells the discount in its own words when the prices are half of what follows', () => {
    // the fixture's level is 249 (not half of 399): here it is 199, as it is for a learner who has paid for nothing
    enableTariffs({ ...announced(), level: { ...tariffsInfo().level, price: 199 } })
    const w = mount(PriceChangeStrip, { global: { stubs } })
    expect(flat(w.get('[data-test="price-strip-sticker"]').text())).toBe('Скидка 50%')
    expect(flat(w.get('[data-test="price-strip-msg"]').text())).toBe('Полцены до 27 ноября')
    expect(w.get('[data-test="price-strip-go"]').text()).toBe('Забрать')
  })

  it('has no cross: it cannot be closed', () => {
    enableTariffs(announced())
    const w = mount(PriceChangeStrip, { global: { stubs } })
    expect(w.find('[data-test="price-strip-close"]').exists()).toBe(false)
    expect(w.find('button').exists()).toBe(false)
  })

  it('reports the showing and the press', async () => {
    enableTariffs(announced())
    const track = vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
    const w = mount(PriceChangeStrip, { global: { stubs } })
    expect(track).toHaveBeenCalledWith('view', 'price_banner')
    await w.get('[data-test="price-strip-go"]').trigger('click')
    expect(track).toHaveBeenCalledWith('click', 'price_banner')
  })

  it('floats stickers and the two current prices behind the text, each tilted its own way', () => {
    enableTariffs(announced())
    const lines = mount(PriceChangeStrip, { global: { stubs } }).findAll('[data-test="price-strip-floaters"] li')
    expect(lines).toHaveLength(5)
    expect(lines.filter((l) => l.find('img').exists())).toHaveLength(3)
    expect(flat(lines.filter((l) => !l.find('img').exists()).map((l) => l.text()).join('|'))).toBe('249 ₽|1 000 ₽')
    expect(new Set(lines.map((l) => (l.element as HTMLElement).style.getPropertyValue('--tilt'))).size).toBe(5)
  })

  it('takes the place of the advert for the next level while it runs', () => {
    localStorage.removeItem('srpski.promo.dismissed')
    enableTariffs()
    expect(mount(TariffsPromo, { global: { stubs } }).find('[data-test="tariffs-promo"]').exists()).toBe(true)
    enableTariffs(announced())
    expect(mount(TariffsPromo, { global: { stubs } }).find('[data-test="tariffs-promo"]').exists()).toBe(false)
  })
})

describe('PriceChangePoster', () => {
  it('sets each price against the one to come and names the day', () => {
    enableTariffs(announced())
    const w = mount(PriceChangePoster)
    expect(w.get('[data-test="price-poster-date"]').text()).toContain('27')
    expect(flat(w.get('[data-test="price-poster-sub"]').text())).toContain('С 28 ноября дороже')
    expect(flat(w.get('[data-test="price-poster-level"]').text())).toContain('249 ₽')
    expect(flat(w.get('[data-test="price-poster-level"]').text())).toContain('399 ₽')
    expect(flat(w.get('[data-test="price-poster-year"]').text())).toContain('1 000 ₽')
    expect(flat(w.get('[data-test="price-poster-year"]').text())).toContain('2 000 ₽')
    expect(w.get('[data-test="price-poster-go"]').attributes('href')).toBe('#tariff-cards')
  })

  it('sells the discount, with the prices to come, when the prices are half of what follows', () => {
    enableTariffs({ ...announced(), level: { ...tariffsInfo().level, price: 199 } })
    const w = mount(PriceChangePoster)
    expect(flat(w.get('[data-test="price-poster-title"]').text())).toBe('Успей забрать за полцены')
    expect(flat(w.get('[data-test="price-poster-sub"]').text())).toContain('новые уроки навсегда — 399 ₽, всё на год — 2 000 ₽')
    expect(w.get('[data-test="price-poster-go"]').text()).toBe('Забрать')
    expect(flat(w.get('[data-test="price-poster-level"]').text())).toContain('НОВЫЕ УРОКИ')
    expect(flat(w.get('[data-test="price-poster-year"]').text())).toContain('ВСЁ НА ГОД')
  })

  it('lets only stickers rise behind it: the prices are in front, so none behind', () => {
    enableTariffs(announced())
    const lines = mount(PriceChangePoster).findAll('[data-test="price-poster-floaters"] li')
    expect(lines).toHaveLength(3)
    expect(lines.every((l) => l.find('img').exists())).toBe(true)
  })

  it('does not promise a discount', () => {
    enableTariffs(announced())
    const text = flat(mount(PriceChangePoster).text())
    expect(text).toContain('ничего не снижаем')
    expect(text).not.toMatch(/скидк|−\d+%/)
  })

  it('is not there without an announcement', () => {
    enableTariffs()
    expect(mount(PriceChangePoster).find('[data-test="price-poster"]').exists()).toBe(false)
  })
})

// A learner who has a tariff has bought what is on offer: no banner of the campaign at all, on any screen.
describe('nothing for a learner who has a tariff', () => {
  const until = () => new Date(Date.now() + 100 * day).toISOString()
  const cases: [string, Partial<TariffsInfo>][] = [
    ['a running year', { ...announced(), year: { price: 1000, old_price: 2000, until: until() } }],
    ['a level paid for in rubles', { ...announced(), has_paid_level: true }],
  ]

  it.each(cases)('has none with %s', (_name, over) => {
    localStorage.clear()
    const store = enableTariffs(over)
    expect(store.hasTariff).toBe(true)
    expect(store.campaign).toBeNull()
    expect(mount(PriceChangeStrip, { global: { stubs } }).find('[data-test="price-strip"]').exists()).toBe(false)
    expect(mount(PriceChangeBanner).find('[data-test="price-banner"]').exists()).toBe(false)
    expect(mount(PriceChangePoster).find('[data-test="price-poster"]').exists()).toBe(false)
  })

  it("is the owner's own account the one exception", () => {
    const store = enableTariffs({ ...announced({ preview: true }), has_paid_level: true })
    expect(store.campaign).not.toBeNull()
    expect(mount(PriceChangeStrip, { global: { stubs } }).find('[data-test="price-strip"]').exists()).toBe(true)
  })

  it('reports each banner under its own source, the strip with the screen it stood on', () => {
    enableTariffs(announced())
    const track = vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
    mount(PriceChangeStrip, { props: { place: 'course' }, global: { stubs } })
    mount(PriceChangeBanner)
    expect(track).toHaveBeenCalledWith('view', 'price_banner', 'course')
    expect(track).toHaveBeenCalledWith('view', 'price_page')
  })
})

describe('PriceChangeBanner', () => {
  it('is the small one: the discount and the countdown with seconds, one short line and the prices, with no button', () => {
    enableTariffs({ ...announced(), level: { ...tariffsInfo().level, price: 199 } })
    const w = mount(PriceChangeBanner)
    expect(flat(w.get('[data-test="price-banner-sticker"]').text())).toBe('Скидка 50%')
    expect(w.get('[data-test="price-banner-timer"]').text()).toMatch(/\d дн \d\d:\d\d:\d\d/)
    expect(flat(w.get('[data-test="price-banner-title"]').text())).toBe('Успей забрать за полцены')
    expect(flat(w.get('[data-test="price-banner-sub"]').text())).toBe('До 27 ноября — по-старому, с 28 ноября цены вырастут вдвое.')
    expect(flat(w.get('[data-test="price-banner-level"]').text())).toContain('199 ₽')
    expect(flat(w.get('[data-test="price-banner-level"]').text())).toContain('399 ₽')
    expect(flat(w.get('[data-test="price-banner-year"]').text())).toContain('2 000 ₽')
    // no button: the cards are right below the banner
    expect(w.find('[data-test="price-banner-go"]').exists()).toBe(false)
    expect(w.find('a, button').exists()).toBe(false)
  })

  it('lets stickers and the two current prices rise behind it', () => {
    enableTariffs(announced())
    const lines = mount(PriceChangeBanner).findAll('[data-test="price-banner-floaters"] li')
    expect(lines).toHaveLength(5)
    expect(lines.filter((l) => !l.find('img').exists())).toHaveLength(2)
  })

  it('is not there without an announcement or with a running year', () => {
    enableTariffs()
    expect(mount(PriceChangeBanner).find('[data-test="price-banner"]').exists()).toBe(false)
    const until = new Date(Date.now() + 100 * day).toISOString()
    enableTariffs({ ...announced(), year: { price: 1000, old_price: 2000, until } })
    expect(mount(PriceChangeBanner).find('[data-test="price-banner"]').exists()).toBe(false)
  })
})
