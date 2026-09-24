const METRIKA_ID = 112668663

type YmFn = (id: number, action: string, goal: string) => void

/**
 * Fires a Yandex.Metrika goal. No-op if the visitor hasn't accepted the
 * cookie banner yet — metrika-init.js (and therefore window.ym) only loads
 * after consent, see public/cookie-consent.js.
 */
export function reachGoal(goal: string) {
  const ym = (window as unknown as { ym?: YmFn }).ym
  ym?.(METRIKA_ID, 'reachGoal', goal)
}
