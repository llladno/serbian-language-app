// Latin -> Cyrillic for Serbian text, applied at the very last moment: in the
// component that paints a string. Everything else (answers sent to the server,
// the key of an audio clip, a dictionary lookup) keeps working on the Latin
// original, so nothing here may ever be fed back into the data.
//
// In the course Serbian is always Latin and Russian is always Cyrillic, so every
// Latin run in a displayed string is Serbian and only those runs are converted;
// Cyrillic text is returned byte for byte (a blanket Unicode normalisation would
// split Russian й and ё). scripts/check_cyrillic.py runs the same rules over the
// whole course, and scripts/translit_vectors.json keeps the two implementations
// identical.
export type Script = 'latin' | 'cyrillic'

const LETTERS: Record<string, string> = {
  a: 'а', b: 'б', c: 'ц', č: 'ч', ć: 'ћ', d: 'д', đ: 'ђ', e: 'е', f: 'ф', g: 'г', h: 'х',
  i: 'и', j: 'ј', k: 'к', l: 'л', m: 'м', n: 'н', o: 'о', p: 'п', r: 'р', s: 'с', š: 'ш',
  t: 'т', u: 'у', v: 'в', z: 'з', ž: 'ж',
}
// lj, nj and dž are single letters (and single code points in the rare ǉ ǌ ǆ).
const DIGRAPHS: Record<string, string> = { lj: 'љ', nj: 'њ', dž: 'џ' }
const LIGATURES: Record<string, string> = {
  ǆ: 'џ', ǅ: 'Џ', Ǆ: 'Џ', ǉ: 'љ', ǈ: 'Љ', Ǉ: 'Љ', ǌ: 'њ', ǋ: 'Њ', Ǌ: 'Њ',
}

// Words where the pair really is two letters (in + jekcija, nad + živeti). A
// marker between them stops the digraph from forming; it is dropped again.
const SPLIT = '\u0000'
const TWO_LETTERS: RegExp[] = [/(in)(?=jekc)/gi, /(kon)(?=junkt)/gi, /(nad)(?=živ)/gi, /(od)(?=žvak)/gi, /(pod)(?=župan)/gi]

// A Latin run: one Latin letter plus its combining marks (U+0300-U+036F),
// repeated. Precomposed č ć š ž đ and accented vowels sit in Latin-1 / Latin
// Extended-A (U+00C0-U+017F); U+01C4-U+01CC are the digraph ligatures.
const LATIN_RUN = /(?:[A-Za-zÀ-ſǄ-ǌ][̀-ͯ]*)+/g

function recompose(run: string): string {
  // Stress is a combining acute (U+0301) after a vowel or a syllabic r. A
  // consonant is never stressed, so c + acute is the letter ć, and c/s/z + caron
  // (U+030C) is č š ž.
  return run
    .normalize('NFD')
    .replace(/([cC])́/g, (_, c: string) => (c === 'c' ? 'ć' : 'Ć'))
    .replace(/([csz])̌/gi, (_, c: string) => ({ c: 'č', C: 'Č', s: 'š', S: 'Š', z: 'ž', Z: 'Ž' })[c] ?? c)
}

function convertRun(run: string): string {
  let s = recompose(run)
  for (const re of TWO_LETTERS) s = s.replace(re, `$1${SPLIT}`)
  let out = ''
  for (let i = 0; i < s.length; i++) {
    const ch = s[i]
    if (ch === SPLIT) continue
    const lig = LIGATURES[ch]
    if (lig) {
      out += lig
      continue
    }
    const pair = DIGRAPHS[(ch + (s[i + 1] ?? '')).toLowerCase()]
    if (pair) {
      out += ch === ch.toLowerCase() ? pair : pair.toUpperCase()
      i++
      continue
    }
    const low = ch.toLowerCase()
    const cyr = LETTERS[low]
    out += cyr ? (ch === low ? cyr : cyr.toUpperCase()) : ch
  }
  return out
}

export function toCyrillic(text: string): string {
  return text.replace(LATIN_RUN, convertRun)
}

/** The text as it should be shown in the learner's script. Never store the result. */
export function toScript(text: string | null | undefined, script: Script): string {
  if (!text) return ''
  return script === 'cyrillic' ? toCyrillic(text) : text
}
