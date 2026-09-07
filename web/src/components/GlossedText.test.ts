import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import GlossedText from './GlossedText.vue'
import { api } from '../api'
import { _resetLookupCache } from '../lib/lookup'
import { _resetAddedToReview } from '../lib/review'

vi.mock('../api', () => ({ api: { lookup: vi.fn(), addToReview: vi.fn() } }))

const RouterLinkStub = {
  props: ['to'],
  template: `<a :href="to && to.query ? '/vocab?q=' + to.query.q : ''"><slot /></a>`,
}
const mountGT = (text: string) =>
  mount(GlossedText, { props: { text }, global: { stubs: { RouterLink: RouterLinkStub } } })
const clickWord = (w: ReturnType<typeof mountGT>, text: string) =>
  w.findAll('span.cursor-pointer').find((s) => s.text() === text)!.trigger('click')

beforeEach(() => {
  vi.mocked(api.lookup).mockReset()
  vi.mocked(api.addToReview).mockReset()
  vi.mocked(api.addToReview).mockResolvedValue({ status: 'added' })
  _resetLookupCache()
  _resetAddedToReview()
})

describe('GlossedText', () => {
  it('renders only letter runs as clickable, rejoining to the original text', () => {
    const w = mountGT('___ košta kafa? — Sto dinara.')
    expect(w.text()).toContain('___ košta kafa?')
    expect(w.findAll('span.cursor-pointer').map((s) => s.text())).toEqual([
      'košta',
      'kafa',
      'Sto',
      'dinara',
    ])
  })

  it('looks a word up and shows its dictionary card', async () => {
    vi.mocked(api.lookup).mockResolvedValue({
      query: 'košta',
      partial: false,
      matches: [{ id: 'kostati', latin: 'koštati', cyrillic: 'коштати', ru: 'стоить' }],
    })
    const w = mountGT('Koliko košta?')
    await clickWord(w, 'košta')
    await flushPromises()
    expect(api.lookup).toHaveBeenCalledWith('košta')
    expect(w.text()).toContain('стоить')
    expect(w.text()).toContain('коштати')
  })

  it('falls back to a dictionary link when the word is unknown', async () => {
    vi.mocked(api.lookup).mockResolvedValue({ query: 'xyz', partial: false, matches: [] })
    const w = mountGT('xyz abc')
    await clickWord(w, 'xyz')
    await flushPromises()
    expect(w.text()).toContain('нет в словаре курса')
    expect(w.find('a').attributes('href')).toBe('/vocab?q=xyz')
  })

  it('closes the card on Escape', async () => {
    vi.mocked(api.lookup).mockResolvedValue({ query: 'x', partial: false, matches: [] })
    const w = mountGT('x y')
    await clickWord(w, 'x')
    await flushPromises()
    expect(w.text()).toContain('нет в словаре курса')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(w.text()).not.toContain('нет в словаре курса')
  })

  it('keeps the card open on scroll while the word stays in view', async () => {
    vi.mocked(api.lookup).mockResolvedValue({
      query: 'x',
      partial: false,
      matches: [{ id: 'x', latin: 'x', cyrillic: '', ru: 'икс' }],
    })
    const w = mountGT('x y')
    await clickWord(w, 'x')
    await flushPromises()
    window.dispatchEvent(new Event('scroll'))
    await flushPromises()
    expect(w.text()).toContain('икс')
  })

  it('adds a matched word to review and marks the button done', async () => {
    vi.mocked(api.lookup).mockResolvedValue({
      query: 'košta',
      partial: false,
      matches: [{ id: 'kostati', latin: 'koštati', cyrillic: 'коштати', ru: 'стоить' }],
    })
    const w = mountGT('Koliko košta?')
    await clickWord(w, 'košta')
    await flushPromises()

    const btn = w.findAll('button').find((b) => b.text().includes('в повторение'))!
    await btn.trigger('click')
    await flushPromises()

    expect(api.addToReview).toHaveBeenCalledWith('kostati')
    expect(w.text()).toContain('в очереди')
    expect(w.findAll('button').some((b) => b.text().includes('в повторение'))).toBe(false)
  })

  it('shows no add-to-review button when the word is unknown', async () => {
    vi.mocked(api.lookup).mockResolvedValue({ query: 'xyz', partial: false, matches: [] })
    const w = mountGT('xyz abc')
    await clickWord(w, 'xyz')
    await flushPromises()
    expect(w.findAll('button').some((b) => b.text().includes('в повторение'))).toBe(false)
  })

  it('does not keep global listeners after the card is closed', async () => {
    vi.mocked(api.lookup).mockResolvedValue({ query: 'x', partial: false, matches: [] })
    const add = vi.spyOn(window, 'addEventListener')
    const remove = vi.spyOn(window, 'removeEventListener')
    const w = mountGT('x y')
    await clickWord(w, 'x')
    await flushPromises()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    const scrollAdds = add.mock.calls.filter((c) => c[0] === 'scroll').length
    const scrollRemoves = remove.mock.calls.filter((c) => c[0] === 'scroll').length
    expect(scrollAdds).toBe(1)
    expect(scrollRemoves).toBeGreaterThanOrEqual(scrollAdds)
    add.mockRestore()
    remove.mockRestore()
  })
})
