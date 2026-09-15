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
  // A Telegram-only account has no email at all, so email_verified is false
  // by construction (nothing to verify) - only force the gate when there is
  // an actual unverified email on the account, never merely "not true".
  if (session.user.email && !session.user.email_verified) return { path: '/verify' }
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
    { path: '/profile', name: 'profile', component: ProfileView, meta: { wide: true } },
    { path: '/course', name: 'course', component: CourseView },
    { path: '/lesson/:id', name: 'lesson', component: LessonView, meta: { hideTabBar: true } },
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
