# Auth Frontend (Part 3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the name-based, no-password login with the real session-cookie
auth the backend (Part 2, merged) already serves: login/register/verify/forgot/reset
screens, a personal `/profile` page replacing the dashboard, and dormant-until-configured
Telegram sign-in (Mini App silent auto-login + a real Login Widget button).

**Architecture:** A new Pinia `session` store owns `{user, loading}` and wraps every
`/api/auth/*` and `/api/me*` call; `api.ts` drops the `X-User` header entirely and
relies on the `HttpOnly` session cookie (`credentials: 'same-origin'`). The router
gains a `beforeEach` guard that waits for the initial session fetch, then routes
based on `user` + `email_verified`. The whole app is built and shipped as ONE
cutover (final task) once every new screen exists standalone; earlier tasks only
add new, unreferenced files so the app keeps building and passing tests after
every commit.

**Tech Stack:** Vue 3 `<script setup>`, Pinia, vue-router 4, Vitest + `@vue/test-utils`,
Tailwind (existing `.card`/`.btn`/`.field` utility classes, `--accent` etc. CSS vars).
No new npm dependencies.

**Spec:** `docs/superpowers/specs/2026-09-09-auth-postgres-profile-design.md`
(Section 4 "Фронтенд" is authoritative for scope; Sections 3.2/3.3 for the exact
backend contract this plan consumes — that backend is done, reviewed, and live at
HEAD `01a403b`).

## Global Constraints

- **Backend is done and frozen for this plan.** Every `/api/auth/*` and `/api/me*`
  endpoint, request/response shape, and error code below is copied verbatim from
  the shipped code (`server/internal/api/{auth,me,api,dto,middleware}.go`) — do
  not invent fields that aren't there and do not change backend behavior except
  where a task explicitly says so (only Task 5 touches Go code, and only to add
  one field to the already-public `/api/health`).
- **The `X-User` bridge stays server-side for one more release** (spec rollout
  step 5, a separate deploy gated on confirming this frontend is live in prod).
  This plan's job is to stop the *frontend* from sending it (Task 1) — do not
  remove `requireAuth`'s bridge branch in `server/internal/api/middleware.go`.
- **The session cookie is `HttpOnly`.** The frontend can never read it directly;
  all session state comes from `GET /api/auth/session` / the auth-endpoint
  response bodies. Every `fetch` to `/api/*` must pass `credentials: 'same-origin'`.
- **Error codes vs. messages:** the backend mixes real Russian sentences (e.g.
  the login 401 body `"неверная почта или пароль"`) with literal English codes
  meant for the client to interpret (`"wrong_password"`, `"email_unverified"`,
  `"invalid_token"`, `"telegram_disabled"`, …). Never render a raw backend error
  string that isn't already a finished Russian sentence — map codes through
  `web/src/lib/authErrors.ts` (Task 1). This app's UI text is Russian throughout
  (see `CLAUDE.md`) and that must hold for every new screen.
- **Task ordering matters and must not be reshuffled:** Tasks 1–4 only add new,
  currently-unreferenced files (or modify `api.ts`/`types.ts` in ways nothing
  else depends on yet) — the app must build, type-check, and pass `npm test`
  after every single task, including these. Task 5 (Telegram) modifies files
  from Tasks 1 and 2. **Task 6 is the one atomic cutover** — router, `App.vue`,
  `AppNav.vue`, the `/people`→`/rating` rename, and deleting `account.ts` /
  `LoginGate.vue` / `DashboardView.vue` all land together, because none of
  those seven files can be independently reviewed without the others (flipping
  the whole app from name-based to session-based auth is one behavior change).
- **CSP already allows what Task 5 needs.** `script-src 'self' https://telegram.org`
  was added in commit `01a403b` for the Mini App SDK — the Login Widget script
  (`https://telegram.org/js/telegram-widget.js`) is same-origin-to-that-directive
  and needs no further CSP change.
- **No new npm packages.** Everything (including the Telegram widget) is loaded
  via a plain `<script>` tag or built from what's already a dependency.
- Go: `go build`/`go vet`/`go test ./server/...` must stay green after Task 5.
  Frontend: `cd web && npx vue-tsc -b --noEmit` (type-check) and `npm test` must
  stay green after every task — **with one known, ruled exception:** from
  Task 1 through Task 6, `vue-tsc` reports exactly two errors in
  `LoginGate.vue` (`api.listAccounts`/`api.createAccount` no longer exist —
  Task 1 removes them per this plan). This is expected and already ruled on
  (see the ledger): those two backend routes (`GET`/`POST /api/users`) were
  already removed and 404ing since Part 2, so the calls were already dead at
  runtime; `LoginGate.vue` is deleted outright in Task 6, which is what
  actually resolves the type error. Do not "fix" `LoginGate.vue` before
  Task 6 — that would mean redesigning login-screen UX that Task 6 owns.
  Every task's type-check step should see *only* those same two errors; any
  other `vue-tsc` error is a real regression.

---

## Task 1: Session store, `api.ts` rewrite, error-code mapping

**Files:**
- Create: `web/src/stores/session.ts`
- Create: `web/src/stores/session.test.ts`
- Create: `web/src/lib/authErrors.ts`
- Create: `web/src/lib/authErrors.test.ts`
- Modify: `web/src/api.ts` (full rewrite of the request layer and the endpoint list)
- Modify: `web/src/api.test.ts` (drop the `X-User` test, add credential/401 tests)
- Modify: `web/src/types.ts` (add `SessionUser`, `SessionDevice`, `Me`)

**Interfaces:**
- Produces: `useSessionStore()` → `{ user: Ref<SessionUser|null>, loading: Ref<boolean>,
  fetchSession(), login(email,password), register(email,password,name), logout(),
  logoutAll(), forgot(email), reset(token,password), resendVerification(email) }`.
- Produces: `api.register/login/logout/logoutAll/session/resendVerification/
  forgotPassword/resetPassword/telegramLogin/getMe/patchMe/setPassword/
  linkTelegram/unlinkTelegram/deleteSession/deleteMe` — exact signatures in the
  code below. `Task 5` adds `api.health()` later; do not add it here.
- Produces: `authErrorMessage(e: unknown): string`, `isEmailUnverified(e: unknown): boolean`
  from `lib/authErrors.ts` — every later auth screen imports these.
- Consumes: nothing new (this task is the foundation).

- [ ] **Step 1: Add the new types**

Append to `web/src/types.ts`:

```ts
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
```

- [ ] **Step 2: Rewrite `web/src/api.ts`**

Replace the whole file:

```ts
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
  register: (email: string, password: string, name: string) =>
    request<{ status: string }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, name }),
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
    request<SessionUser>('/auth/telegram', { method: 'POST', body: JSON.stringify(payload) }),

  getMe: () => request<Me>('/me'),
  patchMe: (name: string) => request<SessionUser>('/me', { method: 'PATCH', body: JSON.stringify({ name }) }),
  setPassword: (payload: { current?: string; new: string; email?: string }) =>
    request<{ status: string }>('/me/password', { method: 'POST', body: JSON.stringify(payload) }),
  linkTelegram: (payload: Record<string, unknown>) =>
    request<SessionUser>('/me/link/telegram', { method: 'POST', body: JSON.stringify(payload) }),
  unlinkTelegram: () => request<void>('/me/telegram', { method: 'DELETE' }),
  deleteSession: (id: string) => request<void>(`/me/sessions/${id}`, { method: 'DELETE' }),
  deleteMe: (password?: string) =>
    request<void>('/me', { method: 'DELETE', body: JSON.stringify({ password }) }),

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
```

Note what's gone: the `getAccount`/`clearAccount` import, the `X-User` header,
`listAccounts`, `createAccount`. `web/src/account.ts` itself is **not** deleted
yet — `App.vue`, `LoginGate.vue`, `AppNav.vue`, `PeopleView.vue` and
`DashboardView.vue` still import it and must keep building until Task 6.

- [ ] **Step 3: Create `web/src/lib/authErrors.ts`**

```ts
// Maps the backend's error vocabulary to Russian UI text. auth.go/me.go mix a
// few already-Russian sentences (the login 401) with literal English codes
// meant for the client to switch on ("wrong_password", "email_unverified").
// Anything unrecognized falls back to a generic message — never leak an
// English backend string into this Russian-only UI.
import { ApiError } from '../api'

const MESSAGES: Record<string, string> = {
  'invalid email': 'Некорректный email',
  'password must be 8 to 128 characters': 'Пароль должен быть от 8 до 128 символов',
  'password must be 8-128 characters': 'Пароль должен быть от 8 до 128 символов',
  'name must be 1 to 40 characters': 'Имя должно быть от 1 до 40 символов',
  'bad request body': 'Некорректный запрос',
  'too many attempts': 'Слишком много попыток, попробуйте позже',
  wrong_password: 'Неверный пароль',
  email_required: 'Укажите email',
  invalid_token: 'Ссылка недействительна или устарела',
  reauth_required: 'Нужен свежий вход — войдите ещё раз и повторите',
  telegram_disabled: 'Вход через Telegram пока не настроен',
  bad_telegram_auth: 'Не удалось подтвердить вход через Telegram',
  telegram_taken: 'Этот Telegram уже привязан к другому аккаунту',
  only_login_method: 'Это единственный способ входа — сначала задайте пароль',
  not_linked: 'Telegram не привязан',
  'no session': 'Сессия истекла — войдите снова',
  'session required': 'Сессия истекла — войдите снова',
  'internal error': 'Что-то пошло не так, попробуйте ещё раз',
}

const FALLBACK = 'Что-то пошло не так, попробуйте ещё раз'

export function authErrorMessage(e: unknown): string {
  if (e instanceof ApiError) {
    // login's bad-credentials body is already a finished Russian sentence
    // (badCredentials in auth.go) — pass anything Cyrillic through untouched.
    if (/[а-яё]/i.test(e.message)) return e.message
    return MESSAGES[e.message] ?? FALLBACK
  }
  return FALLBACK
}

// email_unverified is routed to /verify rather than shown inline, so callers
// check for it before falling back to authErrorMessage.
export function isEmailUnverified(e: unknown): boolean {
  return e instanceof ApiError && e.status === 403 && e.message === 'email_unverified'
}
```

- [ ] **Step 4: Create `web/src/lib/authErrors.test.ts`**

```ts
import { describe, it, expect } from 'vitest'
import { authErrorMessage, isEmailUnverified } from './authErrors'
import { ApiError } from '../api'

describe('authErrorMessage', () => {
  it('maps a known English code to Russian', () => {
    expect(authErrorMessage(new ApiError(403, 'wrong_password'))).toBe('Неверный пароль')
  })
  it('passes an already-Russian message through untouched', () => {
    expect(authErrorMessage(new ApiError(401, 'неверная почта или пароль'))).toBe(
      'неверная почта или пароль',
    )
  })
  it('falls back to a generic message for an unrecognized code', () => {
    expect(authErrorMessage(new ApiError(500, 'some new backend string'))).toBe(
      'Что-то пошло не так, попробуйте ещё раз',
    )
  })
  it('falls back for a non-ApiError', () => {
    expect(authErrorMessage(new Error('network down'))).toBe('Что-то пошло не так, попробуйте ещё раз')
  })
})

describe('isEmailUnverified', () => {
  it('is true only for the exact 403 email_unverified shape', () => {
    expect(isEmailUnverified(new ApiError(403, 'email_unverified'))).toBe(true)
    expect(isEmailUnverified(new ApiError(401, 'email_unverified'))).toBe(false)
    expect(isEmailUnverified(new ApiError(403, 'wrong_password'))).toBe(false)
  })
})
```

