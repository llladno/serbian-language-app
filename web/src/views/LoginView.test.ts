import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LoginView from './LoginView.vue'
import { useSessionStore } from '../stores/session'
import { api, ApiError } from '../api'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push }),
  RouterLink: { template: '<a><slot /></a>' },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  push.mockClear()
})
afterEach(() => vi.restoreAllMocks())

describe('LoginView', () => {
  it('logs in and navigates to /profile by default', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'login').mockResolvedValue(undefined)
    const w = mount(LoginView)
    await w.find('input[type="email"]').setValue('g@example.com')
    await w.find('input[type="password"]').setValue('secret123')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(session.login).toHaveBeenCalledWith('g@example.com', 'secret123')
    expect(push).toHaveBeenCalledWith('/profile')
  })

  it('shows the generic error message on wrong credentials', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'login').mockRejectedValue(new ApiError(401, 'неверная почта или пароль'))
    const w = mount(LoginView)
    await w.find('input[type="email"]').setValue('g@example.com')
    await w.find('input[type="password"]').setValue('wrongpass')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(w.text()).toContain('неверная почта или пароль')
  })

  it('routes to /verify with the email on email_unverified', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'login').mockRejectedValue(new ApiError(403, 'email_unverified'))
    const w = mount(LoginView)
    await w.find('input[type="email"]').setValue('g@example.com')
    await w.find('input[type="password"]').setValue('secret123')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(push).toHaveBeenCalledWith({ path: '/verify', query: { email: 'g@example.com' } })
  })

  it('shows the Telegram button once health reports a bot id', async () => {
    vi.spyOn(api, 'health').mockResolvedValue({ status: 'ok', content_stale: false, telegram_bot_id: '42' })
    const w = mount(LoginView)
    await flushPromises()
    expect(w.text()).toContain('Войти через Telegram')
  })

  it('hides the Telegram button when health reports no bot id', async () => {
    vi.spyOn(api, 'health').mockResolvedValue({ status: 'ok', content_stale: false, telegram_bot_id: '' })
    const w = mount(LoginView)
    await flushPromises()
    expect(w.text()).not.toContain('Войти через Telegram')
  })

  it('opens the bot chat, polls, and logs in once the webhook resolves the token', async () => {
    vi.useFakeTimers()
    vi.spyOn(api, 'health').mockResolvedValue({ status: 'ok', content_stale: false, telegram_bot_id: '42' })
    vi.spyOn(api, 'telegramLoginStart').mockResolvedValue({
      url: 'https://t.me/ucimoappbot?start=tok123',
      token: 'tok123',
    })
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null)
    const poll = vi
      .spyOn(api, 'telegramPoll')
      .mockResolvedValueOnce({ status: 'pending' })
      .mockResolvedValueOnce({
        status: 'ok',
        user: { id: '1', name: 'Г', email: '', email_verified: false, telegram: { linked: true, username: 'g' } },
      })

    const w = mount(LoginView)
    await flushPromises()
    await w.find('button.btn-telegram').trigger('click')
    await flushPromises()
    expect(openSpy).toHaveBeenCalledWith('https://t.me/ucimoappbot?start=tok123', '_blank')

    await vi.advanceTimersByTimeAsync(1500)
    expect(poll).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1500)
    expect(poll).toHaveBeenCalledTimes(2)
    await flushPromises()

    expect(push).toHaveBeenCalledWith('/profile')
    vi.useRealTimers()
  })
})
