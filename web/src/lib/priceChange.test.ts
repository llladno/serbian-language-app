import { describe, it, expect } from 'vitest'
import { campaignCopy, countdownParts, firstDay, formatCountdown, lastDay } from './priceChange'

describe('formatCountdown', () => {
  it('counts days, then the clock, and drops the days on the last day', () => {
    const day = 86_400_000
    expect(formatCountdown(17 * day + 12 * 3_600_000 + 34 * 60_000 + 7000)).toBe('17 дн 12:34:07')
    expect(formatCountdown(17 * day + 12 * 3_600_000 + 34 * 60_000 + 7000, false)).toBe('17 дн 12:34')
    expect(formatCountdown(5 * 3_600_000 + 9 * 60_000 + 3000)).toBe('05:09:03')
  })

  it('never goes below zero and drops fractions of a second', () => {
    expect(formatCountdown(-5000)).toBe('00:00:00')
    expect(formatCountdown(999)).toBe('00:00:00')
  })
})

describe('countdownParts', () => {
  it('splits the seconds off the rest', () => {
    expect(countdownParts(5 * 86_400_000 + 12 * 3_600_000 + 34 * 60_000 + 7000)).toEqual({ head: '5 дн 12:34', sec: '07' })
    expect(countdownParts(61_000)).toEqual({ head: '00:01', sec: '01' })
    expect(countdownParts(-1)).toEqual({ head: '00:00', sec: '00' })
  })
})

describe('the days of the change', () => {
  // The change is at the end of Friday 27 November in Belgrade (23:00 UTC).
  const at = Date.parse('2026-11-27T23:00:00Z')

  it('names the last day of the old prices in Belgrade time, whatever zone the phone is in', () => {
    expect(lastDay(at)).toEqual({ day: '27', month: 'ноября' })
  })

  it('names the first day of the new prices', () => {
    expect(firstDay(at)).toBe('28 ноября')
  })
})

describe('campaignCopy', () => {
  // The change is at the end of Sunday 15 November in Belgrade.
  const at = Date.parse('2026-11-15T23:00:00Z')
  const flat = (t: string) => t.replace(/\u00a0/g, ' ')

  it('sells the discount when both prices are half of what follows, with the date and the prices to come', () => {
    const c = campaignCopy({ levelNow: 199, levelNext: 399, yearNow: 1000, yearNext: 2000, hasLevel: true, at })
    expect(flat(c.sticker)).toBe('Скидка 50%')
    expect(flat(c.strip)).toBe('Полцены до 15 ноября')
    expect(flat(c.title)).toBe('Успей забрать за полцены')
    expect(flat(c.sub)).toBe(
      'С 16 ноября цены вырастут вдвое: новые уроки навсегда — 399 ₽, всё на год — 2 000 ₽. До 15 ноября — по-старому.',
    )
    expect(c.stripButton).toBe('Забрать')
    expect(c.posterButton).toBe('Забрать')
  })

  it('is half for the real prices: 199 to 399 and 1 000 to 1 999', () => {
    const c = campaignCopy({ levelNow: 199, levelNext: 399, yearNow: 1000, yearNext: 1999, hasLevel: true, at })
    expect(c.sticker).toBe('Скидка 50%')
    expect(flat(c.sub)).toContain('новые уроки навсегда — 399 ₽, всё на год — 1 999 ₽')
  })

  it('claims no half for a learner who has paid for a level: 299 to 399 is a quarter', () => {
    const c = campaignCopy({ levelNow: 299, levelNext: 399, yearNow: 1000, yearNext: 1999, hasLevel: true, at })
    expect(c.sticker).toBe('Цены растут')
    expect(c.title).not.toMatch(/полцены/)
  })

  it('names only the year once every level is open', () => {
    const c = campaignCopy({ levelNow: 0, levelNext: 0, yearNow: 1000, yearNext: 2000, hasLevel: false, at })
    expect(flat(c.title)).toBe('Успей забрать за полцены')
    expect(flat(c.sub)).toContain('всё на год — 2 000 ₽')
    expect(c.sub).not.toContain('новые уроки')
  })

  it('claims no percentage when the prices are not half of what follows', () => {
    const c = campaignCopy({ levelNow: 249, levelNext: 399, yearNow: 1000, yearNext: 2000, hasLevel: true, at })
    expect(c.sticker).toBe('Цены растут')
    expect(flat(c.strip)).toBe('До 15 ноября цены прежние')
    expect(flat(c.sub)).toContain('С 16 ноября дороже')
    expect(`${c.sticker}${c.title}${c.sub}${c.posterButton}`).not.toMatch(/полцены|скидк|%/i)
  })

  it('never calls a tariff «Уровень» in its words', () => {
    for (const levelNow of [199, 249]) {
      const c = campaignCopy({ levelNow, levelNext: 399, yearNow: 1000, yearNext: 2000, hasLevel: true, at })
      expect(Object.values(c).join(' ')).not.toMatch(/уровень|уровня/i)
    }
  })

  it('says one «!» at most, and no hype', () => {
    const c = campaignCopy({ levelNow: 199, levelNext: 399, yearNow: 1000, yearNext: 2000, hasLevel: true, at })
    const all = Object.values(c).join(' ')
    expect((all.match(/!/g) ?? []).length).toBeLessThanOrEqual(1)
    expect(all).not.toMatch(/революц|уникальн|лучш/i)
  })
})
