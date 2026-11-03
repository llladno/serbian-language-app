import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ScrubBar from './ScrubBar.vue'

function bar(props: Record<string, unknown> = {}) {
  const w = mount(ScrubBar, { props, slots: { default: '<div style="height:6px"></div>' }, attachTo: document.body })
  // jsdom has no layout: a strip 200 px wide that starts at x = 100.
  w.element.getBoundingClientRect = () => ({ left: 100, width: 200, right: 300, top: 0, bottom: 10, height: 10, x: 100, y: 0, toJSON() {} }) as DOMRect
  return w
}
// jsdom has no PointerEvent: a MouseEvent carries the same clientX under the pointer events' names.
async function fire(w: ReturnType<typeof bar>, type: string, x = 0) {
  w.element.dispatchEvent(new MouseEvent(type, { clientX: x, bubbles: true }))
  await w.vm.$nextTick()
}

describe('ScrubBar', () => {
  it('reports the place on press, on every move and on release, as a share of its width', async () => {
    const w = bar()
    await fire(w, 'pointerdown', 150)
    await fire(w, 'pointermove', 200)
    await fire(w, 'pointerup', 250)
    expect(w.emitted('move')).toEqual([[0.25], [0.5]])
    expect(w.emitted('commit')).toEqual([[0.75]])
  })

  it('a tap is a press and a release in the same place', async () => {
    const w = bar()
    await fire(w, 'pointerdown', 180)
    await fire(w, 'pointerup', 180)
    expect(w.emitted('move')).toEqual([[0.4]])
    expect(w.emitted('commit')).toEqual([[0.4]])
  })

  it('stays inside 0..1 when the finger runs past the ends', async () => {
    const w = bar()
    await fire(w, 'pointerdown', 40)
    await fire(w, 'pointerup', 900)
    expect(w.emitted('move')).toEqual([[0]])
    expect(w.emitted('commit')).toEqual([[1]])
  })

  it('ignores a move that was not pressed first, and everything when disabled', async () => {
    const free = bar()
    await fire(free, 'pointermove', 150)
    expect(free.emitted('move')).toBeUndefined()
    const off = bar({ disabled: true })
    await fire(off, 'pointerdown', 150)
    await fire(off, 'pointerup', 150)
    expect(off.emitted('move')).toBeUndefined()
    expect(off.emitted('commit')).toBeUndefined()
  })

  it('a cancelled touch still lets go where it was last', async () => {
    const w = bar()
    await fire(w, 'pointerdown', 150)
    await fire(w, 'pointermove', 200)
    await fire(w, 'pointercancel')
    expect(w.emitted('commit')).toEqual([[0.5]])
  })
})