- [ ] **Step 5: Create `web/src/stores/session.ts`**

```ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'
import type { SessionUser } from '../types'

export const useSessionStore = defineStore('session', () => {
  const user = ref<SessionUser | null>(null)
  const loading = ref(true)

  async function fetchSession() {
    loading.value = true
    try {
      user.value = await api.session()
    } catch {
      user.value = null
    } finally {
      loading.value = false
    }
  }

  async function login(email: string, password: string) {
    user.value = await api.login(email, password)
  }

  async function register(email: string, password: string, name: string) {
    await api.register(email, password, name)
  }

  async function logout() {
    await api.logout()
    user.value = null
  }

  async function logoutAll() {
    await api.logoutAll()
    user.value = null
  }

  async function forgot(email: string) {
    await api.forgotPassword(email)
  }

  async function reset(token: string, password: string) {
    user.value = await api.resetPassword(token, password)
  }

  async function resendVerification(email: string) {
    await api.resendVerification(email)
  }

  return {
    user,
    loading,
    fetchSession,
    login,
    register,
    logout,
    logoutAll,
    forgot,
    reset,
    resendVerification,
  }
})
```

- [ ] **Step 6: Create `web/src/stores/session.test.ts`**

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from './session'
import { api, ApiError } from '../api'
import type { SessionUser } from '../types'

function user(overrides: Partial<SessionUser> = {}): SessionUser {
  return {
    id: 'usr_1',
    name: 'Гриша',
    email: 'g@example.com',
    email_verified: true,
    telegram: { linked: false, username: '' },
    ...overrides,
  }
}

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('useSessionStore', () => {
  it('fetchSession fills user on success', async () => {
    vi.spyOn(api, 'session').mockResolvedValue(user())
    const s = useSessionStore()
    expect(s.loading).toBe(true)
    await s.fetchSession()
    expect(s.loading).toBe(false)
    expect(s.user?.name).toBe('Гриша')
  })

  it('fetchSession clears user on a 401', async () => {
    vi.spyOn(api, 'session').mockRejectedValue(new ApiError(401, 'no session'))
    const s = useSessionStore()
    await s.fetchSession()
    expect(s.user).toBeNull()
    expect(s.loading).toBe(false)
  })

  it('login sets user from the response', async () => {
    vi.spyOn(api, 'login').mockResolvedValue(user({ name: 'Алина' }))
    const s = useSessionStore()
    await s.login('a@example.com', 'secret123')
    expect(s.user?.name).toBe('Алина')
  })

  it('logout clears user', async () => {
    vi.spyOn(api, 'logout').mockResolvedValue(undefined)
    const s = useSessionStore()
    s.user = user()
    await s.logout()
    expect(s.user).toBeNull()
  })

  it('register does not touch user state', async () => {
    vi.spyOn(api, 'register').mockResolvedValue({ status: 'ok' })
    const s = useSessionStore()
    await s.register('a@example.com', 'secret123', 'Аня')
    expect(s.user).toBeNull()
  })

  it('reset autologs in from the response', async () => {
    vi.spyOn(api, 'resetPassword').mockResolvedValue(user({ name: 'Гриша' }))
    const s = useSessionStore()
    await s.reset('tok', 'newpass123')
    expect(s.user?.name).toBe('Гриша')
  })
})
```

- [ ] **Step 7: Rewrite `web/src/api.test.ts`**

Replace the whole file (drops the `X-User` test, adds credentials + 401-redirect coverage):

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { api, ApiError } from './api'

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.unstubAllGlobals())

describe('api', () => {
  it('parses JSON on 200', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ title: 'X', phases: [], lessons: [] }), { status: 200 }),
      ),
    )
    const c = await api.course()
    expect(c.title).toBe('X')
  })

  it('throws ApiError with server message on non-2xx', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'nope' }), { status: 404 })),
    )
    await expect(api.lesson('zz')).rejects.toMatchObject({ status: 404, message: 'nope' })
    await expect(api.lesson('zz')).rejects.toBeInstanceOf(ApiError)
  })

  it('builds query strings for vocab filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('[]', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await api.vocab({ lesson: '02', q: 'raditi' })
    expect(fetchMock.mock.calls[0][0]).toBe('/api/vocab?lesson=02&q=raditi')
  })

  it('returns undefined for 204', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))
    await expect(api.completeLesson('01')).resolves.toBeUndefined()
  })

  it('posts step status', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)
    await expect(api.setStepStatus('01', '01.2', 'done')).resolves.toBeUndefined()
    expect(fetchMock.mock.calls[0][0]).toBe('/api/lessons/01/steps/01.2')
    expect(fetchMock.mock.calls[0][1]).toMatchObject({
      method: 'POST',
      body: JSON.stringify({ status: 'done' }),
    })
  })

  it('sends same-origin credentials and no X-User header', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('{"phases":[]}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await api.progress()
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/progress')
    expect(init.credentials).toBe('same-origin')
    expect(init.headers['X-User']).toBeUndefined()
  })

  it('on a 401 from a non-auth path, clears the session and redirects to /login', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'no session' }), { status: 401 })),
    )
    const { useSessionStore } = await import('./stores/session')
    const { default: router } = await import('./router')
    const session = useSessionStore()
    session.user = {
      id: '1',
      name: 'Х',
      email: '',
      email_verified: false,
      telegram: { linked: false, username: '' },
    }
    const pushSpy = vi.spyOn(router, 'push')

    await expect(api.progress()).rejects.toBeInstanceOf(ApiError)
    expect(session.user).toBeNull()
    expect(pushSpy).toHaveBeenCalledWith(expect.objectContaining({ path: '/login' }))
  })

  it('does not redirect on a 401 from an /auth/ path (e.g. wrong-password login)', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'неверная почта или пароль' }), { status: 401 }),
      ),
    )
    const { default: router } = await import('./router')
    const pushSpy = vi.spyOn(router, 'push')

    await expect(api.login('a@b.com', 'wrong')).rejects.toMatchObject({ status: 401 })
    expect(pushSpy).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 8: Run the frontend suite and type-check**

```bash
cd web && npx vitest run src/api.test.ts src/stores/session.test.ts src/lib/authErrors.test.ts && npx vue-tsc -b --noEmit
```

Expected: all new/changed tests pass. The type-check reports exactly two
errors, both in `LoginGate.vue` (`api.listAccounts`/`api.createAccount` no
longer exist) — see the Global Constraints note on this: it's a known, ruled,
Task-6-terminated exception, not something to fix here.

- [ ] **Step 9: Commit**

```bash
git add web/src/api.ts web/src/api.test.ts web/src/types.ts web/src/stores/session.ts web/src/stores/session.test.ts web/src/lib/authErrors.ts web/src/lib/authErrors.test.ts
git commit -m "feat(web): session store, api.ts on cookies, error-code mapping"
```

---

## Task 2: `AuthShell`, `LoginView`, `RegisterView`

These are new, unreferenced files — nothing imports them yet, so the app's
current behavior (still the old `LoginGate`/`account.ts` flow) is unaffected.

**Files:**
- Create: `web/src/views/AuthShell.vue`
- Create: `web/src/views/LoginView.vue`
- Create: `web/src/views/LoginView.test.ts`
- Create: `web/src/views/RegisterView.vue`
- Create: `web/src/views/RegisterView.test.ts`

**Interfaces:**
- Consumes: `useSessionStore()` (Task 1), `authErrorMessage`/`isEmailUnverified` (Task 1).
- Produces: `AuthShell` (`<script setup> defineProps<{title?: string}>()`, default
  slot) — reused by every remaining auth screen in Tasks 3–4.

- [ ] **Step 1: Create `web/src/views/AuthShell.vue`**

```vue
<script setup lang="ts">
defineProps<{ title?: string }>()
</script>

<template>
  <div class="mx-auto mt-16 max-w-sm px-4">
    <p class="serbian text-center text-4xl font-semibold">Српски</p>
    <p v-if="title" class="mt-1 text-center text-[var(--muted)]">{{ title }}</p>
    <div class="mt-6">
      <slot />
    </div>
  </div>
</template>
```

- [ ] **Step 2: Create `web/src/views/LoginView.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AuthShell from './AuthShell.vue'
import { useSessionStore } from '../stores/session'
import { authErrorMessage, isEmailUnverified } from '../lib/authErrors'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()

const email = ref('')
const password = ref('')
const busy = ref(false)
const error = ref<string | null>(null)

function goNext() {
  const next = typeof route.query.next === 'string' ? route.query.next : '/profile'
  router.push(next)
}

async function submit() {
  if (busy.value) return
  busy.value = true
  error.value = null
  try {
    await session.login(email.value.trim(), password.value)
    goNext()
  } catch (e) {
    if (isEmailUnverified(e)) {
      router.push({ path: '/verify', query: { email: email.value.trim() } })
      return
    }
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="Вход">
    <form class="space-y-3" @submit.prevent="submit">
      <input v-model="email" type="email" class="field w-full" placeholder="email" required autofocus />
      <input
        v-model="password"
        type="password"
        class="field w-full"
        placeholder="пароль"
        required
        minlength="8"
        maxlength="128"
      />
      <button class="btn btn-primary w-full" :disabled="busy">Войти</button>
    </form>
    <p v-if="error" class="mt-2 text-sm text-[var(--bad)]">{{ error }}</p>

    <div class="mt-4 flex justify-between text-sm">
      <RouterLink to="/forgot" class="text-[var(--accent)]">забыли пароль?</RouterLink>
      <RouterLink to="/register" class="text-[var(--accent)]">регистрация</RouterLink>
    </div>
  </AuthShell>
</template>
```

(Task 5 adds a Telegram Login Widget button to this file — do not build it here.)

- [ ] **Step 3: Create `web/src/views/LoginView.test.ts`**

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LoginView from './LoginView.vue'
import { useSessionStore } from '../stores/session'
import { ApiError } from '../api'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push }),
  RouterLink: { template: '<a><slot /></a>' },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  push.mockClear()
})
afterEach(() => vi.restoreAllMocks())

describe('LoginView', () => {
  it('logs in and navigates to /profile by default', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'login').mockResolvedValue(undefined)
    const w = mount(LoginView)
    await w.find('input[type="email"]').setValue('g@example.com')
    await w.find('input[type="password"]').setValue('secret123')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(session.login).toHaveBeenCalledWith('g@example.com', 'secret123')
    expect(push).toHaveBeenCalledWith('/profile')
  })

  it('shows the generic error message on wrong credentials', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'login').mockRejectedValue(new ApiError(401, 'неверная почта или пароль'))
    const w = mount(LoginView)
    await w.find('input[type="email"]').setValue('g@example.com')
    await w.find('input[type="password"]').setValue('wrongpass')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(w.text()).toContain('неверная почта или пароль')
  })

  it('routes to /verify with the email on email_unverified', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'login').mockRejectedValue(new ApiError(403, 'email_unverified'))
    const w = mount(LoginView)
    await w.find('input[type="email"]').setValue('g@example.com')
    await w.find('input[type="password"]').setValue('secret123')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(push).toHaveBeenCalledWith({ path: '/verify', query: { email: 'g@example.com' } })
  })
})
```

- [ ] **Step 4: Create `web/src/views/RegisterView.vue`**

```vue
<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AuthShell from './AuthShell.vue'
import { useSessionStore } from '../stores/session'
import { authErrorMessage } from '../lib/authErrors'

const router = useRouter()
const session = useSessionStore()

const name = ref('')
const email = ref('')
const password = ref('')
const busy = ref(false)
const error = ref<string | null>(null)

const strength = computed(() => {
  const n = password.value.length
  if (n === 0) return null
  if (n < 8) return { label: 'слишком короткий', color: 'var(--bad)' }
  if (n < 12) return { label: 'нормальный', color: 'var(--muted)' }
  return { label: 'надёжный', color: 'var(--good)' }
})

async function submit() {
  if (busy.value) return
  busy.value = true
  error.value = null
  try {
    await session.register(email.value.trim(), password.value, name.value.trim())
    router.push({ path: '/verify', query: { email: email.value.trim() } })
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="Регистрация">
    <form class="space-y-3" @submit.prevent="submit">
      <input v-model="name" class="field w-full" placeholder="имя" required maxlength="40" autofocus />
      <input v-model="email" type="email" class="field w-full" placeholder="email" required />
      <div>
        <input
          v-model="password"
          type="password"
          class="field w-full"
          placeholder="пароль"
          required
          minlength="8"
          maxlength="128"
        />
        <p v-if="strength" class="mt-1 text-xs" :style="{ color: strength.color }">{{ strength.label }}</p>
      </div>
      <button class="btn btn-primary w-full" :disabled="busy">Зарегистрироваться</button>
    </form>
    <p v-if="error" class="mt-2 text-sm text-[var(--bad)]">{{ error }}</p>

    <p class="mt-4 text-center text-sm">
      Уже есть аккаунт? <RouterLink to="/login" class="text-[var(--accent)]">войти</RouterLink>
    </p>
  </AuthShell>
</template>
```

- [ ] **Step 5: Create `web/src/views/RegisterView.test.ts`**

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import RegisterView from './RegisterView.vue'
import { useSessionStore } from '../stores/session'
import { ApiError } from '../api'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
  RouterLink: { template: '<a><slot /></a>' },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  push.mockClear()
})
afterEach(() => vi.restoreAllMocks())

