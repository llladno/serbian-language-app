import { describe, it, expect } from 'vitest'
import { CHEERS, bandFor, cheerFor } from './cheers'

describe('cheers', () => {
  it('splits the percentages into the four bands', () => {
    expect(bandFor(100)).toBe('perfect')
    expect(bandFor(99)).toBe('high')
    expect(bandFor(80)).toBe('high')
    expect(bandFor(79)).toBe('good')
    expect(bandFor(60)).toBe('good')
    expect(bandFor(59)).toBe('low')
    expect(bandFor(0)).toBe('low')
  })

  it('has five distinct phrases for every band', () => {
    for (const list of Object.values(CHEERS)) {
      expect(list).toHaveLength(5)
      expect(new Set(list).size).toBe(5)
    }
  })

  it('picks a phrase of the band, and can reach all of them', () => {
    expect(cheerFor(100, () => 0)).toBe(CHEERS.perfect[0])
    expect(cheerFor(85, () => 0.999)).toBe(CHEERS.high[4])
    expect(cheerFor(70, () => 0.5)).toBe(CHEERS.good[2])
    expect(cheerFor(10, () => 1)).toBe(CHEERS.low[4]) // never runs off the end
  })
})
