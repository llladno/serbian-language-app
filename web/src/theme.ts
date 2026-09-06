// Theme = 'system' | 'light' | 'dark', persisted locally, applied via
// data-theme on <html>. 'system' removes the attribute so prefers-color-scheme
// decides.
import { ref } from 'vue'

export type Theme = 'system' | 'light' | 'dark'
const KEY = 'srpski.theme'

function read(): Theme {
  try {
    const v = localStorage.getItem(KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch {
    /* ignore */
  }
  return 'system'
}

export const theme = ref<Theme>(read())

export function applyTheme(t: Theme = theme.value) {
  const root = document.documentElement
  if (t === 'system') root.removeAttribute('data-theme')
  else root.setAttribute('data-theme', t)
}

export function setTheme(t: Theme) {
  theme.value = t
  try {
    localStorage.setItem(KEY, t)
  } catch {
    /* ignore */
  }
  applyTheme(t)
}

const ORDER: Theme[] = ['system', 'light', 'dark']
export function cycleTheme() {
  setTheme(ORDER[(ORDER.indexOf(theme.value) + 1) % ORDER.length])
}

export const THEME_META: Record<Theme, { icon: string; label: string }> = {
  system: { icon: '◐', label: 'как в системе' },
  light: { icon: '☀', label: 'светлая' },
  dark: { icon: '☾', label: 'тёмная' },
}
