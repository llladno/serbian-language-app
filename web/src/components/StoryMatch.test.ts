import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import StoryMatch from './StoryMatch.vue'
import { RIGHT_BEAT_MS, WRONG_BEAT_MS } from '../lib/useStoryVerdict'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

function mountMatch(check: (p: Record<string, string>) => Promise<{ correct: boolean; answer: string; pairs?: [string, string][] }>) {
  return mount(StoryMatch, {
    props: { prompt: 'Соедините пары', left: ['Zdravo', 'Hvala'], right: ['Привет', 'Спасибо'], check },
  })
}
async function pair(w: ReturnType<typeof mountMatch>, l: string, r: string) {
  await w.findAll('[data-test="story-left"]').find((b) => b.text() === l)!.trigger('click')
  await w.findAll('[data-test="story-right"]').find((b) => b.text() === r)!.trigger('click')
}

describe('StoryMatch', () => {
  it('cannot be checked until every phrase has a pair, and sends the pairs', async () => {
    const check = vi.fn().mockResolvedValue({ correct: true, answer: '' })
    const w = mountMatch(check)
    await pair(w, 'Zdravo', 'Привет')
    expect(w.find('[data-test="story-submit"]').attributes('disabled')).toBeDefined()
    await pair(w, 'Hvala', 'Спасибо')
    await w.find('[data-test="story-submit"]').trigger('click')
    await flushPromises()
    expect(check).toHaveBeenCalledWith({ Zdravo: 'Привет', Hvala: 'Спасибо' })
    expect(w.find('[data-test="story-verdict"]').exists()).toBe(true)
    vi.advanceTimersByTime(RIGHT_BEAT_MS + 50)
    expect(w.emitted('done')).toEqual([[true]])
  })

  it('works from the right column first, and a tap on a made pair undoes it', async () => {
    const w = mountMatch(vi.fn())
    await w.findAll('[data-test="story-right"]').find((b) => b.text() === 'Спасибо')!.trigger('click')
    await w.findAll('[data-test="story-left"]').find((b) => b.text() === 'Hvala')!.trigger('click')
    await pair(w, 'Zdravo', 'Привет')
    expect(w.find('[data-test="story-submit"]').attributes('disabled')).toBeUndefined()
    await w.findAll('[data-test="story-left"]').find((b) => b.text() === 'Hvala')!.trigger('click')
    expect(w.find('[data-test="story-submit"]').attributes('disabled')).toBeDefined()
  })

  it('lists the right pairs after a mistake and then goes on', async () => {
    const w = mountMatch(
      vi.fn().mockResolvedValue({ correct: false, answer: '', pairs: [['Zdravo', 'Привет'], ['Hvala', 'Спасибо']] }),
    )
    await pair(w, 'Zdravo', 'Спасибо')
    await pair(w, 'Hvala', 'Привет')
    await w.find('[data-test="story-submit"]').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Есть ошибки')
    expect(w.text()).toContain('Zdravo')
    expect(w.text()).toContain('Привет')
    vi.advanceTimersByTime(WRONG_BEAT_MS + 50)
    expect(w.emitted('done')).toEqual([[false]])
  })
})
