export type LessonStatus = 'not_started' | 'in_progress' | 'done'

export interface Phase {
  id: string
  title: string
  lessons: string[]
  // A level the learner has not unlocked: its lessons cannot be opened.
  locked?: boolean
  // Only on a level that is for sale: what it costs now, and the full price
  // when a sale is on.
  price?: number
  price_effective?: number
  discount_percent?: number
}

export interface LessonRef {
  id: string
  title: string
  subtitle: string
  planned: boolean
  status: LessonStatus
}

export interface Course {
  title: string
  phases: Phase[]
  lessons: LessonRef[]
}

export type StepKind = 'teach' | 'practice' | 'reading' | 'checkpoint' | 'dialogue'

// Turn is one line of a dialogue step. For a "me" turn the server sends sr/ru
// only once the exercise has an attempt — before that the line would spoil the
// answer.
export interface Turn {
  who: 'npc' | 'me'
  sr?: string
  ru?: string
  audio?: string
  exercise_id?: string
}

export interface Step {
  id: string
  kind: StepKind
  title: string
  markdown?: string
  markdown_ru?: string
  exercise_ids?: string[]
  status: LessonStatus
  scene?: string
  voice?: 'f' | 'm'
  turns?: Turn[]
}

export interface Lesson {
  id: string
  title: string
  subtitle: string
  planned: boolean
  markdown: string
  reading?: string
  reading_ru?: string
  status: LessonStatus
  steps?: Step[]
}

export interface LookupResult {
  query: string
  matches: Vocab[]
  partial: boolean
}

export type ExerciseType =
  | 'translate'
  | 'fill_blank'
  | 'fix_error'
  | 'conjugate'
  | 'listen'
  | 'choice'
  | 'word_bank'
  | 'match'

export interface Exercise {
  id: string
  type: ExerciseType
  prompt: string
  forms?: string[]
  meta?: string
  audio?: string
  text?: string // listen, audio off: the Serbian sentence to translate
  options?: string[]
  bank?: string[]
  left?: string[]
  right?: string[]
  explain?: string
}

export interface ExerciseBlock {
  id: string
  title: string
  instruction?: string
  exercises: Exercise[]
}

export interface Chunk {
  text: string
  ok: boolean
}

export interface FormResult {
  ok: boolean
  expected: string
  diff?: Chunk[]
}

export interface CheckResult {
  ok: boolean
  diff?: Chunk[]
  expected?: string
  explain?: string
  forms?: FormResult[]
  match?: Record<string, boolean>
  near_miss?: boolean
  line?: string
  line_ru?: string
  line_audio?: string
}

export interface CheckPayload {
  answer?: string
  answers?: string[]
  pairs?: Record<string, string>
  ru?: boolean // listen, audio off: the answer is a Russian translation
}

export interface LessonAttempt {
  answer: string
  correct: boolean
}
export type LessonAttempts = Record<string, LessonAttempt>

export interface Vocab {
  id: string
  latin: string
  cyrillic: string
  transcription?: string
  ru: string
  note?: string
  lesson?: string
  pos?: string
  gender?: string
  aspect?: string
  tags?: string[]
  emoji?: string
  image?: string
  audio?: string
  example_sr?: string
  example_ru?: string
}

export interface FalseFriend {
  id: string
  sr: string
  transcription?: string
  means: string
  not?: string
  correct?: string
  group: string
  emoji?: string
  image?: string
}

export interface ReviewCard {
  card_id: string
  kind: 'vocab' | 'ff' | 'gram'
  front: string
  cyrillic?: string
  transcription?: string
  back: string
  note?: string
  state: string
  emoji?: string
  image?: string
  audio?: string
  example_sr?: string
  example_ru?: string
  // 4 shuffled candidate translations (one of them === back), sent only for
  // a first-encounter ("new") card — see ReviewView's recognition quiz.
  options?: string[]
  preview: Record<'again' | 'hard' | 'good' | 'easy', number>
  // kind === 'gram' only: one item picked at random from the card, to type
  // and have checked — see ReviewView's grammar drill and
  // api.gradeGram/GramCheckResult.
  item_prompt?: string
  item_index?: number
}

