import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import type { RouteLocationNormalized } from 'vue-router'
import { resolveGuard, waitForFreshSession } from './router'
import { useSessionStore } from './stores/session'
import type { SessionUser } from './types'

function route(name: string, fullPath = '/' + name): RouteLocationNormalized {
  return { name, fullPath } as RouteLocationNormalized
}
function verifiedUser(overrides: Partial<SessionUser> = {}): SessionUser {
  return {
    id: '1',
    name: 'Г',
    email: 'g@x.com',
    email_verified: true,
    telegram: { linked: false, username: '' },
    ...overrides,
  }
}

beforeEach(() => setActivePinia(createPinia()))

describe('resolveGuard', () => {
  it('sends a logged-out visitor from a private route to /login with next=', () => {
    const session = useSessionStore()
    session.user = null
    expect(resolveGuard(route('profile', '/profile'), session)).toEqual({
      path: '/login',
      query: { next: '/profile' },
    })
  })

  it('lets a logged-out visitor reach every public auth route', () => {
    const session = useSessionStore()
    session.user = null
    for (const name of ['login', 'register', 'verify', 'forgot', 'reset']) {
      expect(resolveGuard(route(name), session)).toBe(true)
    }
  })

  it('sends a logged-in, verified visitor away from login/register/forgot/reset to /profile', () => {
    const session = useSessionStore()
    session.user = verifiedUser()
    for (const name of ['login', 'register', 'forgot', 'reset']) {
      expect(resolveGuard(route(name), session)).toEqual({ path: '/profile' })
    }
  })

  it('still lets a logged-in visitor reach /verify', () => {
    const session = useSessionStore()
    session.user = verifiedUser()
    expect(resolveGuard(route('verify'), session)).toBe(true)
  })

  it('sends a logged-in, unverified visitor to /verify from any other route', () => {
    const session = useSessionStore()
    session.user = verifiedUser({ email_verified: false })
    expect(resolveGuard(route('profile'), session)).toEqual({ path: '/verify' })
  })

  it('lets a logged-in, verified visitor through to a private route', () => {
    const session = useSessionStore()
    session.user = verifiedUser()
    expect(resolveGuard(route('profile'), session)).toBe(true)
  })

  it('lets a Telegram-only visitor (no email at all) through without forcing /verify', () => {
    // A Telegram-only account has email: '' and email_verified: false by
    // construction (there is no email to verify) - the gate must only fire
    // for an account that actually HAS an unconfirmed email, never merely
    // because email_verified isn't true. Regression: this account was being
    // bounced to /verify and shown a confusing "enter your email" form.
    const session = useSessionStore()
    session.user = verifiedUser({ email: '', email_verified: false, telegram: { linked: true, username: 'g' } })
    expect(resolveGuard(route('profile'), session)).toBe(true)
  })

  // Tariffs are not open to everyone: the address leads nowhere for an account
  // the server has not opened them for, and the screen never renders.
  it('sends an account without tariffs from /tariffs to /profile', () => {
    const session = useSessionStore()
    session.user = verifiedUser({ features: { tariffs: false } })
    expect(resolveGuard(route('tariffs'), session)).toEqual({ path: '/profile' })
  })

  it('also does so for a session that predates the features flag', () => {
    const session = useSessionStore()
    session.user = verifiedUser()
    expect(resolveGuard(route('tariffs'), session)).toEqual({ path: '/profile' })
  })

  it('lets an account that has tariffs through', () => {
    const session = useSessionStore()
    session.user = verifiedUser({ features: { tariffs: true } })
    expect(resolveGuard(route('tariffs'), session)).toBe(true)
  })

  it('an account with tariffs but an unverified email still goes to /verify first', () => {
    const session = useSessionStore()
    session.user = verifiedUser({ features: { tariffs: true }, email_verified: false })
    expect(resolveGuard(route('tariffs'), session)).toEqual({ path: '/verify' })
  })
})

describe('waitForFreshSession', () => {
  it('is immediate once the server has answered', async () => {
    const session = useSessionStore()
    session.refreshed = true
    await expect(waitForFreshSession(session)).resolves.toBeUndefined()
  })

  // A page painted from the cached account must not decide a closed screen
  // from that cache: it waits until the server has said who this really is.
  it('waits for the server while only the cached account is known', async () => {
    const session = useSessionStore()
    session.refreshed = false
    let done = false
    const p = waitForFreshSession(session).then(() => (done = true))
    await Promise.resolve()
    expect(done).toBe(false)
    session.refreshed = true
    await p
    expect(done).toBe(true)
  })
})
