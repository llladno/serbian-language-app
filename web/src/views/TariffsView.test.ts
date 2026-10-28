import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TariffsView from './TariffsView.vue'
import { api, ApiError } from '../api'
import { useCourseStore } from '../stores/course'
import { useWalletStore } from '../stores/wallet'
import { useToasts } from '../lib/toasts'
import { isTelegram, initData, openExternalLink } from '../telegram'
import { enableTariffs, tariffsInfo } from '../test/tariffsFixture'
import { useLevelChoiceModal } from '../lib/levelChoiceModal'
import type { Course, SessionUser, TariffsInfo } from '../types'

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}))
vi.mock('../telegram', () => ({ isTelegram: vi.fn(), initData: vi.fn(), openExternalLink: vi.fn() }))

enableAutoUnmount(afterEach)

const assign = vi.fn()

// The level part of the quote, with some of it changed (choices, credits, the limit).
const levelOf = (over: Partial<TariffsInfo['level']>): TariffsInfo['level'] => ({ ...tariffsInfo().level, ...over })
const norm = (text: string) => text.replace(/\s+/g, ' ').trim()

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
// bar's query, for the visit back from the bank.
async function mountView(over: Partial<TariffsInfo> = {}, c: Course = course(), search = '', user: Partial<SessionUser> = {}) {
  const info = tariffsInfo(over)
  vi.spyOn(api, 'tariffs').mockResolvedValue(info)
  setActivePinia(createPinia())
  useCourseStore().course = c
  enableTariffs(over, user)
  Object.defineProperty(window, 'location', {
    configurable: true,
    value: { href: 'http://localhost/tariffs' + search, assign },
  })
  const w = mount(TariffsView, { global: { stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } } } })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.stubEnv('DEV', false) // the tests see a production build; the dev-only window has its own tests below
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
  vi.unstubAllEnvs()
  vi.useRealTimers()
  useLevelChoiceModal().closeModal()
})

