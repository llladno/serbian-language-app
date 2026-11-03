import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import StoryBank from './StoryBank.vue'
import { RIGHT_BEAT_MS, WRONG_BEAT_MS } from '../lib/useStoryVerdict'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

function mountBank(check: (c: string) => Promise<{ correct: boolean; answer: string }>) {
  return mount(StoryBank, {
    props: { prompt: 'Соберите по-сербски: «Меня зовут Марко.»', bank: ['Marko', 'se', 'zovem', 'ja'], check },
  })
}
async function tap(w: ReturnType<typeof mountBank>, text: string) {
  await w.findAll('[data-test="story-tile"]').find((b) => b.text() === text)!.trigger('click')
}

describe('StoryBank', () => {
  it('builds the sentence in the order of the taps and sends it', async () => {
    const check = vi.fn().mockResolvedValue({ correct: true, answer: 'Zovem se Marko.' })
    const w = mountBank(check)
    for (const t of ['zovem', 'se', 'Marko']) await tap(w, t)
    expect(w.findAll('[data-test="story-chosen"]').map((b) => b.text())).toEqual(['zovem', 'se', 'Marko'])
    await w.find('[data-test="story-submit"]').trigger('click')
    await flushPromises()
    expect(check).toHaveBeenCalledWith('zovem se Marko')
    expect(w.find('[data-test="story-verdict"]').exists()).toBe(true)
    expect(w.find('[data-test="story-tile"]').exists()).toBe(false)
    vi.advanceTimersByTime(RIGHT_BEAT_MS + 50)
    expect(w.emitted('done')).toEqual([[true]])
  })

  it('takes a tile back when it is tapped in the answer, and cannot be checked while empty', async () => {
    const w = mountBank(vi.fn())
    expect(w.find('[data-test="story-submit"]').attributes('disabled')).toBeDefined()
    await tap(w, 'ja')
    await w.find('[data-test="story-chosen"]').trigger('click')
    expect(w.findAll('[data-test="story-chosen"]')).toHaveLength(0)
    expect(w.findAll('[data-test="story-tile"]')).toHaveLength(4)
  })

  it('shows the right sentence after a wrong one and then goes on', async () => {
    const w = mountBank(vi.fn().mockResolvedValue({ correct: false, answer: 'Zovem se Marko.' }))
    await tap(w, 'ja')
    await w.find('[data-test="story-submit"]').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Не совсем')
    expect(w.text()).toContain('Zovem se Marko.')
    vi.advanceTimersByTime(WRONG_BEAT_MS + 50)
    expect(w.emitted('done')).toEqual([[false]])
  })
})
