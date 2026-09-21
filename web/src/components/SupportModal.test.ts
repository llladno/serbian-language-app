import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import SupportModal from './SupportModal.vue'
import { useSupportModal } from '../lib/supportModal'
import { api, ApiError } from '../api'

function mountModal() {
  // Teleport's real target (document.body) is outside the wrapper's DOM tree;
  // stubbing it keeps the modal inline so it can be found/asserted on.
  return mount(SupportModal, { global: { stubs: { teleport: true } } })
}

beforeEach(() => {
  const { open, message, error, sent } = useSupportModal()
  open.value = false
  message.value = ''
  error.value = null
  sent.value = false
})
afterEach(() => vi.restoreAllMocks())

describe('SupportModal', () => {
  it('renders nothing until the shared state opens it', () => {
    const w = mountModal()
    expect(w.find('textarea').exists()).toBe(false)
  })

  it('sends a trimmed message and shows the thank-you state', async () => {
    const send = vi.spyOn(api, 'sendSupportMessage').mockResolvedValue({ status: 'ok' })
    const { openModal, message } = useSupportModal()
    openModal()
    const w = mountModal()

    message.value = '  Не работает кнопка на уроке 5  '
    await w.vm.$nextTick()
    await w.find('[data-test="send-support"]').trigger('click')
    await flushPromises()

    expect(send).toHaveBeenCalledWith('Не работает кнопка на уроке 5')
    expect(w.text()).toContain('Спасибо')
  })

  it('disables sending a blank message', async () => {
    const send = vi.spyOn(api, 'sendSupportMessage').mockResolvedValue({ status: 'ok' })
    const { openModal } = useSupportModal()
    openModal()
    const w = mountModal()

    const button = w.find('[data-test="send-support"]')
    expect((button.element as HTMLButtonElement).disabled).toBe(true)
    await button.trigger('click')
    expect(send).not.toHaveBeenCalled()
  })

  it('shows an error message when sending fails', async () => {
    vi.spyOn(api, 'sendSupportMessage').mockRejectedValue(new ApiError(500, 'internal error'))
    const { openModal, message } = useSupportModal()
    openModal()
    message.value = 'Привет'
    const w = mountModal()
    await w.vm.$nextTick()

    await w.find('[data-test="send-support"]').trigger('click')
    await flushPromises()

    expect(w.text()).toContain('Что-то пошло не так')
    expect(w.find('textarea').exists()).toBe(true) // form stays open on failure
  })

  it('links to the support Telegram account', () => {
    const { openModal } = useSupportModal()
    openModal()
    const w = mountModal()
    const link = w.find('[data-test="support-telegram"]')
    expect(link.attributes('href')).toBe('https://t.me/ucimosupport')
  })
})
