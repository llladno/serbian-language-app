import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { api, ApiError } from '../api'
import { useSessionStore } from './session'
import { useTariffsStore, payErrorMessage } from './tariffs'
import { sessionUser, tariffsInfo } from '../test/tariffsFixture'

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('useTariffsStore', () => {
  it('asks the server nothing for an account that does not have tariffs', async () => {
    const spy = vi.spyOn(api, 'tariffs')
    useSessionStore().user = sessionUser(false)
    const store = useTariffsStore()
    await store.load()
    expect(spy).not.toHaveBeenCalled()
    expect(store.enabled).toBe(false)
    expect(store.ready).toBe(false)
  })

  it('stays off when nobody is signed in', () => {
    useSessionStore().user = null
    expect(useTariffsStore().enabled).toBe(false)
  })

  it('loads the quote once and reports it', async () => {
    const spy = vi.spyOn(api, 'tariffs').mockResolvedValue(tariffsInfo({ has_paid_level: true }))
    useSessionStore().user = sessionUser(true)
    const store = useTariffsStore()
    await store.load()
    await store.load() // the second call is free
    expect(spy).toHaveBeenCalledTimes(1)
    expect(store.ready).toBe(true)
    expect(store.levelNow).toBe(249)
    expect(store.levelRegular).toBe(299)
    expect(store.yearPrice).toBe(1000)
    expect(store.hasPaidLevel).toBe(true)
    expect(store.yearActive).toBe(false)
  })

  it('a failed price lookup leaves the screens hidden instead of breaking them', async () => {
    vi.spyOn(api, 'tariffs').mockRejectedValue(new ApiError(500, 'internal error'))
    useSessionStore().user = sessionUser(true)
    const store = useTariffsStore()
    await expect(store.load()).resolves.toBeUndefined()
    expect(store.ready).toBe(false)
  })

  it('knows a year is running only until its end date', async () => {
    useSessionStore().user = sessionUser(true)
    const store = useTariffsStore()
    store.info = tariffsInfo({ year: { price: 1000, old_price: 2000, until: new Date(Date.now() + 86_400_000).toISOString() } })
    expect(store.yearActive).toBe(true)
    store.info = tariffsInfo({ year: { price: 1000, old_price: 2000, until: new Date(Date.now() - 86_400_000).toISOString() } })
    expect(store.yearActive).toBe(false)
  })

  // "Has a tariff" is what the profile's advert asks: a running year, or a level
  // paid for in rubles. Зёрнышки-bought levels and a lapsed year are not a tariff.
  it('has a tariff while a year runs or once a level was paid for in rubles', async () => {
    useSessionStore().user = sessionUser(true)
    const store = useTariffsStore()
    const day = 86_400_000
    store.info = tariffsInfo()
    expect(store.hasTariff).toBe(false)
    store.info = tariffsInfo({ year: { price: 1000, old_price: 2000, until: new Date(Date.now() - day).toISOString() } })
    expect(store.hasTariff).toBe(false)
    store.info = tariffsInfo({ year: { price: 1000, old_price: 2000, until: new Date(Date.now() + day).toISOString() } })
    expect(store.hasTariff).toBe(true)
    store.info = tariffsInfo({ has_paid_level: true })
    expect(store.hasTariff).toBe(true)
  })

  // A paid level is a credit the learner spends on a level of their choice.
  it('knows the levels a paid level can open and the credits waiting to be spent', () => {
    useSessionStore().user = sessionUser(true)
    const store = useTariffsStore()
    // an older answer without the two lists reads as none
    store.info = tariffsInfo({ level: { price: 249, regular_price: 299, next: null } })
    expect(store.choices).toEqual([])
    expect(store.credits).toEqual([])
    expect(store.hasCredit).toBe(false)
    const choices = [{ id: '4', title: 'Уровень 4 — Падежи', lessons: 18 }]
    store.info = tariffsInfo({ level: { price: 299, regular_price: 299, next: null, choices, credits: [7] } })
    expect(store.choices).toEqual(choices)
    expect(store.credits).toEqual([7])
    expect(store.hasCredit).toBe(true)
  })

  it('spends a credit on a level and asks the server for the quote again', async () => {
    useSessionStore().user = sessionUser(true)
    const choose = vi
      .spyOn(api, 'chooseLevel')
      .mockResolvedValue({ id: 7, tariff: 'level', status: 'succeeded', phase_title: 'Уровень 4 — Падежи', bonus: 1000 })
    const quote = vi.spyOn(api, 'tariffs').mockResolvedValue(tariffsInfo())
    const store = useTariffsStore()
    const order = await store.chooseLevel(7, '4')
    expect(choose).toHaveBeenCalledWith(7, '4')
    expect(order.phase_title).toBe('Уровень 4 — Падежи')
    expect(quote).toHaveBeenCalledTimes(1) // the credit is gone and the choices have changed
  })

  it('says to choose the paid level first when a new one is refused for it', () => {
    expect(payErrorMessage(new ApiError(409, 'level_pending_choice'))).toMatch(/выберите/i)
  })

  it('refuses to start two payments at once', async () => {
    const create = vi.spyOn(api, 'createOrder').mockImplementation(
      () => new Promise(() => {}), // never answers
    )
    useSessionStore().user = sessionUser(true)
    const store = useTariffsStore()
    void store.pay('level')
    void store.pay('year')
    expect(create).toHaveBeenCalledTimes(1)
  })

  it('reloads the quote once an order turns out to be paid', async () => {
    const spy = vi.spyOn(api, 'tariffs').mockResolvedValue(tariffsInfo())
    vi.spyOn(api, 'order').mockResolvedValue({ id: 3, tariff: 'level', status: 'succeeded' })
    useSessionStore().user = sessionUser(true)
    const store = useTariffsStore()
    await store.checkOrder(3)
    expect(spy).toHaveBeenCalledTimes(1)
  })
})

describe('payErrorMessage', () => {
  it('turns the server\'s codes into something to read', () => {
    expect(payErrorMessage(new ApiError(503, 'payments_unavailable'))).toContain('не подключена')
    expect(payErrorMessage(new ApiError(409, 'nothing_to_buy'))).toContain('уже открыты')
    expect(payErrorMessage(new ApiError(502, 'payment_failed'))).toContain('не ответила')
    expect(payErrorMessage(new Error('boom'))).toContain('Попробуйте ещё раз')
  })
})
