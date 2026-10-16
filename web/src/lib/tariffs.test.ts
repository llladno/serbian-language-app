import { describe, it, expect } from 'vitest'
import { daysLeft, discountPercent, grouped, monthlyYearCost, nextLockedLevel, offerForPhase, perMonth, rub, shortDate, yearSaving } from './tariffs'
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
const QUOTE = { levelPrice: 249, yearPrice: 1000 }

describe('offerForPhase', () => {
  it('offers the level itself when it is the next one to open', () => {
    const o = offerForPhase(course(), '4', QUOTE)
    expect(o).toMatchObject({ kind: 'level', price: 249 })
    expect(o?.level.id).toBe('4')
  })

  it('quotes whatever the server priced the level at (the regular price after a first purchase)', () => {
    expect(offerForPhase(course(), '4', { levelPrice: 299, yearPrice: 1000 })).toMatchObject({ kind: 'level', price: 299 })
  })

  it('offers the year for a later level, because levels open in order', () => {
    const o = offerForPhase(course(), '5', QUOTE)
    expect(o).toMatchObject({ kind: 'year', price: 1000 })
    expect(o?.kind === 'year' && o.first.id).toBe('4')
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

describe('offerForPhase while the monthly limit holds the next level back', () => {
  const FROM = '2026-11-12T10:00:00Z'

  it('answers the next level with the year, and says it is because of the limit', () => {
    const o = offerForPhase(course(), '4', { ...QUOTE, levelFrom: FROM })
    expect(o).toMatchObject({ kind: 'year', why: 'cooldown', from: FROM, price: 1000 })
  })

  it('still answers a later level with the year, for the usual reason', () => {
    expect(offerForPhase(course(), '5', { ...QUOTE, levelFrom: FROM })).toMatchObject({ kind: 'year', why: 'order' })
  })

  it('offers the level as before when nothing holds the learner back', () => {
    expect(offerForPhase(course(), '4', { ...QUOTE, levelFrom: null })).toMatchObject({ kind: 'level', price: 249 })
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
