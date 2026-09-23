// Shared state for the donate modal, mirroring supportModal.ts: opened from
// DonateCard at the bottom of ProfileView when the click happens outside
// Telegram (inside Telegram, DonateCard skips the modal and opens
// TRIBUTE_TELEGRAM_LINK directly — see isTelegram() in ../telegram).
// Module-level refs so the single <DonateModal> instance (mounted once, in
// AppNav) shares state with the card.
import { ref } from 'vue'

export const TRIBUTE_TELEGRAM_LINK = 'https://t.me/tribute/app?startapp=dQWl'
export const TRIBUTE_WEB_LINK = 'https://web.tribute.tg/d/QWl'

const open = ref(false)

function openModal() {
  open.value = true
}
function closeModal() {
  open.value = false
}

export function useDonateModal() {
  return { open, openModal, closeModal }
}
