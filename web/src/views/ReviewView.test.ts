import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ReviewView from './ReviewView.vue'
import { api } from '../api'
import { useSessionStore } from '../stores/session'
import type { GramCheckResult, ReviewCard, SessionUser } from '../types'
import { repeatedLines } from '../test/repeatedLines'

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: {}, query: {} }),
  useRouter: () => ({ push: vi.fn() }),
  RouterLink: { template: '<a><slot /></a>' },
}))

const preview = { again: 0, hard: 1, good: 3, easy: 7 }
const known: ReviewCard = {
  card_id: 'vocab:biti',
  kind: 'vocab',
  front: 'biti',
  cyrillic: 'бити',
  transcription: 'бы́ти',
  back: 'быть',
  note: 'jesam / sam',
  example_sr: 'Ja sam ovde.',
  example_ru: 'Я здесь.',
  state: 'review',
  preview,
}
const quiz: ReviewCard = { ...known, card_id: 'vocab:biti2', state: 'new', options: ['быть', 'бить', 'бить'] }
const falseFriend: ReviewCard = {
  card_id: 'ff:sto',
  kind: 'ff',
  front: 'sto',
  back: 'сто (число)',
  state: 'review',
  preview,
}
const gram: ReviewCard = {
  card_id: 'gram:prezent-am',
  kind: 'gram',
  front: 'prezent na -am',
  back: '',
  note: 'radim, radiš, radi',
  item_prompt: 'ja ___ (raditi)',
  item_index: 0,
  example_sr: 'Ja radim ovde.',
  example_ru: 'Я работаю здесь.',
  state: 'review',
  preview,
}

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

async function open(card: ReviewCard) {
  vi.spyOn(api, 'reviewQueue').mockResolvedValue([card])
  const w = mount(ReviewView)
  await flushPromises()
  return w
}

afterEach(() => vi.restoreAllMocks())

describe('ReviewView in Cyrillic', () => {
  beforeEach(() => readIn('cyrillic'))

  it('leads a known word with its Cyrillic spelling and shows the Latin one second', async () => {
    const w = await open(known)
    const card = w.find('.card')
    expect(card.find('p.serbian').text()).toBe('бити')
    expect(card.text()).toContain('biti')
    await card.trigger('click')
    expect(w.text()).toContain('Ја сам овде.')
    expect(w.text()).toContain('јесам / сам')
    expect(w.text()).toContain('быть')
  })

  it('does the same for the first-encounter quiz', async () => {
    const w = await open(quiz)
    expect(w.find('p.serbian').text()).toBe('бити')
    expect(w.text()).toContain('biti')
  })

  it('shows the quiz options in Cyrillic when a translation names a Serbian word', async () => {
    const w = await open({ ...quiz, options: ['быть (jesam)', 'бить', 'пить'] })
    expect(w.text()).toContain('быть (јесам)')
  })

  it('converts a word that has no authored Cyrillic (a false friend)', async () => {
    const w = await open(falseFriend)
    expect(w.find('p.serbian').text()).toBe('сто')
  })

  it('shows a grammar card, its asked form and the example in Cyrillic', async () => {
    const w = await open(gram)
    expect(w.text()).toContain('презент на -ам')
    expect(w.text()).toContain('радим, радиш, ради')
    expect(w.text()).toContain('ја ___ (радити)')
  })
})

describe('ReviewView script pill', () => {
  it('is in the header, whatever card is on screen', async () => {
    readIn('latin')
    const w = await open(known)
    expect(w.find('[data-test="script-cyrillic"]').text()).toBe('Кир')
  })
})

describe('ReviewView in Latin', () => {
  beforeEach(() => readIn('latin'))

  it('keeps the Latin headword first, as before', async () => {
    const w = await open(known)
    const card = w.find('.card')
    expect(card.find('p.serbian').text()).toBe('biti')
    expect(card.text()).toContain('бити')
    await card.trigger('click')
    expect(w.text()).toContain('Ja sam ovde.')
  })
})

// No line of text appears twice on a card, in any state (see test/repeatedLines.ts).
describe('ReviewView never shows a line twice', () => {
  beforeEach(() => readIn('latin'))
  const click = async (w: Awaited<ReturnType<typeof open>>, text: string) => {
    await w.findAll('button').find((b) => b.text() === text)!.trigger('click')
    await flushPromises()
  }
  const cases: [string, ReviewCard, ((w: Awaited<ReturnType<typeof open>>) => Promise<void>)?][] = [
    ['a known word, hidden', known],
    ['a known word, shown', known, (w) => click(w, 'Показать')],
    ['a false friend, shown', falseFriend, (w) => click(w, 'Показать')],
    ['a new word, before a pick', quiz],
    ['a new word, picked right', quiz, (w) => click(w, 'быть')],
    ['a new word, picked wrong', { ...quiz, options: ['быть', 'бить', 'выть'] }, (w) => click(w, 'выть')],
    ['a grammar card, before an answer', gram],
  ]
  for (const [name, card, act] of cases) {
    it(name, async () => {
      const w = await open(card)
      await act?.(w)
      expect(repeatedLines(w)).toEqual([])
    })
  }

  const gramAnswers: [string, string, GramCheckResult][] = [
    ['right', 'radim', { ok: true, expected: 'radim' } as GramCheckResult],
    ['right in a phrase with a typo', 'Ja radim ovdee.', { ok: true, typo: true, expected: 'Ja radim ovde.' } as GramCheckResult],
    ['wrong', 'radiš', { ok: false, expected: 'radim' } as GramCheckResult],
    ['a near miss', 'radm', { ok: false, near_miss: true, expected: 'radim' } as GramCheckResult],
  ]
  for (const [name, typed, res] of gramAnswers) {
    it(`a grammar card, answered ${name}`, async () => {
      const grade = vi.spyOn(api, 'gradeGram').mockResolvedValue(res)
      const w = await open(gram)
      await w.find('input').setValue(typed)
      await click(w, 'Проверить')
      expect(grade).toHaveBeenCalledTimes(1)
      expect(repeatedLines(w)).toEqual([])
    })
  }
})
