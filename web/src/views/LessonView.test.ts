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

  it('has no button at all until the conversation is over', async () => {
    // lines arrive at once here: the pacing is DialogueStep's own business
    vi.stubGlobal('matchMedia', (q: string) => ({ matches: /reduce/.test(q), media: q, addEventListener() {}, removeEventListener() {} }))
    vi.spyOn(api, 'lesson').mockResolvedValue(structuredClone(dialogueLesson))
    vi.spyOn(api, 'exercises').mockResolvedValue(structuredClone(dialogueBlocks))
    vi.spyOn(api, 'lessonAttempts').mockResolvedValue({})
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, line: 'Jedan hleb, molim.', line_ru: 'Один хлеб, пожалуйста.' })

    const w = mount(LessonView)
    await flushPromises()

    expect(w.text()).toContain('Ты зашёл в пекару.')
    expect(w.text()).toContain('Izvolite?')
    // not a disabled button waiting at the bottom: none
    expect(w.find('button.btn-primary').exists()).toBe(false)

    await w.findAll('button').find((b) => b.text() === 'Jedan hleb, molim.')!.trigger('click')
    await flushPromises()

    expect(w.find('button.btn-primary').exists()).toBe(true)
    expect(w.find('button.btn-primary').attributes('disabled')).toBeUndefined()
    vi.unstubAllGlobals()
  })
})

