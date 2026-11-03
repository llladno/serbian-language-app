// The pure part of the story player: what plays in which order, which word is lit, what a subtitle shows. No DOM
// and no audio here, so every rule is a plain test; the element that plays the clips is in lib/storyEngine.ts.
import type { StoryContent, StoryInlineQuiz, StorySentence, StoryWord } from '../types'

export type PassNo = 1 | 2 | 3

/** The pause between two clips, at normal speed. */
export const GAP_MS = 450
/** How long a card between passes stays on the screen. */
export const CARD_MS = 2200

export type TimelineStep =
  | { type: 'clip'; pass: PassNo; sentence: StorySentence; lang: 'sr' | 'ru'; rate: number; gapMs: number }
  | { type: 'card'; pass: PassNo; title: string; subtitle: string; ms: number }
  | { type: 'quiz'; pass: 3; index: number; sentence: StorySentence; quiz: StoryInlineQuiz }

export const PASS_CARDS: Record<2 | 3, { title: string; subtitle: string }> = {
  2: { title: 'Теперь с переводом', subtitle: 'Предложение по-сербски, потом по-русски' },
  3: { title: 'А теперь с вопросами', subtitle: 'Отвечайте, когда звук остановится' },
}

/** The name of a pass under the progress bar, as a lesson names its step. */
export const PASS_TITLES: Record<PassNo, string> = {
  1: 'Слушаем по-сербски',
  2: 'С переводом',
  3: 'С вопросами',
}

/** The segments of the progress bar: the three passes and the questions at the end. */
export const PROGRESS_STEPS = [{ id: 'p1' }, { id: 'p2' }, { id: 'p3' }, { id: 'q' }]

/**
 * How far into its pass the step at idx is, 0..1: the share of the pass's sentences already behind it. The cards
 * between passes and the questions of pass 3 count as the sentence they belong to.
 */
export function passFraction(timeline: TimelineStep[], idx: number): number {
  const cur = timeline[idx]
  if (!cur) return 0
  const clips = timeline.filter((s) => s.type === 'clip' && s.pass === cur.pass)
  if (!clips.length || cur.type === 'card') return 0
  const done = timeline.slice(0, idx).filter((s) => s.type === 'clip' && s.pass === cur.pass).length
  // Pass 2 plays each sentence twice (Serbian, Russian): count the clips, not the sentences.
  return Math.min(1, (done + (cur.type === 'clip' ? 0.5 : 0)) / clips.length)
}

/**
 * Where a press or a drag at position p (0..1) of the progress bar lands: the start of the sentence that sits there.
 * The bar has four equal segments, the three passes and the questions. A pass is cut into one slice per sentence,
 * and the target is that sentence's Serbian clip (in the second pass the Serbian clip opens the pair, so the pair
 * plays whole). The last segment is the questions: it has no sentence, and the answer is null, which the screen reads
 * as "go straight to the questions". Returns an index into the timeline, or null.
 */
export function seekTarget(timeline: TimelineStep[], p: number): number | null {
  const pos = Math.min(1, Math.max(0, p)) * PROGRESS_STEPS.length
  const seg = Math.min(PROGRESS_STEPS.length - 1, Math.floor(pos))
  if (seg >= 3) return null
  const pass = (seg + 1) as PassNo
  const frac = pos - seg
  const sr: number[] = []
  timeline.forEach((s, i) => {
    if (s.type === 'clip' && s.pass === pass && s.lang === 'sr') sr.push(i)
  })
  if (!sr.length) return 0
  return sr[Math.min(sr.length - 1, Math.floor(frac * sr.length))]
}

/** The index of the clip to go back to: the one before the current sentence in the same pass, or the current one. */
export function backTarget(timeline: TimelineStep[], idx: number): number {
  const cur = timeline[idx]
  if (!cur || cur.type === 'card') return idx
  // On a question, "back" is its own sentence; on a clip it is the clip before it.
  const from = cur.type === 'quiz' ? idx : idx - 1
  for (let i = from; i >= 0; i--) {
    const s = timeline[i]
    if (s.type === 'clip' && s.pass === cur.pass) return i
    if (s.type === 'card') break
  }
  return idx
}

export function allSentences(c: StoryContent): StorySentence[] {
  return c.chapters.flatMap((ch) => ch.sentences)
}

