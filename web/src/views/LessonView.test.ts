import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LessonView from './LessonView.vue'
import { api } from '../api'
import { useToasts } from '../lib/toasts'
import { useRewardModal } from '../lib/rewardModal'
import type { Lesson, ExerciseBlock, Quest } from '../types'

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '01' }, query: {} }),
  useRouter: () => ({ push: vi.fn() }),
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

beforeEach(() => {
  setActivePinia(createPinia())
  useToasts().clearToasts()
  useRewardModal().closeModal()
})
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

  it('shows one exercise at a time inside a practice block and pages through them', async () => {
    const twoEx = structuredClone(lesson)
    twoEx.steps![1].exercise_ids = ['01.2.1', '01.2.2']
    const twoBlocks: ExerciseBlock[] = [
      {
        id: '01.2',
        title: 'Практика',
        exercises: [
          { id: '01.2.1', type: 'translate', prompt: 'Первое' },
          { id: '01.2.2', type: 'translate', prompt: 'Второе' },
        ],
      },
    ]
    vi.spyOn(api, 'lesson').mockResolvedValue(twoEx)
    vi.spyOn(api, 'exercises').mockResolvedValue(twoBlocks)
    vi.spyOn(api, 'lessonAttempts').mockResolvedValue({})
    vi.spyOn(api, 'setStepStatus').mockResolvedValue(undefined)
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true })

    // The lesson's own Далее/Завершить bar only exists while no exercise owns
    // the bottom slot itself (translate-type exercises get their own pinned
    // "Проверить" — see TextAnswer.vue — so only one bottom action shows at once).
    const lessonNavBtn = () => w.findAll('button').find((b) => /Дальше|Завершить/.test(b.text()))
    // jsdom has no layout, so vue-test-utils' isVisible() (which consults
    // bounding boxes) can't be trusted here — check the v-show style directly.
    const hidden = (sel: string) => w.find(sel).attributes('style')?.includes('display: none')

    const w = mount(LessonView)
    await flushPromises()
    await lessonNavBtn()!.trigger('click') // teach -> practice
    await flushPromises()

    // both exercises are mounted (so answered state survives paging back and
    // forth), but only the first one is visible
    expect(hidden('[data-ex="01.2.1"]')).toBeFalsy()
    expect(hidden('[data-ex="01.2.2"]')).toBe(true)
    // ungraded translate exercise owns the bottom slot — the lesson bar steps aside
    expect(lessonNavBtn()).toBeUndefined()

    // answer it via the exercise's own pinned "Проверить", then the lesson bar
    // takes the slot back with "Дальше", which should reveal exercise 2
    await w.find('[data-ex="01.2.1"] input[type="text"]').setValue('Zdravo')
    await w.find('[data-ex="01.2.1"] form').trigger('submit')
    await flushPromises()
    expect(lessonNavBtn()).toBeDefined()
    await lessonNavBtn()!.trigger('click')
    await flushPromises()

    expect(hidden('[data-ex="01.2.1"]')).toBe(true)
    expect(hidden('[data-ex="01.2.2"]')).toBeFalsy()

    // the back arrow should return to the first exercise (still showing its
    // answer, since it stayed mounted) instead of leaving the step
    await w.find('button[aria-label="Назад"]').trigger('click')
    await flushPromises()
    expect(hidden('[data-ex="01.2.1"]')).toBeFalsy()
    expect(w.find('[data-ex="01.2.1"]').text()).toContain('Верно')
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

// Finishing a lesson is where most of the currency is earned: the lesson pays
// a little by itself, and the last lesson of a level also finishes the level.
describe('LessonView, finishing', () => {
  const WALLET = {
    balance: 0,
    currency_one: 'пёрышко',
    currency_few: 'пёрышка',
    currency_many: 'пёрышек',
    streak_days: 1,
  }

  function mountOneStepLesson(quests: Quest[] = []) {
    const oneStep = structuredClone(lesson)
    oneStep.steps = [oneStep.steps![0]]
    vi.spyOn(api, 'lesson').mockResolvedValue(oneStep)
    vi.spyOn(api, 'exercises').mockResolvedValue([])
    vi.spyOn(api, 'lessonAttempts').mockResolvedValue({})
    vi.spyOn(api, 'setStepStatus').mockResolvedValue(undefined)
    vi.spyOn(api, 'wallet').mockResolvedValue(WALLET)
    vi.spyOn(api, 'quests').mockResolvedValue({ quests })
    return mount(LessonView)
  }

  const finishButton = (w: ReturnType<typeof mount>) =>
    w.findAll('button').find((b) => /Завершить/.test(b.text()))

  it('says what the lesson paid', async () => {
    const complete = vi.spyOn(api, 'completeLesson').mockResolvedValue({ status: 'done', reward: 4 })
    const w = mountOneStepLesson()
    await flushPromises()

    await finishButton(w)!.trigger('click')
    await flushPromises()

    expect(complete).toHaveBeenCalledWith('01')
    const { items } = useToasts()
    expect(items.value).toHaveLength(1)
    expect(items.value[0].reward).toBe(4)
    expect(items.value[0].text).toBe('Урок 01')
  })

  it('stays quiet about a lesson that paid nothing', async () => {
    vi.spyOn(api, 'completeLesson').mockResolvedValue({ status: 'done', reward: 0 })
    const w = mountOneStepLesson()
    await flushPromises()

    await finishButton(w)!.trigger('click')
    await flushPromises()

    expect(useToasts().items.value).toHaveLength(0)
  })

  it('claims the level and celebrates it when this was its last lesson', async () => {
    vi.spyOn(api, 'completeLesson').mockResolvedValue({ status: 'done', reward: 4 })
    const claimQuest = vi.spyOn(api, 'claimQuest').mockResolvedValue({ reward: 10, balance: 14 })
    const w = mountOneStepLesson([
      {
        id: 7,
        kind: 'phase_completed',
        title: 'Уровень 1 на 100%',
        description: '',
        target: 100,
        value: 100,
        reward: 10,
        done: true,
        claimed: false,
      },
    ])
    await flushPromises()

    await finishButton(w)!.trigger('click')
    await flushPromises()

    expect(claimQuest).toHaveBeenCalledWith(7)
    const { open, reward, questsLink } = useRewardModal()
    expect(open.value).toBe(true)
    expect(reward.value).toBe(10)
    expect(questsLink.value).toBe(true)
  })
})
