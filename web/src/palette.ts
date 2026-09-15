// Color palette (accent family), independent of light/dark mode. Persisted
// locally, applied via data-palette on <html>. See theme.ts for light/dark.
import { ref } from 'vue'

export type Palette = 'teal' | 'pink' | 'blue'
const KEY = 'srpski.palette'

function read(): Palette {
  try {
    const v = localStorage.getItem(KEY)
    if (v === 'teal' || v === 'pink' || v === 'blue') return v
  } catch {
    /* ignore */
  }
  return 'teal'
}

export const palette = ref<Palette>(read())

export function applyPalette(p: Palette = palette.value) {
  document.documentElement.setAttribute('data-palette', p)
}

export function setPalette(p: Palette) {
  palette.value = p
  try {
    localStorage.setItem(KEY, p)
  } catch {
    /* ignore */
  }
  applyPalette(p)
}

export const PALETTE_META: Record<Palette, { label: string; swatch: string }> = {
  teal: { label: 'бирюзовая', swatch: '#176b68' },
  pink: { label: 'розовая', swatch: '#d6457d' },
  blue: { label: 'синяя', swatch: '#2563b3' },
}
