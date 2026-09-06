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

export interface Lesson {
  id: string
  title: string
  subtitle: string
  planned: boolean
  markdown: string
  status: LessonStatus
}

export type ExerciseType = 'translate' | 'fill_blank' | 'fix_error' | 'conjugate' | 'free'

export interface Exercise {
  id: string
  type: ExerciseType
  prompt: string
  forms?: string[]
  meta?: string
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
  near_miss?: boolean
}

export interface CheckPayload {
  answer?: string
  answers?: string[]
  self?: boolean
}

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
}

export interface FalseFriend {
  id: string
  sr: string
  means: string
  not?: string
  correct?: string
  group: string
}

export interface ReviewCard {
  card_id: string
  kind: 'vocab' | 'ff'
  front: string
  cyrillic?: string
  back: string
  note?: string
  state: string
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
