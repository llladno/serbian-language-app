import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TelegramLoginButton from './TelegramLoginButton.vue'

afterEach(() => {
  vi.restoreAllMocks()
  delete (window as unknown as { Telegram?: unknown }).Telegram
  document.head.innerHTML = ''
})

describe('TelegramLoginButton', () => {
  it('renders nothing until the widget script signals it is ready', async () => {
    const w = mount(TelegramLoginButton, { props: { botId: '42' } })
    expect(w.find('button').exists()).toBe(false)
    window.Telegram = { Login: { auth: vi.fn() } }
    document.head.querySelector('script')?.dispatchEvent(new Event('load'))
    await flushPromises()
    expect(w.find('button').exists()).toBe(true)
  })

  it('opens the Telegram auth popup with bot_id and emits auth on success', async () => {
    const authMock = vi.fn((_opts, cb) => cb({ id: 1, username: 'x', auth_date: 1, hash: 'h' }))
    window.Telegram = { Login: { auth: authMock } }
    const w = mount(TelegramLoginButton, { props: { botId: '42' } })
    document.head.querySelector('script')?.dispatchEvent(new Event('load'))
    await flushPromises()
    await w.find('button').trigger('click')
    expect(authMock).toHaveBeenCalledWith({ bot_id: '42', request_access: true }, expect.any(Function))
    expect(w.emitted('auth')?.[0][0]).toMatchObject({ id: 1, username: 'x' })
  })
})
