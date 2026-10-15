// Which Telegram messages the learner accepts. One store so the profile's
// settings and the tariff screen's "remind me" switch always show the same
// answer. A switch flips at once and rolls back if the server says no: a
// toggle that waits a second for the network feels broken.
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'
import type { BotPrefs } from '../types'

export const useBotPrefsStore = defineStore('botPrefs', () => {
  const prefs = ref<BotPrefs | null>(null)
  const error = ref<string | null>(null)

  async function load(force = false) {
    if (prefs.value && !force) return
    error.value = null
    try {
      prefs.value = await api.botPrefs()
    } catch {
      error.value = 'Не получилось загрузить настройки'
    }
  }

  async function set(key: keyof BotPrefs, value: boolean) {
    if (!prefs.value) return
    const before = prefs.value
    error.value = null
    prefs.value = { ...before, [key]: value }
    try {
      prefs.value = await api.setBotPrefs({ [key]: value })
    } catch {
      prefs.value = before
      error.value = 'Не получилось сохранить, попробуй ещё раз'
    }
  }

  return { prefs, error, load, set }
})
