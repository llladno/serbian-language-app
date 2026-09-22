import { beforeEach, describe, expect, it, vi } from 'vitest'
import { captureAttribution, getStoredAttribution } from './attribution'

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
