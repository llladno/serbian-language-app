import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import RepeatLimitModal from './RepeatLimitModal.vue'
import { enableTariffs } from '../test/tariffsFixture'
import { useSessionStore } from '../stores/session'
import { api } from '../api'

const push = vi.fn()
vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))

const mountModal = (open = true) =>
  mount(RepeatLimitModal, { props: { open }, global: { stubs: { teleport: true } } })

beforeEach(() => {
  setActivePinia(createPinia())
  push.mockClear()
})

describe('RepeatLimitModal', () => {
  it('renders nothing while closed', () => {
    expect(mountModal(false).find('[data-test="repeat-limit"]').exists()).toBe(false)
  })

  it('says one batch a day is free and offers a subscription for repeats without limits', () => {
    enableTariffs()
    const text = mountModal().text()
    expect(text).toContain('Серия на сегодня пройдена')
    expect(text).toContain('Можете оформить подписку, чтобы использовать повторения без ограничений')
  })

  it('takes the learner to the tariffs and closes itself', async () => {
    enableTariffs()
    const w = mountModal()
    const go = w.find('[data-test="repeat-limit-tariffs"]')
    expect(go.text()).toBe('Повторять без ограничений')
    await go.trigger('click')
    expect(push).toHaveBeenCalledWith('/tariffs')
    expect(w.emitted('close')).toHaveLength(1)
  })

  it('keeps the tariffs button active for every account', async () => {
    useSessionStore().user = null
    const w = mountModal()
    const go = w.find('[data-test="repeat-limit-tariffs"]')
    expect(go.exists()).toBe(true)
    expect(go.attributes('disabled')).toBeUndefined()
    await go.trigger('click')
    expect(push).toHaveBeenCalledWith('/tariffs')
  })

  it('has a full-width picture on top', () => {
    const w = mountModal()
    const pic = w.get('[data-test="repeat-limit-scene"]')
    expect(pic.element.tagName).toBe('IMG')
    expect(pic.attributes('alt')).toBe('')
    expect(pic.attributes('src')).toContain('repeat-limit-scene')
  })

  it('closes from the plain button', async () => {
    const w = mountModal()
    await w.find('[data-test="repeat-limit-close"]').trigger('click')
    expect(w.emitted('close')).toHaveLength(1)
  })
})

describe('RepeatLimitModal reports what it shows', () => {
  it('reports a showing when it opens, and a press when the learner goes on to the tariffs', async () => {
    const spy = vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
    enableTariffs()
    const w = mountModal()
    expect(spy).toHaveBeenCalledWith('view', 'repeat_limit')
    await w.get('[data-test="repeat-limit-tariffs"]').trigger('click')
    expect(spy).toHaveBeenLastCalledWith('click', 'repeat_limit')
  })

  it('reports a showing each time it opens again, and nothing while it is closed', async () => {
    const spy = vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
    enableTariffs()
    const w = mountModal(false)
    expect(spy).not.toHaveBeenCalled()
    await w.setProps({ open: true })
    await w.setProps({ open: false })
    await w.setProps({ open: true })
    expect(spy.mock.calls.filter((c) => c[0] === 'view')).toHaveLength(2)
  })
})
