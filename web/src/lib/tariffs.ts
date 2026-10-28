// Tariffs: paying rubles for paid levels. Spec: docs/superpowers/specs/2026-10-14-tariffs-design.md.
//
// Two products, one-off payments, no auto-renew:
//   level - one level the learner has not opened, forever; they choose it after paying
//   year  - every level for 365 days, new ones included
//
// The prices and who has paid come from the server (stores/tariffs.ts); this
// file is the pure part: which level is "next", what to offer, how to show money.
import type { Course, Phase } from '../types'

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

// "1 000 ₽" - a no-break space between the thousands and before the sign, so a
// price never wraps in the middle.
export function rub(n: number): string {
  return `${n.toLocaleString('ru-RU').replace(/\s/g, ' ')} ₽`
}

// "15 000" - thousands grouped with no-break spaces, for amounts of the in-app
// currency (the word after it comes from the wallet store, which knows the
// currency's name and its plural forms).
export function grouped(n: number): string {
  return n.toLocaleString('ru-RU').replace(/\s/g, '\u00a0')
}

// What a year costs per month, rounded to whole rubles.
export function perMonth(yearPrice: number): number {
  return Math.round(yearPrice / 12)
}

// How much cheaper `now` is than `regular`, in whole percent: 249 against 299 is
// 17 (the live prices are the server's: 199 against 299 is 33). 0 when there is no discount, so a caller can use it as the "show a badge" test.
export function discountPercent(now: number, regular: number): number {
  if (regular <= 0 || now >= regular) return 0
  return Math.round((1 - now / regular) * 100)
}

// What twelve months of "Уровень" cost at the fastest pace the tariff allows,
// one level a month: this learner's price for the first, the regular price for
// the other eleven. This is the yardstick for the year, because buying levels
// more often is not possible; the year also covers the levels still to come.
export function monthlyYearCost(levelNow: number, levelRegular: number): number {
  return levelNow + levelRegular * 11
}

// What the year saves against that yardstick, or null when it saves nothing.
export function yearSaving(yearPrice: number, monthlyCost: number): { rub: number; percent: number } | null {
  if (monthlyCost <= 0 || yearPrice >= monthlyCost) return null
  const rub = monthlyCost - yearPrice
  return { rub, percent: Math.round((rub / monthlyCost) * 100) }
}

// What to offer a learner who tried to open `phaseId` without enough зёрнышки.
// "Уровень" is the first tariff and is always what is offered: it is a credit for any closed level the
// learner picks after paying, so it fits the level they tapped as much as the next one.
//
// Two things change what the offer can do:
//  - levels are sold one a month: while the limit holds, the level is still offered but carries the date
//    it can be bought (`from`);
//  - a paid level that has not been spent yet (`hasCredit`): another cannot be bought until the learner
//    has chosen, so the answer is to choose (`kind: 'credit'`).
export type Offer =
  | { kind: 'level'; level: Phase; price: number; from?: string }
  | { kind: 'credit'; level: Phase }

export function offerForPhase(
  course: Course | null,
  phaseId: string,
  quote: { levelPrice: number; levelFrom?: string | null; hasCredit?: boolean },
): Offer | null {
  if (!course) return null
  const planned = new Set(course.lessons.filter((l) => l.planned).map((l) => l.id))
  const asked = course.phases.find((p) => p.id === phaseId)
  if (!asked || !forSale(asked, (id) => planned.has(id))) return null
  if (quote.hasCredit) return { kind: 'credit', level: asked }
  if (quote.levelFrom) return { kind: 'level', level: asked, price: quote.levelPrice, from: quote.levelFrom }
  return { kind: 'level', level: asked, price: quote.levelPrice }
}

// Whole days from `now` until `until` (an ISO date), never below 0.
export function daysLeft(until: string, now: Date = new Date()): number {
  const ms = new Date(until).getTime() - now.getTime()
  return Math.max(0, Math.ceil(ms / 86_400_000))
}

// "12 ноября"
export function shortDate(iso: string): string {
  return new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })
}

// "14 октября 2027"
export function longDate(iso: string): string {
  return new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' })
}

// Whether what was typed can be where a receipt goes: an email, or a phone (a
// Russian number in any of its usual writings, or an international one with a
// plus). The server has the last word; this only keeps an obvious slip from
// costing a round trip, and follows its rules.
export function looksLikeContact(raw: string): boolean {
  const v = raw.trim()
  if (/^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/.test(v)) return true
  const plus = v.startsWith('+')
  const body = plus ? v.slice(1) : v
  if (!/^[\d\s\-().\u00a0]+$/.test(body)) return false
  const digits = body.replace(/\D/g, '')
  if (plus) return digits.length >= 8 && digits.length <= 15 && digits[0] !== '0'
  return (digits.length === 11 && (digits[0] === '8' || digits[0] === '7')) || (digits.length === 10 && digits[0] === '9')
}
