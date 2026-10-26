import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LevelChoiceModal from './LevelChoiceModal.vue'
import { api, ApiError } from '../api'
import { useTariffsStore } from '../stores/tariffs'
import { useWalletStore } from '../stores/wallet'
import { useCourseStore } from '../stores/course'
import { useLevelChoiceModal } from '../lib/levelChoiceModal'
import { enableTariffs } from '../test/tariffsFixture'
import type { LevelChoice, ShopItem } from '../types'

const push = vi.fn()
vi.mock('vue-router', () => ({ useRouter: () => ({ push }), RouterLink: { template: '<a><slot /></a>' } }))

enableAutoUnmount(afterEach)

const choices: LevelChoice[] = [
  { id: '3', title: 'Уровень 4 — Связная речь', lessons: 12 },
  { id: '4', title: 'Уровень 5 — Падежи в жизни', lessons: 18, after: 'Уровень 4 — Связная речь' },
]

function product(ref: string, description: string): ShopItem {
  return { id: Number(ref), kind: 'phase_unlock', ref, title: 'Фаза ' + ref, description, price: 230, price_effective: 230, owned: 0 }
}

function mountModal(over: { choices?: LevelChoice[]; credits?: number[] } = {}) {
  enableTariffs({ level: { price: 299, regular_price: 299, bonus: 1000, next: null, choices: over.choices ?? choices, credits: over.credits ?? [7] } })
  const wallet = useWalletStore()
  wallet.products = [
    product('3', 'Открывает уровень 4: связная речь, прошедшее время'),
    product('4', 'Открывает уровень 5: падежи, прилагательные'),
  ]
  useLevelChoiceModal().openFor()
  return mount(LevelChoiceModal, { global: { stubs: { teleport: true } } })
}

beforeEach(() => {
  setActivePinia(createPinia())
  push.mockClear()
  // after a choice the shop and the wallet are read again: keep that offline
  vi.spyOn(useWalletStore(), 'loadShop').mockResolvedValue(undefined)
  vi.spyOn(useWalletStore(), 'refresh').mockResolvedValue(undefined)
})
afterEach(() => {
  useLevelChoiceModal().closeModal()
  vi.restoreAllMocks()
})

// The list of levels scrolls inside the window and is drawn with a soft scrollbar of its own. On a phone the
// scrollbar is not visible at all, so the list also fades at the edge where there is more to see
// (jsdom has no layout, so the sizes are stubbed).
describe('LevelChoiceModal: the scrolling list', () => {
  const stub = (scrollHeight: number, clientHeight: number) => {
    vi.spyOn(Element.prototype, 'scrollHeight', 'get').mockReturnValue(scrollHeight)
    vi.spyOn(Element.prototype, 'clientHeight', 'get').mockReturnValue(clientHeight)
  }
  const list = (w: ReturnType<typeof mountModal>) => w.get('[data-test="choice-list"]')
  const scrollTo = async (w: ReturnType<typeof mountModal>, top: number) => {
    Object.defineProperty(list(w).element, 'scrollTop', { configurable: true, value: top })
    await list(w).trigger('scroll')
  }

  it('wears the soft scrollbar', () => {
    stub(300, 300)
    expect(list(mountModal()).classes()).toContain('soft-scroll')
  })

  it('fades at the bottom while there is more below, and not at all when everything fits', async () => {
    stub(600, 300)
    const long = mountModal()
    await flushPromises()
    expect(list(long).classes()).toContain('more-below')
    expect(list(long).classes()).not.toContain('more-above')
    long.unmount()

    vi.restoreAllMocks()
    vi.spyOn(useWalletStore(), 'loadShop').mockResolvedValue(undefined)
    vi.spyOn(useWalletStore(), 'refresh').mockResolvedValue(undefined)
    stub(300, 300)
    const short = mountModal()
    await flushPromises()
    expect(list(short).classes()).not.toContain('more-below')
    expect(list(short).classes()).not.toContain('more-above')
  })

  it('follows the scrolling: both edges in the middle, only the top once at the end', async () => {
    stub(600, 300)
    const w = mountModal()
    await flushPromises()
    await scrollTo(w, 150)
    expect(list(w).classes()).toEqual(expect.arrayContaining(['more-above', 'more-below']))
    await scrollTo(w, 300) // scrollTop + clientHeight = scrollHeight
    expect(list(w).classes()).toContain('more-above')
    expect(list(w).classes()).not.toContain('more-below')
    await scrollTo(w, 0)
    expect(list(w).classes()).not.toContain('more-above')
    expect(list(w).classes()).toContain('more-below')
  })
})

