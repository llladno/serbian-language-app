import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LoginView from './LoginView.vue'
import { useSessionStore } from '../stores/session'
import { ApiError } from '../api'

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
})