/**
 * The whole listening in order: pass 1 (Serbian, a little slower for a first listen), pass 2 (a Serbian sentence, then
 * its Russian one), pass 3 (Serbian at the fast rate, stopping for the quick questions). A card announces passes 2 and 3.
 */
export function buildTimeline(c: StoryContent): TimelineStep[] {
  const sentences = allSentences(c)
  const fast = c.fast_rate || 1.25
  const slow = c.slow_rate && c.slow_rate > 0 ? c.slow_rate : 1
  const out: TimelineStep[] = []
  for (const s of sentences) out.push({ type: 'clip', pass: 1, sentence: s, lang: 'sr', rate: slow, gapMs: GAP_MS })
  out.push({ type: 'card', pass: 2, ...PASS_CARDS[2], ms: CARD_MS })
  for (const s of sentences) {
    out.push({ type: 'clip', pass: 2, sentence: s, lang: 'sr', rate: 1, gapMs: GAP_MS })
    out.push({ type: 'clip', pass: 2, sentence: s, lang: 'ru', rate: 1, gapMs: GAP_MS })
  }
  out.push({ type: 'card', pass: 3, ...PASS_CARDS[3], ms: CARD_MS })
  for (const s of sentences) {
    out.push({ type: 'clip', pass: 3, sentence: s, lang: 'sr', rate: fast, gapMs: GAP_MS })
    c.quiz_inline.forEach((q, index) => {
      if (q.after === s.n) out.push({ type: 'quiz', pass: 3, index, sentence: s, quiz: q })
    })
  }
  return out
}

/** The index of the word being spoken at tMs of the clip: the last one that has started, or -1 before the first. */
export function activeWord(words: StoryWord[], tMs: number): number {
  let idx = -1
  for (let i = 0; i < words.length; i++) {
    if (words[i].start_ms <= tMs) idx = i
    else break
  }
  return idx
}

const BLANK = '_____'

/**
 * The words of a sentence with the answer of a quick question hidden, so the learner has to recall it. The answer may
 * be several words ("ne razumem"); it is found case-insensitively, ignoring the punctuation stuck to a word.
 */
export function maskAnswer(sentence: string, answer: string): string[] {
  const words = sentence.split(/\s+/).filter(Boolean)
  const norm = (w: string) => w.toLowerCase().replace(/[^\p{L}]/gu, '')
  const target = answer.split(/\s+/).filter(Boolean).map(norm)
  if (!target.length) return words
  for (let i = 0; i + target.length <= words.length; i++) {
    if (target.every((t, k) => norm(words[i + k]) === t)) {
      const out = [...words]
      for (let k = 0; k < target.length; k++) {
        const w = words[i + k]
        const tail = w.match(/[^\p{L}]+$/u)?.[0] ?? ''
        out[i + k] = BLANK + tail
      }
      return out
    }
  }
  return words
}

export interface Subtitle {
  /** Serbian line and whether it is the one being spoken. */
  sr: { text: string; active: boolean; words: StoryWord[] } | null
  ru: { text: string; active: boolean } | null
}

/**
 * What the subtitles show for a step. Pass 1, the very first time through: Serbian alone, the learner is to listen
 * before reading a translation. Pass 2: both lines, the spoken one bold. Pass 3: Serbian with the Russian small under it.
 */
export function subtitleFor(step: TimelineStep): Subtitle {
  if (step.type === 'card') return { sr: null, ru: null }
  const s = step.sentence
  const sr = { text: s.sr, words: s.sr_words }
  if (step.pass === 1) return { sr: { ...sr, active: true }, ru: null }
  if (step.pass === 2) {
    const ruSpoken = step.type === 'clip' && step.lang === 'ru'
    return { sr: { ...sr, active: !ruSpoken }, ru: { text: s.ru, active: ruSpoken } }
  }
  return { sr: { ...sr, active: true }, ru: { text: s.ru, active: false } }
}

/** A readable length: "≈ 40 сек" under a minute, "≈ 2 мин" after. */
export function durationLabel(ms: number): string {
  const sec = Math.round(ms / 1000)
  if (sec < 60) return `≈ ${Math.max(sec, 1)} сек`
  return `≈ ${Math.round(sec / 60)} мин`
}
