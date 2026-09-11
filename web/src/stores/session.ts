import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'
import type { SessionUser } from '../types'

export const useSessionStore = defineStore('session', () => {
  const user = ref<SessionUser | null>(null)
  const loading = ref(true)

  async function fetchSession() {
    loading.value = true
    try {
      user.value = await api.session()
    } catch {
      user.value = null
    } finally {
      loading.value = false
    }
  }

  async function login(email: string, password: string) {
    user.value = await api.login(email, password)
  }

  async function register(email: string, password: string, name: string) {
    await api.register(email, password, name)
  }

  async function logout() {
    await api.logout()
    user.value = null
  }

  async function logoutAll() {
    await api.logoutAll()
    user.value = null
  }

  async function forgot(email: string) {
    await api.forgotPassword(email)
  }

  async function reset(token: string, password: string) {
    user.value = await api.resetPassword(token, password)
  }

  async function resendVerification(email: string) {
    await api.resendVerification(email)
  }

  return {
    user,
    loading,
    fetchSession,
    login,
    register,
    logout,
    logoutAll,
    forgot,
    reset,
    resendVerification,
  }
})
