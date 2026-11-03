// Where the paid tariffs are shown, and where they are pressed: the log behind the admin's «Воронка тарифов»
// (server: POST /api/me/tariff-events, table tariff_events). Three places sell the tariffs: the banner at the top
// of the profile, the window that says today's «Повторить» is used up, and the windows of a paid level and of a paid story when зёрнышек
// are short. Each reports a showing ('view') and a press on the way to the tariffs ('click'), being closed without going on ('dismiss') and,
// in the windows that name a free way out, taking it ('earn'). The tariffs screen reports the steps of paying under 'checkout' (open, press,
// receipt_view / receipt_send / receipt_cancel; the server adds 'error' for an order it refused). `detail` is a few words on what it was about
// ("phase:2" for the level a window was opened for, "year" for the tariff pressed).
//
// Best-effort analytics: it never throws and never waits, and an account the tariffs are closed to reports nothing
// (the server would answer 404).
import { api } from '../api'
import { useTariffsStore } from '../stores/tariffs'

export type TariffSource = 'profile_banner' | 'repeat_limit' | 'unlock_modal' | 'story_modal'
export type CheckoutKind = 'open' | 'press' | 'receipt_view' | 'receipt_send' | 'receipt_cancel'

export function trackTariff(kind: 'view' | 'click' | 'dismiss' | 'earn', source: TariffSource, detail = ''): void {
  if (!useTariffsStore().enabled) return
  void (detail ? api.trackTariffEvent(kind, source, detail) : api.trackTariffEvent(kind, source)).catch(() => {})
}

/** A step of paying, on the tariffs screen. */
export function trackCheckout(kind: CheckoutKind, detail = ''): void {
  if (!useTariffsStore().enabled) return
  void api.trackTariffEvent(kind, 'checkout', detail).catch(() => {})
}
