// Shared state for the reward modal. Module-level refs (not a factory), same
// as supportModal.ts: a single <RewardModal> lives in AppNav, and anything
// that pays the learner — today the quests screen — opens it from anywhere.
import { ref } from 'vue'

const open = ref(false)
const reward = ref(0)
const title = ref('')
// Set for a reward that paid itself out (a finished level): the learner did
// not come from the quests screen, so the modal offers the way there.
const questsLink = ref(false)

// Called right after the server confirmed the payment, with the reward it
// actually credited — never with what the client expected to get.
function celebrate(amount: number, questTitle: string, withQuestsLink = false) {
  reward.value = amount
  title.value = questTitle
  questsLink.value = withQuestsLink
  open.value = true
}

function closeModal() {
  open.value = false
}

export function useRewardModal() {
  return { open, reward, title, questsLink, celebrate, closeModal }
}
