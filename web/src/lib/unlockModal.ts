// Shared state for the "open this level" modal: raised from a locked level on
// the course screen, rendered once by CourseView. Module-level, like the other
// shared modals.
import { ref } from 'vue'

const phaseId = ref<string | null>(null)

function openFor(id: string) {
  phaseId.value = id
}
function closeModal() {
  phaseId.value = null
}

export function useUnlockModal() {
  return { phaseId, openFor, closeModal }
}
