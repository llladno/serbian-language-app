import { describe, expect, it } from 'vitest'
import type { StoryContent, StorySentence } from '../types'
import { activeWord, backTarget, seekTarget, buildTimeline, durationLabel, maskAnswer, passFraction, subtitleFor } from './storyPlayer'

function sentence(n: number, sr: string, ru: string): StorySentence {
  return {
    n, who: 'Ana', scene: 1, sr, ru,
    sr_audio: `s${n}_sr.mp3`, sr_ms: 1000, sr_words: [],
    ru_audio: `s${n}_ru.mp3`, ru_ms: 1200, ru_words: [],
  }
}

const content: StoryContent = {
  fast_rate: 1.25,
  chapters: [{ scenes: [{ id: 1, image: 'f1.webp' }], sentences: [sentence(1, 'Zdravo!', 'Привет!'), sentence(2, 'Zovem se Marko.', 'Меня зовут Марко.')] }],
  quiz_inline: [{ after: 2, hint: 'зовут', options: ['zovem', 'živim'], mask: 'zovem' }],
  quiz_final: [],
}

describe('buildTimeline', () => {
  const t = buildTimeline(content)

  it('plays Serbian, then a card, then sr/ru in turn, then a card, then fast Serbian with the question', () => {
    expect(t.map((s) => (s.type === 'clip' ? `${s.pass}${s.lang}${s.sentence.n}` : s.type === 'card' ? `card${s.pass}` : `quiz${s.index}`))).toEqual([
      '1sr1', '1sr2', 'card2',
      '2sr1', '2ru1', '2sr2', '2ru2',
      'card3',
      '3sr1', '3sr2', 'quiz0',
    ])
  })

  it('speeds up only the third pass, by the story\'s own rate, and slows only the first', () => {
    const slowed = buildTimeline({ ...content, slow_rate: 0.85 })
    const rates = slowed.filter((s) => s.type === 'clip').map((s) => (s.type === 'clip' ? [s.pass, s.rate] : []))
    expect(rates.filter(([p]) => p === 1).every(([, r]) => r === 0.85)).toBe(true)
    expect(rates.filter(([p]) => p === 2).every(([, r]) => r === 1)).toBe(true)
    expect(rates.filter(([p]) => p === 3).every(([, r]) => r === 1.25)).toBe(true)
  })

  it('plays the first pass at normal speed when the story has no slow rate', () => {
    for (const c of [content, { ...content, slow_rate: 0 }]) {
      const first = buildTimeline(c).filter((s) => s.type === 'clip' && s.pass === 1)
      expect(first.every((s) => s.type === 'clip' && s.rate === 1)).toBe(true)
    }
  })

  it('stops for a question right after the sentence it names, and not on the other passes', () => {
    const i = t.findIndex((s) => s.type === 'quiz')
    expect(t[i - 1]).toMatchObject({ type: 'clip', pass: 3, sentence: { n: 2 } })
    expect(t.filter((s) => s.type === 'quiz')).toHaveLength(1)
  })

  it('falls back to 1.25 when the story has no rate', () => {
    const fast = buildTimeline({ ...content, fast_rate: 0 }).filter((s) => s.type === 'clip' && s.pass === 3)
    expect(fast.every((s) => s.type === 'clip' && s.rate === 1.25)).toBe(true)
  })
})

describe('activeWord', () => {
  const words = [
    { w: 'a', start_ms: 100, end_ms: 400 },
    { w: 'b', start_ms: 500, end_ms: 900 },
  ]
  it('is -1 before the first word, then follows the last word that has started', () => {
    expect(activeWord(words, 0)).toBe(-1)
    expect(activeWord(words, 100)).toBe(0)
    expect(activeWord(words, 450)).toBe(0) // the pause between words keeps the last one lit
    expect(activeWord(words, 700)).toBe(1)
    expect(activeWord([], 700)).toBe(-1)
  })
})

describe('maskAnswer', () => {
  it('hides a word, keeping the punctuation', () => {
    expect(maskAnswer('Zovem se Marko.', 'zovem')).toEqual(['_____', 'se', 'Marko.'])
    expect(maskAnswer('Malo. Učim.', 'malo')).toEqual(['_____.', 'Učim.'])
  })
  it('hides a phrase of several words', () => {
    expect(maskAnswer('Izvini, ne razumem. Polako, molim.', 'ne razumem')).toEqual(['Izvini,', '_____', '_____.', 'Polako,', 'molim.'])
  })
  it('leaves the sentence alone when the answer is not in it', () => {
    expect(maskAnswer('Zdravo!', 'nema')).toEqual(['Zdravo!'])
  })
})

