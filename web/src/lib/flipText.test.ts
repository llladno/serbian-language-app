import { describe, it, expect, vi, afterEach } from 'vitest'
import { flipText } from './flipText'

afterEach(() => {
  vi.unstubAllGlobals()
  delete (Element.prototype as { animate?: unknown }).animate
})

function withAnimate() {
  const animate = vi.fn()
  Element.prototype.animate = animate as never
  return animate
}

describe('flipText', () => {
  it('fades the new text in on every element it is given', () => {
    const animate = withAnimate()
    flipText([document.createElement('div'), document.createElement('p')])
    expect(animate).toHaveBeenCalledTimes(2)
  })

  it('animates opacity only: a transform or filter would pull the fixed bottom bar out of place', () => {
    const animate = withAnimate()
    flipText([document.createElement('div')])
    const frames = animate.mock.calls[0][0] as Record<string, unknown>[]
    const properties = new Set(frames.flatMap((f) => Object.keys(f)))
    expect([...properties]).toEqual(['opacity'])
    expect(frames[0].opacity).toBeLessThan(1)
    expect(frames[frames.length - 1].opacity).toBe(1)
  })

  it('stays short', () => {
    const animate = withAnimate()
    flipText([document.createElement('div')])
    expect((animate.mock.calls[0][1] as { duration: number }).duration).toBeLessThanOrEqual(400)
  })

  it('does nothing for someone who asked for less motion', () => {
    const animate = withAnimate()
    vi.stubGlobal('matchMedia', (q: string) => ({ matches: q.includes('reduce') }))
    flipText([document.createElement('div')])
    expect(animate).not.toHaveBeenCalled()
  })

  it('skips elements that are not on screen and browsers without animations', () => {
    flipText([null, undefined, document.createElement('div')]) // jsdom has no Element.animate
    const animate = withAnimate()
    flipText([null, undefined])
    expect(animate).not.toHaveBeenCalled()
  })
})
