import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// The module keeps its settings in module-level refs, so every test loads a fresh one.
const load = async () => {
  vi.resetModules()
  return await import('./sound')
}

class FakeOsc {
  type = ''
  frequency = { value: 0 }
  start = vi.fn()
  stop = vi.fn()
  connect = vi.fn((to: unknown) => to)
}
class FakeCtx {
  static all: FakeCtx[] = []
  state = 'suspended'
  currentTime = 0
  destination = {}
  oscs: FakeOsc[] = []
  resume = vi.fn(async () => {
    this.state = 'running'
  })
  constructor() {
    FakeCtx.all.push(this)
  }
  createOscillator() {
    const o = new FakeOsc()
    this.oscs.push(o)
    return o
  }
  createGain() {
    return {
      gain: { setValueAtTime: vi.fn(), exponentialRampToValueAtTime: vi.fn() },
      connect: vi.fn((to: unknown) => to),
    }
  }
}
class FakeAudio {
  static all: FakeAudio[] = []
  src = ''
  muted = false
  currentTime = 0
  play = vi.fn(() => Promise.resolve())
  pause = vi.fn()
  constructor() {
    FakeAudio.all.push(this)
  }
}

function telegram(version = '7.0') {
  const notificationOccurred = vi.fn()
  ;(window as unknown as { Telegram: unknown }).Telegram = {
    WebApp: {
      ready() {},
      expand() {},
      initData: '',
      colorScheme: 'light',
      isVersionAtLeast: (v: string) => parseFloat(version) >= parseFloat(v),
      HapticFeedback: { notificationOccurred },
    },
  }
  return notificationOccurred
}

beforeEach(() => {
  FakeCtx.all = []
  FakeAudio.all = []
  localStorage.clear()
  vi.stubGlobal('AudioContext', FakeCtx)
  vi.stubGlobal('Audio', FakeAudio)
})
afterEach(() => {
  vi.unstubAllGlobals()
  delete (window as unknown as { Telegram?: unknown }).Telegram
})

describe('settings', () => {
  it('is on by default, for the sound and for the vibration', async () => {
    const s = await load()
    expect(s.soundOn.value).toBe(true)
    expect(s.hapticsOn.value).toBe(true)
  })

  it('remembers a choice on the device', async () => {
    let s = await load()
    s.setSound(false)
    s.setHaptics(false)
    s = await load()
    expect(s.soundOn.value).toBe(false)
    expect(s.hapticsOn.value).toBe(false)
  })

  it('works when storage is unavailable', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    const s = await load()
    expect(s.soundOn.value).toBe(true)
    expect(() => s.setSound(false)).not.toThrow()
    expect(s.soundOn.value).toBe(false)
    vi.restoreAllMocks()
  })
})

describe('the finishing chime', () => {
  it('plays a rising run of notes', async () => {
    const s = await load()
    s.playWin()
    const ctx = FakeCtx.all[0]
    expect(ctx.oscs.length).toBeGreaterThanOrEqual(3)
    const freqs = ctx.oscs.map((o) => o.frequency.value)
    expect([...freqs].sort((a, b) => a - b)).toEqual(freqs)
    for (const o of ctx.oscs) {
      expect(o.start).toHaveBeenCalledTimes(1)
      expect(o.stop).toHaveBeenCalledTimes(1)
    }
  })

  it('is silent once the sound is switched off', async () => {
    const s = await load()
    s.setSound(false)
    s.playWin()
    expect(FakeCtx.all.flatMap((c) => c.oscs)).toHaveLength(0)
  })

  it('does nothing where the browser has no Web Audio', async () => {
    vi.stubGlobal('AudioContext', undefined)
    const s = await load()
    expect(() => s.playWin()).not.toThrow()
  })
})

describe('the spoken sentence', () => {
  it('plays the clip from /audio', async () => {
    const s = await load()
    s.speakClip('p/abc.mp3')
    expect(FakeAudio.all).toHaveLength(1)
    expect(FakeAudio.all[0].src).toBe('/audio/p/abc.mp3')
    expect(FakeAudio.all[0].play).toHaveBeenCalledTimes(1)
  })

  it('reuses one element, so one sentence cuts off the previous one', async () => {
    const s = await load()
    s.speakClip('p/one.mp3')
    s.speakClip('p/two.mp3')
    expect(FakeAudio.all).toHaveLength(1)
    expect(FakeAudio.all[0].src).toBe('/audio/p/two.mp3')
    expect(FakeAudio.all[0].play).toHaveBeenCalledTimes(2)
  })

  it('is silent once the sound is switched off', async () => {
    const s = await load()
    s.setSound(false)
    s.speakClip('p/abc.mp3')
    expect(FakeAudio.all.flatMap((a) => a.play.mock.calls)).toHaveLength(0)
  })

  it('survives a refused play()', async () => {
    const s = await load()
    s.speakClip('p/one.mp3')
    FakeAudio.all[0].play.mockRejectedValueOnce(new Error('NotAllowedError'))
    expect(() => s.speakClip('p/two.mp3')).not.toThrow()
    await Promise.resolve()
  })
})

describe('unlocking audio on the first touch', () => {
  it('wakes the audio context on the first tap, and only then', async () => {
    const s = await load()
    s.installAudioUnlock()
    expect(FakeCtx.all).toHaveLength(0)
    window.dispatchEvent(new Event('pointerdown'))
    expect(FakeCtx.all).toHaveLength(1)
    expect(FakeCtx.all[0].resume).toHaveBeenCalled()
    window.dispatchEvent(new Event('pointerdown'))
    expect(FakeCtx.all).toHaveLength(1)
  })

  it('readies the voice element in the tap, so a clip started later may play', async () => {
    const s = await load()
    s.installAudioUnlock()
    window.dispatchEvent(new Event('pointerdown'))
    expect(FakeAudio.all).toHaveLength(1)
    expect(FakeAudio.all[0].play).toHaveBeenCalledTimes(1)
    s.speakClip('p/abc.mp3')
    expect(FakeAudio.all).toHaveLength(1)
    expect(FakeAudio.all[0].src).toBe('/audio/p/abc.mp3')
    expect(FakeAudio.all[0].muted).toBe(false)
  })
})

describe('celebrate', () => {
  it('chimes and buzzes the phone inside Telegram', async () => {
    const buzz = telegram()
    const s = await load()
    s.celebrate()
    expect(FakeCtx.all[0].oscs.length).toBeGreaterThan(0)
    expect(buzz).toHaveBeenCalledTimes(1)
    expect(buzz).toHaveBeenCalledWith('success')
  })

  it('keeps the vibration when only the sound is off, and the other way round', async () => {
    const buzz = telegram()
    let s = await load()
    s.setSound(false)
    s.celebrate()
    expect(FakeCtx.all.flatMap((c) => c.oscs)).toHaveLength(0)
    expect(buzz).toHaveBeenCalledTimes(1)

    s = await load()
    s.setSound(true)
    s.setHaptics(false)
    s.celebrate()
    expect(buzz).toHaveBeenCalledTimes(1)
    expect(FakeCtx.all.flatMap((c) => c.oscs).length).toBeGreaterThan(0)
  })

  it('is just the chime outside Telegram', async () => {
    const s = await load()
    expect(() => s.celebrate()).not.toThrow()
    expect(FakeCtx.all[0].oscs.length).toBeGreaterThan(0)
  })

  it('leaves an old Telegram client alone, which has no haptics before 6.1', async () => {
    const buzz = telegram('6.0')
    const s = await load()
    expect(() => s.celebrate()).not.toThrow()
    expect(buzz).not.toHaveBeenCalled()
  })
})
