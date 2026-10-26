// Shared state for the "choose your level" window that follows a paid level: raised on coming back
// from the bank, from the notice in the profile and from the tariffs screen, rendered once by AppNav.
// Module-level, like the other shared modals. Closing it keeps the paid level: it waits to be spent.
import { ref } from 'vue'

const open = ref(false)
// The paid order to spend. null: the oldest one waiting (the tariffs store knows them).
const orderId = ref<number | null>(null)

function openFor(id: number | null = null) {
  orderId.value = id
  open.value = true
}
function closeModal() {
  open.value = false
}

export function useLevelChoiceModal() {
  return { open, orderId, openFor, closeModal }
}
