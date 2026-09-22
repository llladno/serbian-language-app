// First-touch UTM attribution captured from the landing URL. See
// docs/superpowers/specs/2026-09-22-utm-link-tracking-design.md.
const STORAGE_KEY = 'ucimo_attribution'
const VISIT_LOGGED_KEY = 'ucimo_visit_logged'

export interface Attribution {
  utm_source?: string
  utm_medium?: string
  utm_campaign?: string
  utm_content?: string
}

const UTM_FIELDS: (keyof Attribution)[] = ['utm_source', 'utm_medium', 'utm_campaign', 'utm_content']

function fromSearchParams(search: string): Attribution {
  const params = new URLSearchParams(search)
  const attr: Attribution = {}
  for (const field of UTM_FIELDS) {
    const value = params.get(field)
    if (value) attr[field] = value
  }
  return attr
}

/** Returns the first-touch attribution stored for this browser, or {}. */
export function getStoredAttribution(): Attribution {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? (JSON.parse(raw) as Attribution) : {}
  } catch {
    return {}
  }
}

/**
 * Reads UTM parameters off the current URL. When present: fires one
 * fire-and-forget visit beacon per browser session, and — only if no
 * attribution is stored yet for this browser — saves them as the first
 * touch, for a later register()/telegramLogin() call to submit.
 */
export function captureAttribution(): void {
  const attr = fromSearchParams(window.location.search)
  if (Object.keys(attr).length === 0) return

  try {
    if (!sessionStorage.getItem(VISIT_LOGGED_KEY)) {
      sessionStorage.setItem(VISIT_LOGGED_KEY, '1')
      fetch('/api/track-visit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(attr),
        keepalive: true,
      }).catch(() => {
        /* best-effort analytics beacon */
      })
    }
  } catch {
    /* sessionStorage unavailable (private mode etc.) — skip the dedup */
  }

  try {
    if (!localStorage.getItem(STORAGE_KEY)) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(attr))
    }
  } catch {
    /* localStorage unavailable — attribution just won't reach registration */
  }
}
