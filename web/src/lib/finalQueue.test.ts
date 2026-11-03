import { describe, expect, it } from 'vitest'
import { answerFinal, currentIndex, isRetry, startFinal } from './finalQueue'

describe('the questions at the end of a story', () => {
  it('asks each question once when all are right', () => {
    let q = startFinal(3)
    const seen: number[] = []
    for (;;) {
      seen.push(currentIndex(q))
      const r = answerFinal(q, true)
      q = r.queue
      if (r.finished) break
    }
    expect(seen).toEqual([0, 1, 2])
    expect(q.score).toBe(3)
    expect(q.asked).toBe(3)
  })

  it('puts a missed question at the end and asks it again until it is right', () => {
    let q = startFinal(3)
    q = answerFinal(q, false).queue // 0 wrong
    expect(q.order).toEqual([0, 1, 2, 0])
    q = answerFinal(q, true).queue // 1
    q = answerFinal(q, true).queue // 2
    expect(currentIndex(q)).toBe(0)
    expect(isRetry(q)).toBe(true)
    let r = answerFinal(q, false) // 0 wrong again: back to the end once more
    expect(r.finished).toBe(false)
    expect(r.queue.order).toEqual([0, 1, 2, 0, 0])
    r = answerFinal(r.queue, true)
    expect(r.finished).toBe(true)
  })

  it('counts only the first try for the score', () => {
    let q = startFinal(2)
    q = answerFinal(q, false).queue // 0: wrong on the first try
    q = answerFinal(q, true).queue // 1: right
    q = answerFinal(q, true).queue // 0 again: right now
    expect(q.asked).toBe(2)
    expect(q.score).toBe(1)
    expect(q.right).toBe(2)
  })

  it('does not call a question a retry the first time', () => {
    const q = startFinal(2)
    expect(isRetry(q)).toBe(false)
    expect(isRetry(answerFinal(q, true).queue)).toBe(false)
  })

  it('finishes straight away when the last question is right even after misses elsewhere', () => {
    let q = startFinal(1)
    const r = answerFinal(q, false)
    q = r.queue
    expect(r.finished).toBe(false)
    expect(currentIndex(q)).toBe(0)
    expect(answerFinal(q, true).finished).toBe(true)
  })
})
