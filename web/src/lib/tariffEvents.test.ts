import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { api, ApiError } from '../api'
import { trackTariff } from './tariffEvents'
import { enableTariffs, sessionUser } from '../test/tariffsFixture'
import { useSessionStore } from '../stores/session'

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('trackTariff', () => {
  it('reports a showing or a press for an account that is shown the tariffs', () => {
    enableTariffs()
    const spy = vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
    trackTariff('view', 'profile_banner')
    trackTariff('click', 'unlock_modal')
    expect(spy).toHaveBeenNthCalledWith(1, 'view', 'profile_banner')
    expect(spy).toHaveBeenNthCalledWith(2, 'click', 'unlock_modal')
  })

  it('stays silent for an account the tariffs are closed to: the server would answer 404', () => {
    useSessionStore().user = sessionUser(false)
    const spy = vi.spyOn(api, 'trackTariffEvent').mockResolvedValue(undefined)
    trackTariff('view', 'profile_banner')
    expect(spy).not.toHaveBeenCalled()
  })

  it('never lets a failed report reach the learner', async () => {
    enableTariffs()
    vi.spyOn(api, 'trackTariffEvent').mockRejectedValue(new ApiError(500, 'internal error'))
    expect(() => trackTariff('view', 'repeat_limit')).not.toThrow()
    await Promise.resolve() // an unhandled rejection would fail the run
  })
})
