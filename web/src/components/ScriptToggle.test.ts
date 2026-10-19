import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ScriptToggle from './ScriptToggle.vue'
import { api } from '../api'
import { useSessionStore } from '../stores/session'
import type { SessionUser } from '../types'

vi.mock('../telegram', () => ({ isTelegram: vi.fn(), initData: vi.fn() }))

function readIn(script?: 'latin' | 'cyrillic') {
  useSessionStore().user = {
    id: 'u', name: 'Гриша', email: '', email_verified: true,
    telegram: { linked: false, username: '' }, script,
  } as SessionUser
}
const user = (script: 'latin' | 'cyrillic') =>
  ({ id: 'u', name: 'Гриша', email: '', email_verified: true, telegram: { linked: false, username: '' }, script }) as SessionUser

afterEach(() => vi.restoreAllMocks())

describe('ScriptToggle', () => {
  it('shows both scripts and marks the learner’s one', () => {
    readIn('cyrillic')
    const w = mount(ScriptToggle)
    expect(w.find('[data-test="script-latin"]').attributes('aria-pressed')).toBe('false')
    expect(w.find('[data-test="script-cyrillic"]').attributes('aria-pressed')).toBe('true')
  })

  it('treats an account with no choice as Latin', () => {
    readIn(undefined)
    const w = mount(ScriptToggle)
    expect(w.find('[data-test="script-latin"]').attributes('aria-pressed')).toBe('true')
  })

  it('switches to the other script through the account', async () => {
    readIn('latin')
    const spy = vi.spyOn(api, 'setScript').mockResolvedValue(user('cyrillic'))
    const w = mount(ScriptToggle)
    await w.find('[data-test="script-cyrillic"]').trigger('click')
    await flushPromises()
    expect(spy).toHaveBeenCalledWith('cyrillic')
    expect(w.find('[data-test="script-cyrillic"]').attributes('aria-pressed')).toBe('true')
  })

  it('does nothing when the learner taps the script they already have', async () => {
    readIn('latin')
    const spy = vi.spyOn(api, 'setScript').mockResolvedValue(user('latin'))
    const w = mount(ScriptToggle)
    await w.find('[data-test="script-latin"]').trigger('click')
    expect(spy).not.toHaveBeenCalled()
  })

  it('goes back to what it was when the server does not take the change', async () => {
    readIn('latin')
    vi.spyOn(api, 'setScript').mockRejectedValue(new Error('offline'))
    const w = mount(ScriptToggle)
    await w.find('[data-test="script-cyrillic"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="script-latin"]').attributes('aria-pressed')).toBe('true')
  })

  it('has a short form for headers and a full-word form for settings', () => {
    readIn('latin')
    const compact = mount(ScriptToggle, { props: { compact: true } })
    expect(compact.find('[data-test="script-latin"]').text()).toBe('Лат')
    expect(compact.find('[data-test="script-cyrillic"]').text()).toBe('Кир')
    const full = mount(ScriptToggle)
    expect(full.find('[data-test="script-latin"]').text()).toBe('Латиница')
    expect(full.find('[data-test="script-cyrillic"]').text()).toBe('Кириллица')
  })

  it('slides one highlight between the two scripts instead of re-colouring them', async () => {
    readIn('latin')
    const w = mount(ScriptToggle)
    const thumb = w.find('[data-test="script-thumb"]')
    expect(thumb.exists()).toBe(true)
    expect(thumb.classes()).not.toContain('translate-x-full')
    readIn('cyrillic')
    await w.vm.$nextTick()
    expect(thumb.classes()).toContain('translate-x-full')
    readIn('latin')
    await w.vm.$nextTick()
    expect(thumb.classes()).not.toContain('translate-x-full')
  })

  it('animates the slide, and not for someone who asked for less motion', () => {
    readIn('latin')
    const thumb = mount(ScriptToggle).find('[data-test="script-thumb"]')
    expect(thumb.classes()).toContain('transition-transform')
    expect(thumb.classes()).toContain('motion-reduce:transition-none')
  })

  it('leaves the highlight to the thumb: the buttons have no background of their own', () => {
    readIn('cyrillic')
    const w = mount(ScriptToggle)
    for (const b of w.findAll('button')) expect(b.classes().join(' ')).not.toContain('bg-[var(--accent)]')
  })

  it('does not slide the highlight out of the control in either form', () => {
    readIn('cyrillic')
    for (const compact of [true, false]) {
      const w = mount(ScriptToggle, { props: { compact } })
      expect(w.find('[data-test="script-thumb"]').element.parentElement).toBe(w.element)
    }
  })
})
