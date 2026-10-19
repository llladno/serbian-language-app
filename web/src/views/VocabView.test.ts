import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import VocabView from './VocabView.vue'
import { api } from '../api'
import { useSessionStore } from '../stores/session'
import type { FalseFriend, SessionUser, Vocab } from '../types'

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: {}, query: {} }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  RouterLink: { template: '<a><slot /></a>' },
}))

const words: Vocab[] = [
  { id: 'cokolada', latin: 'čokolada', cyrillic: 'чоколада', ru: 'шоколад', note: 'jedna kocka', lesson: '01' },
  { id: 'konj', latin: 'konj', cyrillic: 'коњ', ru: 'лошадь', lesson: '01' },
  { id: 'koji', latin: 'koji', cyrillic: 'који', ru: 'какой (локатив от koji)', lesson: '02' },
]
const traps: FalseFriend[] = [
  { id: 'sto', sr: 'sto', means: 'сто (число)', not: 'стол', correct: 'astal', group: 'top' },
]

function readIn(script: 'latin' | 'cyrillic') {
  useSessionStore().user = {
    id: 'u', name: 'Гриша', email: '', email_verified: true,
    telegram: { linked: false, username: '' }, script,
  } as SessionUser
}

async function open() {
  vi.spyOn(api, 'vocab').mockResolvedValue(structuredClone(words))
  vi.spyOn(api, 'falseFriends').mockResolvedValue(structuredClone(traps))
  const w = mount(VocabView)
  await flushPromises()
  return w
}

afterEach(() => vi.restoreAllMocks())

describe('VocabView follows the learner’s script', () => {
  it('lists the words in Cyrillic for a Cyrillic reader', async () => {
    readIn('cyrillic')
    const w = await open()
    const heads = w.findAll('.card .serbian').map((e) => e.text())
    expect(heads).toEqual(['чоколада', 'коњ', 'који'])
    expect(w.text()).toContain('једна коцка')
  })

  it('lists the words in Latin for a Latin reader', async () => {
    readIn('latin')
    const w = await open()
    expect(w.findAll('.card .serbian').map((e) => e.text())).toEqual(['čokolada', 'konj', 'koji'])
  })

  it('has no script switch of its own any more (it lives in one place, the account)', async () => {
    readIn('cyrillic')
    const w = await open()
    expect(w.findAll('button').some((b) => /^(латиница|кириллица)$/.test(b.text()))).toBe(false)
  })

  it('shows a translation that names a Serbian word in brackets in Cyrillic too', async () => {
    readIn('cyrillic')
    const w = await open()
    expect(w.text()).toContain('какой (локатив от који)')
    expect(w.text()).not.toContain('koji')
  })

  it('has the script pill in its header instead', async () => {
    readIn('latin')
    const w = await open()
    expect(w.find('[data-test="script-cyrillic"]').text()).toBe('Ћир')
  })

  it('drills a word in the learner’s script', async () => {
    readIn('cyrillic')
    const w = await open()
    await w.findAll('button').find((b) => /Учить выборку/.test(b.text()))!.trigger('click')
    expect(w.find('.card.mb-4 .serbian').text()).toBe('чоколада')
  })

  it('shows the false-friend tab in Cyrillic', async () => {
    readIn('cyrillic')
    const w = await open()
    await w.findAll('button').find((b) => b.text().startsWith('Ловушки'))!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('сто')
    expect(w.text()).toContain('астал')
  })
})
