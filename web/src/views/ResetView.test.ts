import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ResetView from './ResetView.vue'
import { useSessionStore } from '../stores/session'

let query: Record<string, string> = { token: 'abc123' }
const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ query }),
  useRouter: () => ({ push }),
  RouterLink: { template: '<a><slot /></a>' },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  query = { token: 'abc123' }
  push.mockClear()
})
afterEach(() => vi.restoreAllMocks())

describe('ResetView', () => {
  it('rejects mismatched passwords without calling the API', async () => {
    const session = useSessionStore()
    const spy = vi.spyOn(session, 'reset')
    const w = mount(ResetView)
    await w.find('input[placeholder="новый пароль"]').setValue('secret123')
    await w.find('input[placeholder="повторите пароль"]').setValue('different1')
    await w.find('form').trigger('submit.prevent')
    expect(spy).not.toHaveBeenCalled()
    expect(w.text()).toContain('не совпадают')
  })

  it('resets and routes to /profile on match', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'reset').mockResolvedValue(undefined)
    const w = mount(ResetView)
    await w.find('input[placeholder="новый пароль"]').setValue('secret123')
    await w.find('input[placeholder="повторите пароль"]').setValue('secret123')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(session.reset).toHaveBeenCalledWith('abc123', 'secret123')
    expect(push).toHaveBeenCalledWith('/profile')
  })

  it('shows an error when there is no token', () => {
    query = {}
    const w = mount(ResetView)
    expect(w.text()).toContain('недействительна')
  })
})
