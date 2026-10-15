// Tariffs: paying rubles for paid levels. Spec: docs/superpowers/specs/2026-10-14-tariffs-design.md.
//
// Two products, one-off payments, no auto-renew:
//   level - the NEXT level the learner has not opened, forever
//   year  - every level for 365 days, new ones included
//
// TARIFF_DEFAULTS stands in for GET /api/tariffs until the backend exists; the
// numbers are the ones agreed on 2026-10-14 and will be admin-editable.
import type { Course, Phase } from '../types'

export interface TariffPrices {
  // Whole rubles. The server will speak kopecks; the screen never needs them.
  level: { price: number; firstPrice: number }
  year: { price: number; oldPrice: number }
}

export const TARIFF_DEFAULTS: TariffPrices = {
  level: { price: 299, firstPrice: 249 },
  year: { price: 1000, oldPrice: 2000 },
}

// A locked level counts as "for sale" once at least one of its lessons is
// written - the same rule CourseView uses to decide between a price tag and
// "скоро".
export function forSale(phase: Phase, planned: (lessonId: string) => boolean): boolean {
  return !!phase.locked && phase.lessons.some((id) => !planned(id))
}

// The level "Уровень" would open: the first one in course order that is locked
// and for sale. null when everything on sale is already open.
export function nextLockedLevel(course: Course | null): Phase | null {
  if (!course) return null
  const planned = new Set(course.lessons.filter((l) => l.planned).map((l) => l.id))
  return course.phases.find((p) => forSale(p, (id) => planned.has(id))) ?? null
}

// 249 for the very first level purchase, 299 afterwards.
export function levelPrice(prices: TariffPrices['level'], hasPaidLevel: boolean): number {
  return hasPaidLevel || prices.firstPrice <= 0 ? prices.price : prices.firstPrice
}

// "1 000 ₽" - a no-break space between the thousands and before the sign, so a
// price never wraps in the middle.
export function rub(n: number): string {
  return `${n.toLocaleString('ru-RU').replace(/\s/g, ' ')} ₽`
}

// What a year costs per month, rounded to whole rubles.
export function perMonth(yearPrice: number): number {
  return Math.round(yearPrice / 12)
}

// What to offer a learner who tried to open `phaseId` without enough зёрнышки.
// "Уровень" only ever opens the NEXT locked level, so asking for a later one
// is answered with the year: it opens every level at once, which is the only
// way to reach level 5 without level 4.
export type Offer =
  | { kind: 'level'; level: Phase; price: number }
  | { kind: 'year'; level: Phase; first: Phase; price: number }

export function offerForPhase(
  course: Course | null,
  phaseId: string,
  prices: TariffPrices,
  hasPaidLevel: boolean,
): Offer | null {
  const next = nextLockedLevel(course)
  const asked = course?.phases.find((p) => p.id === phaseId)
  if (!next || !asked || !asked.locked) return null
  if (asked.id === next.id) return { kind: 'level', level: asked, price: levelPrice(prices.level, hasPaidLevel) }
  return { kind: 'year', level: asked, first: next, price: prices.year.price }
}

// Whole days from `now` until `until` (an ISO date), never below 0.
export function daysLeft(until: string, now: Date = new Date()): number {
  const ms = new Date(until).getTime() - now.getTime()
  return Math.max(0, Math.ceil(ms / 86_400_000))
}

// "14 октября 2027"
export function longDate(iso: string): string {
  return new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' })
}
