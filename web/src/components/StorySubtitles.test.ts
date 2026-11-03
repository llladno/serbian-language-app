import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import StorySubtitles from './StorySubtitles.vue'

const subtitle = { sr: { text: 'Ne, samo keš!', active: true, words: [] }, ru: { text: 'Нет, только наличные.', active: false } }
const mountSub = (extra: Record<string, unknown> = {}) =>
  mount(StorySubtitles, { props: { subtitle, wordIdx: -1, mask: 'keš', ...extra } })

describe('StorySubtitles', () => {
  it('blanks the answer of an open quick question with the same gap as the final questions', () => {
    const w = mountSub()
    expect(w.find('[data-test="story-sr"]').text()).not.toContain('keš')
    const gap = w.find('[data-test="story-gap"]')
    expect(gap.exists()).toBe(true)
    expect(gap.classes()).toContain('gap')
    expect(gap.classes()).not.toContain('gap-filled')
    expect(gap.text()).toBe('')
    // the punctuation stuck to the word stays outside the gap
    expect(w.find('[data-test="story-sr"]').text()).toContain('!')
  })

  it('writes the word back into its gap, in green, once the answer is right', () => {
    const w = mountSub({ reveal: true })
    const gap = w.find('[data-test="story-gap"]')
    expect(gap.text()).toBe('keš')
    expect(gap.classes()).toContain('gap-filled')
    expect(w.find('[data-test="story-sr"]').text()).toContain('keš!')
    // only the gap is filled, the other words are plain
    expect(w.findAll('[data-test="story-gap"]')).toHaveLength(1)
  })

  it('with no open question the line is plain', () => {
    const w = mountSub({ mask: null, reveal: true })
    expect(w.find('[data-test="story-sr"]').text()).toContain('keš')
    expect(w.find('[data-test="story-gap"]').exists()).toBe(false)
  })
})