describe('TariffsView: the two plans', () => {
  it('offers a level of your choice at the first-purchase price with the regular price struck through', async () => {
    const w = await mountView()
    expect(w.get('[data-test="level-title"]').text()).toBe('Уровень на ваш выбор')
    const price = w.get('[data-test="level-price"]').text()
    expect(price).toContain('249')
    expect(price).toContain('299')
    expect(w.get('[data-test="level-price"] .was').text()).toContain('299')
    expect(w.get('[data-test="buy-level"]').text()).toBe('Хочу уровень — 249\u00a0₽')
  })

  it('lists by name, without the numbers, only the levels still left to choose', async () => {
    const w = await mountView()
    expect(w.get('[data-test="level-inside"]').text()).toContain('«Падежи в жизни», «Жизнь на сербском»')
    const left = await mountView({ level: levelOf({ choices: [tariffsInfo().level.choices![1]] }) })
    const inside = left.get('[data-test="level-inside"]').text()
    expect(inside).toContain('«Жизнь на сербском»')
    expect(inside).not.toContain('Падежи в жизни') // already open: not offered
  })

  it('says "and N more" when there are more than three levels to choose from', async () => {
    const many = ['4', '5', '6', '7'].map((id) => ({ id, title: `Уровень ${id} — Тема ${id}`, lessons: 10 }))
    const w = await mountView({ level: levelOf({ choices: many }) })
    expect(w.get('[data-test="level-inside"]').text()).toContain('«Тема 4», «Тема 5», «Тема 6» и ещё 1')
  })

  it('keeps the 249 tariff on show when every level is open, with the button switched off', async () => {
    const w = await mountView({ level: levelOf({ choices: [], next: null }) })
    expect(w.get('[data-test="level-price"]').text()).toContain('249')
    expect(w.find('[data-test="buy-level"]').exists()).toBe(false)
    expect(w.find('[data-test="level-inside"]').exists()).toBe(false)
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

describe('TariffsView: what makes buying worthwhile', () => {
  // The first purchase is cheaper than the regular price; say by how much.
  it('says the first purchase is a discount from 299, in percent', async () => {
    const w = await mountView()
    expect(w.get('[data-test="level-discount"]').text()).toBe('−17%')
    // the prices are in the line above; the note only names the discount
    expect(w.get('[data-test="level-discount-note"]').text().replace(/\s+/g, ' ')).toBe('Скидка 17% на первую покупку')
  })

  it('shows no discount once the regular price applies', async () => {
    const w = await mountView({ has_paid_level: true, level: { price: 299, regular_price: 299, next: null } })
    expect(w.find('[data-test="level-discount"]').exists()).toBe(false)
    expect(w.find('[data-test="level-discount-note"]').exists()).toBe(false)
    expect(w.find('[data-test="level-price"] .was').exists()).toBe(false)
  })

  // The year is set against a level every month for twelve months: the first at this learner's price, the
  // other eleven at the regular one (249 + 11 × 299 = 3 538 with the test prices).
  it('shows how much the year saves against a level every month', async () => {
    const w = await mountView()
    expect(w.get('[data-test="year-badge"]').text()).toBe('Выгода 72%')
  })

  it('draws a thin bar with the year\'s price at the left and "вместо" the twelve-month sum at the right', async () => {
    const w = await mountView()
    expect(norm(w.get('.thin-row b').text())).toBe('1 000 ₽')
    expect(norm(w.get('.thin-row span').text())).toBe('вместо 3 538 ₽')
    expect(norm(w.get('[data-test="tariff-year"]').text())).toContain('≈ 83 ₽ в месяц') // the line stays under the price
  })

  it('counts every month at 299 once the first purchase is behind the learner', async () => {
    const w = await mountView({ has_paid_level: true, level: { price: 299, regular_price: 299, next: null } })
    expect(norm(w.get('[data-test="saving-monthly"]').text())).toBe('вместо 3 588 ₽')
    expect(w.get('[data-test="year-badge"]').text()).toBe('Выгода 72%') // (3 588 − 1 000) / 3 588
  })

  it('claims no saving when the year would not be cheaper', async () => {
    const w = await mountView({ year: { price: 4000, old_price: 0 } })
    expect(w.find('[data-test="year-saving"]').exists()).toBe(false)
    expect(w.find('[data-test="year-badge"]').exists()).toBe(false)
  })

  it('sets out what each plan gives, in bold', async () => {
    const w = await mountView()
    const level = w.get('[data-test="tariff-level"]')
    expect(level.findAll('.perks li')).toHaveLength(3) // forever, unlimited repeats, the зёрнышки
    expect(level.text()).toContain('Навсегда, без подписки')
    expect(level.text()).toContain('само ничего не спишется')
    expect(level.text()).not.toContain('Тратятся в магазине') // the gift line stops at "в подарок к уровню"
    expect(w.get('[data-test="level-monthly"]').text()).toContain('раз в месяц')
    const year = w.get('[data-test="tariff-year"]')
    expect(year.findAll('.perks li')).toHaveLength(4) // every level, unlimited repeats, the зёрнышки, new levels
    expect(year.text()).toContain('Все уровни сразу')
    expect(year.text()).not.toContain('Без очереди') // the line says "Все уровни сразу" and stops
    expect(year.text()).toContain('Новые уровни включены')
  })
})

describe('TariffsView: the зёрнышки that come with a tariff', () => {
  // The currency is named in the admin panel and only known once the wallet has
  // answered, so the tests give it a name.
  async function mountWithCurrency(over: Partial<TariffsInfo> = {}) {
    const w = await mountView(over)
    const wallet = useWalletStore()
    wallet.nameOne = 'зёрнышко'
    wallet.nameFew = 'зёрнышка'
    wallet.nameMany = 'зёрнышек'
    await flushPromises()
    return w
  }

  it('says a level brings 1 000 зёрнышек and the year 15 000', async () => {
    const w = await mountWithCurrency()
    expect(w.get('[data-test="level-bonus"]').text()).toContain('+1\u00a0000 зёрнышек')
    expect(w.get('[data-test="year-bonus"]').text()).toContain('+15\u00a0000 зёрнышек')
  })

  it('gives the year\'s зёрнышки in one short line, with no sum against twelve levels', async () => {
    const w = await mountWithCurrency()
    const text = w.get('[data-test="year-bonus"]').text().replace(/\s+/g, ' ')
    expect(text).toContain('в подарок')
    expect(text).not.toContain('больше')
    expect(text).not.toContain('двенадцать')
  })

  it('centres the one-line gift rows against their icons', async () => {
    const w = await mountWithCurrency()
    for (const row of ['level-bonus', 'year-bonus']) expect(w.get(`[data-test="${row}"]`).classes()).toContain('centered')
  })

  it('says on both cards that repeats have no daily limit', async () => {
    const w = await mountWithCurrency()
    expect(w.get('[data-test="level-repeat"]').text()).toContain('Повторения без лимита')
    expect(w.get('[data-test="year-repeat"]').text()).toContain('Повторения без лимита')
  })

  it('shows no bonus line for a tariff that has none', async () => {
    const w = await mountWithCurrency({
      level: levelOf({ bonus: undefined }),
      year: { price: 1000, old_price: 2000 },
    })
    expect(w.find('[data-test="level-bonus"]').exists()).toBe(false)
    expect(w.find('[data-test="year-bonus"]').exists()).toBe(false)
  })
})

describe('TariffsView: one level a month', () => {
  // A level was paid for recently: the button says when the next opens, and the
  // year, which is not held back, stays on offer.
  const held = (from: string) => ({ level: levelOf({ price: 299, regular_price: 299, available_from: from }) })
  const future = () => new Date(Date.now() + 12 * 86_400_000).toISOString()

  it('replaces the buy button with the date the next level opens', async () => {
    const w = await mountView({ has_paid_level: true, ...held(future()) })
    expect(w.find('[data-test="buy-level"]').exists()).toBe(false)
    const wait = w.get('[data-test="level-wait"]')
    expect(wait.attributes('disabled')).toBeDefined()
    expect(wait.text()).toContain('Следующий уровень — с ')
    const note = w.get('[data-test="level-cooldown"]').text().replace(/\s+/g, ' ')
    expect(note).toContain('через 12 дней')
    expect(note).toContain('откройте все уровни на год')
    expect(w.get('[data-test="level-cooldown"] a').attributes('href')).toBe('#tariff-year')
  })

  it('leaves the year on sale during the wait', async () => {
    const create = vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 8, tariff: 'year', status: 'pending', confirmation_url: 'https://pay.test/confirm/8',
    })
    const w = await mountView({ has_paid_level: true, ...held(future()) })
    expect(w.get('[data-test="buy-year"]').attributes('disabled')).toBeUndefined()
    await w.get('[data-test="buy-year"]').trigger('click')
    await flushPromises()
    expect(create).toHaveBeenCalledWith('year')
  })

  it('is not shown once the date has passed', async () => {
    const w = await mountView({ has_paid_level: true, ...held(new Date(Date.now() - 86_400_000).toISOString()) })
    expect(w.find('[data-test="level-cooldown"]').exists()).toBe(false)
    expect(w.find('[data-test="buy-level"]').exists()).toBe(true)
  })

  it('shows the date when the server says the limit applies although the screen did not know', async () => {
    vi.spyOn(api, 'createOrder').mockRejectedValue(new ApiError(409, 'level_cooldown'))
    const w = await mountView()
    const reload = vi.spyOn(api, 'tariffs').mockResolvedValue(
      tariffsInfo({ has_paid_level: true, ...held(future()) }),
    )
    await w.get('[data-test="buy-level"]').trigger('click')
    await flushPromises()
    expect(useToasts().items.value[0].text).toContain('раз в месяц')
    expect(reload).toHaveBeenCalled()
    expect(w.find('[data-test="level-wait"]').exists()).toBe(true)
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
  it('starts an order and takes the buyer to the bank', async () => {
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
    expect(w.get('[data-test="buy-year"]').text()).not.toContain('на сайте') // the button reads as it does in a browser
    await w.get('[data-test="buy-year"]').trigger('click')
    await flushPromises()
    expect(openExternalLink).toHaveBeenCalledWith('https://pay.test/confirm/9')
    expect(assign).not.toHaveBeenCalled() // the Mini App itself stays where it is
  })

  it('treats Telegram\'s WebApp object without initData as a plain browser', async () => {
    vi.mocked(isTelegram).mockReturnValue(true)
    vi.mocked(initData).mockReturnValue('')
    vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 10, tariff: 'year', status: 'pending', confirmation_url: 'https://pay.test/confirm/10',
    })
    const w = await mountView()
    await w.get('[data-test="buy-year"]').trigger('click')
    await flushPromises()
    expect(assign).toHaveBeenCalledWith('https://pay.test/confirm/10')
    expect(openExternalLink).not.toHaveBeenCalled()
  })

  it('makes both pay buttons the main button', async () => {
    const w = await mountView()
    expect(w.get('[data-test="buy-level"]').classes()).toContain('btn-primary')
    expect(w.get('[data-test="buy-year"]').classes()).toContain('btn-primary')
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

// An account that came in through Telegram has no email: the receipt needs an email or a phone, which a
// window asks for when a pay button is pressed. The window is teleported to <body>, outside the view.
describe('TariffsView: paying without an account email', () => {
  const tgOnly = { email: '', email_verified: false }
  const modal = () => document.body.querySelector('[data-test="receipt-contact-modal"]')
  const field = () => document.body.querySelector<HTMLInputElement>('[data-test="receipt-contact-input"]')
  const error = () => document.body.querySelector('[data-test="receipt-contact-error"]')
  const click = async (selector: string) => {
    document.body.querySelector<HTMLElement>(selector)?.click()
    await flushPromises()
  }
  const type = async (value: string) => {
    field()!.value = value
    field()!.dispatchEvent(new Event('input'))
    await flushPromises()
  }
  const press = async (w: Awaited<ReturnType<typeof mountView>>, button: string) => {
    await w.get(`[data-test="${button}"]`).trigger('click')
    await flushPromises()
  }

  beforeEach(() => localStorage.clear())

  it('keeps the page free of a receipt field and opens the window when a pay button is pressed', async () => {
    const create = vi.spyOn(api, 'createOrder')
    const w = await mountView({}, course(), '', tgOnly)
    expect(w.find('[data-test="receipt-contact"]').exists()).toBe(false)
    expect(modal()).toBeNull()
    await press(w, 'buy-year')
    expect(modal()).not.toBeNull()
    expect(document.body.textContent).toContain('Куда прислать чек?')
    expect(create).not.toHaveBeenCalled()
  })

  it('in a dev build the window opens for an account that has an email too, and sends no contact', async () => {
    vi.stubEnv('DEV', true)
    const create = vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 13, tariff: 'year', status: 'pending', confirmation_url: 'https://pay.test/tb/13',
    })
    const w = await mountView() // the admin account has an email
    await press(w, 'buy-year')
    expect(modal()).not.toBeNull()
    await type('buyer@example.com')
    await click('[data-test="receipt-contact-submit"]')
    expect(create).toHaveBeenCalledWith('year') // the account's own email is the receipt's address, not what was typed
    expect(localStorage.getItem('tariffs.receiptContact')).toBeNull()
  })

  it('?receipt=0 switches that off in a dev build: straight to the bank', async () => {
    vi.stubEnv('DEV', true)
    const create = vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 14, tariff: 'year', status: 'pending', confirmation_url: 'https://pay.test/tb/14',
    })
    const w = await mountView({}, course(), '?receipt=0')
    await press(w, 'buy-year')
    expect(modal()).toBeNull()
    expect(create).toHaveBeenCalledWith('year')
  })

  it('does not ask an account that has an email: the button goes straight to the bank', async () => {
    const create = vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 10, tariff: 'level', status: 'pending', confirmation_url: 'https://pay.test/tb/10',
    })
    const w = await mountView()
    await press(w, 'buy-level')
    expect(modal()).toBeNull()
    expect(create).toHaveBeenCalledWith('level')
  })

  it('sends nothing, and says why, until the contact looks like an email or a phone', async () => {
    const create = vi.spyOn(api, 'createOrder')
    const w = await mountView({}, course(), '', tgOnly)
    await press(w, 'buy-year')
    await click('[data-test="receipt-contact-submit"]') // empty
    expect(error()).not.toBeNull()
    await type('not a contact')
    expect(error()).toBeNull() // the note goes away as soon as the buyer types again
    await click('[data-test="receipt-contact-submit"]')
    expect(error()).not.toBeNull()
    await type('12345')
    await click('[data-test="receipt-contact-submit"]')
    expect(error()).not.toBeNull()
    expect(modal()).not.toBeNull()
    expect(create).not.toHaveBeenCalled()
  })

  it('orders with the typed email, closes the window and keeps the email for next time', async () => {
    const create = vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 11, tariff: 'year', status: 'pending', confirmation_url: 'https://pay.test/tb/11',
    })
    const w = await mountView({}, course(), '', tgOnly)
    await press(w, 'buy-year')
    await type('  buyer@example.com ')
    await click('[data-test="receipt-contact-submit"]')
    expect(create).toHaveBeenCalledWith('year', 'buyer@example.com')
    expect(assign).toHaveBeenCalledWith('https://pay.test/tb/11')
    expect(modal()).toBeNull()
    expect(localStorage.getItem('tariffs.receiptContact')).toBe('buyer@example.com')
    // the next press asks again with it filled in, so a returning buyer only confirms
    const again = await mountView({}, course(), '', tgOnly)
    await press(again, 'buy-level')
    expect(field()?.value).toBe('buyer@example.com')
  })

  it('orders with a typed phone as it was written', async () => {
    const create = vi.spyOn(api, 'createOrder').mockResolvedValue({
      id: 12, tariff: 'level', status: 'pending', confirmation_url: 'https://pay.test/tb/12',
    })
    const w = await mountView({}, course(), '', tgOnly)
    await press(w, 'buy-level')
    await type('8 (903) 123-45-67')
    await click('[data-test="receipt-contact-submit"]')
    expect(create).toHaveBeenCalledWith('level', '8 (903) 123-45-67')
  })

  it('starts nothing when the window is closed', async () => {
    const create = vi.spyOn(api, 'createOrder')
    const w = await mountView({}, course(), '', tgOnly)
    await press(w, 'buy-year')
    await click('[data-test="receipt-contact-close"]')
    expect(modal()).toBeNull()
    expect(create).not.toHaveBeenCalled()
  })

  it('tells the buyer when the server turned the contact down, and asks again on the next press', async () => {
    vi.spyOn(api, 'createOrder').mockRejectedValue(new ApiError(400, 'receipt_contact_invalid'))
    const w = await mountView({}, course(), '', tgOnly)
    await press(w, 'buy-level')
    await type('buyer@example.com')
    await click('[data-test="receipt-contact-submit"]')
    expect(useToasts().items.value[0].text).toContain('почту или телефон')
    await press(w, 'buy-level')
    expect(field()?.value).toBe('buyer@example.com') // the window is back, to be corrected
  })
})

