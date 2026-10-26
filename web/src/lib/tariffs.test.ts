import { describe, it, expect } from 'vitest'
import { daysLeft, discountPercent, grouped, looksLikeContact, monthlyYearCost, nextLockedLevel, offerForPhase, perMonth, rub, shortDate, yearSaving } from './tariffs'
import type { Course } from '../types'

function course(over: Partial<Course> = {}): Course {
  return {
    title: 'Курс',
    phases: [
      { id: '3', title: 'Уровень 3', lessons: ['30'] },
      { id: '4', title: 'Уровень 4', lessons: ['42', '43'], locked: true },
      { id: '5', title: 'Уровень 5', lessons: ['60', '61'], locked: true },
    ],
    lessons: [
      { id: '30', title: 'a', subtitle: '', planned: false, status: 'not_started' },
      { id: '42', title: 'b', subtitle: '', planned: false, status: 'not_started' },
      { id: '43', title: 'c', subtitle: '', planned: false, status: 'not_started' },
      { id: '60', title: 'd', subtitle: '', planned: false, status: 'not_started' },
      { id: '61', title: 'e', subtitle: '', planned: false, status: 'not_started' },
    ],
    ...over,
  }
}

describe('nextLockedLevel', () => {
  it('is the first locked level in course order', () => {
    expect(nextLockedLevel(course())?.id).toBe('4')
  })

  it('skips a level that is already open and takes the next locked one', () => {
    const c = course()
    c.phases[1].locked = false // level 4 bought
    expect(nextLockedLevel(c)?.id).toBe('5')
  })

  it('skips a locked level whose lessons are all still "скоро" (not for sale)', () => {
    const c = course()
    for (const l of c.lessons) if (['42', '43'].includes(l.id)) l.planned = true
    expect(nextLockedLevel(c)?.id).toBe('5')
  })

  it('is null once everything on sale is open, or before the course has loaded', () => {
    const c = course()
    c.phases.forEach((p) => (p.locked = false))
    expect(nextLockedLevel(c)).toBeNull()
    expect(nextLockedLevel(null)).toBeNull()
  })
})

describe('rub and perMonth', () => {
  it('keeps the amount and the sign on one line', () => {
    expect(rub(249)).toBe('249 ₽')
    expect(rub(1000)).toBe('1 000 ₽')
  })

  it('rounds the year to whole rubles a month', () => {
    expect(perMonth(1000)).toBe(83)
    expect(perMonth(2000)).toBe(167)
  })
})

// What the server quotes the learner: the level at the first-purchase price.
const QUOTE = { levelPrice: 249 }

describe('offerForPhase', () => {
  it('offers the level itself when it is the next one to open', () => {
    const o = offerForPhase(course(), '4', QUOTE)
    expect(o).toMatchObject({ kind: 'level', price: 249 })
    expect(o?.level.id).toBe('4')
  })

  it('quotes whatever the server priced the level at (the regular price after a first purchase)', () => {
    expect(offerForPhase(course(), '4', { levelPrice: 299 })).toMatchObject({ kind: 'level', price: 299 })
  })

  // «Уровень» is a credit the learner spends on any closed level, so a later level is offered it too.
  it('offers the level for a later one as well, because the learner chooses which to open', () => {
    const o = offerForPhase(course(), '5', QUOTE)
    expect(o).toMatchObject({ kind: 'level', price: 249 })
    expect(o?.level.id).toBe('5')
  })

  it('offers nothing for a closed level that is not for sale yet (all its lessons are "скоро")', () => {
    const c = course()
    for (const l of c.lessons) if (['60', '61'].includes(l.id)) l.planned = true
    expect(offerForPhase(c, '5', QUOTE)).toBeNull()
  })

  it('offers nothing for a level that is already open, or that does not exist', () => {
    expect(offerForPhase(course(), '3', QUOTE)).toBeNull()
    expect(offerForPhase(course(), '99', QUOTE)).toBeNull()
    expect(offerForPhase(null, '4', QUOTE)).toBeNull()
  })
})

describe('daysLeft', () => {
  const now = new Date('2026-10-14T12:00:00Z')
  it('counts a started day as a day', () => {
    expect(daysLeft('2026-10-15T00:00:00Z', now)).toBe(1)
    expect(daysLeft('2026-11-13T12:00:00Z', now)).toBe(30)
  })
  it('never goes below zero', () => {
    expect(daysLeft('2026-10-01T00:00:00Z', now)).toBe(0)
  })
})

