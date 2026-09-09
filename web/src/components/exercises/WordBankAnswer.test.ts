import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import WordBankAnswer from './WordBankAnswer.vue'
import { api } from '../../api'

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
