import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { mount as rawMount, flushPromises } from '@vue/test-utils'
import ArcGauge from './ArcGauge.vue'

// The drawing itself is checked by eye; what is pinned here is the value shown,
// the clamping and when the gauge says it has landed.
beforeEach(() => {
  // reduced motion: the final frame at once
  vi.stubGlobal('matchMedia', (q: string) => ({ matches: /reduce/.test(q), media: q, addEventListener() {}, removeEventListener() {} }))
})
afterEach(() => vi.unstubAllGlobals())

// The final value is set in onMounted and reaches the DOM a tick later.
async function mount(...args: Parameters<typeof rawMount<typeof ArcGauge>>) {
  const w = rawMount(...args)
  await flushPromises()
  return w
}

describe('ArcGauge', () => {
  it('shows the percentage and says it has landed', async () => {
    const w = await mount(ArcGauge, { props: { percent: 87 } })
    expect(w.find('.num').text()).toContain('87')
    expect(w.attributes('aria-label')).toContain('87%')
    expect(w.emitted('landed')).toHaveLength(1)
  })

  it('draws as much of the stroke as the percentage', async () => {
    const w = await mount(ArcGauge, { props: { percent: 40 } })
    expect(w.find('.fill').attributes('stroke-dasharray')).toBe('40 100')
  })

  it('keeps the figure between 0 and 100', async () => {
    expect((await mount(ArcGauge, { props: { percent: 130 } })).find('.num').text()).toContain('100')
    expect((await mount(ArcGauge, { props: { percent: -5 } })).find('.num').text()).toContain('0')
  })

  it('draws no stroke at all for zero', async () => {
    const w = await mount(ArcGauge, { props: { percent: 0 } })
    expect(w.find('.fill').attributes('style')).toContain('display: none')
  })

  it('turns gold only when asked to', async () => {
    expect((await mount(ArcGauge, { props: { percent: 100, gold: true } })).classes()).toContain('gold')
    expect((await mount(ArcGauge, { props: { percent: 100 } })).classes()).not.toContain('gold')
  })

  // Behind the arc, small, and only for the two best bands.
  async function sparkles(props: Record<string, unknown>) {
    const w = await mount(ArcGauge, { props: props as never })
    const burst = w.findComponent({ name: 'SparkleBurst' })
    return burst.exists() ? { mode: burst.props('mode'), count: burst.findAll('img').length, base: burst.props('base') } : null
  }

  it('makes a full arc glitter', async () => {
    const got = await sparkles({ percent: 100 })
    expect(got?.mode).toBe('twinkle')
    expect(got!.count).toBeGreaterThan(5)
    expect(got!.base).toBe(2) // half the size of the reward modal's
  })

  it('lets exactly five sparkles float up from an arc in the eighties and nineties', async () => {
    for (const percent of [80, 99]) {
      const got = await sparkles({ percent })
      expect(got).toMatchObject({ mode: 'rise', count: 5 })
    }
  })

  it('leaves a lower arc plain, and any arc when told to', async () => {
    expect(await sparkles({ percent: 79 })).toBeNull()
    expect(await sparkles({ percent: 0 })).toBeNull()
    expect(await sparkles({ percent: 100, sparkle: false })).toBeNull()
  })

  it('draws no seed on the arc', async () => {
    const w = await mount(ArcGauge, { props: { percent: 60 } })
    expect(w.find('image').exists()).toBe(false)
  })

  it('has a glow only on a full gold arc, and it is at full strength once the arc is', async () => {
    const gold = await mount(ArcGauge, { props: { percent: 100, gold: true } })
    expect(gold.find('.halo').exists()).toBe(true)
    expect(parseFloat(gold.find('.halo').attributes('style')!.match(/opacity: ([\d.]+)/)![1])).toBeCloseTo(0.8, 2)
    expect((await mount(ArcGauge, { props: { percent: 100 } })).find('.halo').exists()).toBe(false)
  })
})

// The glow must grow with the arc, not switch on when it lands: it follows the
// same clock, so no frame may jump by much.
describe('ArcGauge glow while drawing', () => {
  beforeEach(() => {
    vi.stubGlobal('matchMedia', (q: string) => ({ matches: false, media: q, addEventListener() {}, removeEventListener() {} }))
    vi.useFakeTimers({ toFake: ['requestAnimationFrame', 'cancelAnimationFrame', 'setTimeout', 'clearTimeout', 'performance'] })
  })
  afterEach(() => vi.useRealTimers())

  it('swells smoothly from nothing to full', async () => {
    const w = rawMount(ArcGauge, { props: { percent: 100, gold: true, delay: 0 } })
    const seen: number[] = []
    for (let ms = 0; ms < 3500; ms += 16) {
      vi.advanceTimersByTime(16)
      await flushPromises()
      const style = w.find('.halo').attributes('style') ?? ''
      const m = style.match(/opacity: ([\d.]+)/)
      seen.push(m ? parseFloat(m[1]) : 0)
    }
    expect(seen[0]).toBeLessThan(0.05) // not already lit
    expect(seen[seen.length - 1]).toBeCloseTo(0.8, 2) // and lit at the end
    const steps = seen.slice(1).map((v, i) => v - seen[i])
    expect(Math.min(...steps)).toBeGreaterThanOrEqual(-1e-9) // never dims on the way
    expect(Math.max(...steps)).toBeLessThan(0.05) // no frame jumps
  })
})
