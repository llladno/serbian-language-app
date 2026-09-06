import type {
  Course,
  Lesson,
  ExerciseBlock,
  CheckResult,
  CheckPayload,
  Vocab,
  FalseFriend,
  ReviewCard,
  GradeResult,
  Progress,
} from './types'

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch('/api' + path, {
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
    ...init,
  })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const body = await res.json()
      if (body?.error) msg = body.error
    } catch {
      /* keep statusText */
    }
    throw new ApiError(res.status, msg)
  }
  if (res.status === 204) return undefined as T
  const text = await res.text()
  return text ? (JSON.parse(text) as T) : (undefined as T)
}

function qs(params?: Record<string, string | undefined>): string {
  if (!params) return ''
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v) p.set(k, v)
  }
  const s = p.toString()
  return s ? '?' + s : ''
}

export const api = {
  course: () => request<Course>('/course'),
  lesson: (id: string) => request<Lesson>(`/lessons/${id}`),
  exercises: (id: string) => request<ExerciseBlock[]>(`/lessons/${id}/exercises`),
  check: (lesson: string, exId: string, payload: CheckPayload) =>
    request<CheckResult>(`/lessons/${lesson}/exercises/${exId}/check`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  completeLesson: (id: string) =>
    request<void>(`/lessons/${id}/complete`, { method: 'POST' }),
  vocab: (params?: { lesson?: string; tag?: string; q?: string }) =>
    request<Vocab[]>('/vocab' + qs(params)),
  falseFriends: (params?: { group?: string; q?: string }) =>
    request<FalseFriend[]>('/false-friends' + qs(params)),
  reviewQueue: () => request<ReviewCard[]>('/review/queue'),
  grade: (cardId: string, grade: number) =>
    request<GradeResult>('/review/grade', {
      method: 'POST',
      body: JSON.stringify({ card_id: cardId, grade }),
    }),
  progress: () => request<Progress>('/progress'),
}
