import { describe, it, expect, afterEach, beforeEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TariffsCard from './TariffsCard.vue'
import { useSessionStore } from '../stores/session'
import { useTariffsStore } from '../stores/tariffs'
import { enableTariffs, sessionUser, tariffsInfo } from '../test/tariffsFixture'

enableAutoUnmount(afterEach)
const stubs = { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } }

const day = 86_400_000
const mountCard = () => mount(TariffsCard, { global: { stubs } })

beforeEach(() => setActivePinia(createPinia()))

// The card is the way back into the tariffs screen for a learner who already has a
// tariff. Everybody else gets the advert (TariffsPromo) in its place.
describe('TariffsCard', () => {
  it('points a learner with a running year at their tariff, with the end date', () => {
    const until = new Date(Date.now() + 30 * day).toISOString()
    enableTariffs({ year: { price: 1000, old_price: 2000, until } })
    const w = mountCard()
    expect(w.find('[data-test="open-tariffs"]').attributes('href')).toBe('/tariffs')
    expect(w.find('[data-test="tariffs-card"]').text()).toContain('Мой тариф')
    expect(w.find('[data-test="tariffs-card"]').text()).toContain('Год')
  })

  it('is there for a learner who paid for a level', () => {
    enableTariffs({ has_paid_level: true })
    expect(mountCard().find('[data-test="open-tariffs"]').attributes('href')).toBe('/tariffs')
  })

  it('is not there while the advert is: a learner with no tariff', () => {
    enableTariffs()
    expect(mountCard().find('[data-test="tariffs-card"]').exists()).toBe(false)
  })

  it('is not there for everybody else', () => {
    useSessionStore().user = sessionUser(false)
    useTariffsStore().info = tariffsInfo({ has_paid_level: true })
    expect(mountCard().find('[data-test="tariffs-card"]').exists()).toBe(false)
  })

  it('is not there for an account whose session predates the flag', () => {
    useSessionStore().user = { ...sessionUser(false), features: undefined }
    expect(mountCard().find('[data-test="tariffs-card"]').exists()).toBe(false)
  })
})
