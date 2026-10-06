import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWalletStore } from './wallet'
import { api, ApiError } from '../api'
import { useToasts } from '../lib/toasts'
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

beforeEach(() => {
  setActivePinia(createPinia())
  useToasts().clearToasts()
})
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

describe('wallet store, finishing quests', () => {
  it('announces a quest that just became claimable, once', async () => {
    const quests = vi.spyOn(api, 'quests')
    quests.mockResolvedValueOnce({ quests: [quest({ value: 3, done: false })] })
    const w = useWalletStore()
    const { items } = useToasts()
    await w.loadQuests()

    expect(items.value).toHaveLength(0)

    quests.mockResolvedValue({ quests: [quest({ value: 5, done: true })] })
    await w.loadQuests()
    expect(items.value).toHaveLength(1)
    expect(items.value[0].text).toBe('Пройти 5 уроков')
    expect(items.value[0].reward).toBe(15)

    // Still finished on the next refresh, but it is no longer news.
    await w.loadQuests()
    expect(items.value).toHaveLength(1)
  })

  it('names the rung the quests screen shows, not every rung that crossed', async () => {
    // Ten in a row is finished and waiting. Twenty crosses too, but the screen
    // is still showing ten, so announcing twenty would send the learner
    // looking for a card that is not on it.
    const rung = (id: number, target: number, done: boolean) =>
      quest({ id, kind: 'correct_in_row', title: `${target} правильных подряд`, target, done })
    const quests = vi.spyOn(api, 'quests')
    quests.mockResolvedValueOnce({ quests: [rung(1, 10, true), rung(2, 20, false)] })
    const w = useWalletStore()
    await w.loadQuests()

    quests.mockResolvedValue({ quests: [rung(1, 10, true), rung(2, 20, true)] })
    await w.loadQuests()

    expect(useToasts().items.value).toHaveLength(0)
  })

  it('says nothing on the first load, however much is already finished', async () => {
    vi.spyOn(api, 'quests').mockResolvedValue({ quests: [quest({ done: true })] })
    const w = useWalletStore()
    await w.loadQuests()

    expect(useToasts().items.value).toHaveLength(0)
  })

  it('leaves levels to the modal', async () => {
    const quests = vi.spyOn(api, 'quests')
    const level = (over: Partial<Quest>) =>
      quest({ id: 9, kind: 'phase_completed', title: 'Уровень 1 на 100%', target: 100, ...over })
    quests.mockResolvedValueOnce({ quests: [level({ value: 90, done: false })] })
    const w = useWalletStore()
    await w.loadQuests()

    quests.mockResolvedValue({ quests: [level({ value: 100, done: true })] })
    await w.loadQuests()

    expect(useToasts().items.value).toHaveLength(0)
  })

  it('pays a finished level by itself and reports what it paid', async () => {
    vi.spyOn(api, 'quests').mockResolvedValue({
      quests: [
        quest({ id: 9, kind: 'phase_completed', title: 'Уровень 1 на 100%', target: 100, value: 100, reward: 50 }),
        quest({ id: 1, done: true }),
      ],
    })
    const claimQuest = vi.spyOn(api, 'claimQuest').mockResolvedValue({ reward: 50, balance: 150 })
    const w = useWalletStore()

    const paid = await w.claimFinishedPhases()

    // Only the level: the lesson quest stays for the learner to take.
    expect(claimQuest).toHaveBeenCalledTimes(1)
    expect(claimQuest).toHaveBeenCalledWith(9)
    expect(paid).toEqual({ reward: 50, title: 'Уровень 1 на 100%' })
    expect(w.balance).toBe(150)
  })

  it('pays nothing when no level is finished', async () => {
    vi.spyOn(api, 'quests').mockResolvedValue({ quests: [quest({ done: true })] })
    const claimQuest = vi.spyOn(api, 'claimQuest')
    const w = useWalletStore()

    expect(await w.claimFinishedPhases()).toBeNull()
    expect(claimQuest).not.toHaveBeenCalled()
  })
})
