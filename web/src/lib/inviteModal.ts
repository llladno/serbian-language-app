// Shared state for the "invite a friend" modal: raised from the quest row,
// rendered once by QuestsView. Module-level, like the other shared modals.
import { ref } from 'vue'

const open = ref(false)

function openModal() {
  open.value = true
}
function closeModal() {
  open.value = false
}

export function useInviteModal() {
  return { open, openModal, closeModal }
}
