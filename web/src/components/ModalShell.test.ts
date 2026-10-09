import { describe, it, expect, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ModalShell from './ModalShell.vue'
import RewardModal from './RewardModal.vue'
import { useRewardModal } from '../lib/rewardModal'

// Everything mounted is unmounted before the page is wiped: tearing the body
// down under a live teleport makes Vue write into nodes that are gone.
const mounted: { unmount(): void }[] = []
afterEach(() => {
  while (mounted.length) mounted.pop()!.unmount()
  document.body.innerHTML = ''
  useRewardModal().closeModal()
})

const shell = (props: Record<string, unknown> = {}) => {
  const w = mount(ModalShell, { props: { open: true, ...props }, slots: { default: '<p data-test="content">hi</p>' }, attachTo: document.body })
  mounted.push(w)
  return w
}

describe('ModalShell', () => {
  it('puts the panel in the middle of the screen on every size', () => {
    shell()
    const layer = document.querySelector('.modal-panel')!.parentElement!.parentElement!
    // a box at least as tall as the screen that centres its one child: not the
    // top-on-phones, middle-on-desktops layout the first modals had
    expect(layer.className).toContain('min-h-full')
    expect(layer.className).toContain('items-center')
    expect(layer.className).toContain('justify-center')
    expect(layer.className).not.toContain('items-start')
  })

  it('lets a tall panel scroll rather than cutting its top off', () => {
    shell()
    expect(document.querySelector('.modal-panel')!.closest('.fixed')!.className).toContain('overflow-y-auto')
  })

  it('renders nothing while closed and tells its owner when the backdrop is tapped', async () => {
    const w = shell({ open: false })
    expect(document.querySelector('.modal-panel')).toBeNull()
    await w.setProps({ open: true })
    const layer = document.querySelector('.modal-panel')!.parentElement!.parentElement as HTMLElement
    layer.click()
    expect(w.emitted('close')).toHaveLength(1)
  })

  it('does not close when the panel itself is tapped', () => {
    const w = shell()
    ;(document.querySelector('[data-test="content"]') as HTMLElement).click()
    expect(w.emitted('close')).toBeUndefined()
  })
})

describe('RewardModal', () => {
  const open = () => {
    setActivePinia(createPinia())
    useRewardModal().celebrate(5, 'Задание')
    const w = mount(RewardModal, { attachTo: document.body })
    mounted.push(w)
    return w
  }

  it('sits in the shared centred frame', () => {
    open()
    expect(document.querySelector('.modal-panel')!.parentElement!.parentElement!.className).toContain('items-center')
  })

  it('lights the seeds from behind, not the panel', () => {
    open()
    const glow = document.querySelector('.glow')!
    // the glow is in the picture's own box, the same one as the seeds and
    // the sparkles, and comes first so they are drawn over it
    const box = glow.parentElement!
    expect(box.querySelector('img.pile')).not.toBeNull()
    expect(box.firstElementChild).toBe(glow)
    expect(document.querySelector('.modal-panel')!.parentElement!.querySelector(':scope > .glow')).toBeNull()
  })

  it('spreads a few sparkles over the full width, at the height of the seeds', () => {
    // a random scatter fails this one time in a few; a band by columns never does
    for (let run = 0; run < 25; run++) {
      const w = open()
      const spots = [...document.querySelectorAll('.burst img')].map((i) => ({
        x: parseFloat((i as HTMLElement).style.left),
        y: parseFloat((i as HTMLElement).style.top),
      }))
      expect(spots.length).toBeLessThanOrEqual(10)
      expect(spots.length).toBeGreaterThanOrEqual(6)
      expect(Math.min(...spots.map((s) => s.x))).toBeLessThan(12) // reaches the left edge
      expect(Math.max(...spots.map((s) => s.x))).toBeGreaterThan(88) // and the right one
      expect(spots.filter((s) => s.x < 50).length).toBeGreaterThanOrEqual(3) // neither side is bare
      expect(spots.filter((s) => s.x >= 50).length).toBeGreaterThanOrEqual(3)
      for (const s of spots) {
        expect(s.y).toBeGreaterThanOrEqual(24) // no higher or lower than the seeds
        expect(s.y).toBeLessThanOrEqual(76)
      }
      w.unmount()
      mounted.pop()
      useRewardModal().closeModal()
    }
  })
})
