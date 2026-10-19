import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { api } from '../../api'
import { useSessionStore } from '../../stores/session'
import type { CheckResult, SessionUser } from '../../types'
import ChoiceAnswer from './ChoiceAnswer.vue'
import WordBankAnswer from './WordBankAnswer.vue'
import MatchAnswer from './MatchAnswer.vue'
import TextAnswer from './TextAnswer.vue'
import ConjugateAnswer from './ConjugateAnswer.vue'
import ExerciseBlock from './ExerciseBlock.vue'
import SerbianKeys from '../SerbianKeys.vue'

// What a Cyrillic reader sees, and — just as important — what still goes to the
// server: the answers an exercise sends are always the Latin originals, because
// the server compares them with the Latin answer.

function readIn(script: 'latin' | 'cyrillic') {
  useSessionStore().user = {
    id: 'u',
    name: 'Гриша',
    email: '',
    email_verified: true,
    telegram: { linked: false, username: '' },
    script,
  } as SessionUser
}

const check = (r: Partial<CheckResult>) => vi.spyOn(api, 'check').mockResolvedValue(r as CheckResult)
const btn = (w: VueWrapper, text: string) => w.findAll('button').find((b) => b.text() === text)!

beforeEach(() => readIn('cyrillic'))
afterEach(() => vi.restoreAllMocks())

describe('ChoiceAnswer in Cyrillic', () => {
  const props = {
    lesson: '01',
    exerciseId: 'x',
    prompt: 'Скажи «čokolada»',
    options: ['čokolada', 'ćevapi'],
    explain: '«Ćao» — неформально',
  }

  it('shows the prompt and the options in Cyrillic', () => {
    const w = mount(ChoiceAnswer, { props })
    expect(w.text()).toContain('Скажи «чоколада»')
    expect(w.findAll('button.serbian').map((b) => b.text()).sort()).toEqual(['чоколада', 'ћевапи'])
  })

  it('sends the Latin option to the server and shows the right answer in Cyrillic', async () => {
    const spy = check({ ok: false, expected: 'čokolada', explain: '«Ćao» — неформально' })
    const w = mount(ChoiceAnswer, { props })
    await btn(w, 'ћевапи').trigger('click')
    await flushPromises()
    expect(spy).toHaveBeenCalledWith('01', 'x', { answer: 'ćevapi' })
    expect(w.text()).toContain('Правильно: чоколада')
    expect(w.text()).toContain('«Ћао» — неформально')
  })

  it('stays Latin for a Latin reader', () => {
    readIn('latin')
    const w = mount(ChoiceAnswer, { props })
    expect(w.findAll('button.serbian').map((b) => b.text()).sort()).toEqual(['ćevapi', 'čokolada'])
  })
})

describe('WordBankAnswer in Cyrillic', () => {
  const props = { lesson: '01', exerciseId: 'w', prompt: 'Составь: «Я здесь»', bank: ['Ja', 'sam', 'ovde'] }

  it('shows the chips in Cyrillic and sends the sentence in Latin', async () => {
    const spy = check({ ok: false, expected: 'Ja sam ovde' })
    const w = mount(WordBankAnswer, { props })
    expect(w.findAll('button.serbian').map((b) => b.text()).sort()).toEqual(['Ја', 'овде', 'сам'])
    await btn(w, 'Ја').trigger('click')
    await btn(w, 'сам').trigger('click')
    await btn(w, 'овде').trigger('click')
    await btn(w, 'Проверить').trigger('click')
    await flushPromises()
    expect(spy).toHaveBeenCalledWith('01', 'w', { answer: 'Ja sam ovde' })
    expect(w.text()).toContain('Правильно: Ја сам овде')
  })
})

describe('MatchAnswer in Cyrillic', () => {
  it('shows the Serbian column in Cyrillic, leaves Russian, and sends Latin keys', async () => {
    const spy = check({ ok: true })
    const w = mount(MatchAnswer, {
      props: { lesson: '01', exerciseId: 'm', prompt: 'Соедини', left: ['čokolada', 'konj'], right: ['шоколад', 'лошадь'] },
    })
    expect(w.findAll('button.serbian').map((b) => b.text())).toEqual(expect.arrayContaining(['чоколада', 'коњ', 'шоколад', 'лошадь']))
    await btn(w, 'чоколада').trigger('click')
    await btn(w, 'шоколад').trigger('click')
    await btn(w, 'коњ').trigger('click')
    await btn(w, 'лошадь').trigger('click')
    await btn(w, 'Проверить').trigger('click')
    await flushPromises()
    expect(spy).toHaveBeenCalledWith('01', 'm', { pairs: { čokolada: 'шоколад', konj: 'лошадь' } })
  })
})

