import { describe, it, expect, vi, beforeEach } from 'vitest'
import { addWordToReview, isAddedToReview, _resetAddedToReview } from './review'
import { api } from '../api'

vi.mock('../api', () => ({ api: { addToReview: vi.fn() } }))

beforeEach(() => {
  vi.mocked(api.addToReview).mockReset()
  vi.mocked(api.addToReview).mockResolvedValue({ status: 'added' })
  _resetAddedToReview()
})

describe('addWordToReview', () => {
  it('marks the word as added and skips a second API call', async () => {
    expect(isAddedToReview('kostati')).toBe(false)

    await addWordToReview('kostati')
    expect(isAddedToReview('kostati')).toBe(true)
    expect(api.addToReview).toHaveBeenCalledTimes(1)

    await addWordToReview('kostati')
    expect(api.addToReview).toHaveBeenCalledTimes(1)
  })

  it('does not mark the word when the API call fails', async () => {
    vi.mocked(api.addToReview).mockRejectedValueOnce(new Error('boom'))
    await expect(addWordToReview('x')).rejects.toThrow('boom')
    expect(isAddedToReview('x')).toBe(false)
  })
})
