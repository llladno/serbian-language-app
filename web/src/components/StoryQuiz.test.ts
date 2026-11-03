import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import StoryQuiz from './StoryQuiz.vue'
import { RIGHT_BEAT_MS, SAY_DELAY_MS, WRONG_BEAT_MS } from '../lib/useStoryVerdict'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

function mountQuiz(check: (c: string) => Promise<{ correct: boolean; answer: string }>, extra: Record<string, unknown> = {}) {
  return mount(StoryQuiz, {
    props: { prompt: '«зовут»', eyebrow: 'Какое слово прозвучало?', options: ['zovem', 'živim', 'radim'], optionsSerbian: true, shuffle: false, check, ...extra },
  })
}
const opts = (w: ReturnType<typeof mountQuiz>) => w.findAll('[data-test="story-option"]')

describe('StoryQuiz', () => {
  it('a right answer takes the place of the buttons with a big «Верно» and goes on after a beat', async () => {
    const w = mountQuiz(vi.fn().mockResolvedValue({ correct: true, answer: 'zovem' }))
    await opts(w)[0].trigger('click')
    await flushPromises()
    expect(opts(w)).toHaveLength(0)
    expect(w.find('[data-test="story-verdict"]').text()).toContain('Верно')
    expect(w.emitted('done')).toBeUndefined()
    vi.advanceTimersByTime(RIGHT_BEAT_MS + 50)
    expect(w.emitted('done')).toEqual([[true]])
  })

  it('a wrong answer keeps the buttons, rings the pick and the right one, says the right answer, and waits longer', async () => {
    const w = mountQuiz(vi.fn().mockResolvedValue({ correct: false, answer: 'zovem' }))
    await opts(w)[1].trigger('click')
    await flushPromises()
    const ring = (i: number) => opts(w)[i].classes().find((c) => c.startsWith('ring-[')) ?? null
    expect([ring(0), ring(1), ring(2)]).toEqual(['ring-[var(--good)]', 'ring-[var(--bad)]', null])
    expect(w.text()).toContain('Не то')
    expect(w.text()).toContain('Правильно')
    vi.advanceTimersByTime(RIGHT_BEAT_MS + 50)
    expect(w.emitted('done')).toBeUndefined()
    vi.advanceTimersByTime(WRONG_BEAT_MS - RIGHT_BEAT_MS + 50)
    expect(w.emitted('done')).toEqual([[false]])
  })

  it('a sentence with a gap: the gap stays empty until the answer is right, then the word is written into it', async () => {
    const check = vi.fn().mockResolvedValue({ correct: true, answer: 'keš' })
    const w = mountQuiz(check, { prompt: 'Ne, samo _____!', fillGap: true, promptSerbian: true, options: ['keš', 'račun', 'kusur'] })
    expect(w.find('[data-test="story-gap"]').text()).toBe('')
    expect(w.find('[data-test="story-prompt"]').text()).toContain('Ne, samo')
    await opts(w)[0].trigger('click')
    await flushPromises()
    expect(w.find('[data-test="story-gap"]').text()).toBe('keš')
    expect(w.find('[data-test="story-gap"]').classes()).toContain('gap-filled')
    // the word is not said twice: the verdict does not repeat the answer under the tick
    expect(w.find('[data-test="story-verdict"]').text()).toBe('Верно')
  })

  it('a quick question whose answer is written in the subtitle does not repeat it under «Верно»', async () => {
    const mk = (extra: Record<string, unknown>) =>
      mountQuiz(vi.fn().mockResolvedValue({ correct: true, answer: 'živim' }), { options: ['živim', 'radim'], optionsSerbian: true, ...extra })
    const inPlace = mk({ answerInPlace: true })
    await opts(inPlace)[0].trigger('click')
    await flushPromises()
    expect(inPlace.find('[data-test="story-verdict"]').text()).toBe('Верно')
    // with no gap on screen the word still shows, as before
    const plain = mk({})
    await opts(plain)[0].trigger('click')
    await flushPromises()
    expect(plain.find('[data-test="story-verdict"]').text()).toContain('živim')
  })

  it('a wrong answer leaves the gap empty', async () => {
    const w = mountQuiz(vi.fn().mockResolvedValue({ correct: false, answer: 'keš' }), {
      prompt: 'Ne, samo _____!', fillGap: true, promptSerbian: true, options: ['keš', 'račun', 'kusur'],
    })
    await opts(w)[1].trigger('click')
    await flushPromises()
    expect(w.find('[data-test="story-gap"]').text()).toBe('')
    expect(w.find('[data-test="story-gap"]').classes()).not.toContain('gap-filled')
  })

  it('mixes the options unless told not to, and keeps them all', () => {
    const many = Array.from({ length: 12 }, (_, i) => `o${i}`)
    const w = mountQuiz(vi.fn(), { options: many, shuffle: true })
    const texts = opts(w).map((b) => b.text())
    expect([...texts].sort()).toEqual([...many].sort())
    expect(texts).not.toEqual(many)
  })

  it('mixes the options when `shuffle` is not given at all, which is how the story screen mounts it', () => {
    // A Boolean prop that is absent reads as false in Vue: without an explicit default the screen never mixed anything,
    // the right answer (written first in the story) stayed first, and the tests above hid it by passing shuffle: true.
    const many = Array.from({ length: 12 }, (_, i) => `o${i}`)
    const w = mount(StoryQuiz, { props: { prompt: 'p', options: many, check: vi.fn() } })
    const texts = opts(w).map((b) => b.text())
    expect([...texts].sort()).toEqual([...many].sort())
    expect(texts).not.toEqual(many)
  })

  it('does not take a second answer while the first is being checked', async () => {
    const check = vi.fn(() => new Promise<{ correct: boolean; answer: string }>(() => {})) // never answers
    const w = mountQuiz(check)
    await opts(w)[0].trigger('click')
    await opts(w)[1].trigger('click')
    expect(check).toHaveBeenCalledTimes(1)
  })

  it('counts an unreachable server as a wrong answer instead of stopping the story', async () => {
    const w = mountQuiz(vi.fn().mockRejectedValue(new Error('offline')))
    await opts(w)[0].trigger('click')
    await flushPromises()
    vi.advanceTimersByTime(WRONG_BEAT_MS + 50)
    expect(w.emitted('done')).toEqual([[false]])
  })

  it('says the sentence after the answer and waits for it to end before going on', async () => {
    const play = vi.fn()
    const w = mountQuiz(vi.fn().mockResolvedValue({ correct: true, answer: 'zovem' }), { say: '/s5.mp3', sayMs: 3000, play })
    await opts(w)[0].trigger('click')
    await flushPromises()
    expect(play).not.toHaveBeenCalled() // the verdict comes first
    vi.advanceTimersByTime(SAY_DELAY_MS + 10)
    expect(play).toHaveBeenCalledWith('/s5.mp3')
    vi.advanceTimersByTime(RIGHT_BEAT_MS + 50) // the usual beat is over, the clip is not
    expect(w.emitted('done')).toBeUndefined()
    vi.advanceTimersByTime(3000)
    expect(w.emitted('done')).toEqual([[true]])
  })

  it('says it after a wrong answer too: the lessons say the right phrase either way', async () => {
    const play = vi.fn()
    const w = mountQuiz(vi.fn().mockResolvedValue({ correct: false, answer: 'zovem' }), { say: '/s5.mp3', sayMs: 1000, play })
    await opts(w)[1].trigger('click')
    await flushPromises()
    vi.advanceTimersByTime(SAY_DELAY_MS + 10)
    expect(play).toHaveBeenCalledWith('/s5.mp3')
  })

  it('without a clip it goes on after the usual beat', async () => {
    const w = mountQuiz(vi.fn().mockResolvedValue({ correct: true, answer: 'zovem' }))
    await opts(w)[0].trigger('click')
    await flushPromises()
    vi.advanceTimersByTime(RIGHT_BEAT_MS + 50)
    expect(w.emitted('done')).toEqual([[true]])
  })

  it('a question about a clip plays it on arrival and again from the speaker button', async () => {
    const play = vi.fn()
    const w = mountQuiz(vi.fn(), { audio: '/clip.mp3', play })
    vi.advanceTimersByTime(400)
    expect(play).toHaveBeenCalledWith('/clip.mp3')
    await w.find('[data-test="story-listen-again"]').trigger('click')
    expect(play).toHaveBeenCalledTimes(2)
  })
})
