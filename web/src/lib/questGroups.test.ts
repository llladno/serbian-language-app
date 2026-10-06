import { describe, it, expect } from 'vitest'
import { groupQuests } from './questGroups'
import type { Quest } from '../types'

function quest(over: Partial<Quest> & { id: number; target: number }): Quest {
  return {
    kind: 'lessons_completed',
    title: `Пройти ${over.target} уроков`,
    description: '',
    value: 0,
    reward: 10,
    done: false,
    claimed: false,
    ...over,
  }
}

const ladder = (kind: string, targets: number[], claimedUpTo = 0, value = 0) =>
  targets.map((t, i) =>
    quest({ id: i + 1, kind, target: t, value, claimed: t <= claimedUpTo, done: value >= t }),
  )

describe('groupQuests', () => {
  it('shows one rung of a ladder at a time', () => {
    const g = groupQuests(ladder('lessons_completed', [5, 10, 20, 30], 5, 6))

    expect(g.ready).toHaveLength(0)
    expect(g.active.map((q) => q.target)).toEqual([10])
    expect(g.done).toHaveLength(0)
  })

  it('moves the ladder up as soon as the finished rung is claimed', () => {
    const before = groupQuests(ladder('lessons_completed', [5, 10], 0, 5))
    expect(before.ready.map((q) => q.target)).toEqual([5])

    const after = groupQuests(ladder('lessons_completed', [5, 10], 5, 5))
    expect(after.ready).toHaveLength(0)
    expect(after.active.map((q) => q.target)).toEqual([10])
  })

  it('keeps a rung the learner skipped, so its reward is not swallowed', () => {
    // Both rungs were finished; only the bigger one was claimed.
    const quests = [
      quest({ id: 1, target: 5, value: 12, done: true, claimed: false }),
      quest({ id: 2, target: 10, value: 12, done: true, claimed: true }),
    ]
    expect(groupQuests(quests).ready.map((q) => q.target)).toEqual([5])
  })

  it('reports a finished ladder once, by its top rung', () => {
    const g = groupQuests(ladder('vocab_learned', [30, 100], 100, 120))

    expect(g.ready).toHaveLength(0)
    expect(g.active).toHaveLength(0)
    expect(g.done.map((q) => q.target)).toEqual([100])
  })

  it('keeps a streak ladder on the next rung after the streak breaks', () => {
    // Seven days were reached and paid; then the streak broke, so the counter
    // is back to zero while the claim stands.
    const quests = [
      quest({ id: 1, kind: 'streak_days', target: 7, value: 0, done: true, claimed: true }),
      quest({ id: 2, kind: 'streak_days', target: 30, value: 0, done: false, claimed: false }),
    ]
    const g = groupQuests(quests)

    expect(g.active.map((q) => q.target)).toEqual([30])
    expect(g.active[0].value).toBe(0)
    expect(g.done).toHaveLength(0)
  })

  it('does not fold levels together — they are parallel, not a ladder', () => {
    const quests = [
      quest({ id: 1, kind: 'phase_completed', target: 100, value: 100, done: true, claimed: true }),
      quest({ id: 2, kind: 'phase_completed', target: 100, value: 40 }),
      quest({ id: 3, kind: 'phase_completed', target: 100, value: 0 }),
    ]
    const g = groupQuests(quests)

    expect(g.done).toHaveLength(1)
    expect(g.active).toHaveLength(2)
  })
})
