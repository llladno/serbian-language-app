import { describe, it, expect } from 'vitest'
import shared from './translit_vectors.json'
import { toCyrillic, toScript } from './script'

// The same file drives scripts/check_cyrillic.py: the converter exists in two
// languages and these vectors are what keeps them identical.
describe('toCyrillic: shared vectors', () => {
  for (const v of shared.vectors) {
    it(`${JSON.stringify(v.latin)} -> ${JSON.stringify(v.cyrillic)}`, () => {
      expect(toCyrillic(v.latin)).toBe(v.cyrillic)
    })
  }
})

describe('toCyrillic: rules worth stating on their own', () => {
  it('a decomposed c-acute is the letter ć, never a stressed c', () => {
    expect(toCyrillic('c\u0301evapi')).toBe('ћевапи')
    expect(toCyrillic('ćevapi')).toBe('ћевапи')
  })

  it('keeps a stress mark on the vowel it belongs to, precomposed or not', () => {
    const want = 'мо\u0301лим'
    expect(toCyrillic('mo\u0301lim')).toBe(want)
    expect(toCyrillic('mólim')).toBe(want)
  })

  it('does not touch Russian: й and ё survive byte for byte', () => {
    const ru = 'Это мой друг, ёж и йод. Ещё щи.'
    expect(toCyrillic(ru)).toBe(ru)
  })

  it('does not touch a stress mark on a Cyrillic vowel in a transcription', () => {
    expect(toCyrillic('[мо\u0301лим]')).toBe('[мо\u0301лим]')
  })

  it('leaves characters outside the Serbian alphabet alone', () => {
    expect(toCyrillic('q w x y 123 !?')).toBe('q w x y 123 !?')
  })

  it('treats lj, nj and dž as one letter, whatever the case', () => {
    expect(toCyrillic('lj Lj LJ nj Nj NJ dž Dž DŽ')).toBe('љ Љ Љ њ Њ Њ џ Џ Џ')
  })

  it('splits the digraph in the few words where it is really two letters', () => {
    expect(toCyrillic('injekcija')).toBe('инјекција')
    expect(toCyrillic('Injekcije')).toBe('Инјекције')
    expect(toCyrillic('nadživeti')).toBe('надживети')
  })

  it('is idempotent: converting Cyrillic again changes nothing', () => {
    const once = toCyrillic('Zdravo, kako si? Ćao!')
    expect(toCyrillic(once)).toBe(once)
  })
})

describe('toScript', () => {
  it('returns Latin text untouched in latin mode', () => {
    expect(toScript('čokolada', 'latin')).toBe('čokolada')
  })
  it('converts in cyrillic mode', () => {
    expect(toScript('čokolada', 'cyrillic')).toBe('чоколада')
  })
  it('tolerates a missing value', () => {
    expect(toScript(undefined, 'cyrillic')).toBe('')
    expect(toScript(null, 'cyrillic')).toBe('')
  })
})
