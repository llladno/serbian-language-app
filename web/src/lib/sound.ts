// The sounds the app makes by itself, as opposed to the speaker buttons a learner
// taps: the chime and buzz that close a lesson, and the right sentence read out
// after an answer. "Звук" in the settings turns off both sounds, "Вибрация" the
// buzz; both live on the device, like the theme.
//
// Phones only let a page make sound after a touch, and these sounds start later
// (after the server has answered), so the first touch anywhere is used to wake the
// audio up (installAudioUnlock). The chime is synthesised, so there is no file to
// ship; the sentences are the same clips the speaker buttons play.
import { ref } from 'vue'
import { hapticSuccess } from '../telegram'

const SOUND_KEY = 'srpski.sound'
const HAPTICS_KEY = 'srpski.haptics'

function read(key: string): boolean {
  try {
    return localStorage.getItem(key) !== 'off'
  } catch {
    return true
  }
}
function write(key: string, on: boolean) {
  try {
    localStorage.setItem(key, on ? 'on' : 'off')
  } catch {
    /* the choice just does not outlive the page */
  }
}

export const soundOn = ref(read(SOUND_KEY))
export const hapticsOn = ref(read(HAPTICS_KEY))

export function setSound(on: boolean) {
  soundOn.value = on
  write(SOUND_KEY, on)
}
export function setHaptics(on: boolean) {
  hapticsOn.value = on
  write(HAPTICS_KEY, on)
}

let ctx: AudioContext | null = null
function audioCtx(): AudioContext | null {
  if (typeof AudioContext === 'undefined') return null
  try {
    ctx ??= new AudioContext()
    if (ctx.state === 'suspended') void ctx.resume()
    return ctx
  } catch {
    return null
  }
}

// One element for every spoken sentence. A phone allows an element to play later,
// outside a touch, only if it was started during one, and a fresh Audio() per
// sentence would lose that; reusing it also makes a new sentence cut off the last.
let voice: HTMLAudioElement | null = null
// 44-byte WAV header with no samples: something to "play" during the touch.
const SILENCE = 'data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQAAAAA='

function unlock() {
  audioCtx()
  try {
    voice ??= new Audio()
    voice.muted = true
    voice.src = SILENCE
    void voice
      .play()
      .then(() => {
        // A sentence may have started meanwhile; only the silence is stopped.
        if (voice && voice.src === SILENCE) voice.pause()
      })
      .catch(() => {})
  } catch {
    /* no audio, no unlocking */
  }
}

/** Wake the audio up on the first touch of the page. */
export function installAudioUnlock() {
  const events = ['pointerdown', 'touchend', 'keydown']
  const once = () => {
    for (const e of events) window.removeEventListener(e, once, true)
    unlock()
  }
  for (const e of events) window.addEventListener(e, once, true)
}

// C5 E5 G5 C6, a quick major arpeggio: soft sine notes that ring out on the last.
const CHIME = [523.25, 659.25, 783.99, 1046.5]

/** The chime for a finished lesson. */
export function playWin() {
  if (!soundOn.value) return
  const c = audioCtx()
  if (!c) return
  const t0 = c.currentTime + 0.02
  CHIME.forEach((freq, i) => {
    const osc = c.createOscillator()
    const gain = c.createGain()
    const t = t0 + i * 0.09
    const end = t + (i === CHIME.length - 1 ? 0.7 : 0.3)
    osc.type = 'sine'
    osc.frequency.value = freq
    gain.gain.setValueAtTime(0.0001, t)
    gain.gain.exponentialRampToValueAtTime(0.22, t + 0.015)
    gain.gain.exponentialRampToValueAtTime(0.0001, end)
    osc.connect(gain).connect(c.destination)
    osc.start(t)
    osc.stop(end + 0.05)
  })
}

/** Read a phrase clip (path relative to /audio/) out loud, unless the sound is off. */
export function speakClip(clip: string) {
  if (!soundOn.value) return
  try {
    voice ??= new Audio()
    voice.muted = false
    voice.src = `/audio/${clip}`
    voice.currentTime = 0
    void voice.play().catch(() => {})
  } catch {
    /* a sentence that does not play is not worth an error */
  }
}

/** Everything that marks a lesson as finished: the chime and the buzz. */
export function celebrate() {
  playWin()
  if (hapticsOn.value) hapticSuccess()
}