export interface GradeResult {
  due: string
  interval_days: number
  state: string
}

export interface GramCheckResult {
  ok: boolean
  expected?: string
  near_miss?: boolean
  due: string
  interval_days: number
  state: string
}

export interface PhaseProgress {
  id: string
  title: string
  done: number
  total: number
}

export interface WeakExercise {
  exercise_id: string
  lesson: string
  prompt: string
  wrong: number
  total: number
}

export interface RecentLesson {
  lesson: string
  title: string
  status: LessonStatus
}

export interface DayActivity {
  date: string
  count: number
}

export interface LeaderboardPage {
  rows: LeaderRow[]
  has_more: boolean
}

export interface LeaderboardMe {
  rank: number
  row: LeaderRow
}

export interface LeaderRow {
  name: string
  lessons_done: number
  lessons_total: number
  cards_known: number
  total_cards: number
  streak_days: number
  reviewed_today: number
  last_active: string
}

export interface Progress {
  phases: PhaseProgress[]
  srs: {
    due_today: number
    new_available: number
    reviewed_today: number
    total_cards: number
    known: number
  }
  weak_exercises: WeakExercise[]
  streak_days: number
  recent_lessons: RecentLesson[]
  activity: DayActivity[]
  daily_goal: number
}

export interface SessionUser {
  id: string
  name: string
  email: string
  email_verified: boolean
  telegram: { linked: boolean; username: string }
}

export interface SessionDevice {
  id: string
  user_agent: string
  last_seen_at: string
  current: boolean
}

export interface Me extends SessionUser {
  sessions: SessionDevice[]
}

export interface Health {
  status: string
  content_stale: boolean
  telegram_bot_id: string
}

export interface Notification {
  id: number
  text: string
  created_at: string
  read: boolean
}

export interface NotificationsResponse {
  items: Notification[]
  unread_count: number
}

export interface TelegramStart {
  url: string
  token: string
}

export type TelegramPoll =
  | { status: 'pending' }
  | { status: 'ok'; user?: SessionUser }
  | { status: 'error'; error: string }

// -- Экономика (зёрнышки) --

export interface Wallet {
  balance: number
  // Three Russian forms of the currency name, straight from the admin panel:
  // "1 зёрнышко", "2 зёрнышка", "5 зёрнышек". See lib/plural.ts.
  currency_one: string
  currency_few: string
  currency_many: string
  streak_days: number
}

export interface Quest {
  id: number
  kind: string
  title: string
  description: string
  target: number
  // Where the learner is now; the server recomputes it on every request, so
  // it is never behind what the ledger thinks.
  value: number
  reward: number
  done: boolean
  claimed: boolean
  // Where to go to do a quest that happens outside the app (the channel to
  // subscribe to). Absent once the quest is done.
  url?: string
  // On the subscription quest: this account has no Telegram linked, so the
  // subscription could not be seen however many times the learner subscribes.
  needs_telegram?: boolean
}

// The learner's own invite link. `url` is ready to share as it is.
export interface Referral {
  code: string
  url: string
  // Friends who joined through the link and confirmed their account.
  friends: number
}

export interface ClaimResult {
  reward: number
  balance: number
}

export interface PurchaseResult {
  paid: number
  balance: number
  replayed: boolean
}

export interface ShopItem {
  id: number
  kind: string
  ref: string
  title: string
  description: string
  price: number
  price_effective: number
  discount_percent?: number
  /** How many the learner already owns: 0 for an unlocked level is "not yet". */
  owned: number
}

export interface LessonCompletion {
  status: string
  // What finishing this lesson paid. 0 when the lesson has no reward of its
  // own, or when it had already been finished before.
  reward: number
  // Absent from an older server.
  stats?: LessonStats
}

// What the end-of-lesson screen shows.
export interface LessonStats {
  // Exercises answered at all, and how many of them were missed the first time.
  answered: number
  mistakes: number
  // Share answered right the first time, 0..100. Meaningless when answered is 0.
  percent: number
  // Words the lesson introduces.
  new_words: number
}