describe('LevelChoiceModal', () => {
  it('lists only the levels left to choose, in course order, with what is in them', () => {
    const w = mountModal()
    const cards = w.findAll('[data-test^="choice-card-"]')
    expect(cards.map((c) => c.attributes('data-test'))).toEqual(['choice-card-3', 'choice-card-4'])
    expect(cards[0].text()).toContain('Связная речь')
    expect(cards[0].text()).toContain('Связная речь, прошедшее время')
    expect(cards[0].text()).toMatch(/12\s+уроков/) // a no-break space joins the number and the word
    expect(cards[1].text()).toMatch(/18\s+уроков/)
  })

  it('marks the first level as the next one and warns gently about the others', () => {
    const w = mountModal()
    expect(w.find('[data-test="choice-card-3"]').text()).toContain('Следующий по порядку')
    expect(w.find('[data-test="choice-card-4"]').text()).toContain('Лучше после «Связная речь»')
    expect(w.find('[data-test="choice-card-4"]').text()).not.toContain('Следующий по порядку')
  })

  it('cannot be confirmed before a level is picked, and picking is a free choice of any level', async () => {
    const w = mountModal()
    const confirm = w.find('[data-test="choice-confirm"]')
    expect(confirm.attributes('disabled')).toBeDefined()
    await w.find('[data-test="choice-card-4"] input').setValue(true)
    expect(w.find('[data-test="choice-confirm"]').attributes('disabled')).toBeUndefined()
  })

  it('spends the waiting paid level on the picked one and shows what opened', async () => {
    const spend = vi
      .spyOn(api, 'chooseLevel')
      .mockResolvedValue({ id: 7, tariff: 'level', status: 'succeeded', phase_title: 'Уровень 5 — Падежи в жизни', bonus: 1000 })
    vi.spyOn(api, 'tariffs').mockResolvedValue({
      payments_ready: true,
      has_paid_level: true,
      level: { price: 299, regular_price: 299, next: null, choices: [choices[0]], credits: [] },
      year: { price: 1000, old_price: 2000 },
    })
    const course = vi.spyOn(useCourseStore(), 'load').mockResolvedValue(undefined)
    const w = mountModal()
    await w.find('[data-test="choice-card-4"] input').setValue(true)
    await w.find('[data-test="choice-confirm"]').trigger('click')
    await flushPromises()

    expect(spend).toHaveBeenCalledWith(7, '4')
    expect(course).toHaveBeenCalledWith(true) // the course map shows the level as open
    const done = w.find('[data-test="choice-done"]')
    expect(done.exists()).toBe(true)
    expect(done.text()).toContain('Падежи в жизни')
    expect(w.find('[data-test="choice-confirm"]').exists()).toBe(false)
    expect(w.find('[data-test="choice-thanks"] img.heart').exists()).toBe(true)
  })

  it('takes the learner to the course from the thank-you', async () => {
    vi.spyOn(api, 'chooseLevel').mockResolvedValue({ id: 7, tariff: 'level', status: 'succeeded', phase_title: 'Уровень 4 — Связная речь' })
    vi.spyOn(api, 'tariffs').mockResolvedValue(enableTariffs().info!)
    vi.spyOn(useCourseStore(), 'load').mockResolvedValue(undefined)
    const w = mountModal()
    await w.find('[data-test="choice-card-3"] input').setValue(true)
    await w.find('[data-test="choice-confirm"]').trigger('click')
    await flushPromises()
    await w.find('[data-test="choice-to-course"]').trigger('click')
    expect(push).toHaveBeenCalledWith('/course')
    expect(useLevelChoiceModal().open.value).toBe(false)
  })

  it('says so in words when the credit was already spent, and stays open', async () => {
    vi.spyOn(api, 'chooseLevel').mockRejectedValue(new ApiError(409, 'already_chosen'))
    const w = mountModal()
    await w.find('[data-test="choice-card-3"] input').setValue(true)
    await w.find('[data-test="choice-confirm"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="choice-error"]').text()).toContain('уже потрачен')
    expect(useLevelChoiceModal().open.value).toBe(true)
  })

  it('keeps the paid level when it is closed without choosing', async () => {
    const spend = vi.spyOn(api, 'chooseLevel')
    const w = mountModal()
    await w.find('[data-test="choice-close"]').trigger('click')
    expect(spend).not.toHaveBeenCalled()
    expect(useLevelChoiceModal().open.value).toBe(false)
    expect(useTariffsStore().credits).toEqual([7])
  })

  it('tells the learner to wait when every level is already open', () => {
    const w = mountModal({ choices: [] })
    expect(w.find('[data-test="choice-card-3"]').exists()).toBe(false)
    expect(w.find('[data-test="choice-confirm"]').exists()).toBe(false)
    expect(w.find('[data-test="choice-waiting"]').text()).toMatch(/оплаченный уровень/i)
  })
})