describe('subtitleFor', () => {
  const t = buildTimeline(content)
  const at = (pass: number, lang: string) => t.find((s) => s.type === 'clip' && s.pass === pass && s.lang === lang)!

  it('pass 1, the first time through, shows Serbian alone', () => {
    const sub = subtitleFor(at(1, 'sr'))
    expect(sub.sr).toMatchObject({ active: true })
    expect(sub.ru).toBeNull()
  })
  it('pass 2 shows both lines and lights the one being spoken', () => {
    expect(subtitleFor(at(2, 'sr'))).toMatchObject({ sr: { active: true }, ru: { active: false } })
    expect(subtitleFor(at(2, 'ru'))).toMatchObject({ sr: { active: false }, ru: { active: true } })
  })
  it('pass 3 shows Serbian with the Russian small under it', () => {
    expect(subtitleFor(at(3, 'sr'))).toMatchObject({ sr: { active: true }, ru: { text: 'Привет!', active: false } })
  })
  it('a card has no subtitles', () => {
    expect(subtitleFor(t.find((s) => s.type === 'card')!)).toEqual({ sr: null, ru: null })
  })
})

describe('durationLabel', () => {
  it('speaks in seconds under a minute and minutes after', () => {
    expect(durationLabel(30000)).toBe('≈ 30 сек')
    expect(durationLabel(30)).toBe('≈ 1 сек')
    expect(durationLabel(150000)).toBe('≈ 3 мин')
  })
})

describe('passFraction', () => {
  const t = buildTimeline(content)
  // 0 sr1, 1 sr2, 2 card2, 3 sr1 ru1 sr2 ru2 (3..6), 7 card3, 8 sr1, 9 sr2, 10 quiz
  it('grows through a pass and stays inside 0..1', () => {
    expect(passFraction(t, 0)).toBeCloseTo(0.25)
    expect(passFraction(t, 1)).toBeCloseTo(0.75)
    expect(passFraction(t, 3)).toBeCloseTo(0.125)
    expect(passFraction(t, 6)).toBeCloseTo(0.875)
    expect(passFraction(t, 10)).toBeGreaterThan(passFraction(t, 9))
    expect(passFraction(t, 10)).toBeLessThanOrEqual(1)
  })
  it('is empty on a card and out of range', () => {
    expect(passFraction(t, 2)).toBe(0)
    expect(passFraction(t, 99)).toBe(0)
  })
})

describe('backTarget', () => {
  const t = buildTimeline(content)
  it('goes to the clip before the current one within the pass', () => {
    expect(backTarget(t, 1)).toBe(0)
    expect(backTarget(t, 5)).toBe(4) // second sr clip of pass 2 -> the Russian clip before it
  })
  it('restarts the first clip of a pass instead of crossing a card', () => {
    expect(backTarget(t, 0)).toBe(0)
    expect(backTarget(t, 3)).toBe(3)
    expect(backTarget(t, 8)).toBe(8)
  })
  it('on a question goes to its own sentence', () => {
    expect(backTarget(t, 10)).toBe(9)
  })
})

describe('seekTarget', () => {
  const t = buildTimeline(content)
  // 0 sr1, 1 sr2 | 2 card2 | 3 sr1 4 ru1 5 sr2 6 ru2 | 7 card3 | 8 sr1 9 sr2 10 quiz
  it('lands on the start of the sentence under the finger, in the pass under the finger', () => {
    expect(seekTarget(t, 0)).toBe(0)
    expect(seekTarget(t, 0.1)).toBe(0) // first half of pass 1: sentence 1
    expect(seekTarget(t, 0.2)).toBe(1) // second half: sentence 2
    expect(seekTarget(t, 0.3)).toBe(3) // pass 2, sentence 1
    expect(seekTarget(t, 0.45)).toBe(5) // pass 2, sentence 2
    expect(seekTarget(t, 0.6)).toBe(8) // pass 3, sentence 1
    expect(seekTarget(t, 0.7)).toBe(9)
  })
  it('in pass 2 opens the pair with the Serbian clip, never the Russian one', () => {
    for (const p of [0.26, 0.3, 0.4, 0.45, 0.49]) {
      const step = t[seekTarget(t, p)!]
      expect(step).toMatchObject({ type: 'clip', pass: 2, lang: 'sr' })
    }
  })
  it('the questions segment has no sentence: it answers null, "go to the questions"', () => {
    expect(seekTarget(t, 0.75)).toBeNull()
    expect(seekTarget(t, 0.8)).toBeNull()
    expect(seekTarget(t, 1)).toBeNull()
    expect(seekTarget(t, 5)).toBeNull()
    expect(seekTarget(t, 0.74)).toBe(9) // the last slice of pass 3 is still a sentence
  })
  it('the ends are safe', () => {
    expect(seekTarget(t, -1)).toBe(0)
  })
  it('never lands on a card or a question', () => {
    for (let p = 0; p < 0.75; p += 0.01) expect(t[seekTarget(t, p)!].type).toBe('clip')
  })
})
