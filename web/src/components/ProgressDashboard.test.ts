import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ProgressDashboard from './ProgressDashboard.vue'
import { api } from '../api'
import { useSessionStore } from '../stores/session'
import type { Progress, SessionUser, Vocab } from '../types'

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

describe('ProgressDashboard: repeating exercises', () => {
  it('always has the «Повторить задания» card, with or without mistakes', async () => {
    for (const p of [progress(), progress({ weak_exercises: [{ exercise_id: '01.2.1', lesson: '01', prompt: 'Привет', wrong: 2, total: 3 }] })]) {
      const w = mountDashboard(p)
      await flushPromises()
      const card = w.find('[data-test="repeat-card"]')
      expect(card.attributes('href')).toBe('/repeat')
      expect(card.text()).toContain('Повторить задания')
    }
  })

  it('keeps the words review separate from the exercises review', async () => {
    const w = mountDashboard(progress())
    await flushPromises()
    const hrefs = w.findAll('a').map((a) => a.attributes('href'))
    expect(hrefs).toContain('/review')
    expect(hrefs).toContain('/repeat')
  })

  it('shows no list of mistakes: the batch itself is where they come back', async () => {
    const w = mountDashboard(
      progress({ weak_exercises: [{ exercise_id: '01.2.1', lesson: '01', prompt: 'Привет', wrong: 2, total: 3 }] }),
    )
    await flushPromises()
    expect(w.text()).not.toContain('Стоит повторить')
    expect(w.text()).not.toContain('2/3')
  })
})

describe('ProgressDashboard and the learner’s script', () => {
  const readIn = (script: 'latin' | 'cyrillic') => {
    useSessionStore().user = {
      id: 'u', name: 'Гриша', email: '', email_verified: true,
      telegram: { linked: false, username: '' }, script,
    } as SessionUser
  }
  const wotd: Vocab = { id: 'krompir', latin: 'krompir', cyrillic: 'кромпир', ru: 'картофель' }

  it('leads the word of the day with its Cyrillic spelling and shows the Latin one next to it', async () => {
    readIn('cyrillic')
    const w = mount(ProgressDashboard, { props: { progress: progress(), wotd }, global: { stubs: { RouterLink } } })
    await flushPromises()
    const card = w.findAll('.card').find((c) => c.text().includes('Слово дня'))!
    expect(card.find('.serbian').text()).toBe('кромпир')
    expect(card.text()).toContain('krompir')
  })

  it('shows the translation of the word of the day in Cyrillic when it names a Serbian word', async () => {
    readIn('cyrillic')
    const w = mount(ProgressDashboard, {
      props: { progress: progress(), wotd: { ...wotd, ru: 'картофель (в Сербии: krompir)' } },
      global: { stubs: { RouterLink } },
    })
    await flushPromises()
    expect(w.text()).toContain('картофель (в Сербии: кромпир)')
  })

  it('keeps the Latin first for a Latin reader', async () => {
    readIn('latin')
    const w = mount(ProgressDashboard, { props: { progress: progress(), wotd }, global: { stubs: { RouterLink } } })
    await flushPromises()
    const card = w.findAll('.card').find((c) => c.text().includes('Слово дня'))!
    expect(card.find('.serbian').text()).toBe('krompir')
  })
})
