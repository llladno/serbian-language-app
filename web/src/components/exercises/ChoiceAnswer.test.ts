import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ChoiceAnswer from './ChoiceAnswer.vue'
import { api } from '../../api'
import { phraseKey, _setPhraseIndex } from '../../lib/phraseAudio'

afterEach(() => {
  vi.restoreAllMocks()
  _setPhraseIndex(null)
})

const props = { lesson: '01', exerciseId: '01.2.1', prompt: '«Спасибо» —', options: ['Hvala', 'Molim'] }

describe('ChoiceAnswer', () => {
  it('checks the clicked option and reports success', async () => {
    const check = vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Hvala' })
    const w = mount(ChoiceAnswer, { props })
    await w.findAll('button').find((b) => b.text() === 'Hvala')!.trigger('click')
    await flushPromises()

    expect(check).toHaveBeenCalledWith('01', '01.2.1', { answer: 'Hvala' })
    expect(w.emitted('graded')?.[0]?.[0]).toBe(true)
    expect(w.text()).toContain('Верно')
  })

  it('shows the correct answer after a wrong pick', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: false, expected: 'Hvala' })
    const w = mount(ChoiceAnswer, { props })
    await w.findAll('button').find((b) => b.text() === 'Molim')!.trigger('click')
    await flushPromises()

    expect(w.emitted('graded')?.[0]?.[0]).toBe(false)
    expect(w.text()).toContain('Hvala')
  })

  it('passes the check result and what was picked along with the verdict', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, line: 'Hvala', line_ru: 'Спасибо' })
    const w = mount(ChoiceAnswer, { props })
    await w.findAll('button').find((b) => b.text() === 'Hvala')!.trigger('click')
    await flushPromises()

    expect(w.emitted('graded')?.[0]).toEqual([true, { ok: true, line: 'Hvala', line_ru: 'Спасибо' }, 'Hvala'])
  })

  describe('speaker', () => {
    const speaker = 'button[aria-label="озвучить"]'

    it('never voices the options before an answer (a wrong one must not be heard)', () => {
      _setPhraseIndex([phraseKey('Hvala'), phraseKey('Molim')])
      const w = mount(ChoiceAnswer, { props })
      expect(w.find(speaker).exists()).toBe(false)
    })

    it('after a right pick offers one speaker, for the right phrase', async () => {
      _setPhraseIndex([phraseKey('Hvala'), phraseKey('Molim')])
      vi.spyOn(api, 'check').mockResolvedValue({ ok: true })
      const w = mount(ChoiceAnswer, { props })
      await w.findAll('button').find((b) => b.text() === 'Hvala')!.trigger('click')
      await flushPromises()
      expect(w.findAll(speaker)).toHaveLength(1)
    })

    it('after a wrong pick the speaker reads the correct phrase, not the picked one', async () => {
      _setPhraseIndex([phraseKey('Hvala')])
      vi.spyOn(api, 'check').mockResolvedValue({ ok: false, expected: 'Hvala' })
      const w = mount(ChoiceAnswer, { props })
      await w.findAll('button').find((b) => b.text() === 'Molim')!.trigger('click')
      await flushPromises()
      expect(w.findAll(speaker)).toHaveLength(1)
    })
  })
})
