import { describe, it, expect, afterEach, beforeEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import RewardModal from './RewardModal.vue'
import { useRewardModal } from '../lib/rewardModal'

enableAutoUnmount(afterEach)
const stubs = { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } }

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => useRewardModal().closeModal())

describe('RewardModal', () => {
  it('closes with the cross, which sits above the picture that is drawn after it', () => {
    mount(RewardModal, { global: { stubs }, attachTo: document.body })
    useRewardModal().celebrate(5, 'Пройти 5 уроков')
    return Promise.resolve().then(async () => {
      const cross = document.body.querySelector<HTMLButtonElement>('button[aria-label="Закрыть"]')!
      // the picture box after it is positioned too, and a later positioned box paints over an earlier one with the
      // same z-index, so the cross would be covered and the tap would go to the picture (seen on the phone, 2026-11-10)
      expect(cross.classList.contains('z-10')).toBe(true)
      cross.click()
      expect(useRewardModal().open.value).toBe(false)
    })
  })
})
