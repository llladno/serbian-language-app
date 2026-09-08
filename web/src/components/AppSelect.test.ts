import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AppSelect from './AppSelect.vue'

describe('AppSelect', () => {
  it('opens on click, emits the picked option, and closes', async () => {
    const w = mount(AppSelect, { props: { modelValue: '', options: ['брат', 'мама'] } })
    expect(w.text()).toContain('— выбери —')

    await w.find('button').trigger('click')
    const opts = w.findAll('button').filter((b) => ['брат', 'мама'].includes(b.text()))
    expect(opts).toHaveLength(2)

    await opts[1].trigger('click')
    expect(w.emitted('update:modelValue')?.[0]).toEqual(['мама'])
    expect(w.findAll('button').filter((b) => b.text() === 'брат')).toHaveLength(0) // closed
  })

  it('does not open when disabled', async () => {
    const w = mount(AppSelect, { props: { modelValue: '', options: ['a'], disabled: true } })
    await w.find('button').trigger('click')
    expect(w.findAll('button').filter((b) => b.text() === 'a')).toHaveLength(0)
  })
})
