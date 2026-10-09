// Minimal Telegram Mini App integration: expand to full height, tell Telegram
// we're ready, and keep the header/background colours in sync with our theme.
// All no-ops when not running inside Telegram.
import { theme } from './theme'
import { watch } from 'vue'

interface TgWebApp {
  ready: () => void
  expand: () => void
  colorScheme: 'light' | 'dark'
  initData: string
  setHeaderColor?: (c: string) => void
  setBackgroundColor?: (c: string) => void
  onEvent?: (e: string, cb: () => void) => void
  openTelegramLink?: (url: string) => void
}

function tg(): TgWebApp | undefined {
  return (window as unknown as { Telegram?: { WebApp?: TgWebApp } }).Telegram?.WebApp
}

// Opens a t.me link inside Telegram itself when running as a Mini App, which
// keeps the learner in the app instead of bouncing them to an external browser.
// Returns whether it did: outside Telegram the caller lets the link do its
// ordinary thing.
export function openTelegramLink(url: string): boolean {
  const wa = tg()
  if (!wa?.openTelegramLink) return false
  try {
    wa.openTelegramLink(url)
    return true
  } catch {
    return false
  }
}

export function isTelegram() {
  return !!tg()
}

// The raw, HMAC-signed query string the backend's VerifyInitData checks.
// Empty outside Telegram or before the SDK has populated it.
export function initData(): string {
  return tg()?.initData ?? ''
}

function bgColor() {
  return getComputedStyle(document.documentElement).getPropertyValue('--bg').trim() || '#f7f9f7'
}

export function initTelegram() {
  const wa = tg()
  if (!wa) return
  try {
    wa.ready()
    wa.expand()
    document.documentElement.classList.add('tg')
    const sync = () => {
      const c = bgColor()
      wa.setHeaderColor?.(c)
      wa.setBackgroundColor?.(c)
    }
    sync()
    // re-sync a tick later once fonts/tokens settle, and on theme change
    setTimeout(sync, 150)
    watch(theme, () => setTimeout(sync, 50))
    wa.onEvent?.('themeChanged', sync)
  } catch {
    /* ignore */
  }
}
