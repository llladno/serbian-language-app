import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SurveyModal from './SurveyModal.vue'
import { useSurveyModal } from '../lib/surveyModal'
import { useRewardModal } from '../lib/rewardModal'
import { api } from '../api'

function mountModal() {
  return mount(SurveyModal, { global: { stubs: { teleport: true } } })
}

beforeEach(() => {
  setActivePinia(createPinia())
  const s = useSurveyModal()
  s.open.value = true
  s.reward.value = 10
  s.step.value = 0
  s.rating.value = 0
  s.likes.value = []
  s.pains.value = []
  s.painText.value = ''
  s.purpose.value = ''
  s.wish.value = ''
  useRewardModal().closeModal()
  vi.spyOn(api, 'wallet').mockRejectedValue(new Error('offline'))
})
afterEach(() => vi.restoreAllMocks())

describe('SurveyModal', () => {
  it('names the reward on the greeting', () => {
    const w = mountModal()
    expect(w.find('[data-test="survey-intro"]').text()).toContain('+10')
  })

  it('will not go past a question until it is answered', async () => {
    const w = mountModal()
    await w.find('[data-test="survey-start"]').trigger('click')
    expect(w.find('[data-test="survey-next"]').attributes('disabled')).toBeDefined()
    await w.find('[data-test="rating-4"]').trigger('click')
    expect(w.find('[data-test="survey-next"]').attributes('disabled')).toBeUndefined()
  })

  it('"nothing bothers me" clears the other complaints', async () => {
    const s = useSurveyModal()
    s.togglePain('hard')
    s.togglePain('bugs')
    s.togglePain('nothing')
    expect(s.pains.value).toEqual(['nothing'])
    s.togglePain('hard')
    expect(s.pains.value).toEqual(['hard'])
  })

  it('sends every answer and celebrates the reward', async () => {
    const send = vi.spyOn(api, 'answerSurvey').mockResolvedValue({ status: 'ok', reward: 10 })
    const s = useSurveyModal()
    s.step.value = 5
    s.rating.value = 4
    s.likes.value = ['lessons']
    s.pains.value = ['hard']
    s.painText.value = '  быстро  '
    s.purpose.value = 'move'
    const w = mountModal()
    await w.find('[data-test="wish-speaking"]').trigger('click')
    await w.find('[data-test="survey-submit"]').trigger('click')
    await flushPromises()
    expect(send).toHaveBeenCalledWith({
      rating: 4, likes: ['lessons'], pains: ['hard'], pain_text: 'быстро', purpose: 'move', wish: 'speaking',
    })
    expect(s.open.value).toBe(false)
    expect(useRewardModal().open.value).toBe(true)
  })

  it('closing the window counts as dismissing and pays nothing', async () => {
    const dismiss = vi.spyOn(api, 'dismissSurvey').mockResolvedValue({ status: 'ok' })
    const w = mountModal()
    await w.find('[data-test="survey-close"]').trigger('click')
    await flushPromises()
    expect(dismiss).toHaveBeenCalled()
    expect(useSurveyModal().open.value).toBe(false)
    expect(useRewardModal().open.value).toBe(false)
  })
})
