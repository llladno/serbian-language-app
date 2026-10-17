import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { levelPassed, levelProgress, readCollapsed, writeCollapsed } from './courseCollapse'
import type { LessonRef, Phase } from '../types'

const lesson = (id: string, status: LessonRef['status'], planned = false): LessonRef => ({ id, title: id, subtitle: '', planned, status })
const byId = (...ls: LessonRef[]) => new Map(ls.map((l) => [l.id, l]))
const phase = (lessons: string[], over: Partial<Phase> = {}): Phase => ({ id: '1', title: 'Уровень 1', lessons, ...over })

beforeEach(() => localStorage.clear())
afterEach(() => vi.restoreAllMocks())

describe('levelPassed', () => {
  it('is true once every lesson of the level is done', () => {
    expect(levelPassed(phase(['a', 'b']), byId(lesson('a', 'done'), lesson('b', 'done')))).toBe(true)
  })

  it('is false while any lesson is not done', () => {
    expect(levelPassed(phase(['a', 'b']), byId(lesson('a', 'done'), lesson('b', 'in_progress')))).toBe(false)
    expect(levelPassed(phase(['a', 'b']), byId(lesson('a', 'done'), lesson('b', 'not_started')))).toBe(false)
  })

  // A "скоро" lesson is not written yet: it neither blocks nor counts.
  it('ignores lessons that are not written yet, but needs at least one that is', () => {
    expect(levelPassed(phase(['a', 'b']), byId(lesson('a', 'done'), lesson('b', 'not_started', true)))).toBe(true)
    expect(levelPassed(phase(['a']), byId(lesson('a', 'not_started', true)))).toBe(false)
    expect(levelPassed(phase([]), byId())).toBe(false)
  })

  it('is never true for a level the learner has not unlocked', () => {
    expect(levelPassed(phase(['a'], { locked: true }), byId(lesson('a', 'done')))).toBe(false)
  })
})

describe('levelProgress', () => {
  it('counts the lessons done out of the lessons written', () => {
    const lessons = byId(lesson('a', 'done'), lesson('b', 'in_progress'), lesson('c', 'not_started', true))
    expect(levelProgress(phase(['a', 'b', 'c']), lessons)).toEqual({ done: 1, total: 2 })
  })
})

describe('the remembered choice', () => {
  it('is none until the learner makes one', () => {
    expect(readCollapsed('1')).toBeNull()
  })

  it('remembers folded and unfolded separately for each level', () => {
    writeCollapsed('1', true)
    writeCollapsed('2', false)
    expect(readCollapsed('1')).toBe(true)
    expect(readCollapsed('2')).toBe(false)
    expect(readCollapsed('3')).toBeNull()
  })

  it('ignores a value it did not write', () => {
    localStorage.setItem('course.collapsed.1', 'maybe')
    expect(readCollapsed('1')).toBeNull()
  })

  it('does not break when storage is blocked: nothing is remembered', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    expect(readCollapsed('1')).toBeNull()
    expect(() => writeCollapsed('1', true)).not.toThrow()
  })
})
