import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import RepeatView from './RepeatView.vue'
import { api, ApiError } from '../api'
import type { MistakeItem, RepeatResponse } from '../types'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: {}, query: {} }),
  useRouter: () => ({ push }),
  RouterLink: { template: '<a><slot /></a>' },
}))

const choice = (id: string, prompt: string, lesson = '01'): MistakeItem => ({
  lesson,
  exercise: { id, type: 'choice', prompt, options: ['Da', 'Ne'] },
})
const batch = (...items: MistakeItem[]): RepeatResponse => ({
  total: items.length,
  items,
  unlimited: false,
  session: { size: items.length, remaining: items.length },
})
const unlimitedBatch = (...items: MistakeItem[]): RepeatResponse => ({ ...batch(...items), unlimited: true })

// "Da" is right for every exercise in these tests.
function mockCheck() {
  return vi.spyOn(api, 'check').mockImplementation(async (_l, _e, p) => ({
    ok: p.answer === 'Da',
    expected: 'Da',
  }))
}

const pick = async (w: ReturnType<typeof mount>, option: string) => {
  await w.findAll('button').find((b) => b.text() === option)!.trigger('click')
  await flushPromises()
}
const bar = (w: ReturnType<typeof mount>) =>
  w.findAll('button').find((b) => /Дальше|Завершить|Вернёмся позже|Ещё раз/.test(b.text()))

beforeEach(() => {
  setActivePinia(createPinia())
  push.mockClear()
  localStorage.clear()
})
afterEach(() => vi.restoreAllMocks())

