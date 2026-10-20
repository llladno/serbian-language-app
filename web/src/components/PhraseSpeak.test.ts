import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { nextTick } from 'vue'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import PhraseSpeak from './PhraseSpeak.vue'
import MarkdownView from './MarkdownView.vue'
import PromptSpeak from './PromptSpeak.vue'
import { _setPhraseIndex, phraseKey } from '../lib/phraseAudio'
import { speakClip } from '../lib/sound'

vi.mock('../lib/sound', () => ({ speakClip: vi.fn() }))

// A component left mounted would still react to the next test's clip list.
enableAutoUnmount(afterEach)
beforeEach(() => vi.mocked(speakClip).mockClear())
afterEach(() => _setPhraseIndex(null))
const speaker = 'button[aria-label="озвучить"]'

describe('phrase speakers', () => {
  it('PhraseSpeak shows a button only for a voiced phrase', () => {
    _setPhraseIndex([phraseKey('Govorim ruski.')])
    expect(mount(PhraseSpeak, { props: { text: 'Govorim ruski.' } }).find(speaker).exists()).toBe(true)
    expect(mount(PhraseSpeak, { props: { text: 'Ne znam.' } }).find(speaker).exists()).toBe(false)
  })

  it('PromptSpeak voices only the Serbian quote of a prompt', () => {
    _setPhraseIndex([phraseKey('Govorim ruski.')])
    const w = mount(PromptSpeak, { props: { prompt: '«Привет» и «Govorim ruski.» и «Ona radi ___»' } })
    expect(w.findAll(speaker)).toHaveLength(1)
  })

  it('markdown code spans get a speaker only when voiced', () => {
    _setPhraseIndex([phraseKey('Govorim ruski.')])
    const w = mount(MarkdownView, { props: { source: '- `Govorim ruski.` и `Ne znam.`' } })
    expect(w.findAll('.md-speak')).toHaveLength(1)
    expect(w.find('.md-speak').attributes('data-clip')).toBe(`p/${phraseKey('Govorim ruski.')}.mp3`)
  })

  describe('reading the sentence out by itself', () => {
    it('speaks a voiced phrase once, when it appears', () => {
      _setPhraseIndex([phraseKey('Govorim ruski.')])
      mount(PhraseSpeak, { props: { text: 'Govorim ruski.', autoplay: true } })
      expect(speakClip).toHaveBeenCalledTimes(1)
      expect(speakClip).toHaveBeenCalledWith(`p/${phraseKey('Govorim ruski.')}.mp3`)
    })

    it('waits to be tapped without autoplay', () => {
      _setPhraseIndex([phraseKey('Govorim ruski.')])
      mount(PhraseSpeak, { props: { text: 'Govorim ruski.' } })
      expect(speakClip).not.toHaveBeenCalled()
    })

    it('stays silent for a phrase nobody has voiced', () => {
      _setPhraseIndex([phraseKey('Govorim ruski.')])
      mount(PhraseSpeak, { props: { text: 'Ne znam.', autoplay: true } })
      expect(speakClip).not.toHaveBeenCalled()
    })

    it('does not repeat itself when the text changes under it', async () => {
      _setPhraseIndex([phraseKey('Govorim ruski.'), phraseKey('Ne znam.')])
      const w = mount(PhraseSpeak, { props: { text: 'Govorim ruski.', autoplay: true } })
      await w.setProps({ text: 'Ne znam.' })
      await nextTick()
      expect(speakClip).toHaveBeenCalledTimes(1)
    })

    it('does not read out an old verdict when the clip list turns up late', async () => {
      _setPhraseIndex(null)
      mount(PhraseSpeak, { props: { text: 'Govorim ruski.', autoplay: true } })
      _setPhraseIndex([phraseKey('Govorim ruski.')])
      await nextTick()
      expect(speakClip).not.toHaveBeenCalled()
    })
  })
})
