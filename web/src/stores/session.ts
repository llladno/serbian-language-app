import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'
import type { SessionUser } from '../types'
import { isTelegram, initData } from '../telegram'
import { clearStoredReferral, getStoredReferral } from '../attribution'

// A guess for the very first paint of a reload: who was logged in last time,
// so App.vue can show the real header/nav immediately instead of a blank
// "Загрузка…" screen while fetchSession() confirms it against the server.
// Purely a display cache — access is still decided by the httpOnly session
// cookie on every actual API call, never by this.
const CACHE_KEY = 'ucimo_session_user'

function readCachedUser(): SessionUser | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY)
    return raw ? (JSON.parse(raw) as SessionUser) : null
  } catch {
    return null
  }
}

function writeCachedUser(u: SessionUser | null) {
  try {
    if (u) localStorage.setItem(CACHE_KEY, JSON.stringify(u))
    else localStorage.removeItem(CACHE_KEY)
  } catch {
    // private browsing / storage full — it's only a display cache, fine to skip
  }
}

// A learner who arrived by a friend's link has the code in local storage. The
// code normally rides along with the registration itself; this covers the
// signups that could not carry it (the email confirmed on another device, a
// Telegram login that was not a first one). It is asked once and cleared either
// way: the server decides whether the code still counts, and a stale one is
// not an error for the learner.
async function settleReferral() {
  const code = getStoredReferral()
  if (!code) return
  try {
    await api.claimReferral(code)
  } catch {
    return // offline or signed out: keep the code for the next session
  }
  clearStoredReferral()
}

export const useSessionStore = defineStore('session', () => {
  const user = ref<SessionUser | null>(readCachedUser())
  // Only true when we have no guess to show at all — see readCachedUser().
  const loading = ref(!user.value)

  async function fetchSession() {
    const hadGuess = !!user.value
    try {
      user.value = await api.session()
      writeCachedUser(user.value)
      void settleReferral()
    } catch {
      user.value = null
      writeCachedUser(null)
      if (isTelegram() && initData()) {
        try {
          user.value = await api.telegramLogin({ init_data: initData() })
          writeCachedUser(user.value)
          void settleReferral()
        } catch {
          user.value = null
        }
      }
      // The optimistic guess above turned out wrong (cookie expired or was
      // cleared elsewhere) — the authed shell is already showing over
      // content that will just 401 on every request, so bounce to login
      // instead of leaving it stuck.
      if (hadGuess && !user.value) {
        const { default: router } = await import('../router')
        if (router.currentRoute.value.path !== '/login') {
          router.push({ path: '/login', query: { next: router.currentRoute.value.fullPath } })
        }
      }
    } finally {
      loading.value = false
    }
  }

  async function login(email: string, password: string) {
    user.value = await api.login(email, password)
    writeCachedUser(user.value)
    void settleReferral()
  }

  async function register(email: string, password: string, name: string) {
    await api.register(email, password, name)
  }

  async function logout() {
    await api.logout()
    user.value = null
    writeCachedUser(null)
  }

  async function logoutAll() {
    await api.logoutAll()
    user.value = null
    writeCachedUser(null)
  }

  async function forgot(email: string) {
    await api.forgotPassword(email)
  }

  async function reset(token: string, password: string) {
    user.value = await api.resetPassword(token, password)
    writeCachedUser(user.value)
  }

  async function resendVerification(email: string) {
    await api.resendVerification(email)
  }

  return {
    user,
    loading,
    fetchSession,
    settleReferral,
    login,
    register,
    logout,
    logoutAll,
    forgot,
    reset,
    resendVerification,
  }
})
