import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import StoriesView from './StoriesView.vue'
import { api, ApiError } from '../api'
import { useSessionStore } from '../stores/session'
import { trackTariff } from '../lib/tariffEvents'
import type { StoryListItem } from '../types'

const push = vi.fn()
vi.mock('../lib/tariffEvents', () => ({ trackTariff: vi.fn() }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))
vi.mock('../api', async (orig) => {
  const real = await orig<typeof import('../api')>()
  return {
    ...real,
    api: {
      ...real.api,
      stories: vi.fn(),
      buyStory: vi.fn(),
      storyEvent: vi.fn().mockResolvedValue(undefined),
      wallet: vi.fn().mockResolvedValue({ balance: 6, currency: 'зёрнышко' }),
      tariffs: vi.fn(),
      course: vi.fn(),
    },
  }
})

const demo: StoryListItem = {
  slug: 'demo', title_sr: 'Drago mi je', title_ru: 'Приятно познакомиться', tier: 'short', is_demo: true, cover: '/c.webp',
  duration_ms: 30000, phrases: 14, recommended_lesson: '', difficulty: 2, owned: true, price: 0, reward: 0, pass_reached: 0, completed: false, quiz_score: 0, quiz_total: 0,
}
const paid: StoryListItem = { ...demo, slug: 'paid', title_sr: 'Pekara', is_demo: false, owned: false, price: 10, reward: 0 }

function mountView() {
  return mount(StoriesView, { global: { stubs: { Teleport: true, Transition: false } } })
}

beforeEach(() => {
  setActivePinia(createPinia())
  push.mockReset()
  vi.mocked(api.stories).mockResolvedValue({ items: [demo, paid] })
  vi.mocked(api.buyStory).mockReset()
  vi.mocked(api.wallet).mockResolvedValue({ balance: 6, currency: 'зёрнышко' } as never)
  vi.mocked(api.course).mockResolvedValue({
    title: 'Курс',
    phases: [{ id: '1', title: 'Уровень 1', lessons: ['02d', '04b'] }],
    lessons: [
      { id: '02d', title: 'Как дела? (Kako si?)', subtitle: '', planned: false, status: 'not_started' },
      { id: '04b', title: 'Быть и откуда я (Biti)', subtitle: '', planned: false, status: 'not_started' },
    ],
  } as never)
  try {
    localStorage.clear()
  } catch {
    /* no storage in this environment */
  }
})

