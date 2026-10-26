import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ChoiceAnswer from './ChoiceAnswer.vue'
import { api } from '../../api'
import { phraseKey, _setPhraseIndex } from '../../lib/phraseAudio'
import { speakClip } from '../../lib/sound'

vi.mock('../../lib/sound', () => ({ speakClip: vi.fn() }))


afterEach(() => {
  vi.restoreAllMocks()
  _setPhraseIndex(null)
})

const props = { lesson: '01', exerciseId: '01.2.1', prompt: '«Спасибо» —', options: ['Hvala', 'Molim'] }

describe('ChoiceAnswer', () => {
  it('checks the clicked option and reports success', async () => {
    const check = vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Hvala' })
    const w = mount(ChoiceAnswer, { props })
    await w.findAll('button').find((b) => b.text() === 'Hvala')!.trigger('click')
    await flushPromises()

    expect(check).toHaveBeenCalledWith('01', '01.2.1', { answer: 'Hvala' })
    expect(w.emitted('graded')?.[0]?.[0]).toBe(true)
    expect(w.text()).toContain('Верно')
  })

  it('shows the correct answer after a wrong pick', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: false, expected: 'Hvala' })
    const w = mount(ChoiceAnswer, { props })
    await w.findAll('button').find((b) => b.text() === 'Molim')!.trigger('click')
    await flushPromises()

    expect(w.emitted('graded')?.[0]?.[0]).toBe(false)
    expect(w.text()).toContain('Hvala')
  })

  it('passes the check result and what was picked along with the verdict', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, line: 'Hvala', line_ru: 'Спасибо' })
    const w = mount(ChoiceAnswer, { props })
    await w.findAll('button').find((b) => b.text() === 'Hvala')!.trigger('click')
    await flushPromises()

    expect(w.emitted('graded')?.[0]).toEqual([true, { ok: true, line: 'Hvala', line_ru: 'Спасибо' }, 'Hvala'])
  })

  describe('speaker', () => {
    const speaker = 'button[aria-label="озвучить"]'

    it('never voices the options before an answer (a wrong one must not be heard)', () => {
      _setPhraseIndex([phraseKey('Hvala'), phraseKey('Molim')])
      const w = mount(ChoiceAnswer, { props })
      expect(w.find(speaker).exists()).toBe(false)
    })

    it('after a right pick offers one speaker, for the right phrase', async () => {
      _setPhraseIndex([phraseKey('Hvala'), phraseKey('Molim')])
      vi.spyOn(api, 'check').mockResolvedValue({ ok: true })
      const w = mount(ChoiceAnswer, { props })
      await w.findAll('button').find((b) => b.text() === 'Hvala')!.trigger('click')
      await flushPromises()
      expect(w.findAll(speaker)).toHaveLength(1)
    })

    it('after a wrong pick the speaker reads the correct phrase, not the picked one', async () => {
      _setPhraseIndex([phraseKey('Hvala')])
      vi.spyOn(api, 'check').mockResolvedValue({ ok: false, expected: 'Hvala' })
      const w = mount(ChoiceAnswer, { props })
      await w.findAll('button').find((b) => b.text() === 'Molim')!.trigger('click')
      await flushPromises()
      expect(w.findAll(speaker)).toHaveLength(1)
    })
  })
})

