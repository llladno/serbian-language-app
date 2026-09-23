import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import DonateCard from './DonateCard.vue'
import { useDonateModal, TRIBUTE_TELEGRAM_LINK } from '../lib/donateModal'
import { isTelegram } from '../telegram'

vi.mock('../telegram', () => ({ isTelegram: vi.fn() }))

beforeEach(() => {
  const { open } = useDonateModal()
  open.value = false
  vi.mocked(isTelegram).mockReturnValue(false)
})
afterEach(() => vi.restoreAllMocks())

describe('DonateCard', () => {
  it('opens the shared donate modal when clicked outside Telegram', async () => {
    const { open } = useDonateModal()
    const w = mount(DonateCard)

    expect(open.value).toBe(false)
    await w.find('[data-test="open-donate"]').trigger('click')
    expect(open.value).toBe(true)
  })

  it('opens the Tribute Telegram link directly inside Telegram, without the modal', async () => {
    vi.mocked(isTelegram).mockReturnValue(true)
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const { open } = useDonateModal()
    const w = mount(DonateCard)

    await w.find('[data-test="open-donate"]').trigger('click')

    expect(openSpy).toHaveBeenCalledWith(TRIBUTE_TELEGRAM_LINK, '_blank')
    expect(open.value).toBe(false)
  })
})
