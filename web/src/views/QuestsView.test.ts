import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import QuestsView from './QuestsView.vue'
import { api, ApiError } from '../api'
import { useRewardModal } from '../lib/rewardModal'
import { useToasts } from '../lib/toasts'
import { useInviteModal } from '../lib/inviteModal'
import type { Quest } from '../types'

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}))

// The screen listens for the learner coming back to the app: a component that
// outlived its test would answer the next test's events too.
enableAutoUnmount(afterEach)

function quest(over: Partial<Quest> = {}): Quest {
  return {
    id: 1,
    kind: 'lessons_completed',
    title: 'Пройти 5 уроков',
    description: 'Любые пять уроков курса',
    target: 5,
    value: 5,
    reward: 15,
    done: true,
    claimed: false,
    ...over,
  }
}

const WALLET = {
  balance: 100,
  currency_one: 'зёрнышко',
  currency_few: 'зёрнышка',
  currency_many: 'зёрнышек',
  streak_days: 4,
}

beforeEach(() => {
  // The balance counts up to its value; AnimatedNumber's own test covers the
  // counting, and here it only gets in the way of reading the number.
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: query.includes('prefers-reduced-motion'),
    media: query,
    addEventListener: () => {},
    removeEventListener: () => {},
  }))
  setActivePinia(createPinia())
  useRewardModal().closeModal()
  useInviteModal().closeModal()
  vi.spyOn(api, 'wallet').mockResolvedValue(WALLET)
  vi.spyOn(api, 'shop').mockResolvedValue({ items: [] })
})
afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('QuestsView', () => {
  it('splits quests into claimable, in progress and already paid', async () => {
    vi.spyOn(api, 'quests').mockResolvedValue({
      quests: [
        quest({ id: 1 }),
        quest({ id: 2, kind: 'vocab_learned', title: 'Выучить 30 слов', target: 30, value: 12, done: false }),
        quest({
          id: 3,
          kind: 'telegram_subscribed',
          title: 'Подписка на канал',
          target: 1,
          value: 1,
          claimed: true,
        }),
      ],
    })
    const w = mount(QuestsView)
    await flushPromises()

    expect(w.find('[data-test="wallet-balance"]').text()).toBe('100 зёрнышек')
    expect(w.findAll('[data-test="quest-ready"]')).toHaveLength(1)
    expect(w.text()).toContain('Можно забрать')
    expect(w.text()).toContain('12/30')
    expect(w.text()).toContain('Получено')
    // A claimed quest keeps no button: the only claim button on screen is the
    // one for the quest that is actually payable.
    expect(w.findAll('button')).toHaveLength(1)
  })

  it('celebrates the reward the server paid, not the one the quest advertised', async () => {
    // The admin can change a quest's reward between the list and the claim;
    // the modal has to show what was actually credited.
    vi.spyOn(api, 'quests').mockResolvedValue({ quests: [quest()] })
    vi.spyOn(api, 'claimQuest').mockResolvedValue({ reward: 20, balance: 120 })
    const { open, reward } = useRewardModal()
    const w = mount(QuestsView)
    await flushPromises()

    await w.find('[data-test="claim-1"]').trigger('click')
    await flushPromises()

    expect(api.claimQuest).toHaveBeenCalledWith(1)
    expect(open.value).toBe(true)
    expect(reward.value).toBe(20)
    expect(w.find('[data-test="wallet-balance"]').text()).toBe('120 зёрнышек')
  })

  it('reports a rejected claim and refetches the list instead of celebrating', async () => {
    const quests = vi.spyOn(api, 'quests').mockResolvedValue({ quests: [quest()] })
    vi.spyOn(api, 'claimQuest').mockRejectedValue(new ApiError(409, 'already_claimed'))
    const { open } = useRewardModal()
    const w = mount(QuestsView)
    await flushPromises()

    await w.find('[data-test="claim-1"]').trigger('click')
    await flushPromises()

    expect(open.value).toBe(false)
    expect(w.find('[data-test="claim-error"]').exists()).toBe(true)
    expect(quests).toHaveBeenCalledTimes(2)
  })

  describe('the channel quest', () => {
    const channel = (over: Partial<Quest> = {}) =>
      quest({
        id: 9,
        kind: 'telegram_subscribed',
        title: 'Подписаться на канал',
        description: 'Подпишись на наш Telegram-канал',
        target: 1,
        value: 0,
        done: false,
        url: 'https://t.me/ucimosrb',
        ...over,
      })

    function quietExtras() {
      vi.spyOn(api, 'wallet').mockResolvedValue(WALLET)
      vi.spyOn(api, 'shop').mockResolvedValue({ items: [] })
    }
    const listing = (quests: Quest[]) => vi.spyOn(api, 'quests').mockResolvedValue({ quests })

    beforeEach(() => useToasts().clearToasts())

    // Telegram is linked: straight to the channel.
    it('sends a learner with Telegram linked to the channel, and says nothing underneath', async () => {
      quietExtras()
      listing([channel()])
      const w = mount(QuestsView)
      await flushPromises()
      const go = w.find('[data-test="quest-go"]')
      expect(go.attributes('href')).toBe('https://t.me/ucimosrb')
      expect(go.attributes('target')).toBe('_blank')
      expect(w.find('[data-test="quest-go-bot"]').exists()).toBe(false)
      // the row is the title, the button and nothing else: no explanation text
      expect(w.find('[data-test="quest-link"]').text()).toBe('Подписаться')
      // nothing to re-check until they have been there
      expect(w.find('[data-test="quest-recheck"]').exists()).toBe(false)
    })

    it('has no button for a quest that lives inside the app', async () => {
      quietExtras()
      listing([quest({ id: 2, done: false, value: 1 })])
      const w = mount(QuestsView)
      await flushPromises()
      expect(w.find('[data-test="quest-go"]').exists()).toBe(false)
      expect(w.find('[data-test="quest-go-bot"]').exists()).toBe(false)
    })

    it('offers to check once the learner has been to the channel, and answers with a toast', async () => {
      quietExtras()
      const quests = listing([channel()])
      const w = mount(QuestsView)
      await flushPromises()
      await w.find('[data-test="quest-go"]').trigger('click')

      // not subscribed yet
      await w.find('[data-test="quest-recheck"]').trigger('click')
      await flushPromises()
      expect(useToasts().items.value.map((t) => t.title)).toEqual(['Подписки пока не видно'])
      expect(w.text()).not.toContain('Пока подписки не видно') // the answer is the toast, not a line of text

      // now subscribed
      useToasts().clearToasts()
      quests.mockResolvedValue({ quests: [channel({ done: true, value: 1, url: undefined })] })
      await w.find('[data-test="quest-recheck"]').trigger('click')
      await flushPromises()
      expect(useToasts().items.value.map((t) => t.title)).toEqual(['Подписка найдена'])
      expect(w.find('[data-test="quest-ready"]').exists()).toBe(true)
      expect(w.find('[data-test="quest-go"]').exists()).toBe(false)
    })

    // Already subscribed: the quest just arrives finished.
    it('shows an already subscribed learner the quest as done, with nothing to press', async () => {
      quietExtras()
      listing([channel({ done: true, value: 1, url: undefined })])
      const w = mount(QuestsView)
      await flushPromises()
      expect(w.find('[data-test="quest-ready"]').exists()).toBe(true)
      expect(w.find('[data-test="quest-go"]').exists()).toBe(false)
    })

    // Telegram not linked: to the bot, which does the rest.
    it('sends a learner without Telegram to the bot, linking it for this quest', async () => {
      quietExtras()
      listing([channel({ needs_telegram: true })])
      const start = vi.spyOn(api, 'telegramLinkStart').mockResolvedValue({ url: 'https://t.me/bot?start=x', token: 'x' })
      vi.spyOn(api, 'telegramPoll').mockResolvedValue({ status: 'pending' })
      vi.spyOn(window, 'open').mockReturnValue(null)

      const w = mount(QuestsView)
      await flushPromises()
      expect(w.find('[data-test="quest-go"]').exists()).toBe(false) // no direct link: the bot comes first
      await w.find('[data-test="quest-go-bot"]').trigger('click')
      await flushPromises()

      expect(start).toHaveBeenCalledWith('channel')
      expect(w.find('[data-test="quest-link"]').text()).not.toMatch(/привяз/i) // still no explanation text
    })

    it('says Telegram is linked, and looks again, when the bot has linked it', async () => {
      quietExtras()
      const quests = listing([channel({ needs_telegram: true })])
      vi.spyOn(api, 'telegramLinkStart').mockResolvedValue({ url: 'https://t.me/bot?start=x', token: 'x' })
      vi.spyOn(api, 'telegramPoll').mockResolvedValue({ status: 'ok' })
      vi.spyOn(window, 'open').mockReturnValue(null)
      vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'setTimeout', 'clearTimeout'] })

      const w = mount(QuestsView)
      await flushPromises()
      quests.mockResolvedValue({ quests: [channel()] }) // linked now: the plain channel link
      const before = quests.mock.calls.length
      await w.find('[data-test="quest-go-bot"]').trigger('click')
      await flushPromises()
      await vi.advanceTimersByTimeAsync(2000)
      await flushPromises()
      vi.useRealTimers()

      expect(useToasts().items.value.map((t) => t.title)).toContain('Telegram привязан')
      expect(quests.mock.calls.length).toBeGreaterThan(before)
      expect(w.find('[data-test="quest-go"]').exists()).toBe(true)
    })

    it('tells the learner, by toast, that the quest finished while they were away', async () => {
      quietExtras()
      const quests = listing([channel({ needs_telegram: true })])
      mount(QuestsView)
      await flushPromises()

      // back from the bot, which found the subscription
      quests.mockResolvedValue({ quests: [channel({ done: true, value: 1, url: undefined })] })
      Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
      document.dispatchEvent(new Event('visibilitychange'))
      await flushPromises()

      const toast = useToasts().items.value.find((t) => t.title === 'Задание выполнено')
      expect(toast?.text).toBe('Подписаться на канал')
    })
  })

  describe('the row layout', () => {
    it('puts the reward on the left and the action on the right', async () => {
      vi.spyOn(api, 'quests').mockResolvedValue({
        quests: [
          quest({
            id: 9,
            kind: 'telegram_subscribed',
            title: 'Подписаться на канал',
            target: 1,
            value: 0,
            done: false,
            reward: 20,
            url: 'https://t.me/ucimosrb',
          }),
        ],
      })
      const w = mount(QuestsView)
      await flushPromises()

      const row = w.find('[data-test="quest-active"]')
      const kids = Array.from(row.element.children)
      expect(kids[0].getAttribute('data-test')).toBe('quest-reward')
      expect(kids[0].textContent).toContain('20')
      expect(kids[1].textContent).toContain('Подписаться на канал')
      expect(kids[kids.length - 1].querySelector('[data-test="quest-go"]')).not.toBeNull()
    })

    it('keeps the claim button on the right of a payable quest', async () => {
      vi.spyOn(api, 'quests').mockResolvedValue({ quests: [quest({ reward: 15 })] })
      const w = mount(QuestsView)
      await flushPromises()

      const kids = Array.from(w.find('[data-test="quest-ready"]').element.children)
      expect(kids[0].getAttribute('data-test')).toBe('quest-reward')
      expect(kids[0].textContent).toContain('+15')
      expect(kids[kids.length - 1].getAttribute('data-test')).toBe('claim-1')
    })
  })

  describe('the invite quest', () => {
    const invite = (over: Partial<Quest> = {}) =>
      quest({
        id: 12,
        kind: 'friends_invited',
        title: 'Пригласить друга',
        description: 'Друг пришёл по твоей ссылке',
        target: 1,
        value: 0,
        reward: 30,
        done: false,
        ...over,
      })
    const REFERRAL = { code: 'abcd2345', url: 'https://ucimo.ru/register?ref=abcd2345', friends: 0 }

    it('has an invite button that opens the modal with the personal link', async () => {
      vi.spyOn(api, 'quests').mockResolvedValue({ quests: [invite()] })
      const referral = vi.spyOn(api, 'referral').mockResolvedValue(REFERRAL)
      const w = mount(QuestsView, { attachTo: document.body })
      await flushPromises()

      expect(referral).not.toHaveBeenCalled() // nothing is minted until somebody asks
      await w.find('[data-test="quest-invite"]').trigger('click')
      await flushPromises()

      const link = document.body.querySelector('[data-test="invite-link"]') as HTMLInputElement
      expect(link.value).toBe(REFERRAL.url)
      const tg = document.body.querySelector('[data-test="invite-telegram"]') as HTMLAnchorElement
      expect(tg.href).toBe(
        'https://t.me/share/url?url=' + encodeURIComponent(REFERRAL.url) + '&text=' + encodeURIComponent('Учу сербский в UCIMO — уроки, повторение слов и озвучка. Заходи по моей ссылке:'),
      )
      expect(document.body.textContent).toContain('30')
      w.unmount()
    })

    it('copies the link and says so', async () => {
      vi.spyOn(api, 'quests').mockResolvedValue({ quests: [invite()] })
      vi.spyOn(api, 'referral').mockResolvedValue(REFERRAL)
      const writeText = vi.fn().mockResolvedValue(undefined)
      vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
      const w = mount(QuestsView, { attachTo: document.body })
      await flushPromises()
      await w.find('[data-test="quest-invite"]').trigger('click')
      await flushPromises()

      const btn = document.body.querySelector('[data-test="invite-copy"]') as HTMLButtonElement
      btn.click()
      await flushPromises()

      expect(writeText).toHaveBeenCalledWith(REFERRAL.url)
      expect(btn.textContent).toContain('скопирована')
      w.unmount()
    })

    it('falls back to selecting the link when the clipboard is refused', async () => {
      vi.spyOn(api, 'quests').mockResolvedValue({ quests: [invite()] })
      vi.spyOn(api, 'referral').mockResolvedValue(REFERRAL)
      vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) } })
      const exec = vi.fn().mockReturnValue(true)
      Object.defineProperty(document, 'execCommand', { value: exec, configurable: true })
      const w = mount(QuestsView, { attachTo: document.body })
      await flushPromises()
      await w.find('[data-test="quest-invite"]').trigger('click')
      await flushPromises()

      ;(document.body.querySelector('[data-test="invite-copy"]') as HTMLButtonElement).click()
      await flushPromises()

      expect(exec).toHaveBeenCalledWith('copy')
      w.unmount()
    })

    it('shows a readable error when the link cannot be fetched', async () => {
      vi.spyOn(api, 'quests').mockResolvedValue({ quests: [invite()] })
      vi.spyOn(api, 'referral').mockRejectedValue(new ApiError(500, 'internal'))
      const w = mount(QuestsView, { attachTo: document.body })
      await flushPromises()
      await w.find('[data-test="quest-invite"]').trigger('click')
      await flushPromises()

      expect(document.body.querySelector('[data-test="invite-error"]')).not.toBeNull()
      expect(document.body.querySelector('[data-test="invite-link"]')).toBeNull()
      w.unmount()
    })
  })
})