describe('discountPercent', () => {
  it('is the share taken off, in whole percent', () => {
    expect(discountPercent(249, 299)).toBe(17) // 16.7
    expect(discountPercent(230, 375)).toBe(39)
    expect(discountPercent(150, 300)).toBe(50)
  })
  it('is 0 when there is no discount, so it doubles as the "show a badge" test', () => {
    expect(discountPercent(299, 299)).toBe(0)
    expect(discountPercent(300, 299)).toBe(0)
    expect(discountPercent(100, 0)).toBe(0)
  })
})

describe('the year against twelve months of levels', () => {
  it('costs the first level at this learner\'s price and eleven at the regular one', () => {
    expect(monthlyYearCost(249, 299)).toBe(3538)
    expect(monthlyYearCost(299, 299)).toBe(3588) // after the first purchase every month is 299
  })

  it('saves the difference, as rubles and as a rounded percent', () => {
    expect(yearSaving(1000, 3538)).toEqual({ rub: 2538, percent: 72 })
    expect(yearSaving(1000, 3588)).toEqual({ rub: 2588, percent: 72 })
    expect(yearSaving(1000, 1500)).toEqual({ rub: 500, percent: 33 })
  })

  it('says nothing when the year is not cheaper, instead of claiming a saving', () => {
    expect(yearSaving(1000, 1000)).toBeNull()
    expect(yearSaving(1200, 1000)).toBeNull()
    expect(yearSaving(1000, 0)).toBeNull()
  })
})

// «Уровень» is always the first thing offered. While the monthly limit holds a level back it is still
// offered, with the date it can be bought.
describe('offerForPhase while the monthly limit holds a level back', () => {
  const FROM = '2026-11-12T10:00:00Z'

  it('still offers the level, and carries the date it can be bought', () => {
    expect(offerForPhase(course(), '4', { ...QUOTE, levelFrom: FROM })).toEqual({
      kind: 'level', level: expect.objectContaining({ id: '4' }), price: 249, from: FROM,
    })
  })

  it('does the same for a later level: the limit holds back every level, not only the next', () => {
    expect(offerForPhase(course(), '5', { ...QUOTE, levelFrom: FROM })).toMatchObject({ kind: 'level', from: FROM })
  })

  it('has no date when nothing holds the learner back', () => {
    const o = offerForPhase(course(), '4', { ...QUOTE, levelFrom: null })
    expect(o).toMatchObject({ kind: 'level', price: 249 })
    expect(o && 'from' in o).toBe(false)
  })
})

// A paid level that has not been spent yet: another cannot be bought, so the way forward is to choose.
describe('offerForPhase while a paid level waits to be spent', () => {
  it('points at the paid level instead of selling another one', () => {
    expect(offerForPhase(course(), '4', { ...QUOTE, hasCredit: true })).toMatchObject({ kind: 'credit' })
    expect(offerForPhase(course(), '5', { ...QUOTE, hasCredit: true })).toMatchObject({ kind: 'credit' })
  })

  it('still has nothing to say about a level that is open', () => {
    expect(offerForPhase(course(), '3', { ...QUOTE, hasCredit: true })).toBeNull()
  })

  it('goes before the monthly limit, which the waiting level itself started', () => {
    const FROM = '2026-11-12T10:00:00Z'
    expect(offerForPhase(course(), '4', { ...QUOTE, hasCredit: true, levelFrom: FROM })).toMatchObject({ kind: 'credit' })
  })
})

describe('shortDate', () => {
  it('writes the day and the month in words', () => {
    expect(shortDate('2026-11-12T10:00:00Z')).toBe('12 ноября')
  })
})

describe('grouped', () => {
  it('groups thousands with no-break spaces, so an amount never wraps in the middle', () => {
    expect(grouped(15000)).toBe('15\u00a0000')
    expect(grouped(1000)).toBe('1\u00a0000')
    expect(grouped(375)).toBe('375')
  })
})

describe('looksLikeContact', () => {
  it('accepts an email, or a phone in the usual writings', () => {
    for (const v of ['a@b.ru', ' buyer@example.com ', '8 (903) 123-45-67', '+7 903 123 45 67', '9031234567', '+381 64 123 4567']) {
      expect(looksLikeContact(v), v).toBe(true)
    }
  })

  it('refuses what is neither', () => {
    for (const v of ['', ' ', 'abc', 'a@b', 'Anna <a@b.ru>', '12345', '1234567890', '+0123456789', '903-123-45-6x', '8903123456789']) {
      expect(looksLikeContact(v), v).toBe(false)
    }
  })
})
