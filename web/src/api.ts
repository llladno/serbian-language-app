import type {
  Course,
  Lesson,
  ExerciseBlock,
  CheckResult,
  CheckPayload,
  Vocab,
  LookupResult,
  FalseFriend,
  ReviewCard,
  GradeResult,
  Progress,
  LessonAttempts,
  LeaderRow,
  SessionUser,
  Me,
  Health,
  TelegramStart,
  TelegramPoll,
} from './types'
import { getStoredAttribution } from './attribution'

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {}
  if (init?.body) headers['Content-Type'] = 'application/json'

  const res = await fetch('/api' + path, { ...init, headers, credentials: 'same-origin' })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const body = await res.json()
      if (body?.error) msg = body.error
    } catch {
      /* keep statusText */
    }
    // A 401 from an /auth/* endpoint is a normal, expected response (wrong
    // password, no session yet on GET /auth/session) — the caller handles it.
    // A 401 from anything else means a previously-live session just died;
    // clear it and send the visitor to log back in. Dynamic imports avoid a
    // static import cycle (router -> views -> api -> router / session).
    if (res.status === 401 && !path.startsWith('/auth/')) {
      const [{ useSessionStore }, { default: router }] = await Promise.all([
        import('./stores/session'),
        import('./router'),
      ])
      const session = useSessionStore()
      session.user = null
      if (router.currentRoute.value.path !== '/login') {
        router.push({ path: '/login', query: { next: router.currentRoute.value.fullPath } })
      }
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
  health: () => request<Health>('/health'),
  register: (email: string, password: string, name: string) =>
    request<{ status: string }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, name, ...getStoredAttribution() }),
    }),
  login: (email: string, password: string) =>
    request<SessionUser>('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
  logout: () => request<void>('/auth/logout', { method: 'POST' }),
  logoutAll: () => request<void>('/auth/logout-all', { method: 'POST' }),
  session: () => request<SessionUser>('/auth/session'),
  resendVerification: (email: string) =>
    request<{ status: string }>('/auth/resend-verification', {
      method: 'POST',
      body: JSON.stringify({ email }),
    }),
  forgotPassword: (email: string) =>
    request<{ status: string }>('/auth/forgot', { method: 'POST', body: JSON.stringify({ email }) }),
  resetPassword: (token: string, password: string) =>
    request<SessionUser>('/auth/reset', { method: 'POST', body: JSON.stringify({ token, password }) }),
  telegramLogin: (payload: Record<string, unknown>) =>
    request<SessionUser>('/auth/telegram', {
      method: 'POST',
      body: JSON.stringify({ ...payload, ...getStoredAttribution() }),
    }),
  telegramLoginStart: () => request<TelegramStart>('/auth/telegram/start', { method: 'POST' }),
  telegramPoll: (token: string) => request<TelegramPoll>('/auth/telegram/poll?token=' + encodeURIComponent(token)),

  getMe: () => request<Me>('/me'),
  patchMe: (name: string) => request<SessionUser>('/me', { method: 'PATCH', body: JSON.stringify({ name }) }),
  setPassword: (payload: { current?: string; new: string; email?: string }) =>
    request<{ status: string }>('/me/password', { method: 'POST', body: JSON.stringify(payload) }),
  linkTelegram: (payload: Record<string, unknown>) =>
    request<SessionUser>('/me/link/telegram', { method: 'POST', body: JSON.stringify(payload) }),
  telegramLinkStart: () => request<TelegramStart>('/me/telegram/start', { method: 'POST' }),
  unlinkTelegram: () => request<void>('/me/telegram', { method: 'DELETE' }),
  deleteSession: (id: string) => request<void>(`/me/sessions/${id}`, { method: 'DELETE' }),
  deleteMe: (password?: string) =>
    request<void>('/me', { method: 'DELETE', body: JSON.stringify({ password }) }),
  sendSupportMessage: (message: string) =>
    request<{ status: string }>('/me/support', { method: 'POST', body: JSON.stringify({ message }) }),

  course: () => request<Course>('/course'),
  lesson: (id: string) => request<Lesson>(`/lessons/${id}`),
  exercises: (id: string) => request<ExerciseBlock[]>(`/lessons/${id}/exercises`),
  lessonAttempts: (id: string) => request<LessonAttempts>(`/lessons/${id}/attempts`),
  check: (lesson: string, exId: string, payload: CheckPayload) =>
    request<CheckResult>(`/lessons/${lesson}/exercises/${exId}/check`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  setStepStatus: (lesson: string, step: string, status: 'in_progress' | 'done') =>
    request<void>(`/lessons/${lesson}/steps/${encodeURIComponent(step)}`, {
      method: 'POST',
      body: JSON.stringify({ status }),
    }),
  completeLesson: (id: string) => request<void>(`/lessons/${id}/complete`, { method: 'POST' }),
  resetLesson: (id: string) => request<void>(`/lessons/${id}/reset`, { method: 'POST' }),
  resetExercises: () => request<void>('/reset-exercises', { method: 'POST' }),
  vocab: (params?: { lesson?: string; tag?: string; q?: string }) =>
    request<Vocab[]>('/vocab' + qs(params)),
  falseFriends: (params?: { group?: string; q?: string }) =>
    request<FalseFriend[]>('/false-friends' + qs(params)),
  lookup: (q: string) => request<LookupResult>('/lookup' + qs({ q })),
  reviewQueue: () => request<ReviewCard[]>('/review/queue'),
  grade: (cardId: string, grade: number) =>
    request<GradeResult>('/review/grade', {
      method: 'POST',
      body: JSON.stringify({ card_id: cardId, grade }),
    }),
  addToReview: (vocabId: string) =>
    request<{ status: 'added' | 'already' }>('/review/add', {
      method: 'POST',
      body: JSON.stringify({ vocab_id: vocabId }),
    }),
  progress: () => request<Progress>('/progress'),
  leaderboard: () => request<LeaderRow[]>('/leaderboard'),
}
