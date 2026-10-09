import type { Course, LessonRef } from '../types'

// The lesson that follows `id` in course order, across level boundaries — or
// null when there is nothing to go on to: the last lesson of the course, a
// lesson that is not written yet, or one inside a level the learner has not
// unlocked. The end-of-lesson screen then offers only the way back to the
// course instead of a button that leads to a wall.
export function nextLesson(course: Course | null, id: string): LessonRef | null {
  if (!course) return null
  const order: { id: string; locked: boolean }[] = []
  for (const p of course.phases) {
    for (const lid of p.lessons) order.push({ id: lid, locked: !!p.locked })
  }
  const at = order.findIndex((l) => l.id === id)
  const next = at === -1 ? undefined : order[at + 1]
  if (!next || next.locked) return null
  const ref = course.lessons.find((l) => l.id === next.id)
  return ref && !ref.planned ? ref : null
}
