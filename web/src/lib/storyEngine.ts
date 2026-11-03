// Plays a story timeline (lib/storyPlayer.ts) with one <audio> element. The element is created by start(), which the
// "Слушать" tap calls: phones allow sound only after a touch, and an element that was started by one may switch to the
// next clip by itself afterwards.
import { onBeforeUnmount, ref, shallowRef } from 'vue'
import { activeWord, backTarget, type PassNo, type TimelineStep } from './storyPlayer'

export interface StoryEngineHooks {
  /** A pass has started (1 right at the start, 2 and 3 when their card appears). */
  onPassStart?: (pass: PassNo) => void
  /** A pass has been listened to the end. */
  onPassEnd?: (pass: PassNo) => void
  /** The timeline is over. */
  onEnd?: () => void
}

export function useStoryEngine(getTimeline: () => TimelineStep[], hooks: StoryEngineHooks = {}) {
  const at = (i: number): TimelineStep | undefined => getTimeline()[i]
  const step = shallowRef<TimelineStep | null>(null)
  const stepIndex = ref(-1)
  const wordIdx = ref(-1)
  const playing = ref(false) // the learner has not paused it
  const waitingForQuiz = ref(false)
  const started = ref(false)
  const finished = ref(false)

  let audio: HTMLAudioElement | null = null
  let preload: HTMLAudioElement | null = null
  let timer: ReturnType<typeof setTimeout> | undefined
  let raf = 0
  let pendingNext = false // paused in a gap: the next step starts on resume

  function clipUrl(s: TimelineStep): string | null {
    return s.type === 'clip' ? (s.lang === 'sr' ? s.sentence.sr_audio : s.sentence.ru_audio) : null
  }

  function stopTracking() {
    cancelAnimationFrame(raf)
    raf = 0
  }

  function track() {
    const s = step.value
    if (audio && s && s.type === 'clip') {
      const words = s.lang === 'sr' ? s.sentence.sr_words : s.sentence.ru_words
      wordIdx.value = activeWord(words, audio.currentTime * 1000)
    }
    raf = requestAnimationFrame(track)
  }

  function warm(i: number) {
    for (let k = i; k < Math.min(i + 3, getTimeline().length); k++) {
      const url = clipUrl(at(k)!)
      if (url) {
        preload ??= new Audio()
        preload.preload = 'auto'
        preload.src = url
        return
      }
    }
  }

  function goTo(i: number) {
    clearTimeout(timer)
    stopTracking()
    pendingNext = false
    const prev = at(stepIndex.value)
    const next = at(i)
    if (prev && next && next.pass !== prev.pass) hooks.onPassEnd?.(prev.pass)
    if (!next) {
      if (prev) hooks.onPassEnd?.(prev.pass)
      step.value = null
      playing.value = false
      finished.value = true
      hooks.onEnd?.()
      return
    }
    if (!prev || next.pass !== prev.pass) hooks.onPassStart?.(next.pass)
    stepIndex.value = i
    step.value = next
    wordIdx.value = -1
    waitingForQuiz.value = false
    if (next.type === 'card') {
      timer = setTimeout(() => advance(), next.ms)
      return
    }
    if (next.type === 'quiz') {
      waitingForQuiz.value = true // the audio stays silent until answerDone()
      return
    }
    if (!audio) return
    audio.src = clipUrl(next)!
    audio.playbackRate = next.rate
    audio.currentTime = 0
    void audio.play().catch(() => {
      playing.value = false // blocked: the learner presses play
    })
    raf = requestAnimationFrame(track)
    warm(i + 1)
  }

  function advance() {
    if (!playing.value) {
      pendingNext = true
      return
    }
    goTo(stepIndex.value + 1)
  }

  /** Begin from the first clip. Call from a tap. */
  function start() {
    if (started.value) return
    started.value = true
    audio = new Audio()
    audio.preservesPitch = true
    audio.addEventListener('ended', () => {
      // A clip played on its own (said after an answer) is not part of the timeline: nothing follows it.
      if (finished.value || step.value?.type !== 'clip') return
      stopTracking()
      const s = step.value
      const gap = s && s.type === 'clip' ? s.gapMs / s.rate : 0
      clearTimeout(timer)
      timer = setTimeout(() => advance(), gap)
    })
    playing.value = true
    goTo(0)
  }

  function pause() {
    if (!started.value || !playing.value) return
    playing.value = false
    audio?.pause()
    clearTimeout(timer)
    const s = step.value
    // Paused inside a card or a gap (the clip is over): remember to move on when resumed.
    if (s && (s.type === 'card' || (audio && audio.ended))) pendingNext = true
  }

  function resume() {
    if (!started.value || playing.value || finished.value) return
    playing.value = true
    if (waitingForQuiz.value) return
    if (pendingNext) {
      pendingNext = false
      goTo(stepIndex.value + 1)
      return
    }
    void audio?.play().catch(() => (playing.value = false))
    raf = requestAnimationFrame(track)
  }

  function toggle() {
    if (playing.value) pause()
    else resume()
  }

  /** One sentence back (the arrow in the header): the clip before this one, or this one again at the start of a pass. */
  function back() {
    if (!started.value || finished.value || !audio) return
    const to = backTarget(getTimeline(), stepIndex.value)
    playing.value = true
    goTo(to)
  }

  /**
   * Show the step at i without playing it: the picture, the subtitle, the progress bar. A finger dragged along the bar
   * calls this on every move, so the learner sees where it lands and the audio does not stutter through every clip.
   */
  function preview(i: number) {
    const target = at(i)
    if (!started.value || !target) return
    clearTimeout(timer)
    stopTracking()
    audio?.pause()
    pendingNext = false
    waitingForQuiz.value = false
    finished.value = false
    stepIndex.value = i
    step.value = target
    wordIdx.value = -1
  }

  /** Jump to the start of the step at i and play from there (the finger came off the bar). */
  function seek(i: number) {
    if (!started.value || !at(i)) return
    playing.value = true
    finished.value = false
    goTo(i)
  }

  /** Stop the listening and go to what follows it (the questions at the end), as if the last clip had just ended. */
  function toEnd() {
    if (!started.value || finished.value) return
    clearTimeout(timer)
    stopTracking()
    audio?.pause()
    pendingNext = false
    waitingForQuiz.value = false
    step.value = null
    playing.value = false
    finished.value = true
    hooks.onEnd?.()
  }

  /** Play one clip on its own, on the element the first tap woke up: a question that asks to listen. */
  function playOnce(url: string) {
    if (!audio) return
    clearTimeout(timer)
    stopTracking()
    audio.src = url
    audio.playbackRate = 1
    audio.currentTime = 0
    void audio.play().catch(() => {})
  }

  /** The learner answered the quick question: go on with the next sentence. */
  function answerDone() {
    if (!waitingForQuiz.value) return
    waitingForQuiz.value = false
    goTo(stepIndex.value + 1)
  }

  function destroy() {
    clearTimeout(timer)
    stopTracking()
    audio?.pause()
    audio = null
    preload = null
  }
  onBeforeUnmount(destroy)

  return { step, stepIndex, wordIdx, playing, waitingForQuiz, started, finished, start, toggle, back, preview, seek, toEnd, playOnce, answerDone, destroy }
}
