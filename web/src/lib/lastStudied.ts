// Where the learner was last in the course: the furthest lesson they have started,
// or, with none started, the last one they finished. Statuses are all the course
// screen has (there is no "last opened" time), so this goes by course order.
import type { LessonRef, LessonStatus } from '../types'

export function lastStudiedLesson(lessons: LessonRef[]): LessonRef | undefined {
  const last = (status: LessonStatus) => [...lessons].reverse().find((l) => l.status === status)
  return last('in_progress') ?? last('done')
}
