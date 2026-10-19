import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import MistakesView from './MistakesView.vue'
import { api } from '../api'
import type { MistakeItem, MistakesResponse } from '../types'

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
const batch = (...items: MistakeItem[]): MistakesResponse => ({ total: items.length, items })

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

describe('MistakesView', () => {
  it('shows the first mistake and none of the others', async () => {
    vi.spyOn(api, 'mistakes').mockResolvedValue(batch(choice('a', 'Первое'), choice('b', 'Второе')))
    const w = mount(MistakesView)
    await flushPromises()
    expect(w.text()).toContain('Первое')
    expect(w.text()).not.toContain('Второе')
    expect(w.text()).toContain('0 из 2')
  })

  it('checks an answer against the lesson the exercise came from', async () => {
    vi.spyOn(api, 'mistakes').mockResolvedValue(batch(choice('a', 'Первое', '07')))
    const check = mockCheck()
    const w = mount(MistakesView)
    await flushPromises()
    await pick(w, 'Da')
    expect(check).toHaveBeenCalledWith('07', 'a', { answer: 'Da' })
  })

  it('cannot move on before the exercise is answered', async () => {
    vi.spyOn(api, 'mistakes').mockResolvedValue(batch(choice('a', 'Первое'), choice('b', 'Второе')))
    const w = mount(MistakesView)
    await flushPromises()
    expect(bar(w)!.attributes('disabled')).toBeDefined()
  })

  it('moves to the next mistake after a right answer', async () => {
    vi.spyOn(api, 'mistakes').mockResolvedValue(batch(choice('a', 'Первое'), choice('b', 'Второе')))
    mockCheck()
    const w = mount(MistakesView)
    await flushPromises()
    await pick(w, 'Da')
    await bar(w)!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Второе')
    expect(w.text()).toContain('1 из 2')
  })

  it('brings a wrong answer back at the end instead of dropping it', async () => {
    vi.spyOn(api, 'mistakes').mockResolvedValue(batch(choice('a', 'Первое'), choice('b', 'Второе')))
    mockCheck()
    const w = mount(MistakesView)
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
    expect(w.text()).toContain('Все ошибки исправлены')
  })

  it('reports how many were fixed at the first try', async () => {
    vi.spyOn(api, 'mistakes').mockResolvedValue(batch(choice('a', 'Первое'), choice('b', 'Второе')))
    mockCheck()
    const w = mount(MistakesView)
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

  it('offers the next batch when more mistakes are waiting', async () => {
    const load = vi.spyOn(api, 'mistakes').mockResolvedValueOnce({ total: 3, items: [choice('a', 'Первое')] })
    mockCheck()
    const w = mount(MistakesView)
    await flushPromises()
    await pick(w, 'Da')
    await bar(w)!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Все ошибки исправлены')

    load.mockResolvedValueOnce({ total: 2, items: [choice('b', 'Второе')] })
    await w.findAll('button').find((b) => /Ещё 2 ошибки/.test(b.text()))!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Второе')
    expect(w.text()).toContain('0 из 1')
  })

  it('has no next batch to offer when this was the last of the mistakes', async () => {
    vi.spyOn(api, 'mistakes').mockResolvedValue(batch(choice('a', 'Первое')))
    mockCheck()
    const w = mount(MistakesView)
    await flushPromises()
    await pick(w, 'Da')
    await bar(w)!.trigger('click')
    await flushPromises()
    expect(w.findAll('button').some((b) => /Ещё \d/.test(b.text()))).toBe(false)
  })

  it('says so when there is nothing to fix', async () => {
    vi.spyOn(api, 'mistakes').mockResolvedValue(batch())
    const w = mount(MistakesView)
    await flushPromises()
    expect(w.text()).toContain('Ошибок нет')
    expect(w.findAll('button').some((b) => /Дальше/.test(b.text()))).toBe(false)
  })

  it('offers a retry when the batch cannot be loaded', async () => {
    const load = vi.spyOn(api, 'mistakes').mockRejectedValueOnce(new Error('offline'))
    const w = mount(MistakesView)
    await flushPromises()
    expect(w.text()).toContain('Не получилось загрузить')

    load.mockResolvedValue(batch(choice('a', 'Первое')))
    await w.findAll('button').find((b) => /Повторить/.test(b.text()))!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Первое')
  })

  it('has the script pill in its header', async () => {
    vi.spyOn(api, 'mistakes').mockResolvedValue(batch(choice('a', 'Первое')))
    const w = mount(MistakesView)
    await flushPromises()
    expect(w.find('[data-test="script-cyrillic"]').exists()).toBe(true)
  })

  it('leaves for the profile from the close button', async () => {
    vi.spyOn(api, 'mistakes').mockResolvedValue(batch(choice('a', 'Первое')))
    const w = mount(MistakesView)
    await flushPromises()
    await w.find('button[aria-label="Закрыть"]').trigger('click')
    expect(push).toHaveBeenCalledWith('/profile')
  })

  it('turns dictation into translation once the learner cannot listen', async () => {
    const listen: MistakeItem = {
      lesson: '01',
      exercise: { id: 'l1', type: 'listen', prompt: 'Запиши, что слышишь.', audio: 'a.mp3' },
    }
    const load = vi.spyOn(api, 'mistakes').mockResolvedValueOnce(batch(listen))
    const w = mount(MistakesView)
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
