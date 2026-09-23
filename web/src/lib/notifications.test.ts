import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { renderNotificationText, useNotifications } from './notifications'
import { api } from '../api'

// useNotifications calls onUnmounted, so it must run inside a component
// setup context — same pattern as useTelegramStart.test.ts.
function mountHarness() {
  let exposed!: ReturnType<typeof useNotifications>
  const Harness = defineComponent({
    setup() {
      exposed = useNotifications()
      return () => h('div')
    },
  })
  mount(Harness)
  return exposed
}

afterEach(() => vi.restoreAllMocks())

describe('renderNotificationText', () => {
  it('escapes raw HTML', () => {
    expect(renderNotificationText('<b>hi</b> & bye')).toBe('&lt;b&gt;hi&lt;/b&gt; &amp; bye')
  })

  it('turns [label](url) into a link for http(s) and relative urls', () => {
    const linkClass = 'class="text-[var(--accent)] underline underline-offset-2"'
    expect(renderNotificationText('Смотри [тут](https://ucimo.ru/course)')).toBe(
      `Смотри <a href="https://ucimo.ru/course" target="_blank" rel="noopener" ${linkClass}>тут</a>`,
    )
    expect(renderNotificationText('[Курс](/course)')).toBe(
      `<a href="/course" target="_blank" rel="noopener" ${linkClass}>Курс</a>`,
    )
  })

  it('does not turn a javascript: url into a link', () => {
    const input = '[кликни](javascript:alert(1))'
    expect(renderNotificationText(input)).toBe(
      '[кликни](javascript:alert(1))'.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'),
    )
  })

  it('converts newlines to <br>', () => {
    expect(renderNotificationText('строка1\nстрока2')).toBe('строка1<br>строка2')
  })
})

describe('useNotifications', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('refresh() populates items and unreadCount from the API', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({
      items: [{ id: 1, text: 'hi', created_at: '2026-09-23T00:00:00Z', read: false }],
      unread_count: 1,
    })
    const { items, unreadCount, refresh } = mountHarness()
    await refresh()
    expect(items.value).toHaveLength(1)
    expect(unreadCount.value).toBe(1)
  })

  it('markRead() optimistically clears unreadCount and calls the API', async () => {
    vi.spyOn(api, 'getNotifications').mockResolvedValue({
      items: [{ id: 1, text: 'hi', created_at: '2026-09-23T00:00:00Z', read: false }],
      unread_count: 1,
    })
    const markSpy = vi.spyOn(api, 'markNotificationsRead').mockResolvedValue({ status: 'ok' })
    const { unreadCount, refresh, markRead } = mountHarness()
    await refresh()
    await markRead()
    expect(unreadCount.value).toBe(0)
    expect(markSpy).toHaveBeenCalled()
  })

  it('markRead() is a no-op when nothing is unread', async () => {
    const markSpy = vi.spyOn(api, 'markNotificationsRead').mockResolvedValue({ status: 'ok' })
    const { markRead } = mountHarness()
    await markRead()
    expect(markSpy).not.toHaveBeenCalled()
  })
})
