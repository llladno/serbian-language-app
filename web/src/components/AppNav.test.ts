import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AppNav from './AppNav.vue'
import { useSupportModal } from '../lib/supportModal'

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/profile', meta: {} }),
  RouterLink: { template: '<a><slot /></a>' },
}))

beforeEach(() => {
  const { open, message, error, sent } = useSupportModal()
  open.value = false
  message.value = ''
  error.value = null
  sent.value = false
})

function mountNav() {
  // Teleport's real target (document.body) is outside the wrapper's DOM tree;
  // stubbing it keeps the modal inline so it can be found/asserted on.
  return mount(AppNav, { props: { name: 'Гриша' }, global: { stubs: { teleport: true } } })
}

describe('AppNav', () => {
  it('opens the support modal from the headphones icon', async () => {
    const w = mountNav()
    expect(w.find('textarea').exists()).toBe(false)

    await w.find('[data-test="open-support"]').trigger('click')

    expect(w.find('textarea').exists()).toBe(true)
  })
})
