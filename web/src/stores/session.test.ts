import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from './session'
import { api, ApiError } from '../api'
import type { SessionUser } from '../types'
import { isTelegram, initData } from '../telegram'

vi.mock('../telegram', () => ({ isTelegram: vi.fn(), initData: vi.fn() }))

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

beforeEach(() => {
  localStorage.clear()
  setActivePinia(createPinia())
})
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

  it('starts from a cached guess without waiting, and refreshes it in the background', async () => {
    localStorage.setItem('ucimo_session_user', JSON.stringify(user({ name: 'Cached' })))
    const s = useSessionStore()
    expect(s.loading).toBe(false) // no wait — the cached guess is shown right away
    expect(s.user?.name).toBe('Cached')

    vi.spyOn(api, 'session').mockResolvedValue(user({ name: 'Fresh' }))
    await s.fetchSession()
    expect(s.user?.name).toBe('Fresh')
    expect(JSON.parse(localStorage.getItem('ucimo_session_user')!).name).toBe('Fresh')
  })

  it('redirects to /login when a cached guess turns out to be stale', async () => {
    localStorage.setItem('ucimo_session_user', JSON.stringify(user()))
    vi.spyOn(api, 'session').mockRejectedValue(new ApiError(401, 'no session'))
    const { default: router } = await import('../router')
    const pushSpy = vi.spyOn(router, 'push')

    const s = useSessionStore()
    await s.fetchSession()
    expect(s.user).toBeNull()
    expect(localStorage.getItem('ucimo_session_user')).toBeNull()
    expect(pushSpy).toHaveBeenCalledWith(expect.objectContaining({ path: '/login' }))
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

  it('silently tries Telegram Mini App auto-login when session() fails inside Telegram', async () => {
    vi.mocked(isTelegram).mockReturnValue(true)
    vi.mocked(initData).mockReturnValue('query_id=AA&user=%7B%22id%22%3A1%7D')
    vi.spyOn(api, 'session').mockRejectedValue(new ApiError(401, 'no session'))
    vi.spyOn(api, 'telegramLogin').mockResolvedValue(user({ name: 'TG User' }))
    const s = useSessionStore()
    await s.fetchSession()
    expect(api.telegramLogin).toHaveBeenCalledWith({ init_data: 'query_id=AA&user=%7B%22id%22%3A1%7D' })
    expect(s.user?.name).toBe('TG User')
  })

  it('does not attempt telegram auto-login outside Telegram', async () => {
    vi.mocked(isTelegram).mockReturnValue(false)
    vi.mocked(initData).mockReturnValue('')
    vi.spyOn(api, 'session').mockRejectedValue(new ApiError(401, 'no session'))
    const spy = vi.spyOn(api, 'telegramLogin')
    const s = useSessionStore()
    await s.fetchSession()
    expect(spy).not.toHaveBeenCalled()
    expect(s.user).toBeNull()
  })
})
