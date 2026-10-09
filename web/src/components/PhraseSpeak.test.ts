import { describe, it, expect, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import PhraseSpeak from './PhraseSpeak.vue'
import MarkdownView from './MarkdownView.vue'
import PromptSpeak from './PromptSpeak.vue'
import { _setPhraseIndex, phraseKey } from '../lib/phraseAudio'

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
})
