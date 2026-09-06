// Minimal Telegram Mini App integration: expand to full height, tell Telegram
// we're ready, and keep the header/background colours in sync with our theme.
// All no-ops when not running inside Telegram.
import { theme } from './theme'
import { watch } from 'vue'

interface TgWebApp {
  ready: () => void
  expand: () => void
  colorScheme: 'light' | 'dark'
  setHeaderColor?: (c: string) => void
  setBackgroundColor?: (c: string) => void
  onEvent?: (e: string, cb: () => void) => void
}

function tg(): TgWebApp | undefined {
  return (window as unknown as { Telegram?: { WebApp?: TgWebApp } }).Telegram?.WebApp
}

export function isTelegram() {
  return !!tg()
}

function bgColor() {
  return getComputedStyle(document.documentElement).getPropertyValue('--bg').trim() || '#fdf4f7'
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