describe('RegisterView', () => {
  it('registers then routes to /verify with the email', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'register').mockResolvedValue(undefined)
    const w = mount(RegisterView)
    await w.find('input[placeholder="имя"]').setValue('Аня')
    await w.find('input[type="email"]').setValue('a@example.com')
    await w.find('input[type="password"]').setValue('secret123')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(session.register).toHaveBeenCalledWith('a@example.com', 'secret123', 'Аня')
    expect(push).toHaveBeenCalledWith({ path: '/verify', query: { email: 'a@example.com' } })
  })

  it('shows a mapped error on a validation failure', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'register').mockRejectedValue(new ApiError(400, 'invalid email'))
    const w = mount(RegisterView)
    await w.find('input[placeholder="имя"]').setValue('Аня')
    await w.find('input[type="email"]').setValue('not-an-email')
    await w.find('input[type="password"]').setValue('secret123')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(w.text()).toContain('Некорректный email')
  })
})
```

- [ ] **Step 6: Run and type-check**

```bash
cd web && npx vitest run src/views/LoginView.test.ts src/views/RegisterView.test.ts && npx vue-tsc -b --noEmit
```

- [ ] **Step 7: Commit**

```bash
git add web/src/views/AuthShell.vue web/src/views/LoginView.vue web/src/views/LoginView.test.ts web/src/views/RegisterView.vue web/src/views/RegisterView.test.ts
git commit -m "feat(web): login and register screens"
```

---

## Task 3: `VerifyView`, `ForgotView`, `ResetView`

Same shape as Task 2 — new, unreferenced files.

**Files:**
- Create: `web/src/views/VerifyView.vue`
- Create: `web/src/views/VerifyView.test.ts`
- Create: `web/src/views/ForgotView.vue`
- Create: `web/src/views/ForgotView.test.ts`
- Create: `web/src/views/ResetView.vue`
- Create: `web/src/views/ResetView.test.ts`

**Interfaces:**
- Consumes: `AuthShell` (Task 2), `useSessionStore()` + `api` (Task 1),
  `authErrorMessage` (Task 1).

- [ ] **Step 1: Create `web/src/views/VerifyView.vue`**

`VerifyView` has three modes selected by query params, plus a "gate" default
that targets either the logged-in user's email or an `?email=` query param
(used by `RegisterView`'s post-submit redirect and `LoginView`'s
`email_unverified` redirect — see Task 2):

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AuthShell from './AuthShell.vue'
import { useSessionStore } from '../stores/session'
import { api } from '../api'
import { authErrorMessage } from '../lib/authErrors'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()

const mode = computed<'ok' | 'err' | 'gate'>(() => {
  if (route.query.ok === '1') return 'ok'
  if (route.query.err === '1') return 'err'
  return 'gate'
})

const targetEmail = computed(
  () => session.user?.email || (typeof route.query.email === 'string' ? route.query.email : ''),
)

const emailInput = ref(targetEmail.value)
const busy = ref(false)
const sent = ref(false)
const error = ref<string | null>(null)
const cooldown = ref(0)
let timer: ReturnType<typeof setInterval> | undefined

onMounted(() => {
  if (mode.value === 'gate' && session.user?.email_verified) {
    router.push('/profile')
  }
})

function startCooldown() {
  cooldown.value = 30
  timer = setInterval(() => {
    cooldown.value -= 1
    if (cooldown.value <= 0) clearInterval(timer)
  }, 1000)
}

async function resend() {
  const email = (targetEmail.value || emailInput.value).trim()
  if (!email || busy.value || cooldown.value > 0) return
  busy.value = true
  error.value = null
  try {
    await api.resendVerification(email)
    sent.value = true
    startCooldown()
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="Почта">
    <div v-if="mode === 'ok'" class="text-center">
      <p class="mb-4">Почта подтверждена!</p>
      <RouterLink to="/login" class="btn btn-primary">Войти</RouterLink>
    </div>

    <div v-else-if="mode === 'err'" class="space-y-3">
      <p>Ссылка устарела или уже использована.</p>
      <input v-model="emailInput" type="email" class="field w-full" placeholder="email" />
      <button class="btn btn-primary w-full" :disabled="busy || cooldown > 0" @click="resend">
        {{ cooldown > 0 ? `отправить снова (${cooldown})` : 'отправить снова' }}
      </button>
      <p v-if="sent" class="text-sm text-[var(--good)]">Письмо отправлено.</p>
      <p v-if="error" class="text-sm text-[var(--bad)]">{{ error }}</p>
    </div>

    <div v-else class="space-y-3">
      <template v-if="targetEmail">
        <p>Письмо для подтверждения отправлено на <b>{{ targetEmail }}</b>.</p>
        <button class="btn btn-ghost w-full" :disabled="busy || cooldown > 0" @click="resend">
          {{ cooldown > 0 ? `отправить снова (${cooldown})` : 'отправить снова' }}
        </button>
      </template>
      <template v-else>
        <p>Введите email, чтобы отправить письмо для подтверждения.</p>
        <input v-model="emailInput" type="email" class="field w-full" placeholder="email" />
        <button class="btn btn-primary w-full" :disabled="busy || cooldown > 0" @click="resend">
          {{ cooldown > 0 ? `отправить снова (${cooldown})` : 'отправить письмо' }}
        </button>
      </template>
      <p v-if="sent" class="text-sm text-[var(--good)]">Письмо отправлено.</p>
      <p v-if="error" class="text-sm text-[var(--bad)]">{{ error }}</p>
    </div>
  </AuthShell>
</template>
```

- [ ] **Step 2: Create `web/src/views/VerifyView.test.ts`**

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import VerifyView from './VerifyView.vue'
import { useSessionStore } from '../stores/session'
import { api } from '../api'

let query: Record<string, string> = {}
const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ query }),
  useRouter: () => ({ push }),
  RouterLink: { template: '<a><slot /></a>' },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  query = {}
  push.mockClear()
})
afterEach(() => vi.restoreAllMocks())

describe('VerifyView', () => {
  it('shows the success view for ?ok=1', () => {
    query = { ok: '1' }
    const w = mount(VerifyView)
    expect(w.text()).toContain('подтверждена')
  })

  it('shows the resend form for ?err=1', () => {
    query = { err: '1' }
    const w = mount(VerifyView)
    expect(w.text()).toContain('устарела')
  })

  it('targets the ?email= query param when there is no session', () => {
    query = { email: 'new@example.com' }
    const w = mount(VerifyView)
    expect(w.text()).toContain('new@example.com')
  })

  it('targets the logged-in email and disables resend during cooldown', async () => {
    vi.useFakeTimers()
    const session = useSessionStore()
    session.user = {
      id: '1',
      name: 'Г',
      email: 'g@example.com',
      email_verified: false,
      telegram: { linked: false, username: '' },
    }
    vi.spyOn(api, 'resendVerification').mockResolvedValue({ status: 'ok' })
    const w = mount(VerifyView)
    expect(w.text()).toContain('g@example.com')
    await w.find('button').trigger('click')
    await flushPromises()
    expect(api.resendVerification).toHaveBeenCalledWith('g@example.com')
    expect(w.find('button').attributes('disabled')).toBeDefined()
    vi.useRealTimers()
  })

  it('redirects an already-verified logged-in user away from the gate', () => {
    const session = useSessionStore()
    session.user = {
      id: '1',
      name: 'Г',
      email: 'g@example.com',
      email_verified: true,
      telegram: { linked: false, username: '' },
    }
    mount(VerifyView)
    expect(push).toHaveBeenCalledWith('/profile')
  })
})
```

- [ ] **Step 3: Create `web/src/views/ForgotView.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import AuthShell from './AuthShell.vue'
import { useSessionStore } from '../stores/session'
import { authErrorMessage } from '../lib/authErrors'

const session = useSessionStore()
const email = ref('')
const busy = ref(false)
const sent = ref(false)
const error = ref<string | null>(null)

