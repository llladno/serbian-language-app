export type LessonStatus = 'not_started' | 'in_progress' | 'done'

export interface Phase {
  id: string
  title: string
  lessons: string[]
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
  | 'free'
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
  options?: string[]
  bank?: string[]
  left?: string[]
  right?: string[]
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
  sample?: string
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
  self?: boolean
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
}

export interface FalseFriend {
  id: string
  sr: string
  means: string
  not?: string
  correct?: string
  group: string
  emoji?: string
  image?: string
}

export interface ReviewCard {
  card_id: string
  kind: 'vocab' | 'ff'
  front: string
  cyrillic?: string
  back: string
  note?: string
  state: string
  emoji?: string
  image?: string
  audio?: string
  preview: Record<'again' | 'hard' | 'good' | 'easy', number>
}

export interface GradeResult {
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
