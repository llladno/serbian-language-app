import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TextAnswer from './TextAnswer.vue'
import { api } from '../../api'
import { phraseKey, _setPhraseIndex } from '../../lib/phraseAudio'

afterEach(() => {
  vi.restoreAllMocks()
  _setPhraseIndex(null)
})

const props = { lesson: '01', exerciseId: '01-A-1', type: 'translate' as const, prompt: 'Привет!' }

describe('TextAnswer', () => {
  it('shows the expected answer and a wrong-chunk after a failed check', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({
      ok: false,
      expected: 'Zdravo! Kako si?',
      explain: 'почти',
      diff: [
        { text: 'zdravo', ok: true },
        { text: 'kako', ok: true },
        { text: 'sti', ok: false },
      ],
    })
    const w = mount(TextAnswer, { props })
    await w.find('input').setValue('zdravo kako sti')
    await w.find('form').trigger('submit')
    await flushPromises()

    expect(w.text()).toContain('Zdravo! Kako si?')
    expect(w.text()).toContain('почти')
    expect(w.find('.chunk-wrong').exists()).toBe(true)
    expect(w.emitted('graded')?.[0]?.[0]).toBe(false)
  })

  it('reports success and emits graded true', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Zdravo!' })
    const w = mount(TextAnswer, { props })
    await w.find('input').setValue('Zdravo!')
    await w.find('form').trigger('submit')
    await flushPromises()

    expect(w.text()).toContain('Верно')
    expect(w.emitted('graded')?.[0]?.[0]).toBe(true)
  })

  it('does not submit an empty answer', async () => {
    const spy = vi.spyOn(api, 'check')
    const w = mount(TextAnswer, { props })
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(spy).not.toHaveBeenCalled()
  })

  it('makes the Serbian prompt words clickable for fill_blank', () => {
    const w = mount(TextAnswer, {
      props: { ...props, type: 'fill_blank', prompt: '___ košta kafa?' },
      global: { stubs: { RouterLink: true } },
    })
    expect(w.findAll('span.cursor-pointer').map((s) => s.text())).toEqual(['košta', 'kafa'])
  })

  it('leaves a Russian translate prompt as plain text', () => {
    const w = mount(TextAnswer, { props: { ...props, type: 'translate', prompt: 'Кто это?' } })
    expect(w.find('span.cursor-pointer').exists()).toBe(false)
    expect(w.text()).toContain('Кто это?')
  })

  it('for listen: plays audio, hides the answer text, grades the typed transcription', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Zdravo, kako si?' })
    const w = mount(TextAnswer, {
      props: {
        ...props,
        type: 'listen',
        prompt: '',
        audio: '01-D-1.mp3',
        exerciseId: '01-D-1',
      },
    })
    // an audio control is present, and nothing gives the answer away before submitting
    expect(w.find('button[aria-label], button[title]').exists()).toBe(true)
    expect(w.text()).not.toContain('Zdravo, kako si?')
    expect(w.text()).toContain('Напиши, что слышишь')

    await w.find('input').setValue('Zdravo, kako si?')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.emitted('graded')?.[0]?.[0]).toBe(true)
  })
})

// A dictation for someone who cannot listen: the sentence is shown and
// translated into Russian.
describe('TextAnswer, a dictation read as text', () => {
  const listen = {
    lesson: '03',
    exerciseId: '03.4.1',
    type: 'listen' as const,
    prompt: 'Переведи на русский.',
  }

  it('offers "не могу прослушать" on a dictation and says so when pressed', async () => {
    const w = mount(TextAnswer, { props: listen })
    await w.find('[data-test="cant-listen"]').trigger('click')
    expect(w.emitted('cantListen')).toHaveLength(1)
  })

  it('shows the sentence, takes a Russian answer and checks it as a translation', async () => {
    const check = vi.spyOn(api, 'check').mockResolvedValue({ ok: true })
    const w = mount(TextAnswer, { props: { ...listen, text: 'Dobar dan, kako ste?' } })

    expect(w.find('[data-test="listen-as-text"]').text()).toContain('Dobar dan, kako ste?')
    // nothing to play and no Serbian letters to type
    expect(w.find('[data-test="cant-listen"]').exists()).toBe(false)
    expect(w.findComponent({ name: 'SerbianKeys' }).exists()).toBe(false)

    await w.find('input').setValue('Добрый день, как вы?')
    await w.find('form').trigger('submit')
    await flushPromises()

    expect(check).toHaveBeenCalledWith('03', '03.4.1', { answer: 'Добрый день, как вы?', ru: true })
  })

  it('does not mark the dictation as a translation when the audio is on', async () => {
    const check = vi.spyOn(api, 'check').mockResolvedValue({ ok: true })
    const w = mount(TextAnswer, { props: listen })
    await w.find('input').setValue('Dobar dan')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(check).toHaveBeenCalledWith('03', '03.4.1', { answer: 'Dobar dan' })
  })

  describe('speaker after the verdict', () => {
    const speaker = 'button[aria-label="озвучить"]'
    const submit = async (w: ReturnType<typeof mount>, value: string) => {
      await w.find('input').setValue(value)
      await w.find('form').trigger('submit')
      await flushPromises()
    }

    it('reads the whole gap-filled sentence, not the single missing word', async () => {
      // only the full sentence is voiced; the bare word "iza" has no clip
      _setPhraseIndex([phraseKey('Ja sam iza vama.')])
      vi.spyOn(api, 'check').mockResolvedValue({ ok: true })
      const w = mount(TextAnswer, {
        props: { ...props, type: 'fill_blank' as const, prompt: '«Я за вами» → Ja sam ____ vama.' },
      })
      await submit(w, 'iza')
      expect(w.findAll(speaker)).toHaveLength(1)
    })

    it('has a single speaker on a dictation (its own, on top)', async () => {
      _setPhraseIndex([phraseKey('Odakle si?')])
      vi.spyOn(api, 'check').mockResolvedValue({ ok: true })
      const w = mount(TextAnswer, {
        props: { ...props, type: 'listen' as const, prompt: 'Запиши', audio: '01.8.1.mp3' },
      })
      await submit(w, 'Odakle si?')
      expect(w.findAll(speaker)).toHaveLength(1)
    })
  })
})
