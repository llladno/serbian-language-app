import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ChoiceAnswer from './ChoiceAnswer.vue'
import { api } from '../../api'

afterEach(() => vi.restoreAllMocks())

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

  it('passes the check result along with the verdict', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, line: 'Hvala', line_ru: 'Спасибо' })
    const w = mount(ChoiceAnswer, { props })
    await w.findAll('button').find((b) => b.text() === 'Hvala')!.trigger('click')
    await flushPromises()

    expect(w.emitted('graded')?.[0]).toEqual([true, { ok: true, line: 'Hvala', line_ru: 'Спасибо' }])
  })
})
