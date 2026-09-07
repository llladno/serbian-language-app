// Tracks which dictionary words the learner has pushed into their review queue
// this session, so every GlossedText card for the same word shows it as done.
import { reactive } from 'vue'
import { api } from '../api'

const added = reactive(new Set<string>())

export function isAddedToReview(vocabId: string): boolean {
  return added.has(vocabId)
}

export async function addWordToReview(vocabId: string): Promise<void> {
  if (added.has(vocabId)) return
  await api.addToReview(vocabId)
  added.add(vocabId)
}

// test hook
export function _resetAddedToReview(): void {
  added.clear()
}
