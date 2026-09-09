import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DialogueBubble from './DialogueBubble.vue'

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