describe('RepeatView', () => {
  it('shows the first mistake and none of the others', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue(batch(choice('a', 'Первое'), choice('b', 'Второе')))
    const w = mount(RepeatView)
    await flushPromises()
    expect(w.text()).toContain('Первое')
    expect(w.text()).not.toContain('Второе')
    expect(w.text()).toContain('0 из 2')
  })

  it('checks an answer against the lesson the exercise came from', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue(batch(choice('a', 'Первое', '07')))
    const check = mockCheck()
    const w = mount(RepeatView)
    await flushPromises()
    await pick(w, 'Da')
    expect(check).toHaveBeenCalledWith('07', 'a', { answer: 'Da' })
  })

  it('cannot move on before the exercise is answered', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue(batch(choice('a', 'Первое'), choice('b', 'Второе')))
    const w = mount(RepeatView)
    await flushPromises()
    expect(bar(w)!.attributes('disabled')).toBeDefined()
  })

  it('moves to the next mistake after a right answer', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue(batch(choice('a', 'Первое'), choice('b', 'Второе')))
    mockCheck()
    const w = mount(RepeatView)
    await flushPromises()
    await pick(w, 'Da')
    await bar(w)!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Второе')
    expect(w.text()).toContain('1 из 2')
  })

  it('brings a wrong answer back at the end instead of dropping it', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue(batch(choice('a', 'Первое'), choice('b', 'Второе')))
    mockCheck()
    const w = mount(RepeatView)
    await flushPromises()

    await pick(w, 'Ne') // a: wrong
    expect(bar(w)!.text()).toContain('Вернёмся позже')
    await bar(w)!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Второе')

    await pick(w, 'Da') // b: right
    await bar(w)!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Первое') // a is back
    expect(w.text()).toContain('1 из 2')

    await pick(w, 'Da')
    await bar(w)!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Серия пройдена')
  })

  it('reports how many were fixed at the first try', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue(batch(choice('a', 'Первое'), choice('b', 'Второе')))
    mockCheck()
    const w = mount(RepeatView)
    await flushPromises()
    await pick(w, 'Ne')
    await bar(w)!.trigger('click')
    await flushPromises()
    await pick(w, 'Da')
    await bar(w)!.trigger('click')
    await flushPromises()
    await pick(w, 'Da')
    await bar(w)!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Исправлено 2')
    expect(w.text()).toContain('с первого раза — 1')
  })

  it('offers another batch only to a learner with no daily limit', async () => {
    const load = vi.spyOn(api, 'repeat').mockResolvedValueOnce(unlimitedBatch(choice('a', 'Первое')))
    mockCheck()
    const w = mount(RepeatView)
    await flushPromises()
    await pick(w, 'Da')
    await bar(w)!.trigger('click')
    await flushPromises()
    expect(w.find('[data-test="repeat-again"]').exists()).toBe(true)

    load.mockResolvedValueOnce(unlimitedBatch(choice('b', 'Второе')))
    await w.find('[data-test="repeat-again"]').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Второе')
    expect(w.text()).toContain('0 из 1')
  })

  it('has no second batch to offer a learner on the free plan', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue(batch(choice('a', 'Первое')))
    mockCheck()
    const w = mount(RepeatView)
    await flushPromises()
    await pick(w, 'Da')
    await bar(w)!.trigger('click')
    await flushPromises()
    expect(w.find('[data-test="repeat-again"]').exists()).toBe(false)
    expect(w.text()).toContain('Серия пройдена')
  })

  it('answers the daily limit with the modal about a subscription, not an error', async () => {
    vi.spyOn(api, 'repeat').mockRejectedValue(new ApiError(402, 'repeat_limit'))
    const w = mount(RepeatView, { global: { stubs: { teleport: true } } })
    await flushPromises()
    expect(w.find('[data-test="repeat-limited"]').exists()).toBe(true)
    expect(w.text()).toContain('повторения без ограничений')
    expect(w.text()).not.toContain('Не получилось загрузить')
    await w.find('[data-test="repeat-limit-close"]').trigger('click')
    expect(push).toHaveBeenCalledWith('/profile')
  })

  it('says so when there is nothing to fix', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue(batch())
    const w = mount(RepeatView)
    await flushPromises()
    expect(w.text()).toContain('Пока нечего повторять')
    expect(w.findAll('button').some((b) => /Дальше/.test(b.text()))).toBe(false)
    await w.findAll('button').find((b) => /К курсу/.test(b.text()))!.trigger('click')
    expect(push).toHaveBeenCalledWith('/course')
  })

  it('offers a retry when the batch cannot be loaded', async () => {
    const load = vi.spyOn(api, 'repeat').mockRejectedValueOnce(new Error('offline'))
    const w = mount(RepeatView)
    await flushPromises()
    expect(w.text()).toContain('Не получилось загрузить')

    load.mockResolvedValue(batch(choice('a', 'Первое')))
    await w.findAll('button').find((b) => /Повторить/.test(b.text()))!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Первое')
  })

  it('has the script pill in its header', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue(batch(choice('a', 'Первое')))
    const w = mount(RepeatView)
    await flushPromises()
    expect(w.find('[data-test="script-cyrillic"]').exists()).toBe(true)
  })

  it('leaves for the profile from the close button', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue(batch(choice('a', 'Первое')))
    const w = mount(RepeatView)
    await flushPromises()
    await w.find('button[aria-label="Закрыть"]').trigger('click')
    expect(push).toHaveBeenCalledWith('/profile')
  })

  it('turns dictation into translation once the learner cannot listen', async () => {
    const listen: MistakeItem = {
      lesson: '01',
      exercise: { id: 'l1', type: 'listen', prompt: 'Запиши, что слышишь.', audio: 'a.mp3' },
    }
    const load = vi.spyOn(api, 'repeat').mockResolvedValueOnce(batch(listen))
    const w = mount(RepeatView)
    await flushPromises()

    load.mockResolvedValueOnce(
      batch({ lesson: '01', exercise: { id: 'l1', type: 'listen', prompt: 'Переведи на русский.', text: 'Zdravo' } }),
    )
    await w.findComponent({ name: 'TextAnswer' }).vm.$emit('cantListen')
    await flushPromises()

    expect(load).toHaveBeenLastCalledWith({ audioOff: true })
    expect(w.text()).toContain('Переведи на русский.')
    expect(localStorage.getItem('mistakes.silent')).toBe('1')
  })
})
