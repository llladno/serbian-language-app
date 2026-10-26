import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TariffsPromo from './TariffsPromo.vue'
import { useSessionStore } from '../stores/session'
import { useTariffsStore } from '../stores/tariffs'
import { useWalletStore } from '../stores/wallet'
import { enableTariffs, sessionUser, tariffsInfo } from '../test/tariffsFixture'
import { rub } from '../lib/tariffs'
import { api } from '../api'

enableAutoUnmount(afterEach)
const stubs = { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } }

function mountPromo() {
  return mount(TariffsPromo, { global: { stubs } })
}

const day = 86_400_000

beforeEach(() => setActivePinia(createPinia()))

describe('TariffsPromo', () => {
  it('closes with the cross, reports it, and stays closed on this device', async () => {
    localStorage.removeItem('srpski.promo.dismissed')
    enableTariffs()
    const track = vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
    const w = mountPromo()
    await w.find('[data-test="promo-close"]').trigger('click')
    expect(w.find('[data-test="tariffs-promo"]').exists()).toBe(false)
    expect(track).toHaveBeenCalledWith('dismiss', 'profile_banner')
    expect(localStorage.getItem('srpski.promo.dismissed')).toBe('1')
    expect(mountPromo().find('[data-test="tariffs-promo"]').exists()).toBe(false)
    localStorage.removeItem('srpski.promo.dismissed')
  })

  it('has one button, and it says what the learner gets', () => {
    enableTariffs()
    const w = mountPromo()
    expect(w.find('[data-test="tariffs-promo"]').exists()).toBe(true)
    const button = w.find('[data-test="promo-level"]')
    expect(button.text()).toBe('Хочу зёрнышки и уровень')
    expect(button.attributes('href')).toBe('/tariffs')
    expect(w.find('[data-test="promo-year"]').exists()).toBe(false)
    expect(w.find('[data-test="promo-saves"]').exists()).toBe(false)
    expect(w.findAll('a')).toHaveLength(1)
  })

  // The price is not hidden, it sits right under the button, small, with a Serbian thank-you
  // after it (no gloss: the owner asked for none).
  it('states the price under the button: once, no subscription, then says thanks', () => {
    enableTariffs()
    const w = mountPromo()
    expect(w.find('[data-test="promo-price"]').text()).toBe(`${rub(249)} один раз, подписки нет`)
    // the heart is the picture from the "Поддержать проект" card, not an emoji
    const thanks = w.find('[data-test="promo-thanks"]')
    expect(thanks.text()).toBe('Hvala lepo')
    expect(thanks.find('img.heart').exists()).toBe(true)
    expect(w.text()).not.toContain('💛')
  })

  // The greeting is in the voice of the Telegram channel. It is shown in two places and
  // the stylesheet keeps one of them: on a phone a label in the sky above the bird (so the
  // text panel stays short), on a wide screen the first line of the panel.
  it('greets in Serbian and Russian, with no gloss and no label', () => {
    enableTariffs()
    const w = mountPromo()
    const hello = 'Ćao! 👋 Рады, что вы учите сербский с нами'
    expect(w.find('.hook').text()).toBe(hello)
    expect(w.find('.hook-pill').text()).toBe(hello)
    expect(w.text()).not.toContain('(Привет!)')
    expect(w.text()).not.toContain('(большое спасибо)')
  })

  // The headline is the owner's wording: the зёрнышки and the level, "в подарок". The
  // зёрнышки are shown as the seed glyph after the number, as everywhere in the app,
  // with the word kept for screen readers.
  it('names the зёрнышки (number and seed glyph) and the level as a gift', () => {
    enableTariffs()
    const wallet = useWalletStore()
    wallet.nameOne = 'зёрнышко'
    wallet.nameFew = 'зёрнышка'
    wallet.nameMany = 'зёрнышек'
    const title = mountPromo().find('[data-test="promo-title"]')
    expect(title.find('img.seed').exists()).toBe(true)
    expect(title.find('.sr-only').text()).toBe('зёрнышек')
    expect(title.text().replace(/\s+/g, ' ')).toBe('1 000 зёрнышек и уровень в подарок')
  })

  it('lists what a level gives: it stays forever, the repeats have no limit, the зёрнышки buy other levels', () => {
    enableTariffs()
    const w = mountPromo()
    expect(w.find('[data-test="promo-next"]').text()).toBe('Навсегда, без подписки')
    expect(w.find('[data-test="promo-repeats"]').text()).toBe('Повторения без лимита')
    expect(w.find('[data-test="promo-bonus"]').text()).toBe('Зёрнышки — на другие уровни')
    // this line wears the seed glyph on a gold chip, not a stock icon
    expect(w.find('[data-test="promo-bonus"] .ico-seed img.seed').exists()).toBe(true)
    // the other two wear the pixel icons (infinity, repeat), not the line icons of the icon set
    expect(w.find('[data-test="promo-next"] img.forever').exists()).toBe(true)
    expect(w.find('[data-test="promo-repeats"] img.repeat').exists()).toBe(true)
    expect(w.find('[data-test="tariffs-promo"] svg').exists()).toBe(false)
    expect(w.text()).not.toContain('на год')
  })

  it('offers just the level when it brings no зёрнышки', () => {
    enableTariffs({ level: { price: 249, regular_price: 299, next: { id: '4', title: 'Уровень 4' } } })
    const w = mountPromo()
    expect(w.find('[data-test="promo-title"]').text()).toBe('Следующий уровень навсегда')
    expect(w.find('[data-test="promo-level"]').text()).toBe('Хочу уровень')
    expect(w.find('[data-test="promo-bonus"]').exists()).toBe(false)
  })

  it('has no label above the headline', () => {
    enableTariffs()
    expect(mountPromo().text()).not.toContain('Тарифы')
  })

  it('has nothing to sell when there is no level left to open', () => {
    enableTariffs({ level: { price: 249, regular_price: 299, next: null } })
    expect(mountPromo().find('[data-test="tariffs-promo"]').exists()).toBe(false)
  })

  it('is not shown while the monthly limit holds a level back', () => {
    const from = new Date(Date.now() + 10 * day).toISOString()
    enableTariffs({ level: { price: 249, regular_price: 299, available_from: from, next: { id: '4', title: 'Уровень 4' } } })
    expect(mountPromo().find('[data-test="tariffs-promo"]').exists()).toBe(false)
  })

  it('is not shown while a year is running', () => {
    enableTariffs({ year: { price: 1000, old_price: 2000, until: new Date(Date.now() + day).toISOString() } })
    expect(mountPromo().find('[data-test="tariffs-promo"]').exists()).toBe(false)
  })

  it('is shown again once the year has run out', () => {
    enableTariffs({ year: { price: 1000, old_price: 2000, until: new Date(Date.now() - day).toISOString() } })
    expect(mountPromo().find('[data-test="tariffs-promo"]').exists()).toBe(true)
  })

  it('is not shown to a learner who paid for a level', () => {
    enableTariffs({ has_paid_level: true })
    expect(mountPromo().find('[data-test="tariffs-promo"]').exists()).toBe(false)
  })

  it('is not shown to an account that does not have tariffs', () => {
    useSessionStore().user = sessionUser(false)
    useTariffsStore().info = tariffsInfo()
    expect(mountPromo().find('[data-test="tariffs-promo"]').exists()).toBe(false)
  })

  it('waits for the prices instead of showing an empty offer', () => {
    useSessionStore().user = sessionUser(true)
    expect(mountPromo().find('[data-test="tariffs-promo"]').exists()).toBe(false)
  })
})

// The admin's «Воронка тарифов» counts how many learners were shown the banner, how many times, and who pressed.
describe('TariffsPromo reports what it shows', () => {
  it('reports one showing when the banner appears, and one press when the button is used', async () => {
    const spy = vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
    enableTariffs()
    const w = mountPromo()
    expect(spy).toHaveBeenCalledTimes(1)
    expect(spy).toHaveBeenCalledWith('view', 'profile_banner')
    await w.get('[data-test="promo-level"]').trigger('click')
    expect(spy).toHaveBeenLastCalledWith('click', 'profile_banner')
    expect(spy).toHaveBeenCalledTimes(2)
  })

  it('does not report a showing again while it merely re-renders', async () => {
    const spy = vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
    const store = enableTariffs()
    mountPromo()
    store.info = tariffsInfo() // the quote is read again: the banner stays
    await Promise.resolve()
    expect(spy.mock.calls.filter((c) => c[0] === 'view')).toHaveLength(1)
  })

  it('reports nothing when there is no banner (a tariff is already running)', () => {
    const spy = vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
    enableTariffs({ has_paid_level: true })
    mountPromo()
    expect(spy).not.toHaveBeenCalled()
  })
})
