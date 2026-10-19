import { describe, it, expect } from 'vitest'
import { createMistakesState, keyOf, record, next, isDone, current } from './mistakesQueue'
import type { MistakeItem } from '../types'

const item = (id: string, lesson = '01'): MistakeItem => ({
  lesson,
  exercise: { id, type: 'translate', prompt: `Задание ${id}` },
})

describe('mistakes queue', () => {
  it('starts on the first exercise with nothing answered', () => {
    const s = createMistakesState([item('a'), item('b')])
    expect(current(s)?.exercise.id).toBe('a')
    expect(s.answered).toBeNull()
    expect(s.total).toBe(2)
    expect(isDone(s)).toBe(false)
  })

  it('an empty batch is done at once', () => {
    const s = createMistakesState([])
    expect(isDone(s)).toBe(true)
    expect(current(s)).toBeUndefined()
  })

  it('a right answer moves on and counts as fixed on the first try', () => {
    const s = createMistakesState([item('a'), item('b')])
    record(s, true)
    next(s)
    expect(current(s)?.exercise.id).toBe('b')
    expect(s.fixed).toBe(1)
    expect(s.fixedFirstTry).toBe(1)
    expect(s.answered).toBeNull()
  })

  it('a wrong answer sends the exercise to the end and bumps its round', () => {
    const s = createMistakesState([item('a'), item('b'), item('c')])
    record(s, false)
    next(s)
    expect(s.pending.map((m) => m.exercise.id)).toEqual(['b', 'c', 'a'])
    expect(s.rounds[keyOf(item('a'))]).toBe(1)
    expect(s.fixed).toBe(0)
  })

  it('an exercise fixed only on a later round is fixed but not first-try', () => {
    const s = createMistakesState([item('a'), item('b')])
    record(s, false)
    next(s) // a -> end: [b, a]
    record(s, true)
    next(s) // b fixed first try
    record(s, true)
    next(s) // a fixed on round 2
    expect(isDone(s)).toBe(true)
    expect(s.fixed).toBe(2)
    expect(s.fixedFirstTry).toBe(1)
  })

  it('a lone exercise answered wrong stays on screen for another round', () => {
    const s = createMistakesState([item('a')])
    record(s, false)
    next(s)
    expect(isDone(s)).toBe(false)
    expect(current(s)?.exercise.id).toBe('a')
    expect(s.rounds[keyOf(item('a'))]).toBe(1)
  })

  it('does not move on before the answer is known', () => {
    const s = createMistakesState([item('a'), item('b')])
    next(s)
    expect(current(s)?.exercise.id).toBe('a')
    expect(s.fixed).toBe(0)
  })

  it('tells apart the same exercise id in different lessons', () => {
    expect(keyOf(item('x', '01'))).not.toBe(keyOf(item('x', '02')))
  })
})
