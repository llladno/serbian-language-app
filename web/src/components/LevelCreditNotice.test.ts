import { describe, it, expect, afterEach, beforeEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LevelCreditNotice from './LevelCreditNotice.vue'
import { useSessionStore } from '../stores/session'
import { useTariffsStore } from '../stores/tariffs'
import { useLevelChoiceModal } from '../lib/levelChoiceModal'
import { enableTariffs, sessionUser, tariffsInfo } from '../test/tariffsFixture'

enableAutoUnmount(afterEach)

const waiting = { price: 299, regular_price: 299, next: null, choices: [{ id: '4', title: 'Уровень 5 — Падежи', lessons: 18 }], credits: [7] }

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => useLevelChoiceModal().closeModal())

describe('LevelCreditNotice', () => {
  it('reminds a learner who paid for a level and has not chosen it yet', () => {
    enableTariffs({ has_paid_level: true, level: waiting })
    const w = mount(LevelCreditNotice)
    expect(w.find('[data-test="level-credit"]').exists()).toBe(true)
    expect(w.text()).toContain('Оплаченный уровень ждёт вас')
  })

  it('opens the choice window from its button', async () => {
    enableTariffs({ has_paid_level: true, level: waiting })
    const w = mount(LevelCreditNotice)
    await w.find('[data-test="level-credit-choose"]').trigger('click')
    expect(useLevelChoiceModal().open.value).toBe(true)
  })

  it('is not there when nothing is waiting', () => {
    enableTariffs()
    expect(mount(LevelCreditNotice).find('[data-test="level-credit"]').exists()).toBe(false)
  })

  it('is not there for an account without tariffs, or before the quote has come', () => {
    useSessionStore().user = sessionUser(false)
    useTariffsStore().info = tariffsInfo({ has_paid_level: true, level: waiting })
    expect(mount(LevelCreditNotice).find('[data-test="level-credit"]').exists()).toBe(false)
    useSessionStore().user = sessionUser(true)
    useTariffsStore().info = null
    expect(mount(LevelCreditNotice).find('[data-test="level-credit"]').exists()).toBe(false)
  })
})
