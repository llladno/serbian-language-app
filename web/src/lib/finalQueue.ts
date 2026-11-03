// The questions at the end of a story, asked as the exercises of a lesson are: a question answered wrongly goes to the end of
// the queue and comes again until it is answered right, so the story is only over once every question has been answered right.
// Only the first try of a question counts for the score and the total (the second try is practice, not a second chance at a mark).
export interface FinalQueue {
  order: number[] // indexes into the story's questions, in the order they are asked (a missed one is appended again)
  pos: number // the place in `order` that is on the screen
  tried: number[] // questions that have had their first try
  right: number // answers that were right, retries included: the progress bar follows this
  score: number // questions right on the first try
  asked: number // questions that have had a first try
}

export function startFinal(total: number): FinalQueue {
  return { order: Array.from({ length: total }, (_, i) => i), pos: 0, tried: [], right: 0, score: 0, asked: 0 }
}

/** The index of the question on the screen. */
export function currentIndex(q: FinalQueue): number {
  return q.order[q.pos] ?? 0
}

/** The question on the screen has already been missed once: it has come back. */
export function isRetry(q: FinalQueue): boolean {
  return q.order.indexOf(currentIndex(q)) < q.pos
}

/** The state after the learner answered the question on the screen; `finished` once nothing is left to ask. */
export function answerFinal(q: FinalQueue, correct: boolean): { queue: FinalQueue; finished: boolean } {
  const idx = currentIndex(q)
  const first = !q.tried.includes(idx)
  const next: FinalQueue = {
    order: correct ? q.order : [...q.order, idx],
    pos: q.pos + 1,
    tried: first ? [...q.tried, idx] : q.tried,
    right: q.right + (correct ? 1 : 0),
    score: q.score + (first && correct ? 1 : 0),
    asked: q.asked + (first ? 1 : 0),
  }
  return { queue: next, finished: next.pos >= next.order.length }
}
