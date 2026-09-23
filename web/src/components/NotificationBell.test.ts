import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import NotificationBell from './NotificationBell.vue'
import { api } from '../api'

beforeEach(() => vi.restoreAllMocks())

describe('NotificationBell', () => {
  it('shows the unread dot when there are unread notifications', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({
      items: [{ id: 1, text: 'hi', created_at: '2026-09-23T00:00:00Z', read: false }],
      unread_count: 1,
    })
    const w = mount(NotificationBell)
    await flushPromises()
    expect(w.find('[data-test="notification-dot"]').exists()).toBe(true)
  })

  it('hides the dot when there is nothing unread', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({ items: [], unread_count: 0 })
    const w = mount(NotificationBell)
    await flushPromises()
    expect(w.find('[data-test="notification-dot"]').exists()).toBe(false)
  })

  it('marks everything read and clears the dot when the dropdown opens', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({
      items: [{ id: 1, text: 'hi', created_at: '2026-09-23T00:00:00Z', read: false }],
      unread_count: 1,
    })
    const markSpy = vi.spyOn(api, 'markNotificationsRead').mockResolvedValue({ status: 'ok' })
    const w = mount(NotificationBell)
    await flushPromises()
    expect(w.find('[data-test="notification-dot"]').exists()).toBe(true)

    await w.find('[data-test="open-notifications"]').trigger('click')
    await flushPromises()

    expect(markSpy).toHaveBeenCalled()
    expect(w.find('[data-test="notification-dot"]').exists()).toBe(false)
  })

  it('renders notification text with links via renderNotificationText', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({
      items: [{ id: 1, text: 'Зайди [сюда](https://ucimo.ru)', created_at: '2026-09-23T00:00:00Z', read: false }],
      unread_count: 1,
    })
    vi.spyOn(api, 'markNotificationsRead').mockResolvedValue({ status: 'ok' })
    const w = mount(NotificationBell)
    await flushPromises()
    await w.find('[data-test="open-notifications"]').trigger('click')
    await flushPromises()
    const link = w.find('a[href="https://ucimo.ru"]')
    expect(link.exists()).toBe(true)
    expect(link.text()).toBe('сюда')
  })
})
