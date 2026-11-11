// The poster of the price change (PriceChangePoster) in a window, offered at the end of a lesson while the change is
// announced (migration 041). Like the feedback survey (lib/surveyModal.ts) it waits for a quiet moment and is rendered once, by
// AppNav; the lesson screen asks `offerAfterLesson`.
//
// Who sees it: a learner with no tariff (no running year, no paid level), at most once a day on this device, not
// after the very first lesson (that moment belongs to the swallow's tour) and not on top of the survey or a reward. The
// owner's own account (`priceChange.preview`) sees it every time, whatever else is true.
//
// It can be closed only a second after it opens: the cross is not there before (so a tap that was meant for the lesson
// screen's own buttons cannot close it unseen), and neither a tap beside the window nor Escape closes it until then.
import { ref } from 'vue'
import { useCourseStore } from '../stores/course'
import { useTariffsStore } from '../stores/tariffs'
import { trackTariff } from './tariffEvents'
import { useRewardModal } from './rewardModal'
import { useSurveyModal } from './surveyModal'

/** How long the window stays shut after it opens. */
export const CLOSABLE_AFTER_MS = 1000
/** How long after the lesson ends it opens: the figures of the summary count up first. */
const SHOW_DELAY_MS = 3200
const DAY_KEY = 'srpski.pricemodal.day'

const open = ref(false)
const closable = ref(false)
let showTimer: ReturnType<typeof setTimeout> | undefined
let closableTimer: ReturnType<typeof setTimeout> | undefined

// The learner's own calendar day: "once a day" means once for the day they see.
function today(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
function shownToday(): boolean {
  try {
    return localStorage.getItem(DAY_KEY) === today()
  } catch {
    return false
  }
}
function markShown() {
  try {
    localStorage.setItem(DAY_KEY, today())
  } catch {
    /* it may come again today: a lesson screen is not worth an error */
  }
}

/** Lessons the learner has finished, the one just ended included once the course has been read again. */
function finishedLessons(): number {
  return useCourseStore().course?.lessons.filter((l) => l.status === 'done').length ?? 0
}

function eligible(): boolean {
  const tariffs = useTariffsStore()
  const change = tariffs.campaign // empty for a learner who has a tariff, unless it is the owner's own account
  if (!tariffs.ready || !change) return false
  if (change.preview) return true
  return !shownToday() && finishedLessons() >= 2
}

function show() {
  // The quiet moment is checked when the time comes, not when the lesson ended: the survey opens a moment after it.
  if (open.value || useRewardModal().open.value || useSurveyModal().open.value || !eligible()) return
  markShown()
  closable.value = false
  open.value = true
  clearTimeout(closableTimer)
  closableTimer = setTimeout(() => (closable.value = true), CLOSABLE_AFTER_MS)
  trackTariff('view', 'price_modal')
}

async function offerAfterLesson() {
  const tariffs = useTariffsStore()
  await tariffs.load()
  const change = tariffs.campaign
  if (!tariffs.ready || !change || (!change.preview && shownToday())) return
  clearTimeout(showTimer)
  showTimer = setTimeout(show, SHOW_DELAY_MS)
}

function close() {
  if (!open.value || !closable.value) return
  open.value = false
  trackTariff('dismiss', 'price_modal')
}

/** The announcement is over (the prices have changed): the window goes without a word, whatever the second says. */
function endNow() {
  clearTimeout(showTimer)
  open.value = false
}

/** The gold button: the window goes (the button itself takes the learner to the tariffs). */
function go() {
  trackTariff('click', 'price_modal')
  open.value = false
}

export function usePriceModal() {
  return { open, closable, offerAfterLesson, close, go, endNow }
}
