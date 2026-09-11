import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import RegisterView from './RegisterView.vue'
import { useSessionStore } from '../stores/session'
import { ApiError } from '../api'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
  RouterLink: { template: '<a><slot /></a>' },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  push.mockClear()
})
afterEach(() => vi.restoreAllMocks())

describe('RegisterView', () => {
  it('registers then routes to /verify with the email', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'register').mockResolvedValue(undefined)
    const w = mount(RegisterView)
    await w.find('input[placeholder="имя"]').setValue('Аня')
    await w.find('input[type="email"]').setValue('a@example.com')
    await w.find('input[type="password"]').setValue('secret123')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(session.register).toHaveBeenCalledWith('a@example.com', 'secret123', 'Аня')
    expect(push).toHaveBeenCalledWith({ path: '/verify', query: { email: 'a@example.com' } })
  })

  it('shows a mapped error on a validation failure', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'register').mockRejectedValue(new ApiError(400, 'invalid email'))
    const w = mount(RegisterView)
    await w.find('input[placeholder="имя"]').setValue('Аня')
    await w.find('input[type="email"]').setValue('not-an-email')
    await w.find('input[type="password"]').setValue('secret123')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(w.text()).toContain('Некорректный email')
  })
})
