import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LessonView from './LessonView.vue'
import { api } from '../api'
import type { Lesson, ExerciseBlock } from '../types'

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '01' }, query: {} }),
  RouterLink: { template: '<a><slot /></a>' },
}))

const lesson: Lesson = {
  id: '01',
  title: 'Урок 01',
  subtitle: '',
  planned: false,
  markdown: '',
  status: 'not_started',
  steps: [
    { id: '01.1', kind: 'teach', title: 'Теория', markdown: '# Привет', status: 'not_started' },
    { id: '01.2', kind: 'practice', title: 'Практика', exercise_ids: ['01.2.1'], status: 'not_started' },
    { id: '01.r', kind: 'reading', title: 'Чтение', markdown: 'Zdravo.', markdown_ru: 'Привет.', status: 'not_started' },
  ],
}
const blocks: ExerciseBlock[] = [
  { id: '01.2', title: 'Практика', exercises: [{ id: '01.2.1', type: 'translate', prompt: 'Привет' }] },
]

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('LessonView step player', () => {
  it('shows one step at a time and advances on Дальше', async () => {
    vi.spyOn(api, 'lesson').mockResolvedValue(structuredClone(lesson))
    vi.spyOn(api, 'exercises').mockResolvedValue(structuredClone(blocks))
    vi.spyOn(api, 'lessonAttempts').mockResolvedValue({})
    const setStep = vi.spyOn(api, 'setStepStatus').mockResolvedValue(undefined)

    const w = mount(LessonView)
    await flushPromises()

    // step 1: teach visible, practice not rendered
    expect(w.text()).toContain('Привет')
    expect(w.findComponent({ name: 'ExerciseBlock' }).exists()).toBe(false)

    await w.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(setStep).toHaveBeenCalledWith('01', '01.1', 'done')
    expect(w.findComponent({ name: 'ExerciseBlock' }).exists()).toBe(true)
  })

  it('resumes at the first unfinished step', async () => {
    const partly = structuredClone(lesson)
    partly.steps![0].status = 'done'
    vi.spyOn(api, 'lesson').mockResolvedValue(partly)
    vi.spyOn(api, 'exercises').mockResolvedValue(structuredClone(blocks))
    vi.spyOn(api, 'lessonAttempts').mockResolvedValue({})
    vi.spyOn(api, 'setStepStatus').mockResolvedValue(undefined)

    const w = mount(LessonView)
    await flushPromises()

    expect(w.text()).toContain('Практика')
    expect(w.findComponent({ name: 'ExerciseBlock' }).exists()).toBe(true)
  })
})

describe('LessonView dialogue step', () => {
  const dialogueLesson: Lesson = {
    ...lesson,
    steps: [
      {
        id: '01.d',
        kind: 'dialogue',
        title: 'У пекари',
        scene: 'Ты зашёл в пекару.',
        exercise_ids: ['01.d.1'],
        status: 'not_started',
        turns: [
          { who: 'npc', sr: 'Izvolite?', ru: 'Слушаю вас?' },
          { who: 'me', exercise_id: '01.d.1' },
        ],
      },
    ],
  }
  const dialogueBlocks: ExerciseBlock[] = [
    {
      id: '01.d',
      title: 'У пекари',
      exercises: [
        { id: '01.d.1', type: 'choice', prompt: 'Попроси хлеб', options: ['Jedan hleb, molim.', 'Hvala.'] },
      ],
    },
  ]

  it('renders the chat and gates advancing until the turn is answered', async () => {
    vi.spyOn(api, 'lesson').mockResolvedValue(structuredClone(dialogueLesson))
    vi.spyOn(api, 'exercises').mockResolvedValue(structuredClone(dialogueBlocks))
    vi.spyOn(api, 'lessonAttempts').mockResolvedValue({})
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, line: 'Jedan hleb, molim.', line_ru: 'Один хлеб, пожалуйста.' })

    const w = mount(LessonView)
    await flushPromises()

    expect(w.text()).toContain('Ты зашёл в пекару.')
    expect(w.text()).toContain('Izvolite?')
    expect(w.find('button.btn-primary').attributes('disabled')).toBeDefined()

    await w.findAll('button').find((b) => b.text() === 'Jedan hleb, molim.')!.trigger('click')
    await flushPromises()

    expect(w.find('button.btn-primary').attributes('disabled')).toBeUndefined()
  })
})
