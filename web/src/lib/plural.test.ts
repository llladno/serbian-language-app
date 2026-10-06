import { describe, it, expect } from 'vitest'
import { pluralRu } from './plural'

const feather = (n: number) => `${n} ${pluralRu(n, 'пёрышко', 'пёрышка', 'пёрышек')}`

describe('pluralRu', () => {
  it('picks the form by the last digit', () => {
    expect(feather(1)).toBe('1 пёрышко')
    expect(feather(2)).toBe('2 пёрышка')
    expect(feather(4)).toBe('4 пёрышка')
    expect(feather(5)).toBe('5 пёрышек')
    expect(feather(0)).toBe('0 пёрышек')
  })

  it('uses the "many" form for the teens, whatever the last digit', () => {
    expect(feather(11)).toBe('11 пёрышек')
    expect(feather(12)).toBe('12 пёрышек')
    expect(feather(14)).toBe('14 пёрышек')
  })

  it('looks past the hundreds', () => {
    expect(feather(21)).toBe('21 пёрышко')
    expect(feather(101)).toBe('101 пёрышко')
    expect(feather(111)).toBe('111 пёрышек')
    expect(feather(1002)).toBe('1002 пёрышка')
  })
})
