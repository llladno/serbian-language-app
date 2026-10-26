// Where the paid tariffs are shown, and where they are pressed: the log behind the admin's «Воронка тарифов»
// (server: POST /api/me/tariff-events, table tariff_events). Three places sell the tariffs: the banner at the top
// of the profile, the window that says today's «Повторить» is used up, and the window of a paid level when зёрнышек
// are short. Each reports a showing ('view') and a press on the way to the tariffs ('click').
//
// Best-effort analytics: it never throws and never waits, and an account the tariffs are closed to reports nothing
// (the server would answer 404).
import { api } from '../api'
import { useTariffsStore } from '../stores/tariffs'

export type TariffSource = 'profile_banner' | 'repeat_limit' | 'unlock_modal'

export function trackTariff(kind: 'view' | 'click', source: TariffSource): void {
  if (!useTariffsStore().enabled) return
  void api.trackTariffEvent(kind, source).catch(() => {})
}
