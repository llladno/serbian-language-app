import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TextAnswer from './TextAnswer.vue'
import { api } from '../../api'
import { phraseKey, _setPhraseIndex } from '../../lib/phraseAudio'
import { speakClip } from '../../lib/sound'

vi.mock('../../lib/sound', () => ({ speakClip: vi.fn() }))


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

  it('says «Вы написали:» before the learner\'s own words, with the wrong ones marked', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({
      ok: false,
      expected: 'Zdravo! Kako si?',
      diff: [
        { text: 'zdravo', ok: true },
        { text: 'sti', ok: false },
      ],
    })
    const w = mount(TextAnswer, { props })
    await w.find('input').setValue('zdravo sti')
    await w.find('form').trigger('submit')
    await flushPromises()

    const wrote = w.get('[data-test="wrote"]')
    expect(wrote.text().replace(/\s+/g, ' ')).toBe('Вы написали: zdravo sti')
    expect(wrote.find('.chunk-wrong').text()).toBe('sti')
    expect(w.text()).toContain('Правильно: Zdravo! Kako si?')
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
describe('TextAnswer, soft acceptance and fix_error', () => {
  it('tells the learner how a word is spelled when the answer passed with a typo', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, typo: true, expected: 'Dobro veče' })
    const w = mount(TextAnswer, { props: { ...props, type: 'fill_blank' as const, prompt: 'Dobro ___!' } , global: { stubs: { RouterLink: true } } })
    await w.find('input').setValue('dobro veće')
    await w.find('form').trigger('submit')
    await flushPromises()

    expect(w.text()).toContain('Верно')
    expect(w.text()).toContain('проверь написание')
    expect(w.text()).toContain('Dobro veče')
    expect(w.emitted('graded')?.[0]?.[0]).toBe(true)
  })

  it('shows what the learner wrote next to the right spelling when it passed with a typo', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, typo: true, expected: 'Ovo je sestru' })
    const w = mount(TextAnswer, { props: { ...props, type: 'fill_blank' as const, prompt: 'Ovo je ___.' }, global: { stubs: { RouterLink: true } } })
    await w.find('input').setValue('Ovo je sestra')
    await w.find('form').trigger('submit')
    await flushPromises()

    const note = w.get('[data-test="typo-note"]').text().replace(/\s+/g, ' ')
    expect(note).toContain('Засчитано, но проверь написание')
    expect(note).toContain('Вы написали: Ovo je sestra')
    expect(note).toContain('Правильно: Ovo je sestru')
  })

  it('does not nag about spelling on a plain correct answer', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Zdravo!' })
    const w = mount(TextAnswer, { props })
    await w.find('input').setValue('Zdravo!')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.text()).not.toContain('проверь написание')
  })

  it('starts a fix_error from the wrong sentence, so the learner edits instead of retyping', () => {
    const w = mount(TextAnswer, {
      props: { ...props, type: 'fix_error' as const, prompt: 'Iz Rusije.' },
      global: { stubs: { RouterLink: true } },
    })
    expect((w.find('input').element as HTMLInputElement).value).toBe('Iz Rusije.')
  })

  it('leaves a fix_error empty when its prompt carries a Russian note', () => {
    const w = mount(TextAnswer, {
      props: { ...props, type: 'fix_error' as const, prompt: 'Ana kaže: «Gladan sam.» (Ана — женщина)' },
      global: { stubs: { RouterLink: true } },
    })
    expect((w.find('input').element as HTMLInputElement).value).toBe('')
  })

  it('keeps other types empty', () => {
    const w = mount(TextAnswer, { props })
    expect((w.find('input').element as HTMLInputElement).value).toBe('')
  })
})

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

describe('TextAnswer reads the right sentence out after an answer', () => {
  const clipOf = (t: string) => `p/${phraseKey(t)}.mp3`
  beforeEach(() => vi.mocked(speakClip).mockClear())
  const answerWith = async (extra: Record<string, unknown>, typed: string) => {
    const w = mount(TextAnswer, { props: { ...props, ...extra } })
    await w.find('input').setValue(typed)
    await w.find('form').trigger('submit')
    await flushPromises()
    return w
  }

  it('reads what was typed when it is right', async () => {
    _setPhraseIndex([phraseKey('Zdravo!')])
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Zdravo!' })
    await answerWith({}, 'Zdravo!')
    expect(speakClip).toHaveBeenCalledTimes(1)
    expect(speakClip).toHaveBeenCalledWith(clipOf('Zdravo!'))
  })

  it('reads the right answer, not the typed mistake', async () => {
    _setPhraseIndex([phraseKey('Zdravo!'), phraseKey('Zdrvo!')])
    vi.spyOn(api, 'check').mockResolvedValue({ ok: false, expected: 'Zdravo!' })
    await answerWith({}, 'Zdrvo!')
    expect(speakClip).toHaveBeenCalledTimes(1)
    expect(speakClip).toHaveBeenCalledWith(clipOf('Zdravo!'))
  })

  it('reads the whole sentence of a gap-fill, not the missing word', async () => {
    _setPhraseIndex([phraseKey('Ja sam iza vama.')])
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'iza' })
    await answerWith({ type: 'fill_blank', prompt: 'Ja sam ___ vama.' }, 'iza')
    expect(speakClip).toHaveBeenCalledWith(clipOf('Ja sam iza vama.'))
  })

  it('leaves a dictation alone: its own sentence has just been played', async () => {
    _setPhraseIndex([phraseKey('Zdravo!')])
    vi.spyOn(api, 'check').mockResolvedValue({ ok: true, expected: 'Zdravo!' })
    await answerWith({ type: 'listen', audio: 'x.mp3' }, 'Zdravo!')
    expect(speakClip).not.toHaveBeenCalled()
  })

  it('is silent while an earlier answer is only being looked at', () => {
    _setPhraseIndex([phraseKey('Zdravo!')])
    mount(TextAnswer, { props: { ...props, prior: { answer: 'Zdravo!', correct: true } } })
    expect(speakClip).not.toHaveBeenCalled()
  })
})

