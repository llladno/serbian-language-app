import { describe, it, expect, vi, afterEach } from 'vitest'
import { api, ApiError } from './api'

afterEach(() => {
  vi.unstubAllGlobals()
  try {
    localStorage.clear()
  } catch {
    /* ignore */
  }
})

describe('api', () => {
  it('parses JSON on 200', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ title: 'X', phases: [], lessons: [] }), { status: 200 }),
      ),
    )
    const c = await api.course()
    expect(c.title).toBe('X')
  })

  it('throws ApiError with server message on non-2xx', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'nope' }), { status: 404 })),
    )
    await expect(api.lesson('zz')).rejects.toMatchObject({ status: 404, message: 'nope' })
    await expect(api.lesson('zz')).rejects.toBeInstanceOf(ApiError)
  })

  it('builds query strings for vocab filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('[]', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await api.vocab({ lesson: '02', q: 'raditi' })
    expect(fetchMock.mock.calls[0][0]).toBe('/api/vocab?lesson=02&q=raditi')
  })

  it('returns undefined for 204', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))
    await expect(api.completeLesson('01')).resolves.toBeUndefined()
  })

  it('posts step status', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)
    await expect(api.setStepStatus('01', '01.2', 'done')).resolves.toBeUndefined()
    expect(fetchMock.mock.calls[0][0]).toBe('/api/lessons/01/steps/01.2')
    expect(fetchMock.mock.calls[0][1]).toMatchObject({
      method: 'POST',
      body: JSON.stringify({ status: 'done' }),
    })
  })

  it('attaches the X-User header from the stored account', async () => {
    localStorage.setItem('srpski.account', 'Гриша')
    const fetchMock = vi.fn().mockResolvedValue(new Response('{"phases":[]}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await api.progress()
    const sent = fetchMock.mock.calls[0][1].headers['X-User']
    expect(decodeURIComponent(sent)).toBe('Гриша')
  })
})
