// Folding the levels on the course screen. A level the learner has finished is
// folded by default (there is nothing left in it to open), and any level can be
// folded or unfolded by hand. What the learner chose is remembered in this
// browser, level by level: a remembered choice always beats the default, so a
// finished level the learner unfolded stays unfolded.
import type { LessonRef, Phase } from '../types'

const PREFIX = 'course.collapsed.'

// The remembered choice for a level: true folded, false unfolded, null none yet.
// Storage can be blocked or full (private windows, embedded browsers); then
// nothing is remembered and the defaults apply, which is all that is lost.
export function readCollapsed(phaseId: string): boolean | null {
  try {
    const v = localStorage.getItem(PREFIX + phaseId)
    return v === '1' ? true : v === '0' ? false : null
  } catch {
    return null
  }
}

export function writeCollapsed(phaseId: string, collapsed: boolean): void {
  try {
    localStorage.setItem(PREFIX + phaseId, collapsed ? '1' : '0')
  } catch {
    // not remembered; see above
  }
}

// The lessons of a level that exist yet: a "скоро" lesson is not something the
// learner can have passed.
function written(phase: Phase, byId: Map<string, LessonRef>): LessonRef[] {
  return phase.lessons.map((id) => byId.get(id)).filter((l): l is LessonRef => !!l && !l.planned)
}

// A level is passed once every lesson that is written is done. A level that is
// still locked, or has nothing written, is never passed.
export function levelPassed(phase: Phase, byId: Map<string, LessonRef>): boolean {
  if (phase.locked) return false
  const lessons = written(phase, byId)
  return lessons.length > 0 && lessons.every((l) => l.status === 'done')
}

// "5 из 17": the lessons done out of the lessons written.
export function levelProgress(phase: Phase, byId: Map<string, LessonRef>): { done: number; total: number } {
  const lessons = written(phase, byId)
  return { done: lessons.filter((l) => l.status === 'done').length, total: lessons.length }
}