async function submit() {
  if (busy.value) return
  busy.value = true
  error.value = null
  try {
    await session.forgot(email.value.trim())
    sent.value = true
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="Восстановление пароля">
    <div v-if="sent" class="text-center">
      <p>Если такой аккаунт существует, письмо со ссылкой для сброса пароля отправлено.</p>
    </div>
    <form v-else class="space-y-3" @submit.prevent="submit">
      <input v-model="email" type="email" class="field w-full" placeholder="email" required autofocus />
      <button class="btn btn-primary w-full" :disabled="busy">Отправить ссылку</button>
    </form>
    <p v-if="error" class="mt-2 text-sm text-[var(--bad)]">{{ error }}</p>
    <p class="mt-4 text-center text-sm">
      <RouterLink to="/login" class="text-[var(--accent)]">вернуться ко входу</RouterLink>
    </p>
  </AuthShell>
</template>
```

- [ ] **Step 4: Create `web/src/views/ForgotView.test.ts`**

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ForgotView from './ForgotView.vue'
import { useSessionStore } from '../stores/session'

vi.mock('vue-router', () => ({ RouterLink: { template: '<a><slot /></a>' } }))

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('ForgotView', () => {
  it('shows the generic confirmation after submit, regardless of account existence', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'forgot').mockResolvedValue(undefined)
    const w = mount(ForgotView)
    await w.find('input[type="email"]').setValue('g@example.com')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(session.forgot).toHaveBeenCalledWith('g@example.com')
    expect(w.text()).toContain('отправлено')
  })
})
```

- [ ] **Step 5: Create `web/src/views/ResetView.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AuthShell from './AuthShell.vue'
import { useSessionStore } from '../stores/session'
import { authErrorMessage } from '../lib/authErrors'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()

const token = typeof route.query.token === 'string' ? route.query.token : ''
const password = ref('')
const confirm = ref('')
const busy = ref(false)
const error = ref<string | null>(null)

async function submit() {
  if (busy.value) return
  if (password.value !== confirm.value) {
    error.value = 'Пароли не совпадают'
    return
  }
  busy.value = true
  error.value = null
  try {
    await session.reset(token, password.value)
    router.push('/profile')
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="Новый пароль">
    <p v-if="!token" class="text-[var(--bad)]">Ссылка недействительна.</p>
    <form v-else class="space-y-3" @submit.prevent="submit">
      <input
        v-model="password"
        type="password"
        class="field w-full"
        placeholder="новый пароль"
        required
        minlength="8"
        maxlength="128"
        autofocus
      />
      <input
        v-model="confirm"
        type="password"
        class="field w-full"
        placeholder="повторите пароль"
        required
        minlength="8"
        maxlength="128"
      />
      <button class="btn btn-primary w-full" :disabled="busy">Сохранить</button>
    </form>
    <p v-if="error" class="mt-2 text-sm text-[var(--bad)]">{{ error }}</p>
    <p class="mt-4 text-center text-sm">
      <RouterLink to="/login" class="text-[var(--accent)]">вернуться ко входу</RouterLink>
    </p>
  </AuthShell>
</template>
```

- [ ] **Step 6: Create `web/src/views/ResetView.test.ts`**

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ResetView from './ResetView.vue'
import { useSessionStore } from '../stores/session'

let query: Record<string, string> = { token: 'abc123' }
const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ query }),
  useRouter: () => ({ push }),
  RouterLink: { template: '<a><slot /></a>' },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  query = { token: 'abc123' }
  push.mockClear()
})
afterEach(() => vi.restoreAllMocks())

describe('ResetView', () => {
  it('rejects mismatched passwords without calling the API', async () => {
    const session = useSessionStore()
    const spy = vi.spyOn(session, 'reset')
    const w = mount(ResetView)
    await w.find('input[placeholder="новый пароль"]').setValue('secret123')
    await w.find('input[placeholder="повторите пароль"]').setValue('different1')
    await w.find('form').trigger('submit.prevent')
    expect(spy).not.toHaveBeenCalled()
    expect(w.text()).toContain('не совпадают')
  })

  it('resets and routes to /profile on match', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'reset').mockResolvedValue(undefined)
    const w = mount(ResetView)
    await w.find('input[placeholder="новый пароль"]').setValue('secret123')
    await w.find('input[placeholder="повторите пароль"]').setValue('secret123')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(session.reset).toHaveBeenCalledWith('abc123', 'secret123')
    expect(push).toHaveBeenCalledWith('/profile')
  })

  it('shows an error when there is no token', () => {
    query = {}
    const w = mount(ResetView)
    expect(w.text()).toContain('недействительна')
  })
})
```

- [ ] **Step 7: Run and type-check**

```bash
cd web && npx vitest run src/views/VerifyView.test.ts src/views/ForgotView.test.ts src/views/ResetView.test.ts && npx vue-tsc -b --noEmit
```

- [ ] **Step 8: Commit**

```bash
git add web/src/views/VerifyView.vue web/src/views/VerifyView.test.ts web/src/views/ForgotView.vue web/src/views/ForgotView.test.ts web/src/views/ResetView.vue web/src/views/ResetView.test.ts
git commit -m "feat(web): verify, forgot-password and reset-password screens"
```

---

## Task 4: `ProgressDashboard` (extracted) + `ProfileView`

New, unreferenced files — `DashboardView.vue` is untouched and still live at
`/` until Task 6.

**Files:**
- Create: `web/src/components/ProgressDashboard.vue` (the current `DashboardView.vue`
  template/script, minus the "сбросить прогресс" button, taking `name` as a prop
  instead of reading `getAccount()`)
- Create: `web/src/views/ProfileView.vue`
- Create: `web/src/views/ProfileView.test.ts`

**Interfaces:**
- Produces: `ProgressDashboard` — `defineProps<{ name: string }>()`, no emits.
- Consumes: `api.getMe/patchMe/setPassword/deleteSession/deleteMe/resetExercises`
  (Task 1), `useSessionStore()` (Task 1), `authErrorMessage` (Task 1).

- [ ] **Step 1: Create `web/src/components/ProgressDashboard.vue`**

Identical logic to today's `web/src/views/DashboardView.vue`, with `me` replaced
by a `name` prop and the bottom "сбросить прогресс по заданиям" button/section
removed (it moves to `ProfileView`'s danger zone in Step 2):

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import { useCourseStore } from '../stores/course'
import type { Progress, Vocab, LeaderRow } from '../types'
import ProgressRing from './ProgressRing.vue'
import ActivityHeatmap from './ActivityHeatmap.vue'
import WordMedia from './WordMedia.vue'

const props = defineProps<{ name: string }>()

const progress = ref<Progress | null>(null)
const wotd = ref<Vocab | null>(null)
const leaders = ref<LeaderRow[]>([])
const error = ref<string | null>(null)
const course = useCourseStore()

onMounted(async () => {
  course.load()
  try {
    progress.value = await api.progress()
    api.leaderboard().then((r) => (leaders.value = r)).catch(() => {})
    const vocab = await api.vocab()
    if (vocab.length) {
      const now = new Date()
      const doy = Math.floor((+now - +new Date(now.getFullYear(), 0, 0)) / 86400000)
      wotd.value = vocab[doy % vocab.length]
    }
  } catch (e) {
    error.value = (e as Error).message
  }
})

const continueLesson = computed(() => {
  const p = progress.value
  if (!p) return null
  const inProgress = p.recent_lessons.find((l) => l.status === 'in_progress')
  if (inProgress) return { id: inProgress.lesson, title: inProgress.title, label: 'Продолжить урок' }
  const done = new Set(p.recent_lessons.filter((l) => l.status === 'done').map((l) => l.lesson))
  const next = course.course?.lessons?.find((l) => !l.planned && !done.has(l.id))
  return next ? { id: next.id, title: next.title, label: 'Следующий урок' } : null
})

const totalDone = computed(() =>
  progress.value ? progress.value.phases.reduce((a, p) => a + p.done, 0) : 0,
)
const todayCount = computed(() => {
  const p = progress.value
  if (!p) return 0
  const today = new Date().toISOString().slice(0, 10)
  return p.activity.find((a) => a.date === today)?.count ?? 0
})
const dueTotal = computed(() =>
  progress.value ? progress.value.srs.due_today + progress.value.srs.new_available : 0,
)
</script>

<template>
  <p v-if="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

  <div v-else-if="progress" class="space-y-4">
    <RouterLink to="/review" class="card block p-5 transition hover:-translate-y-0.5">
      <div class="flex items-center gap-4">
        <div class="relative shrink-0">
          <ProgressRing :value="todayCount" :max="progress.daily_goal" :size="76" />
          <div class="absolute inset-0 flex flex-col items-center justify-center leading-none">
            <span class="text-lg font-extrabold">{{ todayCount }}</span>
            <span class="text-[10px] text-[var(--muted)]">/ {{ progress.daily_goal }}</span>
          </div>
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-lg font-extrabold">
            {{ dueTotal > 0 ? 'Повторить слова' : 'Слова на сегодня — всё' }}
          </p>
          <p class="text-sm text-[var(--muted)]">
            <template v-if="dueTotal > 0">
              к повторению <b class="text-[var(--fg)]">{{ progress.srs.due_today }}</b> ·
              новых <b class="text-[var(--fg)]">{{ progress.srs.new_available }}</b>
            </template>
            <template v-else>сделано {{ progress.srs.reviewed_today }} — возвращайся завтра</template>
          </p>
        </div>
        <span class="text-2xl text-[var(--accent)]">→</span>
      </div>
    </RouterLink>

    <div class="card p-5">
      <div class="mb-3 flex items-baseline justify-between">
        <p class="font-bold">
          <span class="text-xl">{{ progress.streak_days }}</span>
          <span class="text-[var(--muted)]">&nbsp;{{ progress.streak_days === 1 ? 'день' : 'дней' }} подряд</span>
          <span v-if="progress.streak_days > 0">&nbsp;🔥</span>
        </p>
        <p class="text-sm text-[var(--muted)]">{{ progress.srs.known }} / {{ progress.srs.total_cards }} закреплено</p>
      </div>
      <ActivityHeatmap :activity="progress.activity" />
    </div>

    <RouterLink
      v-if="continueLesson"
      :to="`/lesson/${continueLesson.id}`"
      class="card block p-4 transition hover:-translate-y-0.5"
    >
      <p class="text-xs uppercase tracking-wide text-[var(--muted)]">{{ continueLesson.label }}</p>
      <p class="text-lg font-bold">{{ continueLesson.id }}. {{ continueLesson.title }}</p>
    </RouterLink>

    <div v-if="wotd" class="card flex gap-4 p-4">
      <WordMedia :image="wotd.image" :emoji="wotd.emoji" :alt="wotd.ru" :size="72" class="shrink-0" />
      <div class="min-w-0">
        <p class="mb-1 text-xs uppercase tracking-wide text-[var(--muted)]">Слово дня</p>
        <p class="serbian text-2xl font-semibold">{{ wotd.latin }}</p>
        <p class="text-[var(--muted)]">{{ wotd.cyrillic }} — {{ wotd.ru }}</p>
        <p v-if="wotd.note" class="mt-0.5 text-sm text-[var(--muted)]">{{ wotd.note }}</p>
      </div>
    </div>

    <div class="card p-5">
      <p class="mb-3 font-bold">Прогресс <span class="text-[var(--muted)]">· {{ totalDone }} / 30 уроков</span></p>
      <div class="flex justify-around">
        <div v-for="ph in progress.phases" :key="ph.id" class="flex flex-col items-center gap-1">
          <div class="relative">
            <ProgressRing :value="ph.done" :max="ph.total" :size="64" :stroke="7" />
            <div class="absolute inset-0 flex items-center justify-center text-sm font-bold">
              {{ ph.done }}/{{ ph.total }}
            </div>
          </div>
          <span class="text-xs text-[var(--muted)]">Фаза {{ ph.id }}</span>
        </div>
      </div>
    </div>

    <RouterLink v-if="leaders.length > 1" to="/people" class="card block p-5 transition hover:-translate-y-0.5">
      <div class="mb-2 flex items-baseline justify-between">
        <p class="font-bold">Люди</p>
        <span class="text-sm text-[var(--accent)]">все →</span>
      </div>
      <ul class="space-y-1.5 text-sm">
        <li v-for="(r, i) in leaders.slice(0, 3)" :key="r.name" class="flex items-baseline gap-2">
          <span class="w-4 font-mono text-[var(--muted)]">{{ i + 1 }}</span>
          <span class="flex-1 font-semibold" :class="r.name === props.name ? 'text-[var(--accent)]' : ''">{{ r.name }}</span>
          <span class="text-[var(--muted)]">{{ r.lessons_done }}/{{ r.lessons_total }} · {{ r.cards_known }} сл. · {{ r.streak_days }}🔥</span>
        </li>
      </ul>
    </RouterLink>

    <div v-if="progress.weak_exercises.length" class="card p-5">
      <p class="mb-2 font-bold">Стоит повторить</p>
      <ul class="space-y-1.5 text-sm">
        <li v-for="w in progress.weak_exercises" :key="w.exercise_id">
          <RouterLink :to="`/lesson/${w.lesson}`" class="flex gap-2 hover:underline">
            <span class="shrink-0 font-mono text-[var(--bad)]">{{ w.wrong }}/{{ w.total }}</span>
            <span class="text-[var(--muted)]">{{ w.prompt || w.exercise_id }}</span>
          </RouterLink>
        </li>
      </ul>
    </div>

    <div
      v-if="totalDone === 0 && progress.srs.reviewed_today === 0"
      class="card p-6 text-center text-[var(--muted)]"
    >
      Начни с <RouterLink to="/lesson/01" class="font-semibold text-[var(--accent)]">урока 01</RouterLink>.
    </div>
  </div>
</template>
```

