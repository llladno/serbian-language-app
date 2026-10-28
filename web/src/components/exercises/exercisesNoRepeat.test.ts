import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { repeatedLines } from '../../test/repeatedLines'
import { api } from '../../api'
import type { CheckResult, LessonAttempt } from '../../types'
import ChoiceAnswer from './ChoiceAnswer.vue'
import TextAnswer from './TextAnswer.vue'
import WordBankAnswer from './WordBankAnswer.vue'
import MatchAnswer from './MatchAnswer.vue'
import ConjugateAnswer from './ConjugateAnswer.vue'
import DialogueStep from '../DialogueStep.vue'
import type { Exercise, Step } from '../../types'

vi.mock('../../lib/sound', () => ({ speakClip: vi.fn(), stopSpeech: vi.fn() }))
afterEach(() => vi.restoreAllMocks())

// No line of text appears twice on an exercise screen (see test/repeatedLines.ts for the
// rule and the report that started it). Every mode of every screen is run through every
// state it can be in.

type State = { name: string; prior?: LessonAttempt; result?: Partial<CheckResult> }

async function answered(w: VueWrapper, result: Partial<CheckResult>, act: (w: VueWrapper) => Promise<void>) {
  const check = vi.spyOn(api, 'check').mockResolvedValue(result as CheckResult)
  await act(w)
  await flushPromises()
  expect(check).toHaveBeenCalledTimes(1) // the screen really is in its answered state
}

describe('a choice screen never shows a line twice', () => {
  const base = { lesson: '11', exerciseId: '11.12.3', options: ['Это магазин.', 'Это рынок.', 'Это мясная лавка.'] }
  const modes = {
    plain: { ...base, prompt: 'Как будет «рынок»?' },
    listen: { ...base, prompt: 'Послушай и выбери перевод.', audio: '11.12.3.mp3' },
    'listen read as text': { ...base, prompt: 'Выбери перевод.', text: 'Ovo je pijaca.' },
  }
  const states: State[] = [
    { name: 'before an answer' },
    { name: 'answered right', result: { ok: true, expected: 'Это рынок.', say: 'Ovo je pijaca.' } },
    { name: 'answered wrong', result: { ok: false, expected: 'Это рынок.', say: 'Ovo je pijaca.' } },
    { name: 'answered earlier, right', prior: { answer: 'Это рынок.', correct: true } },
    { name: 'answered earlier, wrong', prior: { answer: 'Это магазин.', correct: false } },
  ]
  for (const [mode, props] of Object.entries(modes)) {
    for (const st of states) {
      it(`${mode}, ${st.name}`, async () => {
        const w = mount(ChoiceAnswer, { props: { ...props, prior: st.prior } })
        if (st.result) {
          const pick = st.result.ok ? 'Это рынок.' : 'Это магазин.'
          await answered(w, st.result, (w) => w.findAll('button').find((b) => b.text() === pick)!.trigger('click'))
        }
        expect(repeatedLines(w)).toEqual([])
      })
    }
  }
})

describe('a text screen never shows a line twice', () => {
  const base = { lesson: '11', exerciseId: '11.14.1' }
  // Serbian answers, except a dictation read as text: that one is translated into Russian.
  const sr = { right: 'Ovo je pijaca.', typo: 'Ovo je pjaca.', wrong: 'Ovo je', earlierWrong: 'Ovo je pijac.' }
  const ru = { right: 'Это рынок.', typo: 'Это рынк.', wrong: 'Это', earlierWrong: 'Это магазин.' }
  const modes = {
    dictation: { props: { ...base, type: 'listen' as const, prompt: 'Запиши, что услышал.', audio: '11.14.1.mp3' }, a: sr },
    'dictation read as text': { props: { ...base, type: 'listen' as const, prompt: 'Переведи.', text: 'Ovo je pijaca.' }, a: ru },
    translate: { props: { ...base, type: 'translate' as const, prompt: 'Это рынок.' }, a: sr },
    fill_blank: { props: { ...base, type: 'fill_blank' as const, prompt: '«Это рынок» → Ovo je ___.' }, a: sr },
    fix_error: { props: { ...base, type: 'fix_error' as const, prompt: 'Ovo su pijaca.' }, a: sr },
  }
  for (const [mode, { props, a }] of Object.entries(modes)) {
    const states: (State & { typed?: string })[] = [
      { name: 'before an answer' },
      { name: 'answered right', typed: a.right, result: { ok: true, expected: a.right } },
      { name: 'answered right with a typo', typed: a.typo, result: { ok: true, typo: true, expected: a.right } },
      { name: 'answered wrong', typed: a.wrong, result: { ok: false, expected: a.right } },
      { name: 'answered earlier, right', prior: { answer: a.right, correct: true } },
      { name: 'answered earlier, wrong', prior: { answer: a.earlierWrong, correct: false } },
    ]
    for (const st of states) {
      it(`${mode}, ${st.name}`, async () => {
        const w = mount(TextAnswer, { props: { ...props, prior: st.prior } })
        if (st.result) {
          await answered(w, st.result, async (w) => {
            await w.find('input').setValue(st.typed!)
            await w.find('form').trigger('submit')
          })
        }
        expect(repeatedLines(w)).toEqual([])
      })
    }
  }
})

const tap = (w: VueWrapper, text: string) => w.findAll('button').find((b) => b.text() === text)!.trigger('click')

