import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import VerifyView from './VerifyView.vue'
import { useSessionStore } from '../stores/session'
import { api } from '../api'

let query: Record<string, string> = {}
const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ query }),
  useRouter: () => ({ push }),
  RouterLink: { template: '<a><slot /></a>' },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  query = {}
  push.mockClear()
})
afterEach(() => vi.restoreAllMocks())

describe('VerifyView', () => {
  it('shows the success view for ?ok=1', () => {
    query = { ok: '1' }
    const w = mount(VerifyView)
    expect(w.text()).toContain('подтверждена')
  })

  it('shows the resend form for ?err=1', () => {
    query = { err: '1' }
    const w = mount(VerifyView)
    expect(w.text()).toContain('устарела')
  })

  it('targets the ?email= query param when there is no session', () => {
    query = { email: 'new@example.com' }
    const w = mount(VerifyView)
    expect(w.text()).toContain('new@example.com')
  })

  it('targets the logged-in email and disables resend during cooldown', async () => {
    vi.useFakeTimers()
    const session = useSessionStore()
    session.user = {
      id: '1',
      name: 'Г',
      email: 'g@example.com',
      email_verified: false,
      telegram: { linked: false, username: '' },
    }
    vi.spyOn(api, 'resendVerification').mockResolvedValue({ status: 'ok' })
    const w = mount(VerifyView)
    expect(w.text()).toContain('g@example.com')
    await w.find('button').trigger('click')
    await flushPromises()
    expect(api.resendVerification).toHaveBeenCalledWith('g@example.com')
    expect(w.find('button').attributes('disabled')).toBeDefined()
    vi.useRealTimers()
  })

  it('redirects an already-verified logged-in user away from the gate', () => {
    const session = useSessionStore()
    session.user = {
      id: '1',
      name: 'Г',
      email: 'g@example.com',
      email_verified: true,
      telegram: { linked: false, username: '' },
    }
    mount(VerifyView)
    expect(push).toHaveBeenCalledWith('/profile')
  })
})