Note: the `to="/people"` link here still points at the not-yet-renamed route
(Task 6 renames it to `/rating`, updating this file and its label together
with the nav rename — do not rename it here, the route doesn't exist yet).

- [ ] **Step 2: Create `web/src/views/ProfileView.vue`**

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { useSessionStore } from '../stores/session'
import { authErrorMessage } from '../lib/authErrors'
import ProgressDashboard from '../components/ProgressDashboard.vue'
import type { Me } from '../types'

const router = useRouter()
const session = useSessionStore()

const me = ref<Me | null>(null)
const loadError = ref<string | null>(null)

async function loadMe() {
  try {
    me.value = await api.getMe()
  } catch (e) {
    loadError.value = authErrorMessage(e)
  }
}
onMounted(loadMe)

const hasPassword = computed(() => !!me.value?.email)

// -- rename --
const editingName = ref(false)
const nameDraft = ref('')
const nameBusy = ref(false)
const nameError = ref<string | null>(null)
function startEditName() {
  nameDraft.value = me.value?.name ?? ''
  nameError.value = null
  editingName.value = true
}
async function saveName() {
  const trimmed = nameDraft.value.trim()
  if (!trimmed) return
  nameBusy.value = true
  nameError.value = null
  try {
    const updated = await api.patchMe(trimmed)
    if (me.value) me.value.name = updated.name
    if (session.user) session.user.name = updated.name
    editingName.value = false
  } catch (e) {
    nameError.value = authErrorMessage(e)
  } finally {
    nameBusy.value = false
  }
}

// -- password --
const showPasswordForm = ref(false)
const currentPassword = ref('')
const newPassword = ref('')
const pwEmail = ref('')
const pwBusy = ref(false)
const pwError = ref<string | null>(null)
const pwStatus = ref<string | null>(null)

async function submitPassword() {
  if (newPassword.value.length < 8) {
    pwError.value = 'Пароль должен быть от 8 до 128 символов'
    return
  }
  pwBusy.value = true
  pwError.value = null
  try {
    const res = await api.setPassword({
      current: hasPassword.value ? currentPassword.value : undefined,
      new: newPassword.value,
      email: hasPassword.value ? undefined : pwEmail.value,
    })
    pwStatus.value =
      res.status === 'verify_sent' ? 'Проверьте почту — письмо для подтверждения отправлено' : 'Пароль изменён'
    showPasswordForm.value = false
    currentPassword.value = ''
    newPassword.value = ''
  } catch (e) {
    pwError.value = authErrorMessage(e)
  } finally {
    pwBusy.value = false
  }
}

// -- devices --
async function revokeSession(id: string) {
  if (!confirm('Выйти на этом устройстве?')) return
  await api.deleteSession(id)
  await loadMe()
}

async function logout() {
  await session.logout()
  router.push('/login')
}
async function logoutAll() {
  await session.logoutAll()
  router.push('/login')
}

// -- danger zone --
async function resetExercises() {
  if (!confirm('Сбросить весь прогресс по заданиям (ответы и отметки уроков)? Карточки слов останутся.')) return
  await api.resetExercises()
  location.reload()
}

const deletePassword = ref('')
const deleteBusy = ref(false)
const deleteError = ref<string | null>(null)
async function deleteAccount() {
  if (!confirm('Удалить аккаунт безвозвратно? Это нельзя отменить.')) return
  deleteBusy.value = true
  deleteError.value = null
  try {
    await api.deleteMe(deletePassword.value || undefined)
    session.user = null
    router.push('/login')
  } catch (e) {
    deleteError.value = authErrorMessage(e)
  } finally {
    deleteBusy.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <p v-if="loadError" class="card p-4 text-[var(--bad)]">{{ loadError }}</p>

    <div v-else-if="me" class="card space-y-4 p-5">
      <div class="flex items-center justify-between gap-2">
        <template v-if="!editingName">
          <p class="text-xl font-extrabold">{{ me.name }}</p>
          <button class="text-sm text-[var(--accent)]" @click="startEditName">изменить</button>
        </template>
        <form v-else class="flex flex-1 gap-2" @submit.prevent="saveName">
          <input v-model="nameDraft" class="field flex-1" maxlength="40" autofocus />
          <button class="btn btn-primary" :disabled="nameBusy">Сохранить</button>
          <button type="button" class="btn btn-ghost" @click="editingName = false">Отмена</button>
        </form>
      </div>
      <p v-if="nameError" class="text-sm text-[var(--bad)]">{{ nameError }}</p>

      <div class="flex items-center gap-2 text-sm">
        <span class="text-[var(--muted)]">Почта:</span>
        <span v-if="me.email">{{ me.email }}</span>
        <span v-else class="text-[var(--muted)]">не указана</span>
        <span
          v-if="me.email"
          class="rounded-full px-2 py-0.5 text-xs font-semibold"
          :style="
            me.email_verified
              ? { background: 'color-mix(in srgb, var(--good) 16%, transparent)', color: 'var(--good)' }
              : { background: 'var(--bg-soft)', color: 'var(--muted)' }
          "
        >
          {{ me.email_verified ? 'подтверждена' : 'не подтверждена' }}
        </span>
      </div>

      <div class="flex items-center gap-2 text-sm">
        <span class="text-[var(--muted)]">Telegram:</span>
        <span v-if="me.telegram.linked">@{{ me.telegram.username || '—' }}</span>
        <span v-else class="text-[var(--muted)]">не привязан</span>
      </div>

      <div>
        <button v-if="!showPasswordForm" class="btn btn-ghost" @click="showPasswordForm = true">
          {{ hasPassword ? 'Сменить пароль' : 'Задать пароль' }}
        </button>
        <form v-else class="mt-2 space-y-2" @submit.prevent="submitPassword">
          <input
            v-if="hasPassword"
            v-model="currentPassword"
            type="password"
            class="field w-full"
            placeholder="текущий пароль"
          />
          <input
            v-if="!hasPassword"
            v-model="pwEmail"
            type="email"
            class="field w-full"
            placeholder="email для входа по паролю"
          />
          <input
            v-model="newPassword"
            type="password"
            class="field w-full"
            placeholder="новый пароль"
            minlength="8"
            maxlength="128"
          />
          <div class="flex gap-2">
            <button class="btn btn-primary" :disabled="pwBusy">Сохранить</button>
            <button type="button" class="btn btn-ghost" @click="showPasswordForm = false">Отмена</button>
          </div>
        </form>
        <p v-if="pwError" class="mt-1 text-sm text-[var(--bad)]">{{ pwError }}</p>
        <p v-if="pwStatus" class="mt-1 text-sm text-[var(--good)]">{{ pwStatus }}</p>
      </div>

      <div>
        <p class="mb-2 text-sm font-bold">Устройства</p>
        <ul class="space-y-1.5">
          <li v-for="d in me.sessions" :key="d.id" class="flex items-center justify-between gap-2 text-sm">
            <span class="min-w-0 truncate text-[var(--muted)]">
              {{ d.user_agent || 'неизвестное устройство' }}
              <span v-if="d.current" class="text-[var(--accent)]">— это устройство</span>
            </span>
            <button class="shrink-0 text-xs text-[var(--bad)]" @click="revokeSession(d.id)">выйти</button>
          </li>
        </ul>
      </div>

      <div class="flex flex-wrap gap-2 border-t border-[var(--border)] pt-3">
        <button class="btn btn-ghost" @click="logout">Выйти</button>
        <button class="btn btn-ghost" @click="logoutAll">Выйти везде</button>
      </div>
    </div>

    <ProgressDashboard v-if="me" :name="me.name" />

    <div v-if="me" class="card space-y-3 p-5">
      <p class="font-bold text-[var(--bad)]">Опасная зона</p>
      <button class="text-sm text-[var(--muted)] hover:text-[var(--bad)]" @click="resetExercises">
        сбросить прогресс по заданиям
      </button>
      <div class="space-y-2 border-t border-[var(--border)] pt-3">
        <input
          v-if="hasPassword"
          v-model="deletePassword"
          type="password"
          class="field w-full"
          placeholder="пароль для подтверждения"
        />
        <button
          class="btn"
          style="background: var(--bad); color: #fff"
          :disabled="deleteBusy"
          @click="deleteAccount"
        >
          Удалить аккаунт
        </button>
        <p v-if="deleteError" class="text-sm text-[var(--bad)]">{{ deleteError }}</p>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 3: Create `web/src/views/ProfileView.test.ts`**

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ProfileView from './ProfileView.vue'
import { api } from '../api'
import type { Me } from '../types'

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  RouterLink: { template: '<a><slot /></a>' },
}))
vi.stubGlobal('confirm', () => true)

