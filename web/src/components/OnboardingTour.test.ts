import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { api } from '../api'
import OnboardingTour from './OnboardingTour.vue'
import { useOnboarding } from '../lib/onboarding'
import { useSessionStore } from '../stores/session'
import { sessionUser } from '../test/tariffsFixture'

vi.mock('../api', () => ({
  api: { onboarding: vi.fn(), answerOnboarding: vi.fn(), wallet: vi.fn(), quests: vi.fn() },
}))
const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ fullPath: '/course', path: '/course', meta: {} }),
  useRouter: () => ({ push }),
}))

enableAutoUnmount(afterEach)

// A line takes TYPE_MS to type and BEAT_MS more before «Дальше» works (see OnboardingTour.vue).
const LINE = 1500 + 600 + 100

beforeEach(() => {
  vi.useFakeTimers()
  vi.mocked(api.onboarding).mockReset().mockResolvedValue({ due: true, reward: 5 })
  vi.mocked(api.answerOnboarding).mockReset().mockResolvedValue({ reward: 5, balance: 5 })
  vi.mocked(api.wallet)
    .mockReset()
    .mockResolvedValue({ balance: 0, currency_one: 'зёрнышко', currency_few: 'зёрнышка', currency_many: 'зёрнышек', streak_days: 0 })
  vi.mocked(api.quests).mockReset().mockResolvedValue({ quests: [] })
  push.mockReset()
  useOnboarding().close()
})
afterEach(() => {
  vi.useRealTimers()
  document.body.innerHTML = ''
})

async function opened() {
  const w = mount(OnboardingTour, { attachTo: document.body })
  await useOnboarding().checkDue()
  await flushPromises()
  return w
}

// jsdom lays nothing out: give a nav button a size so the tour finds it.
function navButton(key: string) {
  const el = document.createElement('a')
  el.setAttribute('data-tour', key)
  el.getBoundingClientRect = () => ({ left: 10, top: 700, width: 60, height: 40, right: 70, bottom: 740, x: 10, y: 700, toJSON: () => ({}) })
  document.body.appendChild(el)
}

describe('OnboardingTour', () => {
  it('is not on screen until the tour starts', () => {
    const w = mount(OnboardingTour)
    expect(w.find('[data-test="tour"]').exists()).toBe(false)
  })

  it('opens with the offer: «Показать» and «Отложить на потом», the reward and the gloss', async () => {
    const w = await opened()
    await vi.advanceTimersByTimeAsync(LINE)
    expect(w.find('[data-test="tour-start"]').text()).toBe('Показать')
    expect(w.find('[data-test="tour-later"]').text()).toBe('Отложить на потом')
    expect(w.find('[data-test="tour-bubble"]').text()).toContain('подарю 5 зёрнышек')
    expect(w.find('[data-test="tour-bubble"]').text()).toContain('Ćao — привет.')
  })

  it('a tap on the empty screen does not answer the offer', async () => {
    const w = await opened()
    await vi.advanceTimersByTimeAsync(LINE)
    await w.find('[data-test="tour"]').trigger('click')
    expect(useOnboarding().index.value).toBe(0)
  })

  it('a tap while a line is typing finishes the line but does not skip it', async () => {
    const w = await opened()
    await vi.advanceTimersByTimeAsync(LINE)
    await w.find('[data-test="tour-start"]').trigger('click')
    expect(useOnboarding().index.value).toBe(1)
    await vi.advanceTimersByTimeAsync(300) // still typing
    await w.find('[data-test="tour"]').trigger('click')
    expect(useOnboarding().index.value).toBe(1)
    expect(w.find('[data-test="tour-bubble"]').text()).toContain('Здесь находятся уроки.')
  })

  it('does not move on during the beat after a line, then a tap anywhere does', async () => {
    const w = await opened()
    await vi.advanceTimersByTimeAsync(LINE)
    await w.find('[data-test="tour-start"]').trigger('click')
    await vi.advanceTimersByTimeAsync(1600) // typed, still inside the beat
    await w.find('[data-test="tour"]').trigger('click')
    expect(useOnboarding().index.value).toBe(1)
    await vi.advanceTimersByTimeAsync(LINE)
    await w.find('[data-test="tour"]').trigger('click')
    expect(useOnboarding().index.value).toBe(2)
  })

  it('lights the buttons the swallow talks about', async () => {
    navButton('course')
    const w = await opened()
    await vi.advanceTimersByTimeAsync(LINE)
    expect(w.findAll('[data-test="tour-frame"]')).toHaveLength(0) // the offer lights nothing
    await w.find('[data-test="tour-start"]').trigger('click')
    await vi.advanceTimersByTimeAsync(50)
    expect(w.findAll('[data-test="tour-frame"]')).toHaveLength(1)
  })

  it('«Отложить на потом» and Escape both put the tour off', async () => {
    let w = await opened()
    await vi.advanceTimersByTimeAsync(LINE)
    await w.find('[data-test="tour-later"]').trigger('click')
    expect(api.answerOnboarding).toHaveBeenCalledWith('postpone')
    expect(w.find('[data-test="tour"]').exists()).toBe(false)

    vi.mocked(api.answerOnboarding).mockClear()
    w = await opened()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(api.answerOnboarding).toHaveBeenCalledWith('postpone')
  })

  it('the last line says thanks, offers no way to put it off, and closes on a tap', async () => {
    const w = await opened()
    const t = useOnboarding()
    for (let i = 0; i < 6; i++) t.next()
    await vi.advanceTimersByTimeAsync(LINE)
    expect(w.find('[data-test="tour-next"]').text()).toBe('Спасибо!')
    expect(w.find('[data-test="tour-later"]').exists()).toBe(false)
    expect(w.find('[data-test="tour-bubble"]').text()).toContain('Держи свои 5 зёрнышек')
    expect(w.find('.prize').text()).toContain('+5 зёрнышек')
    await w.find('[data-test="tour-next"]').trigger('click')
    expect(w.find('[data-test="tour"]').exists()).toBe(false)
  })

  it('walks the learner through the screens and puts them back where they started', async () => {
    const w = await opened()
    const t = useOnboarding()
    t.next() // course: already on /course
    await flushPromises()
    expect(push).not.toHaveBeenCalled()
    t.next() // stories
    await flushPromises()
    expect(push).toHaveBeenLastCalledWith('/stories')
    t.next()
    await flushPromises()
    expect(push).toHaveBeenLastCalledWith('/review')
    // the lesson tour ends: the screen she came from is back
    push.mockClear()
    await vi.advanceTimersByTimeAsync(LINE)
    await w.find('[data-test="tour-later"]').trigger('click')
    await flushPromises()
    expect(push).not.toHaveBeenCalled() // route mock never leaves /course, so nothing to go back to
  })

  it('lights the words and the dictionary as one frame', async () => {
    navButton('review')
    navButton('vocab')
    const t = useOnboarding()
    const w = await opened()
    t.next()
    t.next()
    t.next() // words
    await vi.advanceTimersByTimeAsync(600)
    expect(w.findAll('[data-test="tour-frame"]')).toHaveLength(1)
  })

  it('shows the Serbian words in Cyrillic for a learner who reads Cyrillic', async () => {
    useSessionStore().user = sessionUser(false, { script: 'cyrillic' })
    const w = await opened()
    await vi.advanceTimersByTimeAsync(LINE)
    const text = w.find('[data-test="tour-bubble"]').text()
    expect(text).toContain('Ћао!')
    expect(text).not.toContain('Ćao!')
  })
})
