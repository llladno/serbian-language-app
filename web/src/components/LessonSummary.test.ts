import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LessonSummary from './LessonSummary.vue'
import { CHEERS, bandFor } from '../lib/cheers'
import type { LessonRef, LessonStats } from '../types'

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  vi.stubGlobal('matchMedia', (q: string) => ({ matches: /reduce/.test(q), media: q, addEventListener() {}, removeEventListener() {} }))
})
afterEach(() => vi.unstubAllGlobals())

const stats = (over: Partial<LessonStats> = {}): LessonStats => ({ answered: 10, mistakes: 2, percent: 80, new_words: 12, ...over })
const next: LessonRef = { id: '02', title: 'Урок 02', subtitle: '', planned: false, status: 'not_started' }
// The figures are set when the arc lands, a tick after mounting.
const show = async (props: Record<string, unknown>) => {
  const w = mount(LessonSummary, { props: { title: 'Урок 01', reward: 4, stats: stats(), next, ...props } })
  await flushPromises()
  return w
}

describe('LessonSummary', () => {
  it('shows the figures the lesson produced', async () => {
    const w = await show({})
    expect(w.text()).toContain('Урок пройден')
    expect(w.text()).toContain('Урок 01')
    expect(w.find('[data-test="stat-reward"]').text()).toContain('4')
    expect(w.find('[data-test="stat-words"]').text()).toContain('12')
    expect(w.find('[data-test="stat-words"]').text()).toContain('новых слов')
    expect(w.find('[data-test="stat-mistakes"]').text()).toContain('2')
    expect(w.find('[data-test="stat-mistakes"]').text()).toContain('ошибки')
  })

  it('names things in the right form', async () => {
    const w = await show({ reward: 1, stats: stats({ new_words: 1, mistakes: 5 }) })
    expect(w.find('[data-test="stat-words"]').text()).toContain('новое слово')
    expect(w.find('[data-test="stat-mistakes"]').text()).toContain('ошибок')
  })

  it('leaves out a figure that would read zero, except the mistakes', async () => {
    const w = await show({ reward: 0, stats: stats({ new_words: 0, mistakes: 0, percent: 100 }) })
    expect(w.find('[data-test="stat-reward"]').exists()).toBe(false)
    expect(w.find('[data-test="stat-words"]').exists()).toBe(false)
    expect(w.find('[data-test="stat-mistakes"]').text()).toContain('0')
  })

  it('encourages with a phrase from the band of the result', async () => {
    for (const percent of [100, 90, 70, 30]) {
      const w = await show({ stats: stats({ percent }) })
      expect(CHEERS[bandFor(percent)]).toContain(w.find('[data-test="summary-cheer"]').text())
    }
  })

  it('offers the next lesson first and the course second', async () => {
    const w = await show({})
    expect(w.find('[data-test="summary-next"]').attributes('href')).toBe('/lesson/02')
    expect(w.find('[data-test="summary-course"]').classes()).toContain('btn-ghost')
  })

  it('has only the way back to the course when there is nothing next', async () => {
    const w = await show({ next: null })
    expect(w.find('[data-test="summary-next"]').exists()).toBe(false)
    expect(w.find('[data-test="summary-course"]').classes()).toContain('btn-primary')
  })

  it('has no arc, phrase or mistakes for a lesson with nothing to answer', async () => {
    const w = await show({ stats: stats({ answered: 0, mistakes: 0, percent: 0 }) })
    expect(w.findComponent({ name: 'ArcGauge' }).exists()).toBe(false)
    expect(w.find('[data-test="summary-cheer"]').exists()).toBe(false)
    expect(w.find('[data-test="stat-mistakes"]').exists()).toBe(false)
    expect(w.find('[data-test="stat-reward"]').exists()).toBe(true)
  })

  it('brings in the rest of the screen once the arc is drawn', async () => {
    const w = await show({})
    expect(w.classes()).toContain('revealed')
  })

  it('copes with a server that sends no stats at all', async () => {
    const w = await show({ stats: undefined })
    expect(w.text()).toContain('Урок пройден')
    expect(w.find('[data-test="stat-reward"]').exists()).toBe(true)
  })
})
