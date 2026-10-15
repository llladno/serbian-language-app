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
    expect(payErrorMessage(new Error('boom'))).toContain('Попробуй ещё раз')
  })
})
