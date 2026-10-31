import { createRouter, createWebHistory, type RouteLocationNormalized } from 'vue-router'
import { watch } from 'vue'

declare module 'vue-router' {
  interface RouteMeta {
    // Wider main-content column (see App.vue) — for dashboard-like views
    // that benefit from more horizontal space than the reading-width default.
    wide?: boolean
    // Hides AppNav's mobile bottom tab bar — for full-screen views (the
    // lesson player) that draw their own header/back-navigation and want
    // the extra vertical room.
    hideTabBar?: boolean
  }
}
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
import RatingView from './views/RatingView.vue'
import RepeatView from './views/RepeatView.vue'
import QuestsView from './views/QuestsView.vue'
import TariffsView from './views/TariffsView.vue'
import DevPayView from './views/DevPayView.vue'
import { useSessionStore } from './stores/session'

const PUBLIC_AUTH_ROUTES = new Set(['login', 'register', 'verify', 'forgot', 'reset'])

// Where the app opens for someone who is signed in.
const HOME = '/course'
// The screens a learner lands on when they come in, as opposed to ones they open on purpose.
const LANDING_ROUTES = new Set(['course', 'profile'])
// A learner who has not started anything is taken into the first lesson once per page load; after that the course
// map and the profile open normally, so going back out of the lesson does not bounce them in again.
let startLessonOffered = false
export function resetStartLessonOffer() {
  startLessonOffered = false
}

// Exported standalone (not folded into beforeEach) so it can be unit tested as
// a pure function against a session store, without a real router/navigation.
export function resolveGuard(to: RouteLocationNormalized, session: ReturnType<typeof useSessionStore>) {
  const name = to.name as string
  if (!session.user) {
    return PUBLIC_AUTH_ROUTES.has(name) ? true : { path: '/login', query: { next: to.fullPath } }
  }
  if (name === 'verify') return true
  if (PUBLIC_AUTH_ROUTES.has(name)) return { path: HOME }
  // A Telegram-only account has no email at all, so email_verified is false
  // by construction (nothing to verify) - only force the gate when there is
  // an actual unverified email on the account, never merely "not true".
  if (session.user.email && !session.user.email_verified) return { path: '/verify' }
  if (session.user.start_lesson && LANDING_ROUTES.has(name) && !startLessonOffered) {
    startLessonOffered = true
    return { path: '/lesson/' + session.user.start_lesson }
  }
  // Not open to everyone yet: the server's features flag says who may see it.
  if (name === 'tariffs' && !session.user.features?.tariffs) return { path: '/profile' }
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

// Like waitForSession, but for a screen that only some accounts may see: the
// cached account the page painted with is not enough to decide, so wait for the
// server's own answer about who this is.
export function waitForFreshSession(session: ReturnType<typeof useSessionStore>): Promise<void> {
  if (session.refreshed) return Promise.resolve()
  return new Promise((resolve) => {
    const unwatch = watch(
      () => session.refreshed,
      (refreshed) => {
        if (refreshed) {
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
    { path: '/', redirect: HOME },
    { path: '/profile', name: 'profile', component: ProfileView, meta: { wide: true } },
    { path: '/course', name: 'course', component: CourseView },
    { path: '/lesson/:id', name: 'lesson', component: LessonView, meta: { hideTabBar: true } },
    { path: '/review', name: 'review', component: ReviewView },
    { path: '/repeat', name: 'repeat', component: RepeatView, meta: { hideTabBar: true } },
    { path: '/mistakes', redirect: '/repeat' }, // the old «Работа над ошибками» link
    { path: '/vocab', name: 'vocab', component: VocabView },
    { path: '/false-friends', redirect: { path: '/vocab', query: { tab: 'traps' } } },
    { path: '/rating', name: 'rating', component: RatingView },
    { path: '/quests', name: 'quests', component: QuestsView },
    { path: '/tariffs', name: 'tariffs', component: TariffsView },
    // The stand-in cash desk of PAYMENTS_DEV (local development); the server answers 404 where it is off.
    { path: '/dev/pay', name: 'dev-pay', component: DevPayView, meta: { hideTabBar: true } },
  ],
})

router.beforeEach(async (to) => {
  const session = useSessionStore()
  await waitForSession(session)
  // The cached account the page painted with never carries the first-lesson offer, so a learner who came back to an
  // account that has not started yet would be left on the course map: wait for the server's own answer once.
  if (to.name === 'tariffs' || (LANDING_ROUTES.has(to.name as string) && !startLessonOffered)) {
    await waitForFreshSession(session)
  }
  return resolveGuard(to, session)
})

export default router