function me(overrides: Partial<Me> = {}): Me {
  return {
    id: '1',
    name: 'Гриша',
    email: '',
    email_verified: false,
    telegram: { linked: true, username: 'llladnooo' },
    sessions: [{ id: 'abc123456789', user_agent: 'Chrome', last_seen_at: '', current: true }],
    ...overrides,
  }
}

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('ProfileView', () => {
  it('renders the account and offers to set a password for a Telegram-only account', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a')) // ProgressDashboard's own fetch; irrelevant here
    const w = mount(ProfileView)
    await flushPromises()
    expect(w.text()).toContain('Гриша')
    expect(w.text()).toContain('Задать пароль')
  })

  it('renames the account via patchMe', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    const patch = vi.spyOn(api, 'patchMe').mockResolvedValue({
      id: '1',
      name: 'Новое имя',
      email: '',
      email_verified: false,
      telegram: { linked: true, username: 'llladnooo' },
    })
    const w = mount(ProfileView)
    await flushPromises()
    await w.find('button').trigger('click') // "изменить"
    await w.find('input.field').setValue('Новое имя')
    await w.find('form').trigger('submit.prevent')
    await flushPromises()
    expect(patch).toHaveBeenCalledWith('Новое имя')
    expect(w.text()).toContain('Новое имя')
  })

  it('revokes a device session', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me())
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    const del = vi.spyOn(api, 'deleteSession').mockResolvedValue(undefined)
    const w = mount(ProfileView)
    await flushPromises()
    const revoke = w.findAll('button').find((b) => b.text() === 'выйти')
    await revoke!.trigger('click')
    await flushPromises()
    expect(del).toHaveBeenCalledWith('abc123456789')
  })

  it('deletes the account and clears the session', async () => {
    vi.spyOn(api, 'getMe').mockResolvedValue(me({ email: 'g@example.com', email_verified: true }))
    vi.spyOn(api, 'progress').mockRejectedValue(new Error('n/a'))
    const del = vi.spyOn(api, 'deleteMe').mockResolvedValue(undefined)
    const w = mount(ProfileView)
    await flushPromises()
    await w.find('input[placeholder="пароль для подтверждения"]').setValue('secret123')
    const buttons = w.findAll('button')
    await buttons[buttons.length - 1].trigger('click')
    await flushPromises()
    expect(del).toHaveBeenCalledWith('secret123')
  })
})
```

- [ ] **Step 4: Run and type-check**

```bash
cd web && npx vitest run src/views/ProfileView.test.ts && npx vue-tsc -b --noEmit
```

- [ ] **Step 5: Commit**

```bash
git add web/src/components/ProgressDashboard.vue web/src/views/ProfileView.vue web/src/views/ProfileView.test.ts
git commit -m "feat(web): extract ProgressDashboard, add ProfileView (account card + devices + danger zone)"
```

---

## Task 5: Telegram — bot id plumbing, Login Widget button, Mini App auto-login

This is the only task touching Go code: one field on the already-public
`GET /api/health`, derived from the already-configured (but currently empty)
`TELEGRAM_BOT_TOKEN`. It is not a secret — it becomes the public bot id the
moment a bot exists, same as `t.me/<bot>`'s numeric id is discoverable via
Telegram's own Bot API.

**Files:**
- Modify: `server/internal/config/config.go` (add `TelegramBotID()`)
- Modify: `server/internal/config/config_test.go` (add `TestTelegramBotID`)
- Modify: `server/internal/api/api.go` (health handler gains the field)
- Modify: `server/internal/api/api_test.go` (add a health test)
- Modify: `web/src/types.ts` (add `Health`)
- Modify: `web/src/api.ts` (add `api.health()`)
- Modify: `web/src/telegram.ts` (expose `initData()`)
- Modify: `web/src/stores/session.ts` + `session.test.ts` (Mini App silent auto-login)
- Create: `web/src/components/TelegramLoginButton.vue`
- Create: `web/src/components/TelegramLoginButton.test.ts`
- Modify: `web/src/views/LoginView.vue` + `LoginView.test.ts` (wire the button in)

**Interfaces:**
- Produces: `Config.TelegramBotID() string` (Go) — empty when disabled or malformed.
- Produces: `Health { status, content_stale, telegram_bot_id }` (TS), `api.health()`.
- Produces: `initData(): string` from `telegram.ts`.
- Produces: `TelegramLoginButton` — `defineProps<{botId: string}>()`,
  `defineEmits<{auth: [payload: Record<string, unknown>]}>()`.

- [ ] **Step 1: `Config.TelegramBotID()`**

In `server/internal/config/config.go`, add after `TelegramEnabled`:

```go
// TelegramBotID returns the bot's numeric id — the segment before ":" in
// TELEGRAM_BOT_TOKEN — for the client-side Telegram Login Widget, which opens
// its auth popup with this id (not the token). Not a secret: once a bot
// exists, its id is public (part of the bot's own API surface, same as its
// t.me link). Empty when Telegram sign-in is not configured or the token is
// malformed.
func (c Config) TelegramBotID() string {
	id, _, ok := strings.Cut(c.TelegramBotToken, ":")
	if !ok || id == "" {
		return ""
	}
	return id
}
```

- [ ] **Step 2: Test it**

In `server/internal/config/config_test.go`, add:

```go
func TestTelegramBotID(t *testing.T) {
	clearEnv(t)
	if got := Load().TelegramBotID(); got != "" {
		t.Fatalf("TelegramBotID() with no token: got %q, want empty", got)
	}

	clearEnv(t)
	t.Setenv("TELEGRAM_BOT_TOKEN", "123456:abcdef-secret")
	if got := Load().TelegramBotID(); got != "123456" {
		t.Fatalf("TelegramBotID(): got %q, want %q", got, "123456")
	}

	clearEnv(t)
	t.Setenv("TELEGRAM_BOT_TOKEN", "malformed-no-colon")
	if got := Load().TelegramBotID(); got != "" {
		t.Fatalf("TelegramBotID() on malformed token: got %q, want empty", got)
	}
}
```

Run: `go test ./server/internal/config/... -run TestTelegramBotID -v` — expect PASS.

- [ ] **Step 3: Expose it on `GET /api/health`**

In `server/internal/api/api.go`, replace the `health` handler:

```go
func (h handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "ok",
		"content_stale":   h.Stale(),
		"telegram_bot_id": h.Config.TelegramBotID(),
	})
}
```

- [ ] **Step 4: Test it**

In `server/internal/api/api_test.go`, add:

```go
func TestHealthExposesTelegramBotID(t *testing.T) {
	h, _ := newTestAPIWith(t, func(d *Deps) {
		d.Config = config.Config{AppBaseURL: testBaseURL, TelegramBotToken: "42:secret"}
	})
	rr := do(h, "GET", "/api/health", "")
	if rr.Code != 200 {
		t.Fatal(rr.Code)
	}
	got := decodeBody[map[string]any](t, rr)
	if got["telegram_bot_id"] != "42" {
		t.Fatalf("telegram_bot_id = %v, want 42", got["telegram_bot_id"])
	}
}
```

Run: `go test ./server/... -run TestHealth -v` — expect both `TestHealth` and
`TestHealthExposesTelegramBotID` PASS. Then `go build ./... && go vet ./... && go test ./server/... -count=1` clean.

- [ ] **Step 5: Frontend — `Health` type and `api.health()`**

In `web/src/types.ts`, add:

```ts
export interface Health {
  status: string
  content_stale: boolean
  telegram_bot_id: string
}
```

In `web/src/api.ts`: add `Health` to the type-only import from `./types`, and
add to the `api` object (any position, e.g. right before `register`):

```ts
  health: () => request<Health>('/health'),
```

- [ ] **Step 6: `telegram.ts` gains `initData()`**

In `web/src/telegram.ts`, add `initData: string` to the `TgWebApp` interface
and a new export:

```ts
interface TgWebApp {
  ready: () => void
  expand: () => void
  colorScheme: 'light' | 'dark'
  initData: string
  setHeaderColor?: (c: string) => void
  setBackgroundColor?: (c: string) => void
  onEvent?: (e: string, cb: () => void) => void
}
```

```ts
// The raw, HMAC-signed query string the backend's VerifyInitData checks.
// Empty outside Telegram or before the SDK has populated it.
export function initData(): string {
  return tg()?.initData ?? ''
}
```

- [ ] **Step 7: Session store — silent Mini App auto-login**

In `web/src/stores/session.ts`, add the import and extend `fetchSession`:

```ts
import { isTelegram, initData } from '../telegram'
```

```ts
  async function fetchSession() {
    loading.value = true
    try {
      user.value = await api.session()
    } catch {
      user.value = null
      if (isTelegram() && initData()) {
        try {
          user.value = await api.telegramLogin({ init_data: initData() })
        } catch {
          user.value = null
        }
      }
    } finally {
      loading.value = false
    }
  }
```

- [ ] **Step 8: Test it**

In `web/src/stores/session.test.ts`, add the mock at the top of the file (after
the existing imports) and two tests:

```ts
import { isTelegram, initData } from '../telegram'

vi.mock('../telegram', () => ({ isTelegram: vi.fn(), initData: vi.fn() }))
```

```ts
  it('silently tries Telegram Mini App auto-login when session() fails inside Telegram', async () => {
    vi.mocked(isTelegram).mockReturnValue(true)
    vi.mocked(initData).mockReturnValue('query_id=AA&user=%7B%22id%22%3A1%7D')
    vi.spyOn(api, 'session').mockRejectedValue(new ApiError(401, 'no session'))
    vi.spyOn(api, 'telegramLogin').mockResolvedValue(user({ name: 'TG User' }))
    const s = useSessionStore()
    await s.fetchSession()
    expect(api.telegramLogin).toHaveBeenCalledWith({ init_data: 'query_id=AA&user=%7B%22id%22%3A1%7D' })
    expect(s.user?.name).toBe('TG User')
  })

  it('does not attempt telegram auto-login outside Telegram', async () => {
    vi.mocked(isTelegram).mockReturnValue(false)
    vi.mocked(initData).mockReturnValue('')
    vi.spyOn(api, 'session').mockRejectedValue(new ApiError(401, 'no session'))
    const spy = vi.spyOn(api, 'telegramLogin')
    const s = useSessionStore()
    await s.fetchSession()
    expect(spy).not.toHaveBeenCalled()
    expect(s.user).toBeNull()
  })
```

(Place these inside the existing `describe('useSessionStore', ...)` block.)

- [ ] **Step 9: Create `web/src/components/TelegramLoginButton.vue`**

Uses the Login Widget's *programmatic* API (`Telegram.Login.auth`), which
takes the numeric `bot_id` — not the bot's `@username` — so nothing beyond the
already-planned `TELEGRAM_BOT_TOKEN` is needed to make this real:

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'

const props = defineProps<{ botId: string }>()
const emit = defineEmits<{ auth: [payload: Record<string, unknown>] }>()
const ready = ref(false)

declare global {
  interface Window {
    Telegram?: {
      Login?: {
        auth: (
          opts: { bot_id: string; request_access?: boolean },
          cb: (data: Record<string, unknown> | false) => void,
        ) => void
      }
    }
  }
}

function loadScript(): Promise<void> {
  if (window.Telegram?.Login) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const s = document.createElement('script')
    s.src = 'https://telegram.org/js/telegram-widget.js?22'
    s.async = true
    s.onload = () => resolve()
    s.onerror = () => reject(new Error('telegram widget script failed to load'))
    document.head.appendChild(s)
  })
}

onMounted(async () => {
  try {
    await loadScript()
    ready.value = !!window.Telegram?.Login
  } catch {
    ready.value = false
  }
})

function click() {
  if (!window.Telegram?.Login) return
  window.Telegram.Login.auth({ bot_id: props.botId, request_access: true }, (data) => {
    if (data) emit('auth', data)
  })
}
</script>

<template>
  <button v-if="ready" type="button" class="btn btn-ghost w-full" @click="click">
    Войти через Telegram
  </button>
</template>
```

- [ ] **Step 10: Create `web/src/components/TelegramLoginButton.test.ts`**

