import { describe, it, expect } from 'vitest'
import { nextLesson } from './nextLesson'
import type { Course } from '../types'

const ref = (id: string, planned = false) => ({ id, title: id, subtitle: '', planned, status: 'not_started' as const })
const course = (locked = false, planned = false): Course => ({
  title: 'Курс',
  phases: [
    { id: '1', title: 'Один', lessons: ['01', '02'] },
    { id: '2', title: 'Два', lessons: ['03'], locked },
  ],
  lessons: [ref('01'), ref('02'), ref('03', planned)],
})

describe('nextLesson', () => {
  it('goes to the next lesson in the same level', () => {
    expect(nextLesson(course(), '01')?.id).toBe('02')
  })
  it('crosses into the next level when it is open', () => {
    expect(nextLesson(course(), '02')?.id).toBe('03')
  })
  it('stops at a level that is not unlocked', () => {
    expect(nextLesson(course(true), '02')).toBeNull()
  })
  it('stops at a lesson that is not written yet', () => {
    expect(nextLesson(course(false, true), '02')).toBeNull()
  })
  it('has nothing after the last lesson, or before the course has loaded', () => {
    expect(nextLesson(course(), '03')).toBeNull()
    expect(nextLesson(null, '01')).toBeNull()
    expect(nextLesson(course(), 'nope')).toBeNull()
  })
})
