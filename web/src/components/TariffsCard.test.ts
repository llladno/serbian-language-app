import { describe, it, expect, afterEach, beforeEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TariffsCard from './TariffsCard.vue'
import { useSessionStore } from '../stores/session'
import { sessionUser } from '../test/tariffsFixture'

enableAutoUnmount(afterEach)
const stubs = { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } }

beforeEach(() => setActivePinia(createPinia()))

describe('TariffsCard', () => {
  it('is the way into the tariffs screen for an account that has it', () => {
    useSessionStore().user = sessionUser(true)
    const w = mount(TariffsCard, { global: { stubs } })
    expect(w.find('[data-test="open-tariffs"]').attributes('href')).toBe('/tariffs')
  })

  it('is not there for everybody else', () => {
    useSessionStore().user = sessionUser(false)
    expect(mount(TariffsCard, { global: { stubs } }).find('[data-test="tariffs-card"]').exists()).toBe(false)
  })

  it('is not there for an account whose session predates the flag', () => {
    useSessionStore().user = { ...sessionUser(false), features: undefined }
    expect(mount(TariffsCard, { global: { stubs } }).find('[data-test="tariffs-card"]').exists()).toBe(false)
  })
})
