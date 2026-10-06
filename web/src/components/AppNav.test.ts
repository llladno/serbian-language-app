import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AppNav from './AppNav.vue'
import { useSupportModal } from '../lib/supportModal'
import { useRewardModal } from '../lib/rewardModal'
import { useWalletStore } from '../stores/wallet'
import { api } from '../api'
import type { Quest } from '../types'

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/profile', meta: {} }),
  RouterLink: { template: '<a><slot /></a>' },
}))

beforeEach(() => {
  // The chip counts up to its value; AnimatedNumber's own test covers the
  // counting, and here it only gets in the way of reading the number.
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: query.includes('prefers-reduced-motion'),
    media: query,
    addEventListener: () => {},
    removeEventListener: () => {},
  }))
  setActivePinia(createPinia())
  const { open, message, error, sent } = useSupportModal()
  open.value = false
  message.value = ''
  error.value = null
  sent.value = false
  useRewardModal().closeModal()
  vi.spyOn(api, 'quests').mockResolvedValue({ quests: [] })
  vi.spyOn(api, 'wallet').mockResolvedValue({
    balance: 137,
    currency_one: 'пёрышко',
    currency_few: 'пёрышка',
    currency_many: 'пёрышек',
    streak_days: 4,
  })
})
afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function mountNav() {
  // Teleport's real target (document.body) is outside the wrapper's DOM tree;
  // stubbing it keeps the modal inline so it can be found/asserted on.
  return mount(AppNav, { props: { name: 'Гриша' }, global: { stubs: { teleport: true } } })
}

describe('AppNav', () => {
  it('opens the support modal from the headphones icon', async () => {
    const w = mountNav()
    expect(w.find('textarea').exists()).toBe(false)

    await w.find('[data-test="open-support"]').trigger('click')

    expect(w.find('textarea').exists()).toBe(true)
  })

  it('shows the wallet balance in the header', async () => {
    const w = mountNav()
    await flushPromises()

    expect(w.find('[data-test="wallet-chip"]').text()).toBe('137')
  })

  it('marks the chip with a star while a finished quest is waiting', async () => {
    const finished: Quest = {
      id: 1,
      kind: 'lessons_completed',
      title: 'Пройти 5 уроков',
      description: '',
      target: 5,
      value: 5,
      reward: 15,
      done: true,
      claimed: false,
    }
    vi.mocked(api.quests).mockResolvedValue({ quests: [finished] })
    const w = mountNav()
    await flushPromises()

    expect(w.findComponent({ name: 'ClaimableStar' }).exists()).toBe(true)

    // Taking it puts the star away.
    useWalletStore().quests[0].claimed = true
    await flushPromises()
    expect(w.findComponent({ name: 'ClaimableStar' }).exists()).toBe(false)
  })

  it('offers the way to the quests only for a reward that paid itself out', async () => {
    const w = mountNav()
    await flushPromises()

    useRewardModal().celebrate(50, 'Уровень 1 на 100%', true)
    await flushPromises()
    expect(w.find('[data-test="reward-to-quests"]').exists()).toBe(true)

    useRewardModal().closeModal()
    useRewardModal().celebrate(15, 'Пройти 5 уроков')
    await flushPromises()
    expect(w.find('[data-test="reward-to-quests"]').exists()).toBe(false)
  })

  it('renders the reward modal, so a claim from any screen can celebrate', async () => {
    const w = mountNav()
    await flushPromises()
    expect(w.find('[data-test="reward-amount"]').exists()).toBe(false)

    useRewardModal().celebrate(15, 'Пройти 5 уроков')
    await flushPromises()

    expect(w.find('[data-test="reward-quest"]').text()).toBe('Пройти 5 уроков')
    expect(w.find('[data-test="reward-amount"]').text()).toBe('+15 пёрышек')
  })
})
