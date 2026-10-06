import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import AnimatedNumber from './AnimatedNumber.vue'

function stubReducedMotion(reduce: boolean) {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: reduce && query.includes('prefers-reduced-motion'),
    media: query,
    addEventListener: () => {},
    removeEventListener: () => {},
  }))
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('AnimatedNumber', () => {
  it('counts to the new value and stops exactly on it', async () => {
    stubReducedMotion(false)
    const w = mount(AnimatedNumber, { props: { value: 100, duration: 60 } })
    await w.setProps({ value: 160 })

    await vi.waitFor(() => expect(w.text()).toBe('160'), { timeout: 3000 })
  })

  it('jumps straight to the value when motion is unwelcome', async () => {
    stubReducedMotion(true)
    const w = mount(AnimatedNumber, { props: { value: 100 } })

    await w.setProps({ value: 160 })
    await flushPromises()

    expect(w.text()).toBe('160')
  })

  it('counts through the format, so the word follows the number', async () => {
    stubReducedMotion(true)
    const w = mount(AnimatedNumber, {
      props: { value: 1, format: (n: number) => `${n} пёрышек` },
    })

    await w.setProps({ value: 5 })
    await flushPromises()

    expect(w.text()).toBe('5 пёрышек')
  })
})
