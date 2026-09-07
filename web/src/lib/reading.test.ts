import { describe, it, expect } from 'vitest'
import { tokenize } from './reading'

describe('tokenize', () => {
  it('splits a sentence into word and non-word tokens that rejoin to the original', () => {
    const src = 'Zdravo, ja sam Milan!'
    const toks = tokenize(src)
    expect(toks.map((t) => t.text).join('')).toBe(src)
    expect(toks.filter((t) => t.word).map((t) => t.text)).toEqual([
      'Zdravo',
      'ja',
      'sam',
      'Milan',
    ])
  })

  it('keeps Serbian latin diacritics inside a word', () => {
    expect(tokenize('Čujemo se, đačka šžč.').filter((t) => t.word).map((t) => t.text)).toEqual([
      'Čujemo',
      'se',
      'đačka',
      'šžč',
    ])
  })

  it('handles Cyrillic', () => {
    expect(tokenize('Здраво свете').filter((t) => t.word).map((t) => t.text)).toEqual([
      'Здраво',
      'свете',
    ])
  })

  it('treats an internal hyphen as part of one word', () => {
    expect(tokenize('Novi-Sad je grad').filter((t) => t.word).map((t) => t.text)).toEqual([
      'Novi-Sad',
      'je',
      'grad',
    ])
  })

  it('preserves paragraph breaks as non-word tokens', () => {
    const toks = tokenize('Prvi.\n\nDrugi.')
    expect(toks.map((t) => t.text).join('')).toBe('Prvi.\n\nDrugi.')
  })
})
