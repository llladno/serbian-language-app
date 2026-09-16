// Shared /start deep-link flow for both "Войти через Telegram" (LoginView)
// and "Привязать Telegram" (ProfileView): generate a one-time token, open the
// bot chat in a new tab, poll until the webhook resolves it. See
// docs/superpowers/specs/2026-09-14-telegram-bot-login-design.md.
import { onUnmounted, ref } from 'vue'
import { api, ApiError } from '../api'
import { authErrorMessage } from './authErrors'
import type { SessionUser } from '../types'

const POLL_INTERVAL_MS = 1500

export function useTelegramStart(kind: 'login' | 'link') {
  const busy = ref(false)
  const error = ref<string | null>(null)
  let timer: ReturnType<typeof setInterval> | undefined

  function stop() {
    if (timer) clearInterval(timer)
    timer = undefined
    busy.value = false
  }
  onUnmounted(stop)

  async function start(onSuccess: (user?: SessionUser) => void) {
    if (busy.value) return
    busy.value = true
    error.value = null

    let token: string
    let popup: Window | null = null
    try {
      const res = kind === 'login' ? await api.telegramLoginStart() : await api.telegramLinkStart()
      token = res.token
      popup = window.open(res.url, '_blank')
    } catch (e) {
      busy.value = false
      error.value = authErrorMessage(e)
      return
    }

    timer = setInterval(async () => {
      try {
        const res = await api.telegramPoll(token)
        if (res.status === 'pending') return
        stop()
        popup?.close()
        if (res.status === 'ok') {
          onSuccess(res.user)
        } else {
          error.value = authErrorMessage(new ApiError(200, res.error))
        }
      } catch (e) {
        stop()
        error.value = authErrorMessage(e)
      }
    }, POLL_INTERVAL_MS)
  }

  return { busy, error, start }
}
