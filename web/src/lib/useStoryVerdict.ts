// What every question of a story does with the answer, in one place. A right answer takes the place of the controls:
// the box keeps the height the controls had (no jump), «Верно» grows in the middle of it, and after a beat the story
// goes on. A wrong one leaves the controls where they are, marked, with the right answer shown, and waits longer so
// it can be read. Both then say `done`.
import { ref } from 'vue'
import type { StoryVerdict } from '../types'

export const RIGHT_BEAT_MS = 1200
export const WRONG_BEAT_MS = 1900
export const SAY_DELAY_MS = 280
export const SAY_TAIL_MS = 450

export function useStoryVerdict(done: (correct: boolean) => void) {
  const verdict = ref<StoryVerdict | null>(null)
  /** The element that holds the controls: its height is kept while «Верно» stands in for it. */
  const controls = ref<HTMLElement | null>(null)
  const lockedHeight = ref(0)
  // A function ref for the template (`:ref="setControls"`).
  const setControls = (el: unknown) => {
    controls.value = el instanceof HTMLElement ? el : null
  }

  /**
   * Show the verdict, and go on after a beat. With a clip to say (sayMs > 0) the beat is stretched to hold it: the
   * clip starts a moment after the verdict appears (SAY_DELAY_MS) and the story waits for it to end.
   */
  function settle(v: StoryVerdict, sayMs = 0) {
    lockedHeight.value = controls.value?.offsetHeight ?? 0
    verdict.value = v
    const beat = v.correct ? RIGHT_BEAT_MS : WRONG_BEAT_MS
    setTimeout(() => done(v.correct), sayMs > 0 ? Math.max(beat, SAY_DELAY_MS + sayMs + SAY_TAIL_MS) : beat)
  }
  function reset() {
    verdict.value = null
  }
  return { verdict, lockedHeight, setControls, settle, reset }
}
