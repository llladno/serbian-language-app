import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import DialogueStep from './DialogueStep.vue'
import { api } from '../api'
import type { Step, Exercise } from '../types'

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  vi.useRealTimers()
})
// Most of these tests are about what is on the screen, not how it gets there,
// so lines arrive at once (reduced motion); the pacing has its own block below.
const reducedMotion = (on = true) =>
  vi.stubGlobal('matchMedia', (q: string) => ({ matches: on && /reduce/.test(q), media: q, addEventListener() {}, removeEventListener() {} }))
beforeEach(() => {
  localStorage.clear()
  reducedMotion()
})

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

  it('has no autoplay toggle', () => {
    const w = mount(DialogueStep, { props })
    expect(w.find('[data-test="toggle-autoplay"]').exists()).toBe(false)
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

describe('DialogueStep mistakes', () => {
  it('marks a wrong answer in the chat and keeps its explanation', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({
      ok: false,
      line: 'Jedan hleb, molim.',
      line_ru: 'Один хлеб, пожалуйста.',
      explain: '«molim» — просьба.',
    })
    const w = mount(DialogueStep, { props })
    await w.findAll('button').find((b) => b.text() === 'Jedan hleb, hvala.')!.trigger('click')
    await flushPromises()

    expect(w.find('[data-test="correction"]').text()).toContain('Нужно было ответить так')
    expect(w.text()).toContain('«molim» — просьба.')
    // what was answered, the right line, and its meaning, all in one place
    expect(w.find('[data-test="given"]').text()).toContain('Jedan hleb, hvala.')
    expect(w.text()).toContain('Jedan hleb, molim.')
    expect(w.text()).toContain('Один хлеб, пожалуйста.')
  })

  it('leaves a correct answer unmarked', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, line: 'Jedan hleb, molim.' })
    const w = mount(DialogueStep, { props })
    await w.findAll('button').find((b) => b.text() === 'Jedan hleb, molim.')!.trigger('click')
    await flushPromises()

    expect(w.find('[data-test="correction"]').exists()).toBe(false)
    expect(w.find('[data-test="given"]').exists()).toBe(false)
  })

  it('marks a turn answered wrongly in an earlier session', () => {
    const answeredStep: Step = {
      ...step,
      turns: [
        step.turns![0],
        { who: 'me', exercise_id: '05.9.1', sr: 'Jedan hleb, molim.', ru: 'Один хлеб, пожалуйста.' },
        step.turns![2],
      ],
    }
    const w = mount(DialogueStep, {
      props: { ...props, step: answeredStep, priors: { '05.9.1': { answer: 'Jedan hleb, hvala.', correct: false } } },
    })
    expect(w.find('[data-test="correction"]').exists()).toBe(true)
    expect(w.find('[data-test="given"]').text()).toContain('Jedan hleb, hvala.')
  })

  it('still explains a mistake after the page was reloaded', () => {
    const answeredStep: Step = {
      ...step,
      turns: [
        step.turns![0],
        { who: 'me', exercise_id: '05.9.1', sr: 'Jedan hleb, molim.', ru: 'Один хлеб, пожалуйста.' },
        step.turns![2],
      ],
    }
    const withNote = [{ ...exercises[0], explain: '«molim» — просьба.' }]
    const w = mount(DialogueStep, {
      props: { ...props, step: answeredStep, exercises: withNote, priors: { '05.9.1': { answer: 'Jedan hleb, hvala.', correct: false } } },
    })
    expect(w.text()).toContain('«molim» — просьба.')
  })

  it('does not call a mistake "ты ответил": the wording fits anyone', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: false, line: 'Jedan hleb, molim.' })
    const w = mount(DialogueStep, { props })
    await w.findAll('button').find((b) => b.text() === 'Jedan hleb, hvala.')!.trigger('click')
    await flushPromises()
    expect(w.text()).not.toMatch(/ответил\b/)
  })
})

