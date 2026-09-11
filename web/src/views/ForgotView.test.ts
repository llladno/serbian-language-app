import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ForgotView from './ForgotView.vue'
import { useSessionStore } from '../stores/session'

vi.mock('vue-router', () => ({ RouterLink: { template: '<a><slot /></a>' } }))

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('ForgotView', () => {
  it('shows the generic confirmation after submit, regardless of account existence', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'forgot').mockResolvedValue(undefined)
    const w = mount(ForgotView)
    await w.find('input[type="email"]').setValue('g@example.com')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(session.forgot).toHaveBeenCalledWith('g@example.com')
    expect(w.text()).toContain('отправлено')
  })
})
