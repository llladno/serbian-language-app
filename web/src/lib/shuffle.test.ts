import { describe, expect, it } from 'vitest'
import { shuffled } from './shuffle'

describe('shuffled', () => {
  it('keeps every item and does not touch the original', () => {
    const src = ['a', 'b', 'c', 'd']
    const out = shuffled(src)
    expect([...out].sort()).toEqual(['a', 'b', 'c', 'd'])
    expect(src).toEqual(['a', 'b', 'c', 'd'])
  })

  it('is deterministic for a given random source', () => {
    const seq = [0.1, 0.9, 0.5]
    let i = 0
    const rnd = () => seq[i++ % seq.length]
    expect(shuffled([1, 2, 3, 4], rnd)).toEqual(shuffled([1, 2, 3, 4], ((): (() => number) => { let k = 0; return () => seq[k++ % seq.length] })()))
  })

  it('puts the first of three options in each place about a third of the time', () => {
    const N = 30000
    const places = [0, 0, 0]
    for (let t = 0; t < N; t++) places[shuffled(['right', 'x', 'y']).indexOf('right')]++
    for (const n of places) expect(Math.abs(n / N - 1 / 3)).toBeLessThan(0.02)
  })

  it('handles empty and single lists', () => {
    expect(shuffled([])).toEqual([])
    expect(shuffled(['only'])).toEqual(['only'])
  })
})
