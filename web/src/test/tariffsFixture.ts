// Shared setup for the tests of the tariffs screens: the account the server says
// may use them, and the prices it would quote.
import { useSessionStore } from '../stores/session'
import { useTariffsStore } from '../stores/tariffs'
import type { SessionUser, TariffsInfo } from '../types'

export function sessionUser(tariffs: boolean, over: Partial<SessionUser> = {}): SessionUser {
  return {
    id: 'u1',
    name: 'Гриша',
    email: 'g@example.com',
    email_verified: true,
    telegram: { linked: false, username: '' },
    features: { tariffs },
    ...over,
  }
}

// The prices here are an illustrative quote, not the live ones (the server decides
// those, see migration 027): the tests only need numbers that differ and add up.
export function tariffsInfo(over: Partial<TariffsInfo> = {}): TariffsInfo {
  return {
    payments_ready: true,
    has_paid_level: false,
    level: {
      price: 249,
      regular_price: 299,
      bonus: 1000,
      next: { id: '4', title: 'Уровень 4 — Падежи в жизни' },
      // the levels a paid level can open, in course order, and the paid levels not yet spent
      choices: [
        { id: '4', title: 'Уровень 4 — Падежи в жизни', lessons: 18 },
        { id: '5', title: 'Уровень 5 — Жизнь на сербском', lessons: 18, after: 'Уровень 4 — Падежи в жизни' },
      ],
      credits: [],
    },
    year: { price: 1000, old_price: 2000, bonus: 15000 },
    ...over,
  }
}

// Turns tariffs on for the account and puts the quote in the store, as a loaded
// screen would have it.
export function enableTariffs(over: Partial<TariffsInfo> = {}, user: Partial<SessionUser> = {}) {
  useSessionStore().user = sessionUser(true, user)
  const store = useTariffsStore()
  store.info = tariffsInfo(over)
  return store
}