describe('TextAnswer in Cyrillic', () => {
  it('shows a Serbian prompt, the diff and the right answer in Cyrillic', async () => {
    check({
      ok: false,
      expected: 'Ja sam ovde',
      diff: [
        { text: 'ja', ok: true },
        { text: 'sma', ok: false },
      ],
    })
    const w = mount(TextAnswer, {
      props: { lesson: '01', exerciseId: 't', type: 'translate', prompt: 'Скажи «ja sam ovde» по-сербски' },
    })
    expect(w.text()).toContain('Скажи «ја сам овде» по-сербски')
    await w.find('input').setValue('ja sma')
    await btn(w, 'Проверить').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Правильно: Ја сам овде')
    expect(w.find('.chunk-wrong').text()).toBe('сма')
  })

  it('sends exactly what was typed, whichever script it is in', async () => {
    const spy = check({ ok: true })
    const w = mount(TextAnswer, {
      props: { lesson: '01', exerciseId: 't', type: 'translate', prompt: 'Привет' },
    })
    await w.find('input').setValue('ћао')
    await btn(w, 'Проверить').trigger('click')
    await flushPromises()
    expect(spy).toHaveBeenCalledWith('01', 't', { answer: 'ћао' })
  })

  it('shows the sentence of a dictation turned into a translation in Cyrillic', () => {
    const w = mount(TextAnswer, {
      props: { lesson: '01', exerciseId: 'l', type: 'listen', prompt: 'Переведи на русский.', text: 'Zdravo, kako si?' },
    })
    expect(w.find('[data-test="listen-as-text"]').text()).toContain('Здраво, како си?')
  })

  it('shows what the learner wrote earlier in their script', () => {
    const w = mount(TextAnswer, {
      props: {
        lesson: '01',
        exerciseId: 't',
        type: 'translate',
        prompt: 'Привет',
        prior: { answer: 'zdravo', correct: false },
      },
    })
    expect(w.text()).toContain('ты писал: здраво')
  })
})

describe('ConjugateAnswer in Cyrillic', () => {
  it('shows the verb, the pronouns and the corrections in Cyrillic and sends Latin', async () => {
    const spy = check({
      ok: false,
      forms: [
        { ok: true, expected: 'govorim' },
        { ok: false, expected: 'govoriš' },
      ],
    })
    const w = mount(ConjugateAnswer, {
      props: { lesson: '01', exerciseId: 'c', prompt: 'govoriti', forms: ['ja', 'ti'], meta: 'тип I' },
    })
    expect(w.text()).toContain('говорити')
    expect(w.text()).toContain('ја')
    expect(w.text()).toContain('ти')
    const inputs = w.findAll('input')
    await inputs[0].setValue('govorim')
    await inputs[1].setValue('govorim')
    await btn(w, 'Проверить').trigger('click')
    await flushPromises()
    expect(spy).toHaveBeenCalledWith('01', 'c', { answers: ['govorim', 'govorim'] })
    expect(w.text()).toContain('ти → говориш')
  })

  it('shows the note next to the verb in Cyrillic too (it names Serbian forms)', () => {
    const w = mount(ConjugateAnswer, {
      props: { lesson: '01', exerciseId: 'c', prompt: 'nemati', forms: ['ja'], meta: 'nemati — так же, + nemam/nemaš/nema…' },
    })
    expect(w.text()).toContain('немати — так же, + немам/немаш/нема…')
    expect(w.text()).not.toContain('nemam')
  })

  it('leaves a roman numeral written with the roman-numeral character alone', () => {
    const w = mount(ConjugateAnswer, {
      props: { lesson: '01', exerciseId: 'c', prompt: 'govoriti', forms: ['ja'], meta: 'тип \u2160' },
    })
    expect(w.text()).toContain('тип \u2160')
  })
})

describe('ExerciseBlock in Cyrillic', () => {
  it('shows the instruction above the exercises in Cyrillic', () => {
    const w = mount(ExerciseBlock, {
      props: { lesson: '01', block: { id: 'b', title: 'Блок', instruction: 'Ответь по образцу «ja sam ovde».', exercises: [] } },
    })
    expect(w.text()).toContain('Ответь по образцу «ја сам овде».')
  })
})

describe('SerbianKeys', () => {
  const keys = (w: VueWrapper) => w.findAll('button').map((b) => b.text())

  it('offers č ć đ š ž to a Latin reader', () => {
    readIn('latin')
    expect(keys(mount(SerbianKeys))).toEqual(['č', 'ć', 'đ', 'š', 'ž'])
  })

  it('offers the letters a Russian keyboard lacks to a Cyrillic reader', () => {
    expect(keys(mount(SerbianKeys))).toEqual(['ђ', 'ћ', 'ј', 'љ', 'њ', 'џ'])
  })
})
