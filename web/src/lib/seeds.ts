// The seed sprites: one icon, and three that stand in for the size of the
// balance: a few seeds, a heap, a sack. All four are tiny pixel-art WebPs
// (docs/seed-effects/crop-seeds.py builds them from the raw renders) and are
// always drawn with `image-rendering: pixelated`, so they stay crisp at any size.
import seed from '../assets/currency/seed.webp'
import seedsFew from '../assets/currency/seeds-few.webp'
import seedsHandful from '../assets/currency/seeds-handful.webp'
import seedsSack from '../assets/currency/seeds-sack.webp'

export const SEED = seed

// An empty wallet gets the single seed rather than the smallest scatter: a
// heap of nothing would be a lie, and the lone seed reads as "начни".
export function pileFor(balance: number): string {
  if (balance <= 0) return seed
  if (balance < 100) return seedsFew
  if (balance < 1000) return seedsHandful
  return seedsSack
}

export type SparkleSize = 's' | 'm' | 'l' | 'xl'
export type SparkleColor = 'pale' | 'gold' | 'amber' | 'cream'

// Native sprite size in pixels, from docs/seed-effects/gen_sparkles.py.
export const SPARKLE_PX: Record<SparkleSize, number> = { s: 5, m: 7, l: 9, xl: 13 }

const SPARKLE_URLS = import.meta.glob('../assets/currency/sparkles/*.png', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>

export function sparkleUrl(size: SparkleSize, color: SparkleColor): string {
  return SPARKLE_URLS[`../assets/currency/sparkles/${size}-${color}.png`]
}
