import { describe, it, expect, vi, beforeEach } from 'vitest'
import { lookupWord, _resetLookupCache } from './lookup'
import { api } from '../api'

vi.mock('../api', () => ({ api: { lookup: vi.fn() } }))

beforeEach(() => {
  vi.mocked(api.lookup).mockReset()
  _resetLookupCache()
})

describe('lookupWord', () => {
  it('caches by normalized word so repeated calls hit the API once', async () => {
    vi.mocked(api.lookup).mockResolvedValue({ query: 'Radiš', partial: false, matches: [] })

    const a = await lookupWord('Radiš')
    const b = await lookupWord('radiš')
    const c = await lookupWord('  RADIŠ ')

    expect(api.lookup).toHaveBeenCalledTimes(1)
    expect(b).toBe(a)
    expect(c).toBe(a)
  })

  it('dedupes concurrent calls for the same word', async () => {
    vi.mocked(api.lookup).mockImplementation(
      () => new Promise((r) => setTimeout(() => r({ query: 'x', partial: false, matches: [] }), 5)),
    )
    await Promise.all([lookupWord('x'), lookupWord('x'), lookupWord('x')])
    expect(api.lookup).toHaveBeenCalledTimes(1)
  })

  it('returns an empty result on API error and does not cache the failure', async () => {
    vi.mocked(api.lookup).mockRejectedValueOnce(new Error('boom'))
    const first = await lookupWord('boom')
    expect(first.matches).toEqual([])

    vi.mocked(api.lookup).mockResolvedValueOnce({
      query: 'boom',
      partial: false,
      matches: [{ id: 'b', latin: 'boom', cyrillic: '', ru: 'бум' }],
    })
    const second = await lookupWord('boom')
    expect(second.matches).toHaveLength(1)
  })
})
