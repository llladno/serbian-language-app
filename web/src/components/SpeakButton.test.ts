import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { nextTick } from 'vue'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import SpeakButton from './SpeakButton.vue'

// A fake <audio>: records what it was asked to play and lets a test finish or
// interrupt the clip the way the browser does.
class FakeAudio {
  static all: FakeAudio[] = []
  listeners: Record<string, Array<() => void>> = {}
  currentTime = 0
  paused = true
  src: string
  constructor(src: string) {
    this.src = src
    FakeAudio.all.push(this)
  }
  addEventListener(name: string, fn: () => void) {
    ;(this.listeners[name] ??= []).push(fn)
  }
  fire(name: string) {
    this.listeners[name]?.forEach((f) => f())
  }
  play() {
    this.paused = false
    return Promise.resolve()
  }
  pause() {
    if (this.paused) return
    this.paused = true
    this.fire('pause')
  }
}

enableAutoUnmount(afterEach)
beforeEach(() => {
  FakeAudio.all = []
  vi.stubGlobal('Audio', FakeAudio)
})
afterEach(() => vi.unstubAllGlobals())

describe('SpeakButton', () => {
  it('plays the clip it is showing now, not the one it showed first', async () => {
    const w = mount(SpeakButton, { props: { src: 'a.mp3' } })
    await w.find('button').trigger('click')
    await w.setProps({ src: 'b.mp3' })
    await w.find('button').trigger('click')
    const last = FakeAudio.all.filter((a) => !a.paused).at(-1)
    expect(last?.src).toBe('/audio/b.mp3')
    expect(FakeAudio.all.filter((a) => !a.paused)).toHaveLength(1)
  })

  it('stops the old clip when the word changes under it', async () => {
    const w = mount(SpeakButton, { props: { src: 'a.mp3' } })
    await w.find('button').trigger('click')
    await w.setProps({ src: 'b.mp3' })
    expect(FakeAudio.all[0].paused).toBe(true)
    expect(w.find('button').attributes('aria-label')).toBe('озвучить')
  })

  it('goes back to idle when a clip is interrupted', async () => {
    const w = mount(SpeakButton, { props: { src: 'a.mp3' } })
    await w.find('button').trigger('click')
    expect(w.find('button').attributes('aria-label')).toBe('играет')
    FakeAudio.all[0].pause()
    await nextTick()
    expect(w.find('button').attributes('aria-label')).toBe('озвучить')
  })

  it('stops playing when it leaves the page', async () => {
    const w = mount(SpeakButton, { props: { src: 'a.mp3' } })
    await w.find('button').trigger('click')
    w.unmount()
    expect(FakeAudio.all[0].paused).toBe(true)
  })
})
