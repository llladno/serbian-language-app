import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import SoundSettings from './SoundSettings.vue'
import { hapticsOn, setHaptics, setSound, soundOn } from '../lib/sound'

// telegram-web-app.js is loaded in every browser, so the SDK object exists
// outside Telegram too; only inside the Mini App does it carry initData.
function sdk(initData: string) {
  ;(window as unknown as { Telegram: unknown }).Telegram = {
    WebApp: { ready() {}, expand() {}, initData, colorScheme: 'light' },
  }
}
const inTelegram = () => sdk('query_id=AAH&user=%7B%7D&hash=abc')

beforeEach(() => {
  localStorage.clear()
  setSound(true)
  setHaptics(true)
})
afterEach(() => {
  delete (window as unknown as { Telegram?: unknown }).Telegram
})

const sound = '[data-test="setting-sound"]'
const haptics = '[data-test="setting-haptics"]'

describe('SoundSettings', () => {
  it('shows the sound as on by default', () => {
    const w = mount(SoundSettings)
    expect((w.find(sound).element as HTMLInputElement).checked).toBe(true)
  })

  it('says what the sound covers: the chime and the spoken sentences', () => {
    const w = mount(SoundSettings)
    expect(w.text()).toContain('Звук')
    expect(w.text()).toContain('озвучка')
  })

  it('switches the sound off and on, and the choice is kept', async () => {
    const w = mount(SoundSettings)
    await w.find(sound).setValue(false)
    expect(soundOn.value).toBe(false)
    expect(localStorage.getItem('srpski.sound')).toBe('off')
    await w.find(sound).setValue(true)
    expect(soundOn.value).toBe(true)
  })

  it('reflects a sound that was already off', () => {
    setSound(false)
    const w = mount(SoundSettings)
    expect((w.find(sound).element as HTMLInputElement).checked).toBe(false)
  })

  it('offers the vibration only inside the Mini App, not in a browser that merely has the SDK', () => {
    expect(mount(SoundSettings).find(haptics).exists()).toBe(false)
    sdk('')
    expect(mount(SoundSettings).find(haptics).exists()).toBe(false)
    inTelegram()
    expect(mount(SoundSettings).find(haptics).exists()).toBe(true)
  })

  it('switches the vibration off without touching the sound', async () => {
    inTelegram()
    const w = mount(SoundSettings)
    await w.find(haptics).setValue(false)
    expect(hapticsOn.value).toBe(false)
    expect(localStorage.getItem('srpski.haptics')).toBe('off')
    expect(soundOn.value).toBe(true)
  })
})
