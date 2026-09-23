import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import DonateModal from './DonateModal.vue'
import { useDonateModal, TRIBUTE_TELEGRAM_LINK, TRIBUTE_WEB_LINK } from '../lib/donateModal'

function mountModal() {
  // Teleport's real target (document.body) is outside the wrapper's DOM tree;
  // stubbing it keeps the modal inline so it can be found/asserted on.
  return mount(DonateModal, { global: { stubs: { teleport: true } } })
}

beforeEach(() => {
  const { open } = useDonateModal()
  open.value = false
})

describe('DonateModal', () => {
  it('renders nothing until the shared state opens it', () => {
    const w = mountModal()
    expect(w.find('[data-test="donate-telegram"]').exists()).toBe(false)
  })

  it('links both buttons to the Tribute donation tier', () => {
    const { openModal } = useDonateModal()
    openModal()
    const w = mountModal()

    expect(w.find('[data-test="donate-telegram"]').attributes('href')).toBe(TRIBUTE_TELEGRAM_LINK)
    expect(w.find('[data-test="donate-web"]').attributes('href')).toBe(TRIBUTE_WEB_LINK)
  })

  it('closes when the close button is clicked', async () => {
    const { open, openModal } = useDonateModal()
    openModal()
    const w = mountModal()

    await w.find('[title="Закрыть"]').trigger('click')
    expect(open.value).toBe(false)
  })
})