```ts
import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TelegramLoginButton from './TelegramLoginButton.vue'

afterEach(() => {
  vi.restoreAllMocks()
  delete (window as unknown as { Telegram?: unknown }).Telegram
  document.head.innerHTML = ''
})

describe('TelegramLoginButton', () => {
  it('renders nothing until the widget script signals it is ready', async () => {
    const w = mount(TelegramLoginButton, { props: { botId: '42' } })
    expect(w.find('button').exists()).toBe(false)
    window.Telegram = { Login: { auth: vi.fn() } }
    document.head.querySelector('script')?.dispatchEvent(new Event('load'))
    await flushPromises()
    expect(w.find('button').exists()).toBe(true)
  })

  it('opens the Telegram auth popup with bot_id and emits auth on success', async () => {
    const authMock = vi.fn((_opts, cb) => cb({ id: 1, username: 'x', auth_date: 1, hash: 'h' }))
    window.Telegram = { Login: { auth: authMock } }
    const w = mount(TelegramLoginButton, { props: { botId: '42' } })
    document.head.querySelector('script')?.dispatchEvent(new Event('load'))
    await flushPromises()
    await w.find('button').trigger('click')
    expect(authMock).toHaveBeenCalledWith({ bot_id: '42', request_access: true }, expect.any(Function))
    expect(w.emitted('auth')?.[0][0]).toMatchObject({ id: 1, username: 'x' })
  })
})
```

- [ ] **Step 11: Wire the button into `LoginView.vue`**

In `web/src/views/LoginView.vue`, add imports:

```ts
import { api } from '../api'
import TelegramLoginButton from '../components/TelegramLoginButton.vue'
```

Add state and a handler (near the other refs / functions):

```ts
const botId = ref('')
api.health().then((h) => (botId.value = h.telegram_bot_id)).catch(() => {})

async function onTelegramAuth(payload: Record<string, unknown>) {
  error.value = null
  try {
    session.user = await api.telegramLogin(payload)
    goNext()
  } catch (e) {
    error.value = authErrorMessage(e)
  }
}
```

In the template, add the button between the error paragraph and the footer links:

```vue
    <TelegramLoginButton v-if="botId" class="mt-3" :bot-id="botId" @auth="onTelegramAuth" />
```

- [ ] **Step 12: Extend `LoginView.test.ts`**

Add (inside the existing `describe('LoginView', ...)` block):

```ts
  it('shows the Telegram button once health reports a bot id, and logs in on auth', async () => {
    vi.spyOn(api, 'health').mockResolvedValue({ status: 'ok', content_stale: false, telegram_bot_id: '42' })
    vi.spyOn(api, 'telegramLogin').mockResolvedValue({
      id: '1',
      name: 'Г',
      email: '',
      email_verified: false,
      telegram: { linked: true, username: 'g' },
    })
    const w = mount(LoginView)
    await flushPromises()
    const btn = w.findComponent({ name: 'TelegramLoginButton' })
    expect(btn.exists()).toBe(true)
    await btn.vm.$emit('auth', { id: 1 })
    await flushPromises()
    expect(api.telegramLogin).toHaveBeenCalledWith({ id: 1 })
    expect(push).toHaveBeenCalledWith('/profile')
  })
```

Add `import { api } from '../api'` to the test file's imports if not already present.

- [ ] **Step 13: Run everything and type-check**

```bash
go build ./... && go vet ./... && go test ./server/... -count=1
cd web && npx vitest run && npx vue-tsc -b --noEmit
```

Expected: all green.

- [ ] **Step 14: Commit**

```bash
git add server/internal/config/config.go server/internal/config/config_test.go server/internal/api/api.go server/internal/api/api_test.go web/src/types.ts web/src/api.ts web/src/telegram.ts web/src/stores/session.ts web/src/stores/session.test.ts web/src/components/TelegramLoginButton.vue web/src/components/TelegramLoginButton.test.ts web/src/views/LoginView.vue web/src/views/LoginView.test.ts
git commit -m "feat: expose telegram_bot_id, add Login Widget button + Mini App silent auto-login"
```

---

## Task 6: Cutover — router, `App.vue`, `AppNav.vue`, `/rating` rename, cleanup

**This is the one task where the app's actual auth flow flips over.** All
seven changes below land in a single commit: none of them is independently
reviewable (a router with the new guard but the old `AppNav` referencing
`account.ts`, or vice versa, is simply a broken app). Every view this task's
router references already exists (Tasks 2–4); every consumer of `account.ts`
is rewritten in this same task, which is what finally makes it safe to delete.

**Files:**
- Modify: `web/src/router.ts` (full rewrite — new route table + `beforeEach` guard)
- Create: `web/src/router.test.ts`
- Modify: `web/src/App.vue` (full rewrite)
- Create: `web/src/App.test.ts`
- Modify: `web/src/components/AppNav.vue` (full rewrite)
- Rename: `web/src/views/PeopleView.vue` → `web/src/views/RatingView.vue` (rewrite
  to use `useSessionStore()` instead of `getAccount()`)
- Modify: `web/src/components/ProgressDashboard.vue` (the `/people` card link
  and its "Люди" label → `/rating` / "Рейтинг", for consistency with the nav rename)
- Modify: `web/src/views/ReviewView.vue` (its two `to="/"` "На главную" links →
  `to="/profile"` / "В профиль" — "/" is no longer a real destination, `/profile`
  is home now)
- Delete: `web/src/views/DashboardView.vue`
- Delete: `web/src/components/LoginGate.vue`
- Delete: `web/src/account.ts`

**Interfaces:**
- Produces: `resolveGuard(to, session)` from `router.ts` — pure function, unit
  tested directly without a real router/navigation cycle.
- Consumes: every view from Tasks 1–5.

- [ ] **Step 1: Rewrite `web/src/router.ts`**

```ts
import { createRouter, createWebHistory, type RouteLocationNormalized } from 'vue-router'
import { watch } from 'vue'
import LoginView from './views/LoginView.vue'
import RegisterView from './views/RegisterView.vue'
import VerifyView from './views/VerifyView.vue'
import ForgotView from './views/ForgotView.vue'
import ResetView from './views/ResetView.vue'
import ProfileView from './views/ProfileView.vue'
import CourseView from './views/CourseView.vue'
import LessonView from './views/LessonView.vue'
import ReviewView from './views/ReviewView.vue'
import VocabView from './views/VocabView.vue'
import FalseFriendsView from './views/FalseFriendsView.vue'
import RatingView from './views/RatingView.vue'
import { useSessionStore } from './stores/session'

const PUBLIC_AUTH_ROUTES = new Set(['login', 'register', 'verify', 'forgot', 'reset'])

// Exported standalone (not folded into beforeEach) so it can be unit tested as
// a pure function against a session store, without a real router/navigation.
export function resolveGuard(to: RouteLocationNormalized, session: ReturnType<typeof useSessionStore>) {
  const name = to.name as string
  if (!session.user) {
    return PUBLIC_AUTH_ROUTES.has(name) ? true : { path: '/login', query: { next: to.fullPath } }
  }
  if (name === 'verify') return true
  if (PUBLIC_AUTH_ROUTES.has(name)) return { path: '/profile' }
  if (!session.user.email_verified) return { path: '/verify' }
  return true
}

// Blocks the FIRST navigation until App.vue's onMounted fetchSession() (kicked
// off before the router resolves the initial route) has settled, so the guard
// above never runs against the store's transient loading=true default.
function waitForSession(session: ReturnType<typeof useSessionStore>): Promise<void> {
  if (!session.loading) return Promise.resolve()
  return new Promise((resolve) => {
    const unwatch = watch(
      () => session.loading,
      (loading) => {
        if (!loading) {
          unwatch()
          resolve()
        }
      },
    )
  })
}

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    { path: '/login', name: 'login', component: LoginView },
    { path: '/register', name: 'register', component: RegisterView },
    { path: '/verify', name: 'verify', component: VerifyView },
    { path: '/forgot', name: 'forgot', component: ForgotView },
    { path: '/reset', name: 'reset', component: ResetView },
    { path: '/', redirect: '/profile' },
    { path: '/profile', name: 'profile', component: ProfileView },
    { path: '/course', name: 'course', component: CourseView },
    { path: '/lesson/:id', name: 'lesson', component: LessonView },
    { path: '/review', name: 'review', component: ReviewView },
    { path: '/vocab', name: 'vocab', component: VocabView },
    { path: '/false-friends', name: 'false-friends', component: FalseFriendsView },
    { path: '/rating', name: 'rating', component: RatingView },
  ],
})

router.beforeEach(async (to) => {
  const session = useSessionStore()
  await waitForSession(session)
  return resolveGuard(to, session)
})

export default router
```

- [ ] **Step 2: Create `web/src/router.test.ts`**

```ts
import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import type { RouteLocationNormalized } from 'vue-router'
import { resolveGuard } from './router'
import { useSessionStore } from './stores/session'
import type { SessionUser } from './types'

function route(name: string, fullPath = '/' + name): RouteLocationNormalized {
  return { name, fullPath } as RouteLocationNormalized
}
function verifiedUser(overrides: Partial<SessionUser> = {}): SessionUser {
  return {
    id: '1',
    name: 'Г',
    email: 'g@x.com',
    email_verified: true,
    telegram: { linked: false, username: '' },
    ...overrides,
  }
}

beforeEach(() => setActivePinia(createPinia()))

describe('resolveGuard', () => {
  it('sends a logged-out visitor from a private route to /login with next=', () => {
    const session = useSessionStore()
    session.user = null
    expect(resolveGuard(route('profile', '/profile'), session)).toEqual({
      path: '/login',
      query: { next: '/profile' },
    })
  })

  it('lets a logged-out visitor reach every public auth route', () => {
    const session = useSessionStore()
    session.user = null
    for (const name of ['login', 'register', 'verify', 'forgot', 'reset']) {
      expect(resolveGuard(route(name), session)).toBe(true)
    }
  })

  it('sends a logged-in, verified visitor away from login/register/forgot/reset to /profile', () => {
    const session = useSessionStore()
    session.user = verifiedUser()
    for (const name of ['login', 'register', 'forgot', 'reset']) {
      expect(resolveGuard(route(name), session)).toEqual({ path: '/profile' })
    }
  })

  it('still lets a logged-in visitor reach /verify', () => {
    const session = useSessionStore()
    session.user = verifiedUser()
    expect(resolveGuard(route('verify'), session)).toBe(true)
  })

  it('sends a logged-in, unverified visitor to /verify from any other route', () => {
    const session = useSessionStore()
    session.user = verifiedUser({ email_verified: false })
    expect(resolveGuard(route('profile'), session)).toEqual({ path: '/verify' })
  })

  it('lets a logged-in, verified visitor through to a private route', () => {
    const session = useSessionStore()
    session.user = verifiedUser()
    expect(resolveGuard(route('profile'), session)).toBe(true)
  })
})
```

- [ ] **Step 3: Rewrite `web/src/App.vue`**

```vue
<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { useSessionStore } from './stores/session'
import AppNav from './components/AppNav.vue'

const session = useSessionStore()
onMounted(() => {
  session.fetchSession()
})
</script>

<template>
  <div v-if="session.loading" class="flex min-h-screen items-center justify-center text-[var(--muted)]">
    Загрузка…
  </div>
  <template v-else-if="session.user">
    <AppNav :name="session.user.name" />
    <main class="mx-auto max-w-3xl px-4 pt-5 pb-[calc(5rem+env(safe-area-inset-bottom))] sm:pb-16">
      <RouterView v-slot="{ Component, route }">
        <div
          :key="route.name === 'lesson' ? route.fullPath : (route.name as string)"
          class="view-in"
        >
          <component :is="Component" />
        </div>
      </RouterView>
    </main>
  </template>
  <template v-else>
    <RouterView />
  </template>
</template>
```