describe('StoriesView', () => {
  it('lists the stories: the demo as free, a paid one with its price', async () => {
    const w = mountView()
    await flushPromises()
    const badges = w.findAll('[data-test="story-badge"]').map((b) => b.text())
    expect(badges[0]).toBe('Бесплатно')
    expect(badges[1]).toContain('10')
  })

  it('shows the whole time with the questions on a card, no lock and no «три прохода»', async () => {
    const w = mountView()
    await flushPromises()
    const cards = w.findAll('[data-test="story-card"]')
    expect(cards[1].text()).toContain('≈ 30 сек с заданиями')
    expect(cards[1].text()).not.toContain('три прохода')
    expect(w.find('[data-test="story-badge"] svg.lucide-lock').exists()).toBe(false)
    expect(w.text()).toContain('учат понимать живую речь на слух')
  })

  it('offers the level with its gift when the зёрнышки are short and tariffs are open: the offer panel, the price, the gold button', async () => {
    useSessionStore().user = { features: { tariffs: true } } as never
    vi.mocked(api.tariffs).mockResolvedValue({
      level: { bonus: 1000, price: 199, regular_price: 299, next: '2', available_from: null },
      year: {},
    } as never)
    const w = mountView()
    await flushPromises()
    await w.findAll('[data-test="story-card"]')[1].trigger('click')
    await flushPromises()
    expect(w.find('[data-test="story-buy"]').exists()).toBe(false)
    expect(w.find('[data-test="story-short"]').exists()).toBe(true)
    expect(w.find('[data-test="story-offer-title"]').text().replace(/\u00a0/g, ' ')).toContain('и уровень в подарок')
    const fine = w.find('[data-test="story-price-fine"]').text().replace(/\u00a0/g, ' ')
    expect(fine).toContain('199 ₽')
    expect(fine).toContain('299 ₽')
    const pay = w.find('[data-test="story-tariff"]')
    expect(pay.text()).toBe('Получить зёрнышки')
    expect(pay.classes()).toEqual(expect.arrayContaining(['btn', 'btn-seed']))
    expect(trackTariff).toHaveBeenCalledWith('view', 'story_modal')
    await pay.trigger('click')
    expect(trackTariff).toHaveBeenCalledWith('click', 'story_modal')
    expect(push).toHaveBeenCalledWith('/tariffs')
  })

  it('falls back to the plain window while a monthly limit holds a level back', async () => {
    useSessionStore().user = { features: { tariffs: true } } as never
    const later = new Date(Date.now() + 5 * 86_400_000).toISOString()
    vi.mocked(api.tariffs).mockResolvedValue({
      level: { bonus: 1000, price: 199, regular_price: 299, next: '2', available_from: later },
      year: {},
    } as never)
    const w = mountView()
    await flushPromises()
    await w.findAll('[data-test="story-card"]')[1].trigger('click')
    await flushPromises()
    expect(w.find('[data-test="story-offer"]').exists()).toBe(false)
    expect(w.find('[data-test="story-buy"]').attributes('disabled')).toBeDefined()
  })

  it('puts a difficulty plate on every card and filters by it with the chips on the right of the title', async () => {
    const easy: StoryListItem = { ...paid, slug: 'easy', title_sr: 'Dobro veče', difficulty: 1 }
    const hard: StoryListItem = { ...paid, slug: 'hard', title_sr: 'Teška', difficulty: 3 }
    vi.mocked(api.stories).mockResolvedValue({ items: [easy, demo, hard] })
    const w = mountView()
    await flushPromises()
    expect(w.findAll('[data-test="story-level"]').map((p) => p.text())).toEqual(['Лёгкая', 'Средняя', 'Сложная'])
    expect(w.findAll('[data-test="stories-filter"] button').map((b) => b.text())).toEqual(['Все', 'Лёгкие', 'Средние', 'Сложные'])
    expect(w.findAll('[data-test="story-card"]')).toHaveLength(3)

    await w.find('[data-test="stories-filter-3"]').trigger('click')
    expect(w.find('[data-test="stories-filter-3"]').attributes('aria-pressed')).toBe('true')
    const cards = w.findAll('[data-test="story-card"]')
    expect(cards).toHaveLength(1)
    expect(cards[0].text()).toContain('Teška')

    await w.find('[data-test="stories-filter-1"]').trigger('click')
    expect(w.findAll('[data-test="story-card"]')).toHaveLength(1)
    expect(w.find('[data-test="story-card"]').text()).toContain('Dobro veče')

    await w.find('[data-test="stories-filter-0"]').trigger('click')
    expect(w.findAll('[data-test="story-card"]')).toHaveLength(3)
  })

  it('says so when no story has the chosen difficulty yet, and counts a story without one as medium', async () => {
    vi.mocked(api.stories).mockResolvedValue({ items: [{ ...demo, difficulty: 1 }, { ...paid, difficulty: 0 }] })
    const w = mountView()
    await flushPromises()
    expect(w.findAll('[data-test="story-level"]').map((p) => p.text())).toEqual(['Лёгкая', 'Средняя'])
    await w.find('[data-test="stories-filter-3"]').trigger('click')
    expect(w.find('[data-test="stories-filter-empty"]').exists()).toBe(true)
    expect(w.find('[data-test="story-card"]').exists()).toBe(false)
  })

  describe('the warning before a story whose lessons are not finished', () => {
    const ownedRec: StoryListItem = { ...demo, recommended_lesson: '04b' }
    const paidRec: StoryListItem = { ...paid, recommended_lesson: '04b' }
    const press = async (w: ReturnType<typeof mountView>, i: number) => {
      await w.findAll('[data-test="story-card"]')[i].trigger('click')
      await flushPromises()
    }

    it('warns an owned story and says how many lessons are left', async () => {
      vi.mocked(api.stories).mockResolvedValue({ items: [ownedRec] })
      const w = mountView()
      await flushPromises()
      await press(w, 0)
      expect(push).not.toHaveBeenCalled()
      expect(w.find('[data-test="story-warn-text"]').text()).toContain('2 урока')
      expect(w.find('[data-test="story-warn-next"]').exists()).toBe(false)
      expect(api.storyEvent).toHaveBeenCalledWith('demo', 'lessons_warn_view')
    })

    it('«Пройти урок» goes to the first unfinished lesson', async () => {
      vi.mocked(api.stories).mockResolvedValue({ items: [ownedRec] })
      const w = mountView()
      await flushPromises()
      await press(w, 0)
      await w.find('[data-test="story-warn-lesson"]').trigger('click')
      expect(push).toHaveBeenCalledWith('/lesson/02d')
    })

    it('«Продолжить» opens the story and the warning is not repeated for it', async () => {
      vi.mocked(api.stories).mockResolvedValue({ items: [ownedRec] })
      const w = mountView()
      await flushPromises()
      await press(w, 0)
      await w.find('[data-test="story-warn-continue"]').trigger('click')
      expect(push).toHaveBeenCalledWith('/stories/demo')
      push.mockReset()
      await press(w, 0)
      expect(w.find('[data-test="story-warn"]').exists()).toBe(false)
      expect(push).toHaveBeenCalledWith('/stories/demo')
    })

    it('says nothing once the lessons up to the recommended one are finished', async () => {
      vi.mocked(api.course).mockResolvedValue({
        title: 'Курс',
        phases: [{ id: '1', title: 'Уровень 1', lessons: ['02d', '04b'] }],
        lessons: [
          { id: '02d', title: 'a', subtitle: '', planned: false, status: 'done' },
          { id: '04b', title: 'b', subtitle: '', planned: false, status: 'done' },
        ],
      } as never)
      vi.mocked(api.stories).mockResolvedValue({ items: [ownedRec] })
      const w = mountView()
      await flushPromises()
      await press(w, 0)
      expect(w.find('[data-test="story-warn"]').exists()).toBe(false)
      expect(push).toHaveBeenCalledWith('/stories/demo')
    })

    it('shows the зёрнышки first when they are short: the purchase window, no warning', async () => {
      vi.mocked(api.stories).mockResolvedValue({ items: [paidRec] })
      const w = mountView()
      await flushPromises()
      await press(w, 0)
      expect(w.find('[data-test="story-warn"]').exists()).toBe(false)
      expect(w.find('[data-test="story-modal"]').exists()).toBe(true)
      expect(w.find('[data-test="story-short"]').exists()).toBe(true)
    })

    it('warns first when the зёрнышки are enough, and «Продолжить» then opens the purchase window', async () => {
      vi.mocked(api.wallet).mockResolvedValue({
        balance: 25, streak_days: 0, currency_one: 'зёрнышко', currency_few: 'зёрнышка', currency_many: 'зёрнышек',
      } as never)
      vi.mocked(api.stories).mockResolvedValue({ items: [paidRec] })
      const w = mountView()
      await flushPromises()
      await press(w, 0)
      expect(w.find('[data-test="story-warn"]').exists()).toBe(true)
      expect(w.find('[data-test="story-modal"]').exists()).toBe(false)
      await w.find('[data-test="story-warn-continue"]').trigger('click')
      expect(w.find('[data-test="story-enough"]').exists()).toBe(true)
    })
  })

  it('opens a story the learner owns', async () => {
    const w = mountView()
    await flushPromises()
    await w.findAll('[data-test="story-card"]')[0].trigger('click')
    expect(push).toHaveBeenCalledWith('/stories/demo')
  })

  it('asks before buying a locked story, says how many зёрнышек are missing, and will not buy without enough', async () => {
    const w = mountView()
    await flushPromises()
    await w.findAll('[data-test="story-card"]')[1].trigger('click')
    expect(push).not.toHaveBeenCalled()
    expect(w.find('[data-test="story-modal"]').exists()).toBe(true)
    expect(w.find('[data-test="story-buy"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-test="story-short"]').exists()).toBe(true)
  })

  it('draws the purchase modal like the other modals: real buttons, the price on its own line, the currency word from the settings', async () => {
    vi.mocked(api.wallet).mockResolvedValue({
      balance: 6, streak_days: 0, currency_one: 'зёрнышко', currency_few: 'зёрнышка', currency_many: 'зёрнышек',
    } as never)
    const w = mountView()
    await flushPromises()
    await w.findAll('[data-test="story-card"]')[1].trigger('click')
    const buy = w.find('[data-test="story-buy"]')
    expect(buy.classes()).toEqual(expect.arrayContaining(['btn', 'btn-primary']))
    // Short of зёрнышки the shortfall is the headline: the number and the seed picture, the word only for a screen reader.
    expect(w.find('[data-test="story-price"]').exists()).toBe(false)
    expect(w.find('[data-test="story-short"]').find('.lack-num').text()).toBe('4')
    expect(w.find('[data-test="story-short"]').find('img').exists()).toBe(true)
    expect(w.find('[data-test="story-short"]').text()).toContain('Вам не хватает 4')
    expect(w.find('[data-test="story-short"]').find('.sr-only').text()).toBe('зёрнышка')
    // The window itself names no story and gives no time: the card the learner pressed already did.
    expect(w.find('[data-test="story-modal"]').text()).not.toContain('с заданиями')
    expect(w.find('[data-test="story-modal"]').text()).not.toContain('Открыть историю?')
    const later = w.findAll('button').find((b) => b.text() === 'Не сейчас')
    expect(later?.classes()).toEqual(expect.arrayContaining(['btn', 'btn-ghost']))
    await w.find('[data-test="story-earn"]').trigger('click')
    expect(push).toHaveBeenCalledWith('/quests')
  })

  it('says how much is left after buying when the balance is enough', async () => {
    vi.mocked(api.wallet).mockResolvedValue({
      balance: 25, streak_days: 0, currency_one: 'зёрнышко', currency_few: 'зёрнышка', currency_many: 'зёрнышек',
    } as never)
    const w = mountView()
    await flushPromises()
    await w.findAll('[data-test="story-card"]')[1].trigger('click')
    expect(w.find('[data-test="story-buy"]').attributes('disabled')).toBeUndefined()
    expect(w.find('[data-test="story-price"]').text()).toBe('10')
    expect(w.find('[data-test="story-enough"]').text()).toBe('После покупки останется 15 зёрнышек.')
    expect(w.find('[data-test="story-earn"]').exists()).toBe(false)
  })

  it('shows an empty state', async () => {
    vi.mocked(api.stories).mockResolvedValue({ items: [] })
    const w = mountView()
    await flushPromises()
    expect(w.find('[data-test="stories-empty"]').exists()).toBe(true)
  })

  it('says so when the list cannot be loaded', async () => {
    vi.mocked(api.stories).mockRejectedValue(new ApiError(500, 'internal error'))
    const w = mountView()
    await flushPromises()
    expect(w.find('[role="alert"]').exists()).toBe(true)
  })
})
