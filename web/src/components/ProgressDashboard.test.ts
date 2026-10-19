import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ProgressDashboard from './ProgressDashboard.vue'
import { api } from '../api'
import type { Progress } from '../types'

const RouterLink = { props: ['to'], template: '<a :href="to"><slot /></a>' }

function progress(overrides: Partial<Progress> = {}): Progress {
  return {
    phases: [],
    srs: { due_today: 0, new_available: 0, reviewed_today: 0, total_cards: 0, known: 0 },
    weak_exercises: [],
    streak_days: 0,
    recent_lessons: [],
    activity: [],
    daily_goal: 10,
    ...overrides,
  }
}

function mountDashboard(p: Progress) {
  return mount(ProgressDashboard, { props: { progress: p, wotd: null }, global: { stubs: { RouterLink } } })
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.spyOn(api, 'course').mockRejectedValue(new Error('n/a')) // only feeds the "next lesson" card
})
afterEach(() => vi.restoreAllMocks())

describe('ProgressDashboard mistakes', () => {
  it('offers to work through the mistakes when there are some', async () => {
    const w = mountDashboard(
      progress({ weak_exercises: [{ exercise_id: '01.2.1', lesson: '01', prompt: 'Привет', wrong: 2, total: 3 }] }),
    )
    await flushPromises()
    const go = w.findAll('a').find((a) => /Исправить ошибки/.test(a.text()))
    expect(go?.attributes('href')).toBe('/mistakes')
  })

  it('lists the weak exercises as plain text: the button is the way in, not a link per row', async () => {
    const w = mountDashboard(
      progress({ weak_exercises: [{ exercise_id: '01.2.1', lesson: '01', prompt: 'Привет', wrong: 2, total: 3 }] }),
    )
    await flushPromises()
    expect(w.text()).toContain('Стоит повторить')
    expect(w.text()).toContain('2/3')
    expect(w.text()).toContain('Привет')
    expect(w.findAll('a').some((a) => a.attributes('href')?.startsWith('/lesson/') && /Привет/.test(a.text()))).toBe(false)
  })

  it('shows nothing about mistakes when there are none', async () => {
    const w = mountDashboard(progress())
    await flushPromises()
    expect(w.text()).not.toContain('Исправить ошибки')
    expect(w.text()).not.toContain('Стоит повторить')
  })
})
