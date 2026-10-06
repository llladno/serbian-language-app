import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import QuestsView from './QuestsView.vue'
import { api, ApiError } from '../api'
import { useRewardModal } from '../lib/rewardModal'
import type { Quest } from '../types'

function quest(over: Partial<Quest> = {}): Quest {
  return {
    id: 1,
    kind: 'lessons_completed',
    title: 'Пройти 5 уроков',
    description: 'Любые пять уроков курса',
    target: 5,
    value: 5,
    reward: 15,
    done: true,
    claimed: false,
    ...over,
  }
}

const WALLET = {
  balance: 100,
  currency_one: 'пёрышко',
  currency_few: 'пёрышка',
  currency_many: 'пёрышек',
  streak_days: 4,
}

beforeEach(() => {
  setActivePinia(createPinia())
  useRewardModal().closeModal()
  vi.spyOn(api, 'wallet').mockResolvedValue(WALLET)
})
afterEach(() => vi.restoreAllMocks())

describe('QuestsView', () => {
  it('splits quests into claimable, in progress and already paid', async () => {
    vi.spyOn(api, 'quests').mockResolvedValue({
      quests: [
        quest({ id: 1 }),
        quest({ id: 2, kind: 'vocab_learned', title: 'Выучить 30 слов', target: 30, value: 12, done: false }),
        quest({
          id: 3,
          kind: 'telegram_subscribed',
          title: 'Подписка на канал',
          target: 1,
          value: 1,
          claimed: true,
        }),
      ],
    })
    const w = mount(QuestsView)
    await flushPromises()

    expect(w.find('[data-test="wallet-balance"]').text()).toBe('100 пёрышек')
    expect(w.findAll('[data-test="quest-ready"]')).toHaveLength(1)
    expect(w.text()).toContain('Можно забрать')
    expect(w.text()).toContain('12 / 30')
    expect(w.text()).toContain('Получено')
    // A claimed quest keeps no button: the only claim button on screen is the
    // one for the quest that is actually payable.
    expect(w.findAll('button')).toHaveLength(1)
  })

  it('celebrates the reward the server paid, not the one the quest advertised', async () => {
    // The admin can change a quest's reward between the list and the claim;
    // the modal has to show what was actually credited.
    vi.spyOn(api, 'quests').mockResolvedValue({ quests: [quest()] })
    vi.spyOn(api, 'claimQuest').mockResolvedValue({ reward: 20, balance: 120 })
    const { open, reward } = useRewardModal()
    const w = mount(QuestsView)
    await flushPromises()

    await w.find('[data-test="claim-1"]').trigger('click')
    await flushPromises()

    expect(api.claimQuest).toHaveBeenCalledWith(1)
    expect(open.value).toBe(true)
    expect(reward.value).toBe(20)
    expect(w.find('[data-test="wallet-balance"]').text()).toBe('120 пёрышек')
  })

  it('reports a rejected claim and refetches the list instead of celebrating', async () => {
    const quests = vi.spyOn(api, 'quests').mockResolvedValue({ quests: [quest()] })
    vi.spyOn(api, 'claimQuest').mockRejectedValue(new ApiError(409, 'already_claimed'))
    const { open } = useRewardModal()
    const w = mount(QuestsView)
    await flushPromises()

    await w.find('[data-test="claim-1"]').trigger('click')
    await flushPromises()

    expect(open.value).toBe(false)
    expect(w.find('[data-test="claim-error"]').exists()).toBe(true)
    expect(quests).toHaveBeenCalledTimes(2)
  })
})
