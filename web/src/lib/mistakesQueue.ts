// The order of play on the mistakes screen. A wrong answer is not a dead end
// and not a free pass: the exercise goes to the back of the line and comes round
// again until it is answered right. The session ends when the line is empty.
// Plain data and plain functions, so the rules can be tested without a browser.
import type { MistakeItem } from '../types'

export interface MistakesState {
  // Still to fix; pending[0] is the one on screen.
  pending: MistakeItem[]
  // The result for pending[0], or null while it has not been answered.
  answered: boolean | null
  // How many times each exercise went round again, by keyOf().
  rounds: Record<string, number>
  total: number
  fixed: number
  fixedFirstTry: number
}

// An exercise id is only unique within its lesson.
export const keyOf = (m: MistakeItem) => `${m.lesson}/${m.exercise.id}`

export function createMistakesState(items: MistakeItem[]): MistakesState {
  return { pending: [...items], answered: null, rounds: {}, total: items.length, fixed: 0, fixedFirstTry: 0 }
}

export const current = (s: MistakesState): MistakeItem | undefined => s.pending[0]

export const isDone = (s: MistakesState) => s.pending.length === 0

export function record(s: MistakesState, ok: boolean) {
  s.answered = ok
}

// Moves past the exercise on screen: a right answer retires it, a wrong one
// sends it to the back. Does nothing until the exercise has been answered.
export function next(s: MistakesState) {
  const it = s.pending[0]
  if (!it || s.answered === null) return
  s.pending.shift()
  const key = keyOf(it)
  if (s.answered) {
    s.fixed++
    if (!s.rounds[key]) s.fixedFirstTry++
  } else {
    s.rounds[key] = (s.rounds[key] ?? 0) + 1
    s.pending.push(it)
  }
  s.answered = null
}