describe('TariffsView: coming back from the bank', () => {
  // The modal is teleported to <body>, outside the mounted view.
  const thanks = () => document.body.querySelector('[data-test="thanks-modal"]')
  const text = (el: Element | null) => (el?.textContent ?? '').replace(/\s+/g, ' ').trim()

  it('a paid order opens the level, thanks the buyer and refreshes the course', async () => {
    const order = vi.spyOn(api, 'order').mockResolvedValue({
      id: 7, tariff: 'level', status: 'succeeded', phase_title: 'Уровень 4 — Падежи в жизни', bonus: 1000,
    })
    const reload = vi.spyOn(api, 'course').mockResolvedValue(course({ '4': false, '5': true }))
    await mountView({}, course(), '?order=7')
    expect(order).toHaveBeenCalledWith(7)
    expect(reload).toHaveBeenCalled() // the course is fetched again: level 4 is open now
    expect(text(thanks())).toContain('Спасибо, что вы с нами и поддерживаете проект')
    expect(thanks()?.querySelector('img[data-test="thanks-heart"]')).not.toBeNull() // the pixel heart after the thanks
    expect(text(thanks()?.querySelector('[data-test="thanks-opened"]') ?? null)).toBe('Открыт «Уровень 4 — Падежи в жизни»')
    expect(text(thanks()?.querySelector('[data-test="thanks-bonus"]') ?? null)).toContain('+1 000')
    expect(useToasts().items.value).toEqual([]) // the modal says it, not a toast too
  })

  // The tone of the banner, the tariffs screen and the Telegram channel: «вы», never «ты».
  it('speaks to the buyer as «вы», like the rest of the paid screens', async () => {
    vi.spyOn(api, 'order').mockResolvedValue({ id: 8, tariff: 'year', status: 'succeeded', bonus: 15000 })
    await mountView({}, course(), '?order=8')
    const said = text(thanks())
    expect(said).toContain('у вас')
    expect(said).not.toMatch(/(^|[^а-яё])(ты|тебя|тебе|твой|твоя|твои)([^а-яё]|$)/i)
  })

  it('the thank-you has one way forward: the course', async () => {
    vi.spyOn(api, 'order').mockResolvedValue({ id: 7, tariff: 'level', status: 'succeeded' })
    await mountView({}, course(), '?order=7')
    const link = thanks()?.querySelector('[data-test="thanks-course"]')
    expect(link?.getAttribute('href')).toBe('/course')
  })

  it('goes away when closed, and does not come back on its own', async () => {
    vi.spyOn(api, 'order').mockResolvedValue({ id: 7, tariff: 'level', status: 'succeeded' })
    await mountView({}, course(), '?order=7')
    ;(thanks()?.querySelector('[data-test="thanks-close"]') as HTMLElement).click()
    await flushPromises()
    expect(thanks()).toBeNull()
  })

  it('a paid year says that every level is open, until when, and its bonus', async () => {
    vi.spyOn(api, 'order').mockResolvedValue({ id: 8, tariff: 'year', status: 'succeeded', bonus: 15000 })
    const until = new Date('2027-10-14T12:00:00Z').toISOString()
    await mountView({ year: { price: 1000, old_price: 2000, bonus: 15000, until } }, course(), '?order=8')
    expect(text(thanks()?.querySelector('[data-test="thanks-opened"]') ?? null)).toBe('Открыты все уровни до 14 октября 2027 г.')
    expect(text(thanks()?.querySelector('[data-test="thanks-bonus"]') ?? null)).toContain('+15 000')
  })

  it('leaves the bonus line out when the order brought none', async () => {
    vi.spyOn(api, 'order').mockResolvedValue({ id: 7, tariff: 'level', status: 'succeeded', phase_title: 'Уровень 4' })
    await mountView({}, course(), '?order=7')
    expect(thanks()?.querySelector('[data-test="thanks-bonus"]')).toBeNull()
  })

  it('a canceled order says nothing was charged, with no thank-you', async () => {
    vi.spyOn(api, 'order').mockResolvedValue({ id: 7, tariff: 'level', status: 'canceled' })
    await mountView({}, course(), '?order=7')
    expect(thanks()).toBeNull()
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
    expect(thanks()).not.toBeNull()
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

describe('TariffsView: the words and the pictures', () => {
  async function mountWithCurrency() {
    const w = await mountView()
    const wallet = useWalletStore()
    wallet.nameOne = 'зёрнышко'
    wallet.nameFew = 'зёрнышка'
    wallet.nameMany = 'зёрнышек'
    await flushPromises()
    return w
  }

  it('opens with the swallow, a hello in Serbian and a warm title', async () => {
    const w = await mountView()
    const hero = w.get('[data-test="tariffs-hero"]')
    expect(hero.find('img').exists()).toBe(true)
    expect(w.get('[data-test="tariffs-hello"]').text()).toBe('Ćao! 👋 Рады, что вы учите сербский с нами')
    expect(hero.get('h1').text()).toBe('Ласточка летит дальше')
    expect(w.text()).not.toContain('(Привет!)') // no gloss: the owner asked for none
  })

  it('wears the pixel icons: infinity and the repeat arrows, and the seed, not line icons', async () => {
    const w = await mountView()
    expect(w.find('[data-test="level-forever"] img.forever').exists()).toBe(true)
    expect(w.find('[data-test="level-repeat"] img.repeat').exists()).toBe(true)
    expect(w.find('[data-test="year-repeat"] img.repeat').exists()).toBe(true)
    expect(w.find('[data-test="level-bonus"] .ico-seed img.seed').exists()).toBe(true)
  })

  it('says the зёрнышки are a gift, as a number and the seed glyph with the word kept for screen readers', async () => {
    const w = await mountWithCurrency()
    const level = w.get('[data-test="level-bonus"]')
    expect(level.find('img.seed').exists()).toBe(true)
    expect(level.text().replace(/\s+/g, ' ')).toContain('+1 000 зёрнышек в подарок к уровню.')
  })

  it('thanks in Serbian with the pixel heart, under the offer', async () => {
    const w = await mountView()
    const line = w.get('[data-test="thanks-line"]')
    expect(line.text()).toBe('Hvala lepo')
    expect(line.find('img.heart').exists()).toBe(true)
    expect(w.get('[data-test="offer-note"]').text()).toContain('Оплачивая, вы принимаете')
  })

  it('speaks to the reader as "вы", the way the banner and the channel do', async () => {
    const w = await mountView()
    expect(w.text()).not.toMatch(/(^|[^а-яё])(ты|тебя|тебе|тобой|твой|твоё|твои)([^а-яё]|$)/i)
  })

  it('has the warm words and the heart in the running-year card too', async () => {
    const until = new Date(Date.now() + 30 * 86_400_000).toISOString()
    const w = await mountView({ year: { price: 1000, old_price: 2000, until } })
    expect(w.get('[data-test="no-autorenew"]').text()).toContain('вы сами решите')
    expect(w.find('[data-test="my-tariff-thanks"] img.heart').exists()).toBe(true)
  })
})

describe('TariffsView: a paid level is spent on a level of the learner\'s choice', () => {
  const waiting = () => ({ has_paid_level: true, level: levelOf({ price: 299, regular_price: 299, credits: [7] }) })

  it('shows the reminder and swaps the buy button for the way to choose', async () => {
    const w = await mountView(waiting())
    expect(w.find('[data-test="level-credit"]').exists()).toBe(true)
    expect(w.find('[data-test="buy-level"]').exists()).toBe(false)
    expect(w.get('[data-test="level-choose"]').text()).toBe('Выбрать оплаченный уровень')
  })

  it('raises the choice window by itself when the screen opens with a paid level waiting', async () => {
    await mountView(waiting())
    expect(useLevelChoiceModal().open.value).toBe(true)
    expect(useLevelChoiceModal().orderId.value).toBeNull() // the oldest one waiting
  })

  it('does not raise it when nothing is waiting', async () => {
    await mountView()
    expect(useLevelChoiceModal().open.value).toBe(false)
  })

  it('raises it again from the button after it was closed', async () => {
    const w = await mountView(waiting())
    useLevelChoiceModal().closeModal()
    await w.get('[data-test="level-choose"]').trigger('click')
    expect(useLevelChoiceModal().open.value).toBe(true)
  })

  it('opens it for the order just paid when coming back from the bank, instead of the thank-you', async () => {
    vi.spyOn(api, 'order').mockResolvedValue({ id: 7, tariff: 'level', status: 'succeeded', bonus: 1000, choose: true })
    await mountView({}, course(), '?order=7')
    expect(useLevelChoiceModal().open.value).toBe(true)
    expect(useLevelChoiceModal().orderId.value).toBe(7)
    expect(document.body.querySelector('[data-test="thanks-modal"]')).toBeNull()
  })

  it('a refused order for a second level reloads the quote, raises the window and says why', async () => {
    vi.spyOn(api, 'createOrder').mockRejectedValue(new ApiError(409, 'level_pending_choice'))
    const w = await mountView()
    const reload = vi.spyOn(api, 'tariffs').mockResolvedValue(tariffsInfo(waiting()))
    await w.get('[data-test="buy-level"]').trigger('click')
    await flushPromises()
    expect(reload).toHaveBeenCalled()
    expect(useLevelChoiceModal().open.value).toBe(true)
    expect(useToasts().items.value[0].text).toContain('выберите уровень')
  })
})
