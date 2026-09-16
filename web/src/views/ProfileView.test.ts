import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
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

function mountProfile() {
  // Teleport's real target (document.body) is outside the wrapper's DOM tree;
  // stubbing it keeps the settings modal inline so it can be found/asserted on.
  return mount(ProfileView, { global: { stubs: { teleport: true } } })
}

async function openSettings(w: VueWrapper) {
  await w.find('[aria-label="Настройки"]').trigger('click')
  await flushPromises()
}

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('ProfileView', () => {
  it('renders the account and offers to set a password for a Telegram-only account', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a')) // ProgressDashboard's own fetch; irrelevant here
    const w = mountProfile()
    await flushPromises()
    expect(w.text()).toContain('Гриша')
    await openSettings(w)
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
    const w = mountProfile()
    await flushPromises()
    await openSettings(w)
    const renameBtn = w.findAll('button').find((b) => b.text() === 'изменить имя')
    await renameBtn!.trigger('click')
    await w.find('input.field').setValue('Новое имя')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(patch).toHaveBeenCalledWith('Новое имя')
    expect(w.text()).toContain('Новое имя')
  })

  it('shows an error when logout fails', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    // ProfileView calls session.logout() (the store method), not api.logout() directly.
    const session = useSessionStore()
    vi.spyOn(session, 'logout').mockRejectedValue(new ApiError(400, 'no session'))
    const w = mountProfile()
    await flushPromises()
    await openSettings(w)
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

    const w = mountProfile()
    await flushPromises()
    await openSettings(w)
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
})