- [ ] **Step 4: Create `web/src/App.test.ts`**

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import App from './App.vue'
import { useSessionStore } from './stores/session'

vi.mock('vue-router', () => ({
  RouterView: { template: '<div class="router-view-stub" />' },
}))

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('App', () => {
  it('shows a loading state, then the nav once a session resolves', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'fetchSession').mockImplementation(async () => {
      session.user = {
        id: '1',
        name: 'Гриша',
        email: '',
        email_verified: true,
        telegram: { linked: false, username: '' },
      }
      session.loading = false
    })
    const w = mount(App)
    expect(w.text()).toContain('Загрузка')
    await flushPromises()
    expect(w.findComponent({ name: 'AppNav' }).exists()).toBe(true)
  })

  it('renders no nav chrome when logged out', async () => {
    const session = useSessionStore()
    vi.spyOn(session, 'fetchSession').mockImplementation(async () => {
      session.user = null
      session.loading = false
    })
    const w = mount(App)
    await flushPromises()
    expect(w.findComponent({ name: 'AppNav' }).exists()).toBe(false)
  })
})
```

- [ ] **Step 5: Rewrite `web/src/components/AppNav.vue`**

```vue
<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { computed } from 'vue'
import {
  GraduationCap,
  Languages,
  MonitorSmartphone,
  Moon,
  Repeat,
  SunMedium,
  TriangleAlert,
  Trophy,
  User,
} from 'lucide-vue-next'
import { theme, cycleTheme, THEME_META } from '../theme'

defineProps<{ name: string }>()

const route = useRoute()

const links = [
  { to: '/profile', label: 'Профиль', icon: User },
  { to: '/course', label: 'Курс', icon: GraduationCap },
  { to: '/review', label: 'Слова', icon: Repeat },
  { to: '/vocab', label: 'Словарь', icon: Languages },
  { to: '/false-friends', label: 'Ловушки', icon: TriangleAlert },
  { to: '/rating', label: 'Рейтинг', icon: Trophy },
]

const active = computed(() => route.path)
function isActive(to: string) {
  return active.value.startsWith(to)
}

const THEME_ICON = { system: MonitorSmartphone, light: SunMedium, dark: Moon }
const themeMeta = computed(() => THEME_META[theme.value])
const themeIcon = computed(() => THEME_ICON[theme.value])
</script>

<template>
  <header
    class="sticky top-0 z-20 border-b border-[var(--border)] bg-[var(--bg)]/85 backdrop-blur"
    style="padding-top: env(safe-area-inset-top)"
  >
    <div class="mx-auto flex max-w-3xl items-center gap-1 px-3 py-2">
      <nav class="hidden gap-1 sm:flex">
        <RouterLink
          v-for="l in links"
          :key="l.to"
          :to="l.to"
          class="flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold transition"
          :class="
            isActive(l.to)
              ? 'bg-[var(--accent-soft)] text-[var(--accent)]'
              : 'text-[var(--muted)] hover:text-[var(--fg)]'
          "
        >
          <component :is="l.icon" :size="15" :stroke-width="2.25" />{{ l.label }}
        </RouterLink>
      </nav>

      <span class="serbian text-lg font-semibold sm:hidden">Српски</span>

      <button
        class="ml-auto flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-[var(--muted)] transition hover:bg-[var(--bg-soft)] hover:text-[var(--fg)]"
        :title="`Тема: ${themeMeta.label}`"
        @click="cycleTheme()"
      >
        <component :is="themeIcon" :size="17" :stroke-width="2.25" />
      </button>

      <RouterLink
        to="/profile"
        class="flex h-9 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm text-[var(--muted)] transition hover:text-[var(--fg)]"
        title="Профиль"
      >
        <span class="max-w-[7rem] truncate font-semibold text-[var(--fg)]">{{ name }}</span>
      </RouterLink>
    </div>
  </header>

  <nav
    class="fixed inset-x-0 bottom-0 z-20 grid grid-cols-6 border-t border-[var(--border)] bg-[var(--bg)]/95 backdrop-blur sm:hidden"
    style="padding-bottom: env(safe-area-inset-bottom)"
  >
    <RouterLink
      v-for="l in links"
      :key="l.to"
      :to="l.to"
      class="flex flex-col items-center gap-1 py-2 text-[10px] font-medium transition"
      :class="isActive(l.to) ? 'text-[var(--accent)]' : 'text-[var(--muted)]'"
    >
      <component :is="l.icon" :size="19" :stroke-width="2.25" />{{ l.label }}
    </RouterLink>
  </nav>
</template>
```

Note the icon list changed from the original: `Home` (the removed "Главная"
link) and `Users` (the "Люди" nav icon) are gone; `Trophy` (Рейтинг) and `User`
(Профиль) are new. `Repeat` is kept for "Слова" but no longer used for the old
"сменить пользователя" button, which no longer exists.

- [ ] **Step 6: Rename `PeopleView.vue` → `RatingView.vue`**

```bash
git mv web/src/views/PeopleView.vue web/src/views/RatingView.vue
```

Then rewrite it to use the session store instead of `getAccount()`:

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { useSessionStore } from '../stores/session'
import type { LeaderRow } from '../types'

const rows = ref<LeaderRow[]>([])
const error = ref<string | null>(null)
const session = useSessionStore()

onMounted(async () => {
  try {
    rows.value = await api.leaderboard()
  } catch (e) {
    error.value = (e as Error).message
  }
})

function activeLabel(d: string) {
  if (!d) return 'ещё не занимался'
  const today = new Date().toISOString().slice(0, 10)
  const y = new Date(Date.now() - 86400000).toISOString().slice(0, 10)
  if (d === today) return 'сегодня'
  if (d === y) return 'вчера'
  return d
}
</script>

<template>
  <h1 class="mb-1 text-2xl font-extrabold">Рейтинг</h1>
  <p class="mb-4 text-sm text-[var(--muted)]">Прогресс всех, кто занимается по этому курсу.</p>

  <p v-if="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

  <div v-else class="space-y-2">
    <div
      v-for="(r, i) in rows"
      :key="r.name"
      class="card flex items-center gap-3 p-4"
      :class="r.name === session.user?.name ? 'ring-2 ring-[var(--accent)]' : ''"
    >
      <span class="w-5 shrink-0 text-center font-mono text-sm text-[var(--muted)]">{{ i + 1 }}</span>
      <div class="min-w-0 flex-1">
        <p class="font-bold">
          {{ r.name }}
          <span v-if="r.name === session.user?.name" class="text-xs font-normal text-[var(--accent)]">— это ты</span>
        </p>
        <p class="text-sm text-[var(--muted)]">
          {{ r.lessons_done }}/{{ r.lessons_total }} уроков · {{ r.cards_known }} слов ·
          {{ activeLabel(r.last_active) }}
        </p>
      </div>
      <div class="shrink-0 text-right">
        <p class="font-bold">{{ r.streak_days }}<span v-if="r.streak_days" class="ml-0.5">🔥</span></p>
        <p class="text-[10px] text-[var(--muted)]">дней подряд</p>
      </div>
    </div>

    <p v-if="!rows.length" class="card p-6 text-center text-[var(--muted)]">Пока никого.</p>
  </div>
</template>
```

- [ ] **Step 7: Update `ProgressDashboard.vue`'s `/people` reference**

In `web/src/components/ProgressDashboard.vue`, change:

```vue
    <RouterLink v-if="leaders.length > 1" to="/people" class="card block p-5 transition hover:-translate-y-0.5">
      <div class="mb-2 flex items-baseline justify-between">
        <p class="font-bold">Люди</p>
```

to:

```vue
    <RouterLink v-if="leaders.length > 1" to="/rating" class="card block p-5 transition hover:-translate-y-0.5">
      <div class="mb-2 flex items-baseline justify-between">
        <p class="font-bold">Рейтинг</p>
```

- [ ] **Step 8: Update `ReviewView.vue`'s "На главную" links**

In `web/src/views/ReviewView.vue`, change both occurrences of:

```vue
      <RouterLink to="/" class="btn btn-primary">На главную</RouterLink>
```

and

```vue
    <RouterLink to="/" class="btn btn-primary mt-5">На главную</RouterLink>
```

to `to="/profile"` with the label `В профиль` (keep each line's own classes
unchanged, only `to` and the text change).

- [ ] **Step 9: Delete the retired files**

```bash
git rm web/src/views/DashboardView.vue web/src/components/LoginGate.vue web/src/account.ts
```

- [ ] **Step 10: Run everything and type-check**

```bash
cd web && npx vitest run && npx vue-tsc -b --noEmit
```

Expected: the full suite passes, including the earlier tasks' tests (Task 1's
`api.test.ts` 401-redirect test now exercises a router that genuinely has
`/login`), and the type-check is clean (confirms nothing still references
`account.ts`, `DashboardView.vue`, `LoginGate.vue`, or the `/people` path).

- [ ] **Step 11: Manual smoke check**

First the production-shaped path: `cd web && npm run build`, then from the repo
root `go build -o /tmp/serbian-app-smoke ./server`, run it against a scratch
SQLite DB (`-db /tmp/smoke.db -addr :8091`), and in the Browser pane: register
→ check the `LogMailer` stdout link → verify → confirm redirect to `/profile`
with the dashboard rendering, the profile card, and the bottom nav showing
"Профиль"/"Рейтинг". Confirm `/` redirects to `/profile` and a stale/no
session on `/course` bounces to `/login?next=/course`.

Then the **dev-proxy path spec 4.7 flags as a risk**: run `make dev` (Vite on
`:5173` proxying `/api` to the Go server on `:8080`) and repeat the
register→verify→login flow through `localhost:5173`. Confirm in DevTools
(Application → Cookies) that the `session` cookie actually lands on
`localhost:5173` after login through the proxy, and that a page reload keeps
the session (i.e. Vite's proxy is not stripping or mis-scoping `Set-Cookie`).
The backend cookie carries no explicit `Domain` attribute (see
`setSessionCookie` in `middleware.go`), so this is expected to work with the
current `vite.config.ts` unchanged — this step is verification, not a
code change, unless it actually fails, in which case add
`cookieDomainRewrite: ''` to the `/api` proxy entry in `web/vite.config.ts`
and re-verify.

- [ ] **Step 12: Commit**

```bash
git add web/src/router.ts web/src/router.test.ts web/src/App.vue web/src/App.test.ts web/src/components/AppNav.vue web/src/views/RatingView.vue web/src/components/ProgressDashboard.vue web/src/views/ReviewView.vue
git commit -m "feat(web): cutover to session auth — router guards, App.vue, nav, /rating rename; remove account.ts/LoginGate/DashboardView"
```

---

## After Task 6

Part 3 is functionally complete: every screen in spec Section 4 exists, the
`X-User` bridge is no longer sent by the frontend, and Telegram sign-in is
wired end-to-end but dormant until `TELEGRAM_BOT_TOKEN` is set (Part 4). Per
the subagent-driven-development process, run the whole-branch final review
after Task 6 lands, the same way Part 2's did — that review is what actually
closes this plan out, not this document.

Two things explicitly stay out of scope here (see Global Constraints):
removing the `X-User` bridge server-side (spec rollout step 5, gated on
confirming this frontend is live in prod), and SMTP/real Telegram bot
credentials/DB backups (Part 4, needs input from Гриша).
