import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  captureAttribution,
  captureReferral,
  clearStoredReferral,
  getStoredAttribution,
  getStoredReferral,
} from './attribution'

function setUrl(search: string) {
  window.history.replaceState({}, '', '/' + search)
}

describe('captureAttribution', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    vi.restoreAllMocks()
    setUrl('')
  })

  it('does nothing without utm params in the URL', () => {
    const fetchSpy = vi.spyOn(window, 'fetch')
    captureAttribution()
    expect(fetchSpy).not.toHaveBeenCalled()
    expect(getStoredAttribution()).toEqual({})
  })

  it('stores utm params as first-touch and fires a visit beacon', () => {
    const fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(new Response(null, { status: 204 }))
    setUrl('?utm_source=vk&utm_medium=social&utm_campaign=launch')

    captureAttribution()

    expect(getStoredAttribution()).toEqual({ utm_source: 'vk', utm_medium: 'social', utm_campaign: 'launch' })
    expect(fetchSpy).toHaveBeenCalledWith('/api/track-visit', expect.objectContaining({ method: 'POST' }))
  })

  it('does not overwrite an already-stored first touch', () => {
    setUrl('?utm_source=vk&utm_campaign=launch')
    captureAttribution()

    setUrl('?utm_source=instagram&utm_campaign=story')
    captureAttribution()

    expect(getStoredAttribution()).toEqual({ utm_source: 'vk', utm_campaign: 'launch' })
  })

  it('logs the visit beacon only once per browser session', () => {
    const fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(new Response(null, { status: 204 }))
    setUrl('?utm_source=vk&utm_campaign=launch')
    captureAttribution()
    captureAttribution()
    expect(fetchSpy).toHaveBeenCalledTimes(1)
  })
})

describe('invite link', () => {
  beforeEach(() => {
    localStorage.clear()
    setUrl('')
  })

  it('remembers the code from /register?ref=', () => {
    setUrl('?ref=abcd2345')
    captureReferral()
    expect(getStoredReferral()).toBe('abcd2345')
  })

  it('takes the code case-insensitively, as people retype links', () => {
    setUrl('?ref=ABCD2345')
    captureReferral()
    expect(getStoredReferral()).toBe('abcd2345')
  })

  it('ignores junk instead of storing it', () => {
    for (const bad of ['', 'x', '<script>', 'a b c d e f', 'a'.repeat(40)]) {
      setUrl('?ref=' + encodeURIComponent(bad))
      captureReferral()
      expect(getStoredReferral()).toBe('')
    }
  })

  it('lets a newer link replace an older one', () => {
    setUrl('?ref=aaaa2222')
    captureReferral()
    setUrl('?ref=bbbb3333')
    captureReferral()
    expect(getStoredReferral()).toBe('bbbb3333')
  })

  it('keeps the code when the URL no longer has it, until it is cleared', () => {
    setUrl('?ref=abcd2345')
    captureReferral()
    setUrl('')
    captureReferral()
    expect(getStoredReferral()).toBe('abcd2345')
    clearStoredReferral()
    expect(getStoredReferral()).toBe('')
  })
})
