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
})
