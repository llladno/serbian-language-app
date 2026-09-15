import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import MatchAnswer from './MatchAnswer.vue'
import { api } from '../../api'

afterEach(() => vi.restoreAllMocks())

const props = {
  lesson: '01',
  exerciseId: '01.2.5',
  prompt: 'Соедини приветствие и перевод',
  left: ['Zdravo', 'Hvala'],
  right: ['Привет', 'Спасибо'],
}

function find(w: ReturnType<typeof mount>, text: string) {
  return w.findAll('button').find((b) => b.text() === text)!
}

describe('MatchAnswer', () => {
  it('pairs a tapped left word with the next tapped right word, in either order', async () => {
    const w = mount(MatchAnswer, { props })

    // left then right
    await find(w, 'Zdravo').trigger('click')
    await find(w, 'Привет').trigger('click')
    // right then left
    await find(w, 'Спасибо').trigger('click')
    await find(w, 'Hvala').trigger('click')

    const check = vi.spyOn(api, 'check').mockResolvedValue({ ok: true, match: { Zdravo: true, Hvala: true } })
    await find(w, 'Проверить').trigger('click')
    await flushPromises()

    expect(check).toHaveBeenCalledWith('01', '01.2.5', { pairs: { Zdravo: 'Привет', Hvala: 'Спасибо' } })
    expect(w.emitted('graded')?.[0]?.[0]).toBe(true)
  })

  it('keeps Проверить disabled until every word is paired, and re-tapping a pair frees it', async () => {
    const w = mount(MatchAnswer, { props })

    await find(w, 'Zdravo').trigger('click')
    await find(w, 'Привет').trigger('click')
    expect(find(w, 'Проверить').attributes('disabled')).toBeDefined()

    // tapping the paired left word again should free it up, not submit
    await find(w, 'Zdravo').trigger('click')
    await find(w, 'Hvala').trigger('click')
    await find(w, 'Спасибо').trigger('click')
    expect(find(w, 'Проверить').attributes('disabled')).toBeDefined()

    await find(w, 'Zdravo').trigger('click')
    await find(w, 'Привет').trigger('click')
    expect(find(w, 'Проверить').attributes('disabled')).toBeUndefined()
  })
})
