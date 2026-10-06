// The feather sprites: one icon, and three heaps that stand in for the size of
// the balance. All four are tiny pixel-art PNGs (docs/feather-effects/
// crop-feathers.py builds them from the raw renders) and are always drawn with
// `image-rendering: pixelated`, so they stay crisp at any size.
import feather from '../assets/currency/feather.png'
import pile1 from '../assets/currency/pile-1.png'
import pile2 from '../assets/currency/pile-2.png'
import pile3 from '../assets/currency/pile-3.png'

export const FEATHER = feather

// An empty wallet gets the single feather rather than the smallest heap: a
// heap of nothing would be a lie, and the lone feather reads as "начни".
export function pileFor(balance: number): string {
  if (balance <= 0) return feather
  if (balance < 100) return pile1
  if (balance < 1000) return pile2
  return pile3
}

export type SparkleSize = 's' | 'm' | 'l' | 'xl'
export type SparkleColor = 'pale' | 'gold' | 'amber' | 'cream'

// Native sprite size in pixels, from docs/feather-effects/gen_sparkles.py.
export const SPARKLE_PX: Record<SparkleSize, number> = { s: 5, m: 7, l: 9, xl: 13 }

const SPARKLE_URLS = import.meta.glob('../assets/currency/sparkles/*.png', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>

export function sparkleUrl(size: SparkleSize, color: SparkleColor): string {
  return SPARKLE_URLS[`../assets/currency/sparkles/${size}-${color}.png`]
}
