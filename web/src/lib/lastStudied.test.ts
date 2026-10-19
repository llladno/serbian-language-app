import { describe, it, expect } from 'vitest'
import { lastStudiedLesson } from './lastStudied'
import type { LessonRef, LessonStatus } from '../types'

const lessons = (...statuses: LessonStatus[]): LessonRef[] =>
  statuses.map((status, i) => ({ id: String(i + 1).padStart(2, '0'), title: `Урок ${i + 1}`, subtitle: '', planned: false, status }))

describe('lastStudiedLesson', () => {
  it('is the lesson that is started', () => {
    expect(lastStudiedLesson(lessons('done', 'in_progress', 'not_started'))?.id).toBe('02')
  })

  it('is the furthest started lesson when several are', () => {
    expect(lastStudiedLesson(lessons('in_progress', 'done', 'in_progress', 'not_started'))?.id).toBe('03')
  })

  it('prefers a started lesson to a finished one that comes later in the course', () => {
    expect(lastStudiedLesson(lessons('in_progress', 'not_started', 'done'))?.id).toBe('01')
  })

  it('falls back to the last finished lesson when nothing is started', () => {
    expect(lastStudiedLesson(lessons('done', 'done', 'not_started', 'not_started'))?.id).toBe('02')
  })

  it('is nothing for a learner who has not begun', () => {
    expect(lastStudiedLesson(lessons('not_started', 'not_started'))).toBeUndefined()
    expect(lastStudiedLesson([])).toBeUndefined()
  })
})
