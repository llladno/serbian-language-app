import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { api } from '../api'
import { buildSteps, useOnboarding } from './onboarding'

vi.mock('../api', () => ({
  api: { onboarding: vi.fn(), answerOnboarding: vi.fn(), wallet: vi.fn(), quests: vi.fn() },
}))

const amount = (n: number) => `${n} зёрнышек`

beforeEach(() => {
  vi.mocked(api.onboarding).mockReset()
  vi.mocked(api.answerOnboarding).mockReset().mockResolvedValue({ reward: 5, balance: 5 })
  vi.mocked(api.wallet)
    .mockReset()
    .mockResolvedValue({ balance: 0, currency_one: 'зёрнышко', currency_few: 'зёрнышка', currency_many: 'зёрнышек', streak_days: 0 })
  vi.mocked(api.quests).mockReset().mockResolvedValue({ quests: [] })
  useOnboarding().close()
})

describe('the swallow’s lines', () => {
  it('walks the nav in the agreed order, with the reward promised first and paid last', () => {
    const steps = buildSteps('auto', 5, amount)
    expect(steps.map((s) => s.id)).toEqual(['hello', 'course', 'stories', 'words', 'quests', 'profile', 'seeds'])
    expect(steps[0].text).toContain('подарю 5 зёрнышек')
    expect(steps[6].text).toContain('Держи свои 5 зёрнышек')
    // words and the dictionary beside it are one stop
    expect(steps[3].targets).toEqual(['review', 'vocab'])
    // the profile is a stop of its own, and the seeds get a full-screen finale
    expect(steps[5].targets).toEqual(['profile'])
    expect(steps[6].finale).toBe(true)
  })

  it('takes the learner to each screen and points at the thing she talks about there', () => {
    const steps = buildSteps('auto', 5, amount)
    expect(steps.slice(1, 6).map((s) => [s.route, s.focus])).toEqual([
      ['/course', ['lesson']],
      ['/stories', ['story']],
      ['/review', ['words-card']],
      ['/quests', ['wallet']],
      ['/profile', ['progress']],
    ])
  })

  it('goes straight to the first stop when asked for from «Задания»', () => {
    expect(buildSteps('quest', 5, amount).map((s) => s.id)).toEqual(['course', 'stories', 'words', 'quests', 'profile', 'seeds'])
  })

  it('promises and pays nothing once the seeds are taken', () => {
    const steps = buildSteps('auto', 0, amount)
    expect(steps[0].text).not.toContain('подарю')
    expect(steps[6].text).not.toContain('Держи')
  })

  it('glosses the Serbian words', () => {
    const steps = buildSteps('auto', 5, amount)
    expect(steps[0].tr).toBe('Ćao — привет.')
    expect(steps[6].tr).toBe('Srećno — удачи.')
  })
})

describe('starting', () => {
  it('opens by itself when the server says the tour is due', async () => {
    vi.mocked(api.onboarding).mockResolvedValue({ due: true, reward: 5 })
    const t = useOnboarding()
    await t.checkDue()
    expect(t.open.value).toBe(true)
    expect(t.mode.value).toBe('auto')
    expect(t.isOffer.value).toBe(true)
    expect(t.step.value.id).toBe('hello')
  })

  it('stays closed when it is not due, or when the check fails', async () => {
    const t = useOnboarding()
    vi.mocked(api.onboarding).mockResolvedValue({ due: false, reward: 0 })
    await t.checkDue()
    expect(t.open.value).toBe(false)
    vi.mocked(api.onboarding).mockRejectedValue(new Error('offline'))
    await t.checkDue()
    expect(t.open.value).toBe(false)
  })

  it('starts from «Задания» without the opening offer', async () => {
    const t = useOnboarding()
    t.startFromQuest(5)
    await flushPromises()
    expect(t.open.value).toBe(true)
    expect(t.isOffer.value).toBe(false)
    expect(t.step.value.id).toBe('course')
  })
})

describe('finishing and putting off', () => {
  async function started() {
    vi.mocked(api.onboarding).mockResolvedValue({ due: true, reward: 5 })
    const t = useOnboarding()
    await t.checkDue()
    return t
  }

  it('records the tour as done on reaching the last line, once, and refreshes the balance', async () => {
    const t = await started()
    for (let i = 0; i < 6; i++) t.next()
    expect(t.isLast.value).toBe(true)
    await flushPromises()
    expect(api.answerOnboarding).toHaveBeenCalledTimes(1)
    expect(api.answerOnboarding).toHaveBeenCalledWith('done')
    expect(api.wallet).toHaveBeenCalled()
    t.next() // already on the last line: nothing more to do
    expect(api.answerOnboarding).toHaveBeenCalledTimes(1)
  })

  it('puts the tour off on request and does not undo a finished one', async () => {
    const t = await started()
    t.next()
    t.postpone()
    expect(t.open.value).toBe(false)
    expect(api.answerOnboarding).toHaveBeenCalledWith('postpone')

    const done = await started()
    for (let i = 0; i < 6; i++) done.next()
    await flushPromises()
    vi.mocked(api.answerOnboarding).mockClear()
    done.postpone()
    expect(api.answerOnboarding).not.toHaveBeenCalled()
  })

  it('closing without an answer records nothing', async () => {
    const t = await started()
    t.close()
    expect(t.open.value).toBe(false)
    expect(api.answerOnboarding).not.toHaveBeenCalled()
  })
})