describe('DialogueStep pacing', () => {
  const twoLines: Step = {
    ...step,
    turns: [
      { who: 'npc', sr: 'Izvolite?', ru: 'Слушаю вас?' },
      { who: 'me', exercise_id: '05.9.1' },
      { who: 'npc', sr: 'Hvala.', ru: 'Спасибо.' },
      { who: 'npc', sr: 'Doviđenja!', ru: 'До свидания!' },
    ],
  }
  const answer = async (w: ReturnType<typeof mount>) => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, line: 'Jedan hleb, molim.', line_ru: 'Один хлеб, пожалуйста.' })
    await w.findAll('button').find((b) => b.text() === 'Jedan hleb, molim.')!.trigger('click')
    await flushPromises()
  }

  it('says the other side\'s lines one after another, not all at once', async () => {
    reducedMotion(false)
    vi.useFakeTimers()
    const w = mount(DialogueStep, { props: { ...props, step: twoLines } })
    await answer(w)

    // the learner's own line is there straight away, the replies are not
    expect(w.text()).toContain('Jedan hleb, molim.')
    expect(w.text()).not.toContain('Hvala.')

    await vi.advanceTimersByTimeAsync(700)
    expect(w.text()).toContain('Hvala.')
    expect(w.text()).not.toContain('Doviđenja!')

    await vi.advanceTimersByTimeAsync(1500)
    expect(w.text()).toContain('Doviđenja!')
  })

  it('only counts as finished when the last line has been said', async () => {
    reducedMotion(false)
    vi.useFakeTimers()
    const w = mount(DialogueStep, { props: { ...props, step: twoLines } })
    expect((w.vm as unknown as { finished: boolean }).finished).toBe(false)
    await answer(w)
    expect((w.vm as unknown as { finished: boolean }).finished).toBe(false) // answered, but still talking
    await vi.advanceTimersByTimeAsync(3000)
    expect((w.vm as unknown as { finished: boolean }).finished).toBe(true)
  })

  it('fades up what arrives later, and leaves what was there alone', async () => {
    reducedMotion(false)
    vi.useFakeTimers()
    const w = mount(DialogueStep, { props: { ...props, step: twoLines } })
    expect(w.find('[data-index="0"]').classes()).not.toContain('msg-in')
    await answer(w)
    await vi.advanceTimersByTimeAsync(700)
    expect(w.find('[data-index="2"]').classes()).toContain('msg-in')
    // the bubble the answer turned into arrives too
    expect(w.find('[data-index="1"]').classes()).toContain('msg-in')
  })

  it('only fades an open question in: it holds the pinned button, which a transform would unpin', async () => {
    reducedMotion(false)
    vi.useFakeTimers()
    const nextQuestion: Step = {
      ...twoLines,
      turns: [
        ...twoLines.turns!.slice(0, 3),
        { who: 'me', exercise_id: '05.9.2' },
      ],
    }
    const exs: Exercise[] = [
      ...exercises,
      { id: '05.9.2', type: 'choice', prompt: 'Попрощайся', options: ['Doviđenja!', 'Dobar dan!'] },
    ]
    const w = mount(DialogueStep, { props: { ...props, step: nextQuestion, exercises: exs } })
    await answer(w)
    await vi.advanceTimersByTimeAsync(2500)
    const open = w.find('[data-index="3"]')
    expect(open.classes()).toContain('msg-fade')
    expect(open.classes()).not.toContain('msg-in')
  })

  it('shows a dialogue that was already finished straight away, done', () => {
    reducedMotion(false)
    const done: Step = {
      ...twoLines,
      turns: [
        twoLines.turns![0],
        { who: 'me', exercise_id: '05.9.1', sr: 'Jedan hleb, molim.', ru: 'Один хлеб, пожалуйста.' },
        twoLines.turns![2],
        twoLines.turns![3],
      ],
    }
    const w = mount(DialogueStep, {
      props: { ...props, step: done, priors: { '05.9.1': { answer: 'Jedan hleb, molim.', correct: true } } },
    })
    expect(w.text()).toContain('Doviđenja!')
    expect((w.vm as unknown as { finished: boolean }).finished).toBe(true)
    expect(w.find('[data-index="3"]').classes()).not.toContain('msg-in')
  })
})
