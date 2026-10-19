import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import DialogueBubble from './DialogueBubble.vue'
import { useSessionStore } from '../stores/session'
import type { SessionUser } from '../types'

const props = { who: 'npc' as const, sr: 'Izvolite?', ru: 'Слушаю вас?' }

describe('DialogueBubble', () => {
  it('shows the Serbian line and hides the translation until asked', async () => {
    const w = mount(DialogueBubble, { props })
    expect(w.text()).toContain('Izvolite?')
    expect(w.text()).not.toContain('Слушаю вас?')

    await w.find('[data-test="translate"]').trigger('click')
    expect(w.text()).toContain('Слушаю вас?')
  })

  it('shows the translation upfront when showTranslation is set', () => {
    const w = mount(DialogueBubble, { props: { ...props, showTranslation: true } })
    expect(w.text()).toContain('Слушаю вас?')
  })

  it('renders no speaker button without a clip', () => {
    const w = mount(DialogueBubble, { props })
    expect(w.find('button[aria-label="озвучить"]').exists()).toBe(false)
  })

  it('renders a speaker button when a clip exists', () => {
    const w = mount(DialogueBubble, { props: { ...props, audio: '05.9-t1.mp3' } })
    expect(w.find('button[aria-label="озвучить"]').exists()).toBe(true)
  })
})

describe('DialogueBubble in Cyrillic', () => {
  beforeEach(() => {
    useSessionStore().user = {
      id: 'u', name: 'Гриша', email: '', email_verified: true,
      telegram: { linked: false, username: '' }, script: 'cyrillic',
    } as SessionUser
  })

  it('shows the Serbian line in Cyrillic and leaves the translation alone', () => {
    const w = mount(DialogueBubble, { props: { ...props, showTranslation: true } })
    expect(w.text()).toContain('Изволите?')
    expect(w.text()).toContain('Слушаю вас?')
    expect(w.text()).not.toContain('Izvolite')
  })

  it('shows what the learner answered in Cyrillic too, next to the corrected line', () => {
    const w = mount(DialogueBubble, {
      props: { who: 'me' as const, sr: 'Kafu, molim.', ru: 'Кофе, пожалуйста.', wrong: true, given: 'Kafa, molim.' },
    })
    expect(w.find('[data-test="given"]').text()).toContain('«Кафа, молим.»')
    expect(w.text()).toContain('Кафу, молим.')
  })

  it('shows the explanation under a correction in Cyrillic too', () => {
    const w = mount(DialogueBubble, {
      props: { who: 'me' as const, sr: 'Kafu, molim.', ru: 'Кофе, пожалуйста.', wrong: true, given: 'Kafa, molim.', note: '«Kafu» — винительный падеж.' },
    })
    expect(w.text()).toContain('«Кафу» — винительный падеж.')
  })
})
