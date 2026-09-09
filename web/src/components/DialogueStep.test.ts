import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import DialogueStep from './DialogueStep.vue'
import { api } from '../api'
import type { Step, Exercise } from '../types'

afterEach(() => vi.restoreAllMocks())
beforeEach(() => localStorage.clear())

const step: Step = {
  id: '05.9',
  kind: 'dialogue',
  title: 'У пекари',
  scene: 'Ты зашёл в пекару.',
  status: 'not_started',
  turns: [
    { who: 'npc', sr: 'Izvolite?', ru: 'Слушаю вас?' },
    { who: 'me', exercise_id: '05.9.1' },
    { who: 'npc', sr: 'Hvala.', ru: 'Спасибо.' },
  ],
}

const exercises: Exercise[] = [
  {
    id: '05.9.1',
    type: 'choice',
    prompt: 'Попроси хлеб',
    options: ['Jedan hleb, molim.', 'Jedan hleb, hvala.'],
  },
]

const props = { lesson: '05', step, exercises, priors: {} }

describe('DialogueStep', () => {
  it('shows the scene and only the lines up to the active turn', () => {
    const w = mount(DialogueStep, { props })
    expect(w.text()).toContain('Ты зашёл в пекару.')
    expect(w.text()).toContain('Izvolite?')
    expect(w.text()).toContain('Попроси хлеб')
    expect(w.text()).not.toContain('Hvala.')
  })

  it('turns a correct answer into a bubble and advances the conversation', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({
      ok: true,
      line: 'Jedan hleb, molim.',
      line_ru: 'Один хлеб, пожалуйста.',
    })
    const w = mount(DialogueStep, { props })
    await w.findAll('button').find((b) => b.text() === 'Jedan hleb, molim.')!.trigger('click')
    await flushPromises()

    expect(w.emitted('graded')?.[0]).toEqual(['05.9.1', true])
    expect(w.text()).toContain('Hvala.')
  })

  it('keeps the correct line after a wrong answer so the context holds', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({
      ok: false,
      line: 'Jedan hleb, molim.',
      line_ru: 'Один хлеб, пожалуйста.',
      expected: 'Jedan hleb, molim.',
    })
    const w = mount(DialogueStep, { props })
    await w.findAll('button').find((b) => b.text() === 'Jedan hleb, hvala.')!.trigger('click')
    await flushPromises()

    expect(w.emitted('graded')?.[0]).toEqual(['05.9.1', false])
    expect(w.text()).toContain('Jedan hleb, molim.')
    expect(w.text()).toContain('Hvala.')
  })

  it('shows every translation at once when the toggle is on', async () => {
    const w = mount(DialogueStep, { props })
    await w.find('[data-test="toggle-translations"]').trigger('click')
    expect(w.text()).toContain('Слушаю вас?')
  })

  it('remembers the autoplay toggle between dialogues', async () => {
    const w = mount(DialogueStep, { props })
    await w.find('[data-test="toggle-autoplay"]').trigger('click')
    expect(localStorage.getItem('dialogue.autoplay')).toBe('1')

    const again = mount(DialogueStep, { props })
    expect(again.find('[data-test="toggle-autoplay"]').classes().join(' ')).toContain('accent')
  })

  it('reveals a turn answered in an earlier session from the lesson payload', () => {
    const answered: Step = {
      ...step,
      turns: [
        step.turns![0],
        { who: 'me', exercise_id: '05.9.1', sr: 'Jedan hleb, molim.', ru: 'Один хлеб, пожалуйста.' },
        step.turns![2],
      ],
    }
    const w = mount(DialogueStep, {
      props: { ...props, step: answered, priors: { '05.9.1': { answer: 'Jedan hleb, molim.', correct: true } } },
    })
    expect(w.text()).toContain('Jedan hleb, molim.')
    expect(w.text()).toContain('Hvala.')
    expect(w.text()).not.toContain('Попроси хлеб')
  })
})
