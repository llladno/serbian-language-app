import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TextAnswer from './TextAnswer.vue'
import { api } from '../../api'

afterEach(() => vi.restoreAllMocks())

const props = { lesson: '01', exerciseId: '01-A-1', type: 'translate' as const, prompt: 'Привет!' }

describe('TextAnswer', () => {
  it('shows the expected answer and a wrong-chunk after a failed check', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({
      ok: false,
      expected: 'Zdravo! Kako si?',
      explain: 'почти',
      diff: [
        { text: 'zdravo', ok: true },
        { text: 'kako', ok: true },
        { text: 'sti', ok: false },
      ],
    })
    const w = mount(TextAnswer, { props })
    await w.find('input').setValue('zdravo kako sti')
    await w.find('form').trigger('submit')
    await flushPromises()

    expect(w.text()).toContain('Zdravo! Kako si?')
    expect(w.text()).toContain('почти')
    expect(w.find('.chunk-wrong').exists()).toBe(true)
    expect(w.emitted('graded')?.[0]).toEqual([false])
  })

  it('reports success and emits graded true', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Zdravo!' })
    const w = mount(TextAnswer, { props })
    await w.find('input').setValue('Zdravo!')
    await w.find('form').trigger('submit')
    await flushPromises()

    expect(w.text()).toContain('Верно')
    expect(w.emitted('graded')?.[0]).toEqual([true])
  })

  it('does not submit an empty answer', async () => {
    const spy = vi.spyOn(api, 'check')
    const w = mount(TextAnswer, { props })
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(spy).not.toHaveBeenCalled()
  })

  it('makes the Serbian prompt words clickable for fill_blank', () => {
    const w = mount(TextAnswer, {
      props: { ...props, type: 'fill_blank', prompt: '___ košta kafa?' },
      global: { stubs: { RouterLink: true } },
    })
    expect(w.findAll('span.cursor-pointer').map((s) => s.text())).toEqual(['košta', 'kafa'])
  })

  it('leaves a Russian translate prompt as plain text', () => {
    const w = mount(TextAnswer, { props: { ...props, type: 'translate', prompt: 'Кто это?' } })
    expect(w.find('span.cursor-pointer').exists()).toBe(false)
    expect(w.text()).toContain('Кто это?')
  })

  it('for listen: plays audio, hides the answer text, grades the typed transcription', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Zdravo, kako si?' })
    const w = mount(TextAnswer, {
      props: {
        ...props,
        type: 'listen',
        prompt: '',
        audio: '01-D-1.mp3',
        exerciseId: '01-D-1',
      },
    })
    // an audio control is present, and nothing gives the answer away before submitting
    expect(w.find('button[aria-label], button[title]').exists()).toBe(true)
    expect(w.text()).not.toContain('Zdravo, kako si?')
    expect(w.text()).toContain('Напиши, что слышишь')

    await w.find('input').setValue('Zdravo, kako si?')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.emitted('graded')?.[0]).toEqual([true])
  })
})
