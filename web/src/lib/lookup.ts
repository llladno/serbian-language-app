// Shared dictionary lookup for clickable Serbian words. One in-flight request
// and one cached result per word across every GlossedText on the page.
import { api } from '../api'
import type { LookupResult } from '../types'

const cache = new Map<string, LookupResult>()
const inflight = new Map<string, Promise<LookupResult>>()

function key(word: string): string {
  return word.trim().toLowerCase()
}

export async function lookupWord(word: string): Promise<LookupResult> {
  const k = key(word)
  const cached = cache.get(k)
  if (cached) return cached

  const running = inflight.get(k)
  if (running) return running

  const p = api
    .lookup(word.trim())
    .then((res) => {
      cache.set(k, res)
      return res
    })
    .catch((): LookupResult => ({ query: word, matches: [], partial: false }))
    .finally(() => inflight.delete(k))
  inflight.set(k, p)
  return p
}

// test hook
export function _resetLookupCache(): void {
  cache.clear()
  inflight.clear()
}