describe('a word bank screen never shows a line twice', () => {
  const props = { lesson: '11', exerciseId: '11.15.1', prompt: 'Собери: «Это рынок.»', bank: ['Ovo', 'je', 'pijaca.'] }
  const states: State[] = [
    { name: 'before an answer' },
    { name: 'answered right', result: { ok: true, expected: 'Ovo je pijaca.' } },
    { name: 'answered wrong', result: { ok: false, expected: 'Ovo je pijaca.' } },
    { name: 'answered earlier, right', prior: { answer: 'Ovo je pijaca.', correct: true } },
    { name: 'answered earlier, wrong', prior: { answer: 'je Ovo pijaca.', correct: false } },
  ]
  for (const st of states) {
    it(st.name, async () => {
      const w = mount(WordBankAnswer, { props: { ...props, prior: st.prior } })
      if (st.result) {
        await answered(w, st.result, async (w) => {
          for (const word of st.result!.ok ? ['Ovo', 'je', 'pijaca.'] : ['je', 'Ovo', 'pijaca.']) await tap(w, word)
          await tap(w, 'Проверить')
        })
      }
      expect(repeatedLines(w)).toEqual([])
    })
  }
})

describe('a matching screen never shows a line twice', () => {
  const props = { lesson: '11', exerciseId: '11.13.1', prompt: 'Соедини.', left: ['pijaca', 'pekara'], right: ['рынок', 'пекарня'] }
  const states: State[] = [
    { name: 'before an answer' },
    { name: 'answered right', result: { ok: true, match: { pijaca: true, pekara: true } } },
    { name: 'answered with one pair wrong', result: { ok: true, partial: true, match: { pijaca: true, pekara: false } } },
    { name: 'answered wrong', result: { ok: false, match: { pijaca: false, pekara: false } } },
    { name: 'answered earlier', prior: { answer: '', correct: true } },
  ]
  for (const st of states) {
    it(st.name, async () => {
      const w = mount(MatchAnswer, { props: { ...props, prior: st.prior } })
      if (st.result) {
        await answered(w, st.result, async (w) => {
          await tap(w, 'pijaca')
          await tap(w, 'рынок')
          await tap(w, 'pekara')
          await tap(w, 'пекарня')
          await tap(w, 'Проверить')
        })
      }
      expect(repeatedLines(w)).toEqual([])
    })
  }
})

describe('a conjugation screen never shows a line twice', () => {
  const props = { lesson: '06c', exerciseId: '06c.14.1', prompt: 'imati', forms: ['ja', 'ti', 'mi'], meta: 'nemati — так же' }
  const forms = (ok: boolean[], typo = false) =>
    ok.map((o, i) => ({ ok: o, typo: typo && i === 0, expected: ['imam', 'imaš', 'imamo'][i] }))
  const states: State[] = [
    { name: 'before an answer' },
    { name: 'answered right', result: { ok: true, forms: forms([true, true, true]) } },
    { name: 'answered right with a typo', result: { ok: true, forms: forms([true, true, true], true) } },
    { name: 'answered with one form wrong', result: { ok: true, partial: true, forms: forms([true, false, true]) } },
    { name: 'answered wrong', result: { ok: false, forms: forms([false, false, false]), explain: 'imati: imam, imaš…' } },
    { name: 'answered earlier', prior: { answer: 'imam,imaš,imamo', correct: true } },
  ]
  for (const st of states) {
    it(st.name, async () => {
      const w = mount(ConjugateAnswer, { props: { ...props, prior: st.prior } })
      if (st.result) {
        await answered(w, st.result, async (w) => {
          const inputs = w.findAll('input')
          for (const [i, v] of ['imam', 'imaš', 'imamo'].entries()) await inputs[i].setValue(v)
          await tap(w, 'Проверить')
        })
      }
      expect(repeatedLines(w)).toEqual([])
    })
  }
})

describe('a dialogue never shows a line twice', () => {
  const step: Step = {
    id: '11.20',
    kind: 'dialogue',
    title: 'На рынке',
    scene: 'Ты на рынке.',
    status: 'not_started',
    turns: [
      { who: 'npc', sr: 'Izvolite?', ru: 'Слушаю вас?' },
      { who: 'me', exercise_id: '11.20.1' },
      { who: 'npc', sr: 'Izvolite, hvala.', ru: 'Пожалуйста, спасибо.' },
    ],
  }
  const exercises: Exercise[] = [
    { id: '11.20.1', type: 'choice', prompt: 'Попроси кило яблок', options: ['Kilo jabuka, molim.', 'Kilo jabuka, hvala.'] },
  ]
  const line = { line: 'Kilo jabuka, molim.', line_ru: 'Кило яблок, пожалуйста.' }
  const states: (State & { priors?: Record<string, LessonAttempt> })[] = [
    { name: 'before an answer' },
    { name: 'answered right', result: { ok: true, ...line } },
    { name: 'answered wrong', result: { ok: false, expected: 'Kilo jabuka, molim.', explain: 'molim — пожалуйста', ...line } },
    { name: 'answered earlier, wrong', priors: { '11.20.1': { answer: 'Kilo jabuka, hvala.', correct: false } } },
  ]
  for (const st of states) {
    it(st.name, async () => {
      vi.stubGlobal('matchMedia', (q: string) => ({ matches: /reduce/.test(q), media: q, addEventListener() {}, removeEventListener() {} }))
      const w = mount(DialogueStep, { props: { lesson: '11', step, exercises, priors: st.priors ?? {} } })
      if (st.result) {
        const pick = st.result.ok ? 'Kilo jabuka, molim.' : 'Kilo jabuka, hvala.'
        await answered(w, st.result, (w) => tap(w, pick))
      }
      expect(repeatedLines(w)).toEqual([])
      vi.unstubAllGlobals()
    })
  }
})
