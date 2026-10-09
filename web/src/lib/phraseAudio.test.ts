import { describe, it, expect, afterEach } from 'vitest'
import { phraseKey, phraseClip, fillSentence, _setPhraseIndex } from './phraseAudio'

afterEach(() => _setPhraseIndex(null))

describe('phraseKey', () => {
  // vectors computed by scripts/tts.py phrase_key — the two must agree
  it('matches the python generator', () => {
    expect(phraseKey('Govorim ruski.')).toBe('adc562fc95c4145a')
    expect(phraseKey('Šta radiš?')).toBe('0f8a5cb231c832f4')
  })

  it('ignores case, punctuation and spacing', () => {
    expect(phraseKey('  GOVORIM   ruski!! ')).toBe(phraseKey('govorim ruski'))
  })
})

describe('phraseClip', () => {
  it('is undefined until the index is loaded', () => {
    expect(phraseClip('Govorim ruski.')).toBeUndefined()
  })

  it('finds a clip by phrase once indexed, and only for indexed phrases', () => {
    _setPhraseIndex(['adc562fc95c4145a'])
    expect(phraseClip('Govorim ruski.')).toBe('p/adc562fc95c4145a.mp3')
    expect(phraseClip('Ne govorim')).toBeUndefined()
    expect(phraseClip(undefined)).toBeUndefined()
  })
})

describe('fillSentence', () => {
  // same cases as FILL_CASES in scripts/test_tts.py
  it.each([
    ['«Я за вами» → Ja sam ____ vama.', 'iza', 'Ja sam iza vama.'],
    ['Утро, заходишь в пекару: «___ jutro!»', 'Dobro', 'Dobro jutro!'],
    ['Ti ___ student? (ты)', 'si', 'Ti si student?'],
    ['«Горло» → ___', 'grlo', 'grlo'],
    ['«полдень» по-сербски — ___', 'podne', 'podne'],
    ['У меня есть сестра: «Imam ___.»', 'sestru', 'Imam sestru.'],
    ['Привет ___ и ___', 'x', undefined],
  ])('%s + %s', (prompt, word, want) => {
    expect(fillSentence(prompt, word)).toBe(want)
  })
})
