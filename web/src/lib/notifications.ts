// Polling-based unread-notifications state for the bell dropdown
// (NotificationBell.vue). No WebSocket yet — see
// docs/superpowers/specs/2026-09-23-notifications-design.md.
import { onUnmounted, ref } from 'vue'
import { api } from '../api'
import type { Notification } from '../types'

const POLL_INTERVAL_MS = 45000

export function useNotifications() {
  const items = ref<Notification[]>([])
  const unreadCount = ref(0)
  let timer: ReturnType<typeof setInterval> | undefined

  async function refresh() {
    try {
      const res = await api.getNotifications()
      items.value = res.items
      unreadCount.value = res.unread_count
    } catch {
      // Best-effort polling — a transient failure just retries next tick.
    }
  }

  async function markRead() {
    if (unreadCount.value === 0) return
    items.value = items.value.map((n) => ({ ...n, read: true }))
    unreadCount.value = 0
    try {
      await api.markNotificationsRead()
    } catch {
      // Best-effort: the next refresh() restores the true count if this failed.
    }
  }

  function start() {
    refresh()
    timer = setInterval(refresh, POLL_INTERVAL_MS)
  }
  function stop() {
    if (timer) clearInterval(timer)
    timer = undefined
  }
  onUnmounted(stop)

  return { items, unreadCount, refresh, markRead, start, stop }
}

// renderNotificationText escapes text as HTML, then turns [label](url) into
// a link — url must start with "http://", "https://", or "/" (blocks
// javascript: and other unsafe schemes) — and remaining newlines into <br>.
// Deliberately not full markdown (see the design doc): only this one
// pattern is recognized, so admin copy can never render an unexpected
// heading/list/table.
const LINK_PATTERN = /\[([^\]]+)\]\((https?:\/\/[^\s)]+|\/[^\s)]*)\)/g

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export function renderNotificationText(text: string): string {
  const escaped = escapeHtml(text)
  const withLinks = escaped.replace(
    LINK_PATTERN,
    (_match, label: string, url: string) => `<a href="${url}" target="_blank" rel="noopener">${label}</a>`,
  )
  return withLinks.replace(/\n/g, '<br>')
}
