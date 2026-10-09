// One-shot playback of /audio/<clip>, for places that can't host a SpeakButton
// (markdown html). Only one clip plays at a time.
let current: HTMLAudioElement | null = null

export function playClip(clip: string) {
  current?.pause()
  current = new Audio(`/audio/${clip}`)
  current.play().catch(() => {})
}
