import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { useTelegramStart } from './telegramStart'
import { api, ApiError } from '../api'

// useTelegramStart calls onUnmounted, so it must run inside a component setup
// context — wrap it in a tiny throwaway component for every test.
function mountHarness(kind: 'login' | 'link') {
  let exposed!: ReturnType<typeof useTelegramStart>
  const Harness = defineComponent({
    setup() {
      exposed = useTelegramStart(kind)
      return () => h('div')
    },
  })
  mount(Harness)
  return exposed
}

beforeEach(() => vi.useFakeTimers())
afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('useTelegramStart', () => {
  it('opens a blank tab synchronously, points it at the URL, and closes it when done', async () => {
    vi.spyOn(api, 'telegramLoginStart').mockResolvedValue({ url: 'https://t.me/bot?start=tok', token: 'tok' })
    const popup = { close: vi.fn(), location: { href: '' } } as unknown as Window
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(popup)
    const poll = vi
      .spyOn(api, 'telegramPoll')
      .mockResolvedValueOnce({ status: 'pending' })
      .mockResolvedValueOnce({
        status: 'ok',
        user: { id: '1', name: 'Г', email: '', email_verified: true, telegram: { linked: true, username: 'g' } },
      })

    const tg = mountHarness('login')
    const onSuccess = vi.fn()
    await tg.start(onSuccess)
    expect(tg.busy.value).toBe(true)
    // The blank tab must open before the await, so it stays a trusted popup
    // in Safari — never with the real URL directly (that would arrive too
    // late, after the click's user-gesture window has already closed).
    expect(openSpy).toHaveBeenCalledWith('', '_blank')
    expect(openSpy).toHaveBeenCalledTimes(1)
    expect(popup.location.href).toBe('https://t.me/bot?start=tok')

    await vi.advanceTimersByTimeAsync(1500)
    expect(poll).toHaveBeenCalledTimes(1)
    expect(tg.busy.value).toBe(true) // still pending, keep polling
    expect(onSuccess).not.toHaveBeenCalled()
    expect(popup.close).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1500)
    expect(poll).toHaveBeenCalledTimes(2)
    expect(tg.busy.value).toBe(false)
    expect(popup.close).toHaveBeenCalledTimes(1)
    expect(onSuccess).toHaveBeenCalledWith({
      id: '1',
      name: 'Г',
      email: '',
      email_verified: true,
      telegram: { linked: true, username: 'g' },
    })
  })

  it('stops polling and surfaces the mapped error on a poll error result', async () => {
    vi.spyOn(api, 'telegramLinkStart').mockResolvedValue({ url: 'https://t.me/bot?start=tok', token: 'tok' })
    vi.spyOn(window, 'open').mockReturnValue(null)
    vi.spyOn(api, 'telegramPoll').mockResolvedValue({ status: 'error', error: 'telegram_taken' })

    const tg = mountHarness('link')
    const onSuccess = vi.fn()
    await tg.start(onSuccess)
    await vi.advanceTimersByTimeAsync(1500)

    expect(tg.busy.value).toBe(false)
    expect(onSuccess).not.toHaveBeenCalled()
    expect(tg.error.value).toBe('Этот Telegram уже привязан к другому аккаунту')
  })

  it('closes the blank tab and surfaces an error if the start call itself fails', async () => {
    vi.spyOn(api, 'telegramLoginStart').mockRejectedValue(new ApiError(503, 'telegram_disabled'))
    const popup = { close: vi.fn(), location: { href: '' } } as unknown as Window
    vi.spyOn(window, 'open').mockReturnValue(popup)

    const tg = mountHarness('login')
    await tg.start(vi.fn())

    // The blank tab already opened (before the failing request) — it must
    // not be left dangling once the request errors out.
    expect(popup.close).toHaveBeenCalledTimes(1)
    expect(tg.busy.value).toBe(false)
    expect(tg.error.value).toBe('Вход через Telegram пока не настроен')
  })

  it('does not start a second poll loop while one is already busy', async () => {
    vi.spyOn(api, 'telegramLoginStart').mockResolvedValue({ url: 'https://t.me/bot?start=tok', token: 'tok' })
    vi.spyOn(window, 'open').mockReturnValue(null)
    vi.spyOn(api, 'telegramPoll').mockResolvedValue({ status: 'pending' })

    const tg = mountHarness('login')
    await tg.start(vi.fn())
    await tg.start(vi.fn()) // second call while still busy — must be a no-op

    expect(api.telegramLoginStart).toHaveBeenCalledTimes(1)
  })
})
