import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import MarkdownView from './MarkdownView.vue'
import { useSessionStore } from '../stores/session'
import { provideKeepLatin } from '../lib/useScript'
import { _setPhraseIndex, phraseKey } from '../lib/phraseAudio'
import type { SessionUser } from '../types'

vi.mock('../telegram', () => ({ isTelegram: vi.fn(), initData: vi.fn() }))

function signIn(script: 'latin' | 'cyrillic') {
  useSessionStore().user = {
    id: 'u',
    name: 'Гриша',
    email: '',
    email_verified: true,
    telegram: { linked: false, username: '' },
    script,
  } as SessionUser
}

const SOURCE = [
  '## Привет — `Zdravo`',
  '',
  'Скажи `Kako si?` *(как дела)* и **hvala** в конце.',
  '',
  '| Латиница | Звук |',
  '|---|---|',
  '| `č` | ч |',
  '',
  '- `molim` [мо́лим] — пожалуйста',
  '',
  '```',
  'Kilo jabuka, molim.   — Кило яблок.',
  '```',
].join('\n')

beforeEach(() => {
  setActivePinia(createPinia())
  _setPhraseIndex([])
})
afterEach(() => {
  _setPhraseIndex(null)
  vi.restoreAllMocks()
})

describe('MarkdownView in Cyrillic', () => {
  it('shows Serbian in code spans, emphasis, headings and table cells in Cyrillic', () => {
    signIn('cyrillic')
    const w = mount(MarkdownView, { props: { source: SOURCE } })
    const text = w.text()
    expect(text).toContain('Привет — Здраво')
    expect(text).toContain('Како си?')
    expect(text).toContain('хвала')
    expect(w.find('td code').text()).toBe('ч')
    expect(text).toContain('молим')
    expect(w.findAll('code').some((c) => c.text() === 'Здраво')).toBe(true)
  })

  it('leaves Russian and a transcription exactly as written', () => {
    signIn('cyrillic')
    const w = mount(MarkdownView, { props: { source: SOURCE } })
    expect(w.text()).toContain('Скажи')
    expect(w.text()).toContain('[мо́лим] — пожалуйста')
  })

  it('converts the Serbian inside a code block too: the course sets its dialogues there', () => {
    signIn('cyrillic')
    const w = mount(MarkdownView, { props: { source: SOURCE } })
    expect(w.find('pre').text()).toContain('Кило јабука, молим.   — Кило яблок.')
  })

  it('leaves everything in Latin for a Latin reader', () => {
    signIn('latin')
    const w = mount(MarkdownView, { props: { source: SOURCE } })
    expect(w.text()).toContain('Привет — Zdravo')
    expect(w.text()).toContain('Kako si?')
    expect(w.find('td code').text()).toBe('č')
  })

  it('stays in Latin inside a lesson about the letters', () => {
    signIn('cyrillic')
    const Lesson = defineComponent({
      setup() {
        provideKeepLatin(ref(true))
        return () => h(MarkdownView, { source: SOURCE })
      },
    })
    const w = mount(Lesson)
    expect(w.text()).toContain('Привет — Zdravo')
    expect(w.find('td code').text()).toBe('č')
  })

  it('finds the audio clip by the Latin phrase, not by what is on screen', () => {
    signIn('cyrillic')
    _setPhraseIndex([phraseKey('Kako si?')])
    const w = mount(MarkdownView, { props: { source: 'Скажи `Kako si?`' } })
    expect(w.find('code').text()).toBe('Како си?')
    expect(w.find('[data-clip]').attributes('data-clip')).toBe(`p/${phraseKey('Kako si?')}.mp3`)
  })

  it('switches when the learner switches, without a reload', async () => {
    signIn('latin')
    const w = mount(MarkdownView, { props: { source: 'Скажи `čokolada`' } })
    expect(w.text()).toContain('čokolada')
    signIn('cyrillic')
    await w.vm.$nextTick()
    expect(w.text()).toContain('чоколада')
  })
})
