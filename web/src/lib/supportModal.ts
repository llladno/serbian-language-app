// Shared state for the support modal: opened either from the headphones
// icon in AppNav (available on every page) or the card at the bottom of
// ProfileView. Module-level refs (not a factory) so both triggers and the
// single <SupportModal> instance (mounted once, in AppNav) share one state.
import { ref } from 'vue'
import { api } from '../api'
import { authErrorMessage } from './authErrors'

export const SUPPORT_TELEGRAM_URL = 'https://t.me/ucimosupport'
const THANKS_AUTOCLOSE_MS = 2200

const open = ref(false)
const message = ref('')
const busy = ref(false)
const error = ref<string | null>(null)
const sent = ref(false)

function openModal() {
  open.value = true
  sent.value = false
  message.value = ''
  error.value = null
}
function closeModal() {
  open.value = false
}

async function send() {
  const trimmed = message.value.trim()
  if (!trimmed) return
  busy.value = true
  error.value = null
  try {
    await api.sendSupportMessage(trimmed)
    sent.value = true
    setTimeout(() => {
      if (sent.value) open.value = false
    }, THANKS_AUTOCLOSE_MS)
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}

export function useSupportModal() {
  return { open, message, busy, error, sent, openModal, closeModal, send }
}