describe('ChoiceAnswer reads the right phrase out after an answer', () => {
  const clipOf = (t: string) => `p/${phraseKey(t)}.mp3`
  const pick = async (w: ReturnType<typeof mount>, label: string) => {
    await w.findAll('button').find((b) => b.text() === label)!.trigger('click')
    await flushPromises()
  }
  beforeEach(() => vi.mocked(speakClip).mockClear())

  it('after a right pick', async () => {
    _setPhraseIndex([phraseKey('Hvala'), phraseKey('Molim')])
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true })
    await pick(mount(ChoiceAnswer, { props }), 'Hvala')
    expect(speakClip).toHaveBeenCalledTimes(1)
    expect(speakClip).toHaveBeenCalledWith(clipOf('Hvala'))
  })

  it('after a wrong pick it is the right phrase, never the picked one', async () => {
    _setPhraseIndex([phraseKey('Hvala'), phraseKey('Molim')])
    vi.spyOn(api, 'check').mockResolvedValue({ ok: false, expected: 'Hvala' })
    await pick(mount(ChoiceAnswer, { props }), 'Molim')
    expect(speakClip).toHaveBeenCalledTimes(1)
    expect(speakClip).toHaveBeenCalledWith(clipOf('Hvala'))
  })

  it('not while an earlier answer is only being looked at', () => {
    _setPhraseIndex([phraseKey('Hvala')])
    mount(ChoiceAnswer, { props: { ...props, prior: { answer: 'Hvala', correct: true } } })
    expect(speakClip).not.toHaveBeenCalled()
  })

  it('not before an answer', () => {
    _setPhraseIndex([phraseKey('Hvala'), phraseKey('Molim')])
    mount(ChoiceAnswer, { props })
    expect(speakClip).not.toHaveBeenCalled()
  })
})

// «Послушай и выбери»: a choice that comes with a clip (`audio`) instead of
// text, or with the sentence as text (`text`) for a learner who cannot listen.
describe('ChoiceAnswer as a listening task', () => {
  const listenProps = {
    lesson: '02a',
    exerciseId: '02a.3.1',
    prompt: 'Послушай и выбери перевод.',
    options: ['Доброе утро.', 'Добрый вечер.'],
    audio: '02a.3.1.mp3',
  }
  const speaker = 'button[aria-label="озвучить"]'

  it('offers the clip and a way out for someone who cannot listen', async () => {
    const w = mount(ChoiceAnswer, { props: listenProps })
    expect(w.find(speaker).exists()).toBe(true)
    expect(w.find('[data-test="listen-as-text"]').exists()).toBe(false)
    await w.find('[data-test="cant-listen"]').trigger('click')
    expect(w.emitted('cantListen')).toHaveLength(1)
  })

  it('checks the picked meaning like any other choice', async () => {
    const check = vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Доброе утро.', say: 'Dobro jutro.' })
    const w = mount(ChoiceAnswer, { props: listenProps })
    await w.findAll('button').find((b) => b.text() === 'Доброе утро.')!.trigger('click')
    await flushPromises()
    expect(check).toHaveBeenCalledWith('02a', '02a.3.1', { answer: 'Доброе утро.' })
    expect(w.emitted('graded')?.[0]?.[0]).toBe(true)
  })

  it('reveals the sentence that was played once it is answered', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: false, expected: 'Доброе утро.', say: 'Dobro jutro.' })
    const w = mount(ChoiceAnswer, { props: listenProps })
    expect(w.text()).not.toContain('Dobro jutro')
    await w.findAll('button').find((b) => b.text() === 'Добрый вечер.')!.trigger('click')
    await flushPromises()
    expect(w.find('[data-test="listen-said"]').text()).toBe('Dobro jutro.')
    // the clip stays available to hear it again
    expect(w.find(speaker).exists()).toBe(true)
  })

  it('shows the sentence as text, without a clip or the way out, when audio is off', () => {
    const w = mount(ChoiceAnswer, {
      props: { ...listenProps, audio: undefined, text: 'Dobro jutro.', prompt: 'Выбери перевод.' },
    })
    expect(w.find('[data-test="listen-as-text"]').text()).toBe('Dobro jutro.')
    expect(w.find(speaker).exists()).toBe(false)
    expect(w.find('[data-test="cant-listen"]').exists()).toBe(false)
  })

  it('an ordinary choice has no way out and no listening block', () => {
    const w = mount(ChoiceAnswer, { props })
    expect(w.find('[data-test="cant-listen"]').exists()).toBe(false)
    expect(w.find('[data-test="listen-as-text"]').exists()).toBe(false)
  })
})
