import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import WordBankAnswer from './WordBankAnswer.vue'
import { api } from '../../api'
import { phraseKey, _setPhraseIndex } from '../../lib/phraseAudio'
import { speakClip } from '../../lib/sound'

vi.mock('../../lib/sound', () => ({ speakClip: vi.fn() }))


afterEach(() => vi.restoreAllMocks())

const props = {
  lesson: '01',
  exerciseId: '01.4.3',
  prompt: 'Соберите: «Меня зовут Ана»',
  bank: ['Zovem', 'se', 'Ana'],
}

describe('WordBankAnswer', () => {
  it('assembles the sentence in tap order and submits it joined', async () => {
    const check = vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Zovem se Ana' })
    const w = mount(WordBankAnswer, { props })

    for (const word of ['Zovem', 'se', 'Ana']) {
      const chip = w.findAll('button').find((b) => b.text() === word && !b.classes().includes('bg-[var(--accent-soft)]'))
      await chip!.trigger('click')
    }
    await w.findAll('button').find((b) => b.text() === 'Проверить')!.trigger('click')
    await flushPromises()

    expect(check).toHaveBeenCalledWith('01', '01.4.3', { answer: 'Zovem se Ana' })
    expect(w.emitted('graded')?.[0]?.[0]).toBe(true)
  })
})

describe('WordBankAnswer reads the sentence out after an answer', () => {
  const clipOf = (t: string) => `p/${phraseKey(t)}.mp3`
  beforeEach(() => vi.mocked(speakClip).mockClear())
  afterEach(() => _setPhraseIndex(null))

  async function assembleAndCheck(words: string[]) {
    const w = mount(WordBankAnswer, { props })
    for (const word of words) {
      const chip = w.findAll('button').find((b) => b.text() === word && !b.classes().includes('bg-[var(--accent-soft)]'))
      await chip!.trigger('click')
    }
    await w.findAll('button').find((b) => b.text() === 'Проверить')!.trigger('click')
    await flushPromises()
  }

  it('reads what was assembled when it is right', async () => {
    _setPhraseIndex([phraseKey('Zovem se Ana')])
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Zovem se Ana' })
    await assembleAndCheck(['Zovem', 'se', 'Ana'])
    expect(speakClip).toHaveBeenCalledTimes(1)
    expect(speakClip).toHaveBeenCalledWith(clipOf('Zovem se Ana'))
  })

  it('reads the right sentence, not the wrong order, when it is wrong', async () => {
    _setPhraseIndex([phraseKey('Zovem se Ana')])
    vi.spyOn(api, 'check').mockResolvedValue({ ok: false, expected: 'Zovem se Ana' })
    await assembleAndCheck(['se', 'Zovem', 'Ana'])
    expect(speakClip).toHaveBeenCalledTimes(1)
    expect(speakClip).toHaveBeenCalledWith(clipOf('Zovem se Ana'))
  })
})
