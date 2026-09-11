import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { api, ApiError } from './api'

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.unstubAllGlobals())

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

  it('sends same-origin credentials and no X-User header', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('{"phases":[]}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await api.progress()
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/progress')
    expect(init.credentials).toBe('same-origin')
    expect(init.headers['X-User']).toBeUndefined()
  })

  it('on a 401 from a non-auth path, clears the session and redirects to /login', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'no session' }), { status: 401 })),
    )
    const { useSessionStore } = await import('./stores/session')
    const { default: router } = await import('./router')
    const session = useSessionStore()
    session.user = {
      id: '1',
      name: 'Х',
      email: '',
      email_verified: false,
      telegram: { linked: false, username: '' },
    }
    const pushSpy = vi.spyOn(router, 'push')

    await expect(api.progress()).rejects.toBeInstanceOf(ApiError)
    expect(session.user).toBeNull()
    expect(pushSpy).toHaveBeenCalledWith(expect.objectContaining({ path: '/login' }))
  })

  it('does not redirect on a 401 from an /auth/ path (e.g. wrong-password login)', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'неверная почта или пароль' }), { status: 401 }),
      ),
    )
    const { default: router } = await import('./router')
    const pushSpy = vi.spyOn(router, 'push')

    await expect(api.login('a@b.com', 'wrong')).rejects.toMatchObject({ status: 401 })
    expect(pushSpy).not.toHaveBeenCalled()
  })
})
