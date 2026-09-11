import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from './session'
import { api, ApiError } from '../api'
import type { SessionUser } from '../types'

function user(overrides: Partial<SessionUser> = {}): SessionUser {
  return {
    id: 'usr_1',
    name: 'Гриша',
    email: 'g@example.com',
    email_verified: true,
    telegram: { linked: false, username: '' },
    ...overrides,
  }
}

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('useSessionStore', () => {
  it('fetchSession fills user on success', async () => {
    vi.spyOn(api, 'session').mockResolvedValue(user())
    const s = useSessionStore()
    expect(s.loading).toBe(true)
    await s.fetchSession()
    expect(s.loading).toBe(false)
    expect(s.user?.name).toBe('Гриша')
  })

  it('fetchSession clears user on a 401', async () => {
    vi.spyOn(api, 'session').mockRejectedValue(new ApiError(401, 'no session'))
    const s = useSessionStore()
    await s.fetchSession()
    expect(s.user).toBeNull()
    expect(s.loading).toBe(false)
  })

  it('login sets user from the response', async () => {
    vi.spyOn(api, 'login').mockResolvedValue(user({ name: 'Алина' }))
    const s = useSessionStore()
    await s.login('a@example.com', 'secret123')
    expect(s.user?.name).toBe('Алина')
  })

  it('logout clears user', async () => {
    vi.spyOn(api, 'logout').mockResolvedValue(undefined)
    const s = useSessionStore()
    s.user = user()
    await s.logout()
    expect(s.user).toBeNull()
  })

  it('register does not touch user state', async () => {
    vi.spyOn(api, 'register').mockResolvedValue({ status: 'ok' })
    const s = useSessionStore()
    await s.register('a@example.com', 'secret123', 'Аня')
    expect(s.user).toBeNull()
  })

  it('reset autologs in from the response', async () => {
    vi.spyOn(api, 'resetPassword').mockResolvedValue(user({ name: 'Гриша' }))
    const s = useSessionStore()
    await s.reset('tok', 'newpass123')
    expect(s.user?.name).toBe('Гриша')
  })
})
