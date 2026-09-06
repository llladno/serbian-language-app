import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SerbianKeys from './SerbianKeys.vue'

describe('SerbianKeys', () => {
  it('inserts the letter at the caret of the focused input', async () => {
    const input = document.createElement('input')
    document.body.appendChild(input)
    input.value = 'zdrao'
    input.focus()
    input.setSelectionRange(4, 4) // between 'a' and 'o'

    const w = mount(SerbianKeys, { attachTo: document.body })
    const key = w.findAll('button').find((b) => b.text() === 'č')!
    await key.trigger('pointerdown')

    expect(input.value).toBe('zdračo')
    expect(input.selectionStart).toBe(5)

    input.remove()
    w.unmount()
  })

  it('does nothing when no text field is focused', async () => {
    document.body.focus()
    const w = mount(SerbianKeys, { attachTo: document.body })
    await w.findAll('button')[0].trigger('pointerdown')
    // no throw, nothing to assert beyond that
    w.unmount()
  })

  it('replaces the selection', async () => {
    const ta = document.createElement('textarea')
    document.body.appendChild(ta)
    ta.value = 'aXc'
    ta.focus()
    ta.setSelectionRange(1, 2) // select 'X'

    const w = mount(SerbianKeys, { attachTo: document.body })
    await w.findAll('button').find((b) => b.text() === 'ž')!.trigger('pointerdown')

    expect(ta.value).toBe('ažc')

    ta.remove()
    w.unmount()
  })
})
