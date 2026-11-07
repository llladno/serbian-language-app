import { describe, it, expect, beforeEach } from 'vitest'
import { claimVariantFor } from './claimVariant'

beforeEach(() => localStorage.clear())

describe('claimVariantFor', () => {
  it('gives a learner the same wording every time', () => {
    for (const id of ['u1', 'a1b2c3d4', '5f0c8a-77']) {
      expect(claimVariantFor(id)).toBe(claimVariantFor(id))
    }
  })

  it('splits ids, even similar ones, about half and half', () => {
    let buttons = 0
    const n = 2000
    for (let i = 0; i < n; i++) if (claimVariantFor(`user-${i}`) === 'button') buttons++
    expect(buttons / n).toBeGreaterThan(0.45)
    expect(buttons / n).toBeLessThan(0.55)
  })

  it('can be told which wording to show in the dev build, to look at the other one', () => {
    localStorage.setItem('srpski.claimVariant', 'link')
    expect(claimVariantFor('anyone')).toBe('link')
    localStorage.setItem('srpski.claimVariant', 'button')
    expect(claimVariantFor('anyone')).toBe('button')
    localStorage.setItem('srpski.claimVariant', 'nonsense')
    expect(['button', 'link']).toContain(claimVariantFor('anyone'))
  })
})