describe('TextAnswer: what to type', () => {
  const tapQuestion = async (w: ReturnType<typeof mount>) => {
    await w.get('button[aria-label="Подсказка"]').trigger('click')
    return w.text()
  }

  it('has a «?» on every typed exercise that says what kind of answer is wanted', async () => {
    const cases: [string, string][] = [
      ['blank', 'только пропущенное слово'],
      ['word', 'одно слово'],
      ['phrase', 'фразу целиком'],
      ['sentence', 'предложение целиком'],
    ]
    for (const [shape, words] of cases) {
      const w = mount(TextAnswer, { props: { ...props, shape: shape as 'blank' }, global: { stubs: { RouterLink: true } } })
      expect(await tapQuestion(w)).toContain(words)
    }
  })

  it('puts the exercise’s own hint after the line about what to type', async () => {
    const w = mount(TextAnswer, { props: { ...props, shape: 'word' as const, explain: 'Начинается на š.' } })
    const text = await tapQuestion(w)
    expect(text).toContain('одно слово')
    expect(text).toContain('Начинается на š.')
  })

  it('opens the description on its own line below the prompt, in a dictation too', async () => {
    const w = mount(TextAnswer, { props: { ...props, type: 'listen' as const, prompt: 'Запиши, что слышишь.', audio: 'a.mp3', shape: 'phrase' as const } })
    await tapQuestion(w)
    const note = w.get('span.block.w-full')
    expect(note.text()).toContain('фразу целиком')
    // The prompt row wraps, so a full-width note drops under it instead of sitting beside the «?».
    expect(note.element.closest('.flex')!.className).toContain('flex-wrap')
  })

  it('has no «?» when nothing is known about the shape and there is no hint', () => {
    const w = mount(TextAnswer, { props })
    expect(w.find('button[aria-label="Подсказка"]').exists()).toBe(false)
  })
})

describe('TextAnswer: only a piece of the answer', () => {
  it('asks for the whole thing, grades nothing and keeps what was typed', async () => {
    const check = vi.spyOn(api, 'check').mockResolvedValue({ ok: false, format: 'too_short' })
    const w = mount(TextAnswer, { props: { ...props, type: 'fix_error' as const, prompt: 'Iz Rusije.', shape: 'sentence' as const }, global: { stubs: { RouterLink: true } } })
    await w.find('input').setValue('ja sam')
    await w.find('form').trigger('submit')
    await flushPromises()

    expect(check).toHaveBeenCalledTimes(1)
    expect(w.get('[data-test="too-short"]').text()).toContain('Перепиши предложение целиком')
    expect(w.emitted('graded')).toBeUndefined()
    expect((w.find('input').element as HTMLInputElement).value).toBe('ja sam')
    expect(w.text()).not.toContain('Не совсем')
    expect(w.text()).not.toContain('Верно')
  })

  it('says it for a translation too, and goes away as soon as the learner types more', async () => {
    vi.spyOn(api, 'check').mockResolvedValue({ ok: false, format: 'too_short' })
    const w = mount(TextAnswer, { props })
    await w.find('input').setValue('Hvala')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.get('[data-test="too-short"]').text()).toContain('часть ответа')

    await w.find('input').setValue('Hvala, ')
    expect(w.find('[data-test="too-short"]').exists()).toBe(false)
  })

  it('grades normally once the whole answer is sent', async () => {
    const check = vi.spyOn(api, 'check')
    check.mockResolvedValueOnce({ ok: false, format: 'too_short' })
    check.mockResolvedValueOnce({ ok: true, expected: 'Hvala, prijatno!' })
    const w = mount(TextAnswer, { props })
    await w.find('input').setValue('Hvala')
    await w.find('form').trigger('submit')
    await flushPromises()
    await w.find('input').setValue('Hvala, prijatno!')
    await w.find('form').trigger('submit')
    await flushPromises()
    expect(w.text()).toContain('Верно')
    expect(w.emitted('graded')?.[0]?.[0]).toBe(true)
  })
})
