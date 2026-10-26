import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createRouter, createMemoryHistory, RouterView } from 'vue-router'
import { defineComponent, h } from 'vue'
import RepeatView from './RepeatView.vue'
import { api } from '../api'

// The real router again: the cross must really leave the screen, in every state it can be in.
const Page = (name: string) => defineComponent({ render: () => h('div', { 'data-test': name }, name) })
function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/repeat', name: 'repeat', component: RepeatView },
      { path: '/profile', name: 'profile', component: Page('profile') },
      { path: '/course', name: 'course', component: Page('course') },
    ],
  })
}
const item = { lesson: '01', exercise: { id: 'a', type: 'choice' as const, prompt: 'Первое', options: ['Da', 'Ne'] } }

beforeEach(() => {
  setActivePinia(createPinia())
  localStorage.clear()
})
afterEach(() => vi.restoreAllMocks())

describe('closing «Повторить» with the cross', () => {
  it('goes back to the profile while an exercise is on screen', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue({ total: 1, items: [item], unlimited: false, session: { size: 1, remaining: 1 } })
    const router = makeRouter()
    router.push('/repeat')
    await router.isReady()
    const w = mount(defineComponent({ render: () => h(RouterView) }), { global: { plugins: [router], stubs: { teleport: true } } })
    await flushPromises()
    expect(w.text()).toContain('Первое')

    await w.get('button[aria-label="Закрыть"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/profile')
    expect(w.find('[data-test="profile"]').exists()).toBe(true)
  })

  it('the course button of the empty screen goes to the course, not to the profile', async () => {
    vi.spyOn(api, 'repeat').mockResolvedValue({ total: 0, items: [], unlimited: false, session: { size: 0, remaining: 0 } })
    const router = makeRouter()
    router.push('/repeat')
    await router.isReady()
    const w = mount(defineComponent({ render: () => h(RouterView) }), { global: { plugins: [router], stubs: { teleport: true } } })
    await flushPromises()

    await w.findAll('button').find((b) => /К курсу/.test(b.text()))!.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/course')
  })
})
