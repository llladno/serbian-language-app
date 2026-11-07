// A test of two wordings, for the learners who open the window of a paid level with зёрнышки waiting in the quests:
// a button «Забрать N», or a quiet line in place of «Заработать зёрнышки» («Вас ждут N — забрать»). Half and half, by the
// learner's id, so a learner meets the same one every time and the two groups can be told apart afterwards: the window
// reports `v=button` or `v=link` with its showing and its press (tariff_events, kinds claim_view and claim).
export type ClaimVariant = 'button' | 'link'

// Only the dev build honours the stored override, to look at the other wording without finding an id that lands there:
// localStorage.setItem('srpski.claimVariant', 'link' | 'button').
const OVERRIDE_KEY = 'srpski.claimVariant'

// The id is mixed with the finalizer of MurmurHash3 before one bit is taken: the low bit of a plain FNV hash only
// counts the odd characters of the id, which would not split a run of similar ids evenly.
function mix(id: string): number {
  let h = 2166136261
  for (let i = 0; i < id.length; i++) {
    h ^= id.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  h ^= h >>> 16
  h = Math.imul(h, 0x85ebca6b)
  h ^= h >>> 13
  h = Math.imul(h, 0xc2b2ae35)
  h ^= h >>> 16
  return h >>> 0
}

export function claimVariantFor(userId: string): ClaimVariant {
  if (import.meta.env.DEV) {
    try {
      const forced = localStorage.getItem(OVERRIDE_KEY)
      if (forced === 'button' || forced === 'link') return forced
    } catch {
      // storage closed: the split by id stands
    }
  }
  return mix(userId) % 2 === 0 ? 'button' : 'link'
}
