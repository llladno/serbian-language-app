import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWalletStore } from './wallet'
import { api, ApiError } from '../api'
import type { Quest } from '../types'

function quest(over: Partial<Quest> = {}): Quest {
  return {
    id: 1,
    kind: 'lessons_completed',
    title: 'Пройти 5 уроков',
    description: '',
    target: 5,
    value: 5,
    reward: 15,
    done: true,
    claimed: false,
    ...over,
  }
}

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('wallet store', () => {
  it('renders amounts with the currency name the server gave it', async () => {
    vi.spyOn(api, 'wallet').mockResolvedValue({
      balance: 21,
      currency_one: 'пёрышко',
      currency_few: 'пёрышка',
      currency_many: 'пёрышек',
      streak_days: 3,
    })
    const w = useWalletStore()

    expect(w.amount(21)).toBe('21')
    await w.refresh()

    expect(w.balance).toBe(21)
    expect(w.amount(21)).toBe('21 пёрышко')
    expect(w.amount(15)).toBe('15 пёрышек')
  })

  it('takes the balance from the claim response, not from adding up the reward', async () => {
    // The daily drip can credit between two screens, so the server's number
    // wins over balance + reward.
    vi.spyOn(api, 'quests').mockResolvedValue({ quests: [quest()] })
    vi.spyOn(api, 'claimQuest').mockResolvedValue({ reward: 15, balance: 118 })
    const w = useWalletStore()
    await w.loadQuests()
    w.balance = 100

    expect(w.claimable).toBe(1)
    const reward = await w.claim(1)

    expect(reward).toBe(15)
    expect(w.balance).toBe(118)
    expect(w.quests[0].claimed).toBe(true)
    expect(w.claimable).toBe(0)
  })

  it('lets a failed claim through to the caller and leaves the quest unclaimed', async () => {
    vi.spyOn(api, 'quests').mockResolvedValue({ quests: [quest()] })
    vi.spyOn(api, 'claimQuest').mockRejectedValue(new ApiError(409, 'already_claimed'))
    const w = useWalletStore()
    await w.loadQuests()

    await expect(w.claim(1)).rejects.toThrow('already_claimed')
    expect(w.quests[0].claimed).toBe(false)
  })

  it('keeps the last known balance when the wallet request fails', async () => {
    vi.spyOn(api, 'wallet').mockRejectedValue(new ApiError(500, 'internal error'))
    const w = useWalletStore()
    w.balance = 42

    await w.refresh()

    expect(w.balance).toBe(42)
    expect(w.loaded).toBe(false)
  })
})
