import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import App from './App.vue'
import { useSessionStore } from './stores/session'

vi.mock('vue-router', () => ({
  RouterView: { template: '<div class="router-view-stub" />' },
  RouterLink: { template: '<a><slot /></a>' },
  useRoute: () => ({ path: '/profile' }),
}))

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('App', () => {
  it('shows a loading state, then the nav once a session resolves', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'fetchSession').mockImplementation(async () => {
      session.user = {
        id: '1',
        name: 'Гриша',
        email: '',
        email_verified: true,
        telegram: { linked: false, username: '' },
      }
      session.loading = false
    })
    const w = mount(App)
    expect(w.text()).toContain('Загрузка')
    await flushPromises()
    expect(w.findComponent({ name: 'AppNav' }).exists()).toBe(true)
  })

  it('renders no nav chrome when logged out', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'fetchSession').mockImplementation(async () => {
      session.user = null
      session.loading = false
    })
    const w = mount(App)
    await flushPromises()
    expect(w.findComponent({ name: 'AppNav' }).exists()).toBe(false)
  })
})
