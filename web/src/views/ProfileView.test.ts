import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ProfileView from './ProfileView.vue'
import { api, ApiError } from '../api'
import { useSessionStore } from '../stores/session'
import type { Me } from '../types'

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  RouterLink: { template: '<a><slot /></a>' },
}))
vi.stubGlobal('confirm', () => true)

function me(overrides: Partial<Me> = {}): Me {
  return {
    id: '1',
    name: 'Гриша',
    email: '',
    email_verified: false,
    telegram: { linked: true, username: 'llladnooo' },
    sessions: [{ id: 'abc123456789', user_agent: 'Chrome', last_seen_at: '', current: true }],
    ...overrides,
  }
}

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('ProfileView', () => {
  it('renders the account and offers to set a password for a Telegram-only account', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a')) // ProgressDashboard's own fetch; irrelevant here
    const w = mount(ProfileView)
    await flushPromises()
    expect(w.text()).toContain('Гриша')
    expect(w.text()).toContain('Задать пароль')
  })

  it('renames the account via patchMe', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    const patch = vi.spyOn(api, 'patchMe').mockResolvedValue({
      id: '1',
      name: 'Новое имя',
      email: '',
      email_verified: false,
      telegram: { linked: true, username: 'llladnooo' },
    })
    const w = mount(ProfileView)
    await flushPromises()
    await w.find('button').trigger('click') // "изменить"
    await w.find('input.field').setValue('Новое имя')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(patch).toHaveBeenCalledWith('Новое имя')
    expect(w.text()).toContain('Новое имя')
  })

  it('revokes a device session', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    const del = vi.spyOn(api, 'deleteSession').mockResolvedValue(undefined)
    const w = mount(ProfileView)
    await flushPromises()
    const revoke = w.findAll('button').find((b) => b.text() === 'выйти')
    await revoke!.trigger('click')
    await flushPromises()
    expect(del).toHaveBeenCalledWith('abc123456789')
  })

  it('deletes the account and clears the session', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me({ email: 'g@example.com', email_verified: true }))
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    const del = vi.spyOn(api, 'deleteMe').mockResolvedValue(undefined)
    const w = mount(ProfileView)
    await flushPromises()
    await w.find('input[placeholder="пароль для подтверждения"]').setValue('secret123')
    const buttons = w.findAll('button')
    await buttons[buttons.length - 1].trigger('click')
    await flushPromises()
    expect(del).toHaveBeenCalledWith('secret123')
  })

  it('shows an error when revoking a session fails', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    vi.spyOn(api, 'deleteSession').mockRejectedValue(new ApiError(400, 'no session'))
    const w = mount(ProfileView)
    await flushPromises()
    const revoke = w.findAll('button').find((b) => b.text() === 'выйти')
    await revoke!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Сессия истекла — войдите снова')
  })

  it('shows an error when logout fails', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    // ProfileView calls session.logout() (the store method), not api.logout() directly.
    const session = useSessionStore()
    vi.spyOn(session, 'logout').mockRejectedValue(new ApiError(400, 'no session'))
    const w = mount(ProfileView)
    await flushPromises()
    const logoutBtn = w.findAll('button').find((b) => b.text() === 'Выйти')
    await logoutBtn!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Сессия истекла — войдите снова')
  })

  it('links Telegram via the /start flow and refreshes the profile', async () => {
    vi.useFakeTimers()
    vi.spyOn(api, 'getMe')
      .mockResolvedValueOnce(me({ telegram: { linked: false, username: '' } }))
      .mockResolvedValueOnce(me({ telegram: { linked: true, username: 'newlink' } }))
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    vi.spyOn(api, 'telegramLinkStart').mockResolvedValue({
      url: 'https://t.me/ucimoappbot?start=linktok',
      token: 'linktok',
    })
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null)
    const poll = vi
      .spyOn(api, 'telegramPoll')
      .mockResolvedValueOnce({ status: 'pending' })
      .mockResolvedValueOnce({ status: 'ok' })

    const w = mount(ProfileView)
    await flushPromises()
    expect(w.text()).toContain('привязать')

    const linkBtn = w.findAll('button').find((b) => b.text() === 'привязать')
    await linkBtn!.trigger('click')
    await flushPromises()
    expect(openSpy).toHaveBeenCalledWith('https://t.me/ucimoappbot?start=linktok', '_blank')

    await vi.advanceTimersByTimeAsync(1500)
    expect(poll).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1500)
    await flushPromises()

    expect(api.getMe).toHaveBeenCalledTimes(2) // initial load + refresh after link
    expect(w.text()).toContain('newlink')
    vi.useRealTimers()
  })

  it('shows an error when resetting exercise progress fails', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    vi.spyOn(api, 'resetExercises').mockRejectedValue(new ApiError(400, 'internal error'))
    const w = mount(ProfileView)
    await flushPromises()
    const resetBtn = w.findAll('button').find((b) => b.text().includes('сбросить прогресс'))
    await resetBtn!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Что-то пошло не так, попробуйте ещё раз')
  })
})
