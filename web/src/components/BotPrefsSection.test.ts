import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import BotPrefsSection from './BotPrefsSection.vue'
import { api, ApiError } from '../api'
import { useSessionStore } from '../stores/session'
import type { SessionUser } from '../types'

enableAutoUnmount(afterEach)

function user(linked: boolean): SessionUser {
  return { id: 'u1', name: 'Гриша', email: 'g@example.com', email_verified: true, telegram: { linked, username: linked ? 'grisha' : '' } }
}

function mountFor(linked: boolean, props: Record<string, unknown> = {}) {
  useSessionStore().user = user(linked)
  return mount(BotPrefsSection, { props })
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.spyOn(api, 'botPrefs').mockResolvedValue({ learning: true, billing: true, news: false })
})
afterEach(() => vi.restoreAllMocks())

describe('BotPrefsSection', () => {
  it('without a linked Telegram there is nothing to switch, and it says why', async () => {
    const w = mountFor(false)
    await flushPromises()
    expect(w.find('[data-test="bot-prefs-unlinked"]').exists()).toBe(true)
    expect(w.find('[data-test="pref-learning"]').exists()).toBe(false)
    expect(api.botPrefs).not.toHaveBeenCalled()
  })

  it('shows what the account has: study and tariff on, news off until agreed', async () => {
    const w = mountFor(true)
    await flushPromises()
    const on = (k: string) => (w.get(`[data-test="pref-${k}"]`).element as HTMLInputElement).checked
    expect([on('learning'), on('billing'), on('news')]).toEqual([true, true, false])
  })

  it('switching one off sends only that one', async () => {
    const save = vi.spyOn(api, 'setBotPrefs').mockResolvedValue({ learning: false, billing: true, news: false })
    const w = mountFor(true)
    await flushPromises()
    await w.get('[data-test="pref-learning"]').setValue(false)
    await flushPromises()
    expect(save).toHaveBeenCalledWith({ learning: false })
    expect((w.get('[data-test="pref-learning"]').element as HTMLInputElement).checked).toBe(false)
  })

  it('puts the switch back and says so when the save fails', async () => {
    vi.spyOn(api, 'setBotPrefs').mockRejectedValue(new ApiError(500, 'internal error'))
    const w = mountFor(true)
    await flushPromises()
    await w.get('[data-test="pref-learning"]').setValue(false)
    await flushPromises()
    expect((w.get('[data-test="pref-learning"]').element as HTMLInputElement).checked).toBe(true)
    expect(w.get('[data-test="bot-prefs-error"]').text()).toContain('Не получилось сохранить')
  })

  it('can show a single switch, for the tariff screen', async () => {
    const w = mountFor(true, { only: ['billing'] })
    await flushPromises()
    expect(w.find('[data-test="pref-billing"]').exists()).toBe(true)
    expect(w.find('[data-test="pref-learning"]').exists()).toBe(false)
    expect(w.find('[data-test="pref-news"]').exists()).toBe(false)
  })

  it('tells the learner that sign-in and payment messages are not optional, and about /stop', async () => {
    const w = mountFor(true)
    await flushPromises()
    expect(w.text()).toContain('Вход и оплату бот пишет всегда')
    expect(w.text()).toContain('/stop')
  })
})
