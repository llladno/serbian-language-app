// The countdown to an announced price change (migration 041): a black strip at the top of the profile and a poster on
// the tariffs screen and a window after a lesson. The words are `campaignCopy`, below.
//
// The time is the server's: the quote carries the server's clock, the store works out how far the phone's clock is off
// (stores/tariffs.ts, `priceChange.until`), and the change itself is made by the server at the moment, so the zero on
// the screen is the real one. When the timer reaches zero the quote is asked again: the new prices come back and the
// announcement is gone.
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useTariffsStore } from '../stores/tariffs'
import { discountPercent, rub } from './tariffs'

const BELGRADE = 'Europe/Belgrade'

/** "17 дн 12:34:07", or "12:34:07" on the last day. `seconds: false` drops the seconds ("17 дн 12:34"). */
export function formatCountdown(ms: number, seconds = true): string {
  const total = Math.max(0, Math.floor(ms / 1000))
  const days = Math.floor(total / 86_400)
  const two = (n: number) => String(n).padStart(2, '0')
  const clock = [Math.floor((total % 86_400) / 3600), Math.floor((total % 3600) / 60), ...(seconds ? [total % 60] : [])]
    .map(two)
    .join(':')
  return days > 0 ? `${days} дн ${clock}` : clock
}

/**
 * The same time in two pieces, "5 дн 12:34" and "07", so that the seconds can be drawn on their own: they are the part
 * that moves, and the screens make each tick felt (CountdownText).
 */
export function countdownParts(ms: number): { head: string; sec: string } {
  const total = Math.max(0, Math.floor(ms / 1000))
  const days = Math.floor(total / 86_400)
  const two = (n: number) => String(n).padStart(2, '0')
  const clock = `${two(Math.floor((total % 86_400) / 3600))}:${two(Math.floor((total % 3600) / 60))}`
  return { head: days > 0 ? `${days} дн ${clock}` : clock, sec: two(total % 60) }
}

/**
 * The last day of the old prices as the learner reads it, "27 ноября": the minute before the change, in Belgrade time
 * (the learners live there; in Moscow the same moment is already the 28th, which would read as the wrong day).
 */
export function lastDay(at: number): { day: string; month: string } {
  const text = new Date(at - 60_000).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', timeZone: BELGRADE })
  const [day = '', month = ''] = text.split(' ')
  return { day, month }
}

/** The first day of the new prices, "28 ноября". */
export function firstDay(at: number): string {
  return new Date(at).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', timeZone: BELGRADE })
}

/** The words of the campaign, for the strip and the poster (campaignCopy). */
export interface CampaignCopy {
  /** The white sticker. */
  sticker: string
  /** The line of the strip. */
  strip: string
  /** The poster's headline and the lines under it. */
  title: string
  sub: string
  /** One short line for the small banner on the tariffs screen. */
  short: string
  /** The gold buttons: the strip's (short) and the poster's. */
  stripButton: string
  posterButton: string
}

/**
 * What the two tariffs are called in the campaign. The word «Уровень» is left out on purpose (the owner, 2026-11-10: it
 * does not say what the learner gets): a level is «новые уроки», forever, and the year is «всё на год».
 */
export const LEVEL_LABEL = 'Новые уроки'
export const YEAR_LABEL = 'Всё на год'

/**
 * What the campaign says. When both prices are exactly half of what they will be (the case now: 199 → 399, 1 000 →
 * 2 000), it is sold as the discount it is, with the date it ends and the prices that follow it, which really do take over
 * (the server changes them), so the discount is against a price that will be charged. In any other case (a level that has
 * been paid for already is 299 → 399, a quarter, and a different change would not be half either) it falls back to the plain
 * announcement, which claims no percentage. `hasLevel` is false once every level is open: only the year is looked at.
 *
 * The words follow the brand voice (honest, concrete, warm, at most one «!»), written with the marketing skills on
 * 2026-11-10: the discount in the sticker, the benefit and the urge in the headline («Успей забрать за полцены») with a
 * one-word button («Забрать»), the date in the strip, what happens after it in the line under the headline.
 */
export function campaignCopy(q: {
  levelNow: number
  levelNext: number
  yearNow: number
  yearNext: number
  hasLevel: boolean
  at: number
}): CampaignCopy {
  const nb = '\u00a0'
  const last = lastDay(q.at)
  const lastText = `${last.day}${nb}${last.month}`
  const first = firstDay(q.at).replace(' ', nb)
  const yearHalf = discountPercent(q.yearNow, q.yearNext) === 50
  const levelHalf = !q.hasLevel || discountPercent(q.levelNow, q.levelNext) === 50
  if (!(yearHalf && levelHalf)) {
    return {
      sticker: 'Цены растут',
      strip: `До${nb}${lastText} цены прежние`,
      title: `Успейте по${nb}прежним ценам`,
      sub: `Мы ничего не снижаем: просто в${nb}последний раз оставляем цены прежними. С${nb}${first} дороже.`,
      short: `До${nb}${lastText} цены прежние, с${nb}${first} дороже.`,
      stripButton: 'Успеть',
      posterButton: 'Успеть по сегодняшней цене',
    }
  }
  const prices = q.hasLevel
    ? `новые уроки навсегда — ${rub(q.levelNext)}, всё на год — ${rub(q.yearNext)}`
    : `всё на год — ${rub(q.yearNext)}`
  return {
    sticker: 'Скидка 50%',
    strip: `Полцены до${nb}${lastText}`,
    title: `Успей забрать за${nb}полцены`,
    sub: `С${nb}${first} цены вырастут вдвое: ${prices}. До${nb}${lastText} — по-старому.`,
    short: `До${nb}${lastText} — по-старому, с${nb}${first} цены вырастут вдвое.`,
    stripButton: 'Забрать',
    posterButton: 'Забрать',
  }
}

/**
 * The announced change as a screen needs it: the quote (or null), the milliseconds left, and the ticking. Reading the
 * quote again at zero is the tariffs store's job (it does so for every screen), and the screens change back by themselves
 * when the announcement is gone from it.
 */
export function usePriceChange() {
  const tariffs = useTariffsStore()
  const now = ref(Date.now())
  let timer: ReturnType<typeof setInterval> | undefined

  const change = computed(() => tariffs.campaign)
  const remaining = computed(() => (change.value ? Math.max(0, change.value.until - now.value) : 0))

  function tick() {
    now.value = Date.now()
  }

  onMounted(() => {
    timer = setInterval(tick, 1000)
    document.addEventListener('visibilitychange', tick)
  })
  onBeforeUnmount(() => {
    clearInterval(timer)
    document.removeEventListener('visibilitychange', tick)
  })

  // The words of the campaign for this learner's prices.
  const copy = computed(() => {
    const c = change.value
    return c
      ? campaignCopy({
          levelNow: tariffs.levelNow,
          levelNext: c.levelPrice,
          yearNow: tariffs.yearPrice,
          yearNext: c.yearPrice,
          hasLevel: tariffs.choices.length > 0 || tariffs.hasCredit,
          at: c.at,
        })
      : null
  })

  return { change, remaining, copy }
}
