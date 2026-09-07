import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ReadingText from './ReadingText.vue'
import { api } from '../api'
import { _resetLookupCache } from '../lib/lookup'

vi.mock('../api', () => ({ api: { lookup: vi.fn() } }))

const RouterLinkStub = { template: '<a><slot /></a>' }
const mountRT = (props: { serbian: string; translation?: string }) =>
  mount(ReadingText, { props, global: { stubs: { RouterLink: RouterLinkStub } } })

beforeEach(() => {
  vi.mocked(api.lookup).mockReset()
  _resetLookupCache()
})

describe('ReadingText', () => {
  it('renders the Serbian text with clickable words', () => {
    const w = mountRT({ serbian: 'Zdravo, Milane!' })
    expect(w.findAll('span.cursor-pointer').map((s) => s.text())).toEqual(['Zdravo', 'Milane'])
  })

  it('shows the translation only after the details is expanded, and never as clickable words', () => {
    const w = mountRT({ serbian: 'Zdravo.', translation: 'Привет.' })
    expect(w.find('details').text()).toContain('Привет.')
    // the translation text is plain, not part of the glossed span
    expect(w.findAll('span.cursor-pointer').map((s) => s.text())).toEqual(['Zdravo'])
  })

  it('omits the translation block when no translation is given', () => {
    const w = mountRT({ serbian: 'Zdravo.' })
    expect(w.find('details').exists()).toBe(false)
  })

  it('looks up a clicked word through the shared lookup', async () => {
    vi.mocked(api.lookup).mockResolvedValue({
      query: 'Zdravo',
      partial: false,
      matches: [{ id: 'zdravo', latin: 'zdravo', cyrillic: 'здраво', ru: 'привет' }],
    })
    const w = mountRT({ serbian: 'Zdravo svete.' })
    await w.findAll('span.cursor-pointer').find((s) => s.text() === 'Zdravo')!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('привет')
  })
})