// Finishing a lesson is where most of the currency is earned: the lesson pays
// a little by itself, and the last lesson of a level also finishes the level.
describe('LessonView, finishing', () => {
  const WALLET = {
    balance: 0,
    currency_one: 'зёрнышко',
    currency_few: 'зёрнышка',
    currency_many: 'зёрнышек',
    streak_days: 1,
  }

  beforeEach(() => {
    // The arc and the counting are tested on their own; here the final screen.
    vi.stubGlobal('matchMedia', (q: string) => ({ matches: /reduce/.test(q), media: q, addEventListener() {}, removeEventListener() {} }))
  })
  afterEach(() => vi.unstubAllGlobals())

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

  it('closes the lesson with a summary instead of a toast', async () => {
    const complete = vi.spyOn(api, 'completeLesson').mockResolvedValue({
      status: 'done',
      reward: 4,
      stats: { answered: 10, mistakes: 2, percent: 80, new_words: 12 },
    })
    const w = mountOneStepLesson()
    await flushPromises()

    await finishButton(w)!.trigger('click')
    await flushPromises()

    expect(complete).toHaveBeenCalledWith('01')
    expect(useToasts().items.value).toHaveLength(0)
    const summary = w.find('[data-test="lesson-summary"]')
    expect(summary.exists()).toBe(true)
    expect(summary.text()).toContain('Урок пройден')
    expect(summary.text()).toContain('80')
    expect(w.find('[data-test="stat-reward"]').text()).toContain('4')
    expect(w.find('[data-test="stat-words"]').text()).toContain('12')
    expect(w.find('[data-test="stat-mistakes"]').text()).toContain('2')
    // the lesson's own bar is gone: the way on is the summary's
    expect(finishButton(w)).toBeUndefined()
    expect(w.find('[data-test="summary-course"]').exists()).toBe(true)
  })

  it('leaves out the seeds when the lesson paid nothing', async () => {
    vi.spyOn(api, 'completeLesson').mockResolvedValue({
      status: 'done',
      reward: 0,
      stats: { answered: 4, mistakes: 0, percent: 100, new_words: 0 },
    })
    const w = mountOneStepLesson()
    await flushPromises()

    await finishButton(w)!.trigger('click')
    await flushPromises()

    expect(useToasts().items.value).toHaveLength(0)
    expect(w.find('[data-test="stat-reward"]').exists()).toBe(false)
    expect(w.find('[data-test="stat-words"]').exists()).toBe(false)
    // no mistakes is the best news there is, so it stays
    expect(w.find('[data-test="stat-mistakes"]').text()).toContain('0')
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

// A wrong answer sends the exercise to the end of its step, and the step is
// only over once everything in it was answered right. Dialogues and lessons
// that are already finished keep the old, forgiving behaviour.
describe('LessonView, a wrong answer goes to the end', () => {
  const twoLesson = (): Lesson => {
    const l = structuredClone(lesson)
    l.steps![1].exercise_ids = ['01.2.1', '01.2.2']
    return l
  }
  const twoBlocks = (): ExerciseBlock[] => [
    {
      id: '01.2',
      title: 'Практика',
      exercises: [
        { id: '01.2.1', type: 'translate', prompt: 'Первое' },
        { id: '01.2.2', type: 'translate', prompt: 'Второе' },
      ],
    },
  ]

  const navBtn = (w: ReturnType<typeof mount>) =>
    w.findAll('button').find((b) => /Дальше|Завершить|Вернёмся позже|Ещё раз/.test(b.text()))
  const hidden = (w: ReturnType<typeof mount>, ex: string) =>
    w.find(`[data-ex="${ex}"]`).attributes('style')?.includes('display: none')

  async function answer(w: ReturnType<typeof mount>, ex: string, text: string) {
    await w.find(`[data-ex="${ex}"] input[type="text"]`).setValue(text)
    await w.find(`[data-ex="${ex}"] form`).trigger('submit')
    await flushPromises()
  }

  // Mounts the lesson and gets onto the practice step: a lesson in progress
  // resumes there on its own (the teach step is done), a fresh or finished one
  // is walked there by hand.
  async function openPractice(opts: { attempts?: Record<string, { answer: string; correct: boolean }>; status?: Lesson['status']; resume?: boolean } = {}) {
    const l = twoLesson()
    if (opts.status) l.status = opts.status
    if (opts.resume) l.steps![0].status = 'done'
    vi.spyOn(api, 'lesson').mockResolvedValue(l)
    vi.spyOn(api, 'exercises').mockResolvedValue(twoBlocks())
    vi.spyOn(api, 'lessonAttempts').mockResolvedValue(opts.attempts ?? {})
    const setStep = vi.spyOn(api, 'setStepStatus').mockResolvedValue(undefined)
    const w = mount(LessonView)
    await flushPromises()
    if (!opts.resume) {
      await navBtn(w)!.trigger('click') // teach -> practice
      await flushPromises()
    }
    return { w, setStep }
  }

  it('puts a wrong exercise last and asks for it again before the step ends', async () => {
    const check = vi.spyOn(api, 'check')
    const { w, setStep } = await openPractice()

    check.mockResolvedValueOnce({ ok: false, expected: 'Zdravo' })
    await answer(w, '01.2.1', 'nope')
    expect(w.find('[data-ex="01.2.1"]').text()).toContain('Не совсем')
    // no instant redo: the way forward is the lesson's own button
    expect(w.find('[data-ex="01.2.1"] button[title="Переделать"]').exists()).toBe(false)
    expect(navBtn(w)!.text()).toContain('Вернёмся позже')

    await navBtn(w)!.trigger('click')
    await flushPromises()
    expect(hidden(w, '01.2.1')).toBe(true)
    expect(hidden(w, '01.2.2')).toBeFalsy()

    check.mockResolvedValueOnce({ ok: true })
    await answer(w, '01.2.2', 'Drugo')
    await navBtn(w)!.trigger('click') // the second was right, so the first comes back
    await flushPromises()

    expect(setStep).not.toHaveBeenCalledWith('01', '01.2', 'done')
    expect(hidden(w, '01.2.1')).toBeFalsy()
    // asked afresh: no earlier verdict, empty field
    expect(w.find('[data-ex="01.2.1"]').text()).not.toContain('Не совсем')
    expect((w.find('[data-ex="01.2.1"] input[type="text"]').element as HTMLInputElement).value).toBe('')

    check.mockResolvedValueOnce({ ok: true })
    await answer(w, '01.2.1', 'Zdravo')
    await navBtn(w)!.trigger('click')
    await flushPromises()
    expect(setStep).toHaveBeenCalledWith('01', '01.2', 'done')
  })

  it('offers "Ещё раз", not "Дальше", when the wrong one is the only one left', async () => {
    const check = vi.spyOn(api, 'check')
    const { w, setStep } = await openPractice()

    check.mockResolvedValueOnce({ ok: true })
    await answer(w, '01.2.1', 'Prvo')
    await navBtn(w)!.trigger('click')
    await flushPromises()

    check.mockResolvedValueOnce({ ok: false, expected: 'Drugo' })
    await answer(w, '01.2.2', 'nope')
    expect(navBtn(w)!.text()).toContain('Ещё раз')

    await navBtn(w)!.trigger('click')
    await flushPromises()
    expect(setStep).not.toHaveBeenCalledWith('01', '01.2', 'done')
    expect(hidden(w, '01.2.2')).toBeFalsy()
    expect((w.find('[data-ex="01.2.2"] input[type="text"]').element as HTMLInputElement).value).toBe('')
  })

  it('asks again for an exercise that was answered wrongly on an earlier visit', async () => {
    const { w } = await openPractice({
      resume: true,
      attempts: { '01.2.1': { answer: 'staro', correct: false } },
    })
    // the wrong exercise is blank rather than replaying the old mistake
    expect(w.find('[data-ex="01.2.1"]').text()).not.toContain('Был ответ с ошибкой')
    expect((w.find('[data-ex="01.2.1"] input[type="text"]').element as HTMLInputElement).value).toBe('')
    // and it cannot be walked past
    expect(navBtn(w)).toBeUndefined()
  })

  it('does not hold back a lesson that is already finished', async () => {
    const { w } = await openPractice({
      status: 'done',
      attempts: {
        '01.2.1': { answer: 'staro', correct: false },
        '01.2.2': { answer: 'drugo', correct: true },
      },
    })
    // the old mistake is shown as it was, and Дальше just goes on
    expect(w.find('[data-ex="01.2.1"]').text()).toContain('Был ответ с ошибкой')
    expect(navBtn(w)!.text()).toContain('Дальше')
    expect(navBtn(w)!.attributes('disabled')).toBeUndefined()
  })

  it('lets a wrong dialogue line stand', async () => {
    const dialogue: Lesson = {
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
    vi.spyOn(api, 'lesson').mockResolvedValue(dialogue)
    vi.spyOn(api, 'exercises').mockResolvedValue([
      {
        id: '01.d',
        title: 'У пекари',
        exercises: [{ id: '01.d.1', type: 'choice', prompt: 'Попроси хлеб', options: ['Jedan hleb, molim.', 'Hvala.'] }],
      },
    ])
    vi.spyOn(api, 'lessonAttempts').mockResolvedValue({})
    vi.spyOn(api, 'check').mockResolvedValue({ ok: false, expected: 'Jedan hleb, molim.', line: 'Jedan hleb, molim.' })

    const w = mount(LessonView)
    await flushPromises()
    await w.findAll('button').find((b) => b.text() === 'Hvala.')!.trigger('click')
    await flushPromises()

    const btn = w.find('button.btn-primary')
    expect(btn.attributes('disabled')).toBeUndefined()
    expect(btn.text()).toContain('Завершить')
  })
})

// Someone who cannot listen is not let off the dictation: it turns into a
// translation of the same sentence, for the rest of that lesson.
describe('LessonView, cannot listen', () => {
  const dictation = (): Lesson => ({
    ...lesson,
    steps: [
      { id: '01.2', kind: 'practice', title: 'Диктант', exercise_ids: ['01.2.1'], status: 'not_started' },
    ],
  })
  const audioBlocks: ExerciseBlock[] = [
    { id: '01.2', title: 'Диктант', exercises: [{ id: '01.2.1', type: 'listen', prompt: 'Запиши, что слышишь.', audio: '01.2.1.mp3' }] },
  ]
  const textBlocks: ExerciseBlock[] = [
    { id: '01.2', title: 'Диктант', exercises: [{ id: '01.2.1', type: 'listen', prompt: 'Переведи на русский.', text: 'Dobar dan.' }] },
  ]

  beforeEach(() => localStorage.clear())

  function setup(lessonOverride: Lesson = dictation()) {
    vi.spyOn(api, 'lesson').mockResolvedValue(lessonOverride)
    const exercises = vi
      .spyOn(api, 'exercises')
      .mockImplementation(async (_id, opts) => structuredClone(opts?.audioOff ? textBlocks : audioBlocks))
    vi.spyOn(api, 'lessonAttempts').mockResolvedValue({})
    vi.spyOn(api, 'setStepStatus').mockResolvedValue(undefined)
    return exercises
  }

  it('swaps the dictation for a translation and remembers it', async () => {
    const exercises = setup()
    const w = mount(LessonView)
    await flushPromises()
    expect(w.find('[data-test="listen-as-text"]').exists()).toBe(false)

    await w.find('[data-test="cant-listen"]').trigger('click')
    await flushPromises()

    expect(exercises).toHaveBeenLastCalledWith('01', { audioOff: true })
    expect(w.find('[data-test="listen-as-text"]').text()).toContain('Dobar dan.')

    // coming back to the lesson later does not bring the audio back
    const again = mount(LessonView)
    await flushPromises()
    expect(again.find('[data-test="listen-as-text"]').exists()).toBe(true)
  })

  it('brings the audio back once the lesson is finished', async () => {
    const finished = dictation()
    finished.status = 'done'
    localStorage.setItem('lesson.silent.01', '1')
    const exercises = setup(finished)

    const w = mount(LessonView)
    await flushPromises()

    expect(exercises).toHaveBeenLastCalledWith('01', { audioOff: false })
    expect(w.find('[data-test="listen-as-text"]').exists()).toBe(false)
    expect(localStorage.getItem('lesson.silent.01')).toBeNull()
  })

  it('stays a dictation if the text cannot be fetched', async () => {
    const exercises = setup()
    const w = mount(LessonView)
    await flushPromises()
    exercises.mockRejectedValueOnce(new Error('offline'))

    await w.find('[data-test="cant-listen"]').trigger('click')
    await flushPromises()

    expect(w.find('[data-test="cant-listen"]').exists()).toBe(true)
    expect(localStorage.getItem('lesson.silent.01')).toBeNull()
  })
})

