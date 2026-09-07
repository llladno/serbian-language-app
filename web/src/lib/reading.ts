// Tokenizer for the reading-practice text: splits Serbian text (latin or
// cyrillic) into word and non-word tokens. Joining the token texts back
// together reproduces the input exactly, so the component can render each word
// as a clickable span without losing spacing or punctuation.

export interface Token {
  text: string
  word: boolean
}

// A word is a run of Unicode letters, optionally joined by a single internal
// hyphen or apostrophe (Novi-Sad, don't-style forms).
const WORD_RE = /\p{L}+(?:[-'’]\p{L}+)*/gu

export function tokenize(text: string): Token[] {
  const out: Token[] = []
  let last = 0
  for (const m of text.matchAll(WORD_RE)) {
    const start = m.index ?? 0
    if (start > last) out.push({ text: text.slice(last, start), word: false })
    out.push({ text: m[0], word: true })
    last = start + m[0].length
  }
  if (last < text.length) out.push({ text: text.slice(last), word: false })
  return out
}
