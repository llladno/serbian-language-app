import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useScript, provideKeepLatin } from './useScript'
import { useSessionStore } from '../stores/session'
import { api } from '../api'
import type { SessionUser } from '../types'

vi.mock('../telegram', () => ({ isTelegram: vi.fn(), initData: vi.fn() }))

function user(overrides: Partial<SessionUser> = {}): SessionUser {
  return {
    id: 'usr_1',
    name: 'Гриша',
    email: 'g@example.com',
    email_verified: true,
    telegram: { linked: false, username: '' },
    script: 'latin',
    ...overrides,
  }
}

// A component that paints one Serbian string the way every real one will.
const Probe = defineComponent({
  props: { text: { type: String, default: 'čokolada' } },
  setup(props) {
    const { show } = useScript()
    return () => h('span', show(props.text))
  },
})

beforeEach(() => {
  localStorage.clear()
  setActivePinia(createPinia())
})
afterEach(() => vi.restoreAllMocks())

describe('useScript', () => {
  it('shows Latin when nobody is signed in', () => {
    expect(mount(Probe).text()).toBe('čokolada')
  })

  it("shows Latin for an account that has not chosen (or an old cached one with no script)", () => {
    useSessionStore().user = user({ script: undefined })
    expect(mount(Probe).text()).toBe('čokolada')
  })

  it('shows Cyrillic when the account says so', () => {
    useSessionStore().user = user({ script: 'cyrillic' })
    expect(mount(Probe).text()).toBe('чоколада')
  })

  it('follows a change of the setting without remounting', async () => {
    const session = useSessionStore()
    session.user = user({ script: 'latin' })
    const w = mount(Probe)
    expect(w.text()).toBe('čokolada')
    session.user = user({ script: 'cyrillic' })
    await w.vm.$nextTick()
    expect(w.text()).toBe('чоколада')
  })

  it('stays Latin inside a lesson that teaches the letters, whatever the account says', () => {
    useSessionStore().user = user({ script: 'cyrillic' })
    const keep = ref(true)
    const Lesson = defineComponent({
      setup() {
        provideKeepLatin(keep)
        return () => h(Probe)
      },
    })
    const w = mount(Lesson)
    expect(w.text()).toBe('čokolada')
  })

  it('does not let the keep-latin lesson leak to components outside it', () => {
    useSessionStore().user = user({ script: 'cyrillic' })
    const inside = mount(
      defineComponent({
        setup() {
          provideKeepLatin(ref(true))
          return () => h(Probe)
        },
      }),
    )
    expect(inside.text()).toBe('čokolada')
    expect(mount(Probe).text()).toBe('чоколада')
  })

  it('exposes the effective script, for components that branch on it', () => {
    useSessionStore().user = user({ script: 'cyrillic' })
    let seen = ''
    mount(
      defineComponent({
        setup() {
          seen = useScript().script.value
          return () => h('i')
        },
      }),
    )
    expect(seen).toBe('cyrillic')
  })
})

describe('session.setScript', () => {
  it('switches at once, then keeps what the server answered', async () => {
    const session = useSessionStore()
    session.user = user({ script: 'latin' })
    let release: (u: SessionUser) => void = () => {}
    vi.spyOn(api, 'setScript').mockReturnValue(new Promise<SessionUser>((r) => (release = r)))

    const pending = session.setScript('cyrillic')
    expect(session.user?.script).toBe('cyrillic') // optimistic: the text flips before the round trip
    release(user({ script: 'cyrillic' }))
    await pending
    expect(session.user?.script).toBe('cyrillic')
    expect(JSON.parse(localStorage.getItem('ucimo_session_user')!).script).toBe('cyrillic')
  })

  it('puts the old script back when the server refuses', async () => {
    const session = useSessionStore()
    session.user = user({ script: 'latin' })
    vi.spyOn(api, 'setScript').mockRejectedValue(new Error('offline'))
    await expect(session.setScript('cyrillic')).rejects.toThrow('offline')
    expect(session.user?.script).toBe('latin')
  })

  it('asks the server exactly once and with the chosen value', async () => {
    const session = useSessionStore()
    session.user = user()
    const spy = vi.spyOn(api, 'setScript').mockResolvedValue(user({ script: 'cyrillic' }))
    await session.setScript('cyrillic')
    expect(spy).toHaveBeenCalledTimes(1)
    expect(spy).toHaveBeenCalledWith('cyrillic')
  })
})
