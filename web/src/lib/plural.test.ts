import { describe, it, expect } from 'vitest'
import { pluralRu } from './plural'

const seed = (n: number) => `${n} ${pluralRu(n, 'зёрнышко', 'зёрнышка', 'зёрнышек')}`

describe('pluralRu', () => {
  it('picks the form by the last digit', () => {
    expect(seed(1)).toBe('1 зёрнышко')
    expect(seed(2)).toBe('2 зёрнышка')
    expect(seed(4)).toBe('4 зёрнышка')
    expect(seed(5)).toBe('5 зёрнышек')
    expect(seed(0)).toBe('0 зёрнышек')
  })

  it('uses the "many" form for the teens, whatever the last digit', () => {
    expect(seed(11)).toBe('11 зёрнышек')
    expect(seed(12)).toBe('12 зёрнышек')
    expect(seed(14)).toBe('14 зёрнышек')
  })

  it('looks past the hundreds', () => {
    expect(seed(21)).toBe('21 зёрнышко')
    expect(seed(101)).toBe('101 зёрнышко')
    expect(seed(111)).toBe('111 зёрнышек')
    expect(seed(1002)).toBe('1002 зёрнышка')
  })
})
