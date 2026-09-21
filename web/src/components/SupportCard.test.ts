import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import SupportCard from './SupportCard.vue'
import { useSupportModal } from '../lib/supportModal'

beforeEach(() => {
  const { open, message, error, sent } = useSupportModal()
  open.value = false
  message.value = ''
  error.value = null
  sent.value = false
})

describe('SupportCard', () => {
  it('opens the shared support modal when clicked', async () => {
    const { open } = useSupportModal()
    const w = mount(SupportCard)

    expect(open.value).toBe(false)
    await w.find('[data-test="open-support"]').trigger('click')
    expect(open.value).toBe(true)
  })
})
