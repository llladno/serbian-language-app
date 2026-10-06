// Most quests come as a ladder: 5 → 10 → 20 → 30 lessons, 30 → 100 → 300 words.
// Showing every rung at once turns the screen into a wall of the same sentence,
// so a ladder gets one card — the rung the learner is on — and the next rung
// takes its place the moment this one is paid.
import type { Quest } from '../types'

// Kinds whose rows differ only by target. phase_completed is deliberately not
// here: levels run in parallel, each one is its own goal. telegram_subscribed
// has a single row anyway.
const LADDER_KINDS = new Set([
  'lessons_completed',
  'vocab_learned',
  'reviews_done',
  'streak_days',
  'correct_in_row',
])

export interface QuestGroups {
  /** Finished and not yet taken. */
  ready: Quest[]
  /** The rung in progress. */
  active: Quest[]
  /** Ladders with nothing left, and one-off quests already paid. */
  done: Quest[]
}

export function groupQuests(quests: Quest[]): QuestGroups {
  const families = new Map<string, Quest[]>()
  for (const q of quests) {
    // Everything outside a ladder is a family of one.
    const key = LADDER_KINDS.has(q.kind) ? q.kind : `${q.kind}:${q.id}`
    const family = families.get(key)
    if (family) family.push(q)
    else families.set(key, [q])
  }

  const groups: QuestGroups = { ready: [], active: [], done: [] }
  for (const family of families.values()) {
    const rungs = [...family].sort((a, b) => a.target - b.target)
    // The lowest unclaimed rung, even when a higher one is already paid: the
    // learner can claim rungs out of order, and skipping the one they left
    // behind would quietly swallow its reward.
    const current = rungs.find((q) => !q.claimed)
    if (!current) {
      groups.done.push(rungs[rungs.length - 1])
      continue
    }
    if (current.done) groups.ready.push(current)
    else groups.active.push(current)
  }
  return groups
}
