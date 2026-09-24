// Yandex.Metrika goal tracking for the app (web/), gated by the same
// cookie-consent choice as the landing — same origin, so the choice made
// there is readable here via localStorage. The landing's cookie banner is
// the only place consent is ever asked; a visitor who lands directly on
// e.g. /login without going through the landing first never saw it, so
// tracking simply stays off for them too (no second banner here).
const METRIKA_ID = 112668663
const CONSENT_KEY = 'ucimo-cookie-consent'

type YmFn = (id: number, action: string, goal: string) => void

function hasConsent(): boolean {
  try {
    return localStorage.getItem(CONSENT_KEY) === 'accepted'
  } catch {
    return false
  }
}

/** Loads the counter (same /metrika-init.js the landing uses) if consent was already given. */
export function initMetrika(): void {
  if (!hasConsent()) return
  if (document.querySelector('script[data-metrika]')) return
  const s = document.createElement('script')
  s.src = '/metrika-init.js'
  s.setAttribute('data-metrika', '')
  document.body.appendChild(s)
}

/** Fires a goal. No-op if the counter never loaded (no consent). */
export function reachGoal(goal: string): void {
  const ym = (window as unknown as { ym?: YmFn }).ym
  ym?.(METRIKA_ID, 'reachGoal', goal)
}
