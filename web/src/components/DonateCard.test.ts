import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import DonateCard from './DonateCard.vue'
import { useDonateModal, TRIBUTE_TELEGRAM_LINK } from '../lib/donateModal'
import { isTelegram, initData } from '../telegram'

vi.mock('../telegram', () => ({ isTelegram: vi.fn(), initData: vi.fn() }))

beforeEach(() => {
  const { open } = useDonateModal()
  open.value = false
  vi.mocked(isTelegram).mockReturnValue(false)
  vi.mocked(initData).mockReturnValue('')
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

  it('opens the modal, not the Telegram link, when Telegram\'s WebApp object exists but initData is empty', async () => {
    // window.Telegram.WebApp is defined even in a plain browser (the SDK
    // script is loaded unconditionally) — initData is the real signal for
    // an actual Telegram session, same as stores/session.ts.
    vi.mocked(isTelegram).mockReturnValue(true)
    vi.mocked(initData).mockReturnValue('')
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const { open } = useDonateModal()
    const w = mount(DonateCard)

    await w.find('[data-test="open-donate"]').trigger('click')

    expect(openSpy).not.toHaveBeenCalled()
    expect(open.value).toBe(true)
  })

  it('opens the Tribute Telegram link directly inside a real Telegram session, without the modal', async () => {
    vi.mocked(isTelegram).mockReturnValue(true)
    vi.mocked(initData).mockReturnValue('query_id=AA&user=%7B%7D&auth_date=1&hash=abc')
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const { open } = useDonateModal()
    const w = mount(DonateCard)

    await w.find('[data-test="open-donate"]').trigger('click')

    expect(openSpy).toHaveBeenCalledWith(TRIBUTE_TELEGRAM_LINK, '_blank')
    expect(open.value).toBe(false)
  })
})
