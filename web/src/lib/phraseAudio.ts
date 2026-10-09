// Pronunciation clips for whole Serbian phrases (teach-text examples, exercise
// options, reading lines). scripts/tts.py writes content/audio/p/<key>.mp3 and
// lists every key in content/audio/p/index.json; the key is a hash of the
// normalised text, so any component holding a Serbian string can ask for its
// clip without ids being threaded through the API. phraseKey() must stay in
// sync with phrase_key() in scripts/tts.py.
import { ref } from 'vue'

const MASK = (1n << 64n) - 1n
const PRIME = 0x100000001b3n
const encoder = new TextEncoder()

export function phraseNorm(text: string): string {
  return text
    .normalize('NFC')
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, ' ')
    .trim()
}

export function phraseKey(text: string): string {
  let h = 0xcbf29ce484222325n
  for (const b of encoder.encode(phraseNorm(text))) {
    h = ((h ^ BigInt(b)) * PRIME) & MASK
  }
  return h.toString(16).padStart(16, '0')
}

const index = ref<Set<string> | null>(null)
let loading: Promise<void> | null = null

/** Fetch the clip index once; failures leave the app silent, not broken. */
export function loadPhraseIndex(): Promise<void> {
  if (loading) return loading
  loading = (async () => {
    try {
      const res = await fetch('/audio/p/index.json', { cache: 'no-cache' })
      if (!res.ok) return
      index.value = new Set((await res.json()) as string[])
    } catch {
      /* no index → no phrase buttons */
    }
  })()
  return loading
}

/** Clip path (relative to /audio/) for a phrase, or undefined when none exists.
 *  Reactive: re-evaluates once the index has loaded. */
export function phraseClip(text: string | undefined): string | undefined {
  if (!text || !index.value) return undefined
  const key = phraseKey(text)
  return index.value.has(key) ? `p/${key}.mp3` : undefined
}

export function _setPhraseIndex(keys: string[] | null) {
  index.value = keys ? new Set(keys) : null
  loading = keys ? Promise.resolve() : null
}

/** The full Serbian sentence of a fill-in-the-blank prompt with the gap filled,
 *  e.g. «Я за вами» → Ja sam ___ vama. + "iza" → "Ja sam iza vama.".
 *  undefined when no clean Serbian sentence can be cut out. Keep in sync with
 *  fill_sentence() in scripts/tts.py. */
export function fillSentence(prompt: string, word: string): string | undefined {
  if (!word || (prompt.match(/_{2,}/g) ?? []).length !== 1) return undefined
  let t: string
  const quoted = [...prompt.matchAll(/«([^»]*)»/g)].find((m) => m[1].includes('___'))
  if (quoted) t = quoted[1]
  else if (prompt.includes('→')) t = prompt.slice(prompt.lastIndexOf('→') + 1)
  else if (prompt.includes(' — ') && prompt.slice(prompt.lastIndexOf(' — ')).includes('___')) t = prompt.slice(prompt.lastIndexOf(' — ') + 3)
  else t = prompt
  t = t.replace(/\([^)]*\)/g, '').replace(/_{2,}/, word).replace(/\s+/g, ' ').trim()
  return t && !/[\u0400-\u04FF]/.test(t) ? t : undefined
}
