import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from './views/DashboardView.vue'
import CourseView from './views/CourseView.vue'
import LessonView from './views/LessonView.vue'
import ReviewView from './views/ReviewView.vue'
import VocabView from './views/VocabView.vue'
import FalseFriendsView from './views/FalseFriendsView.vue'
import PeopleView from './views/PeopleView.vue'

export default createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    { path: '/', name: 'dashboard', component: DashboardView },
    { path: '/course', name: 'course', component: CourseView },
    { path: '/lesson/:id', name: 'lesson', component: LessonView },
    { path: '/review', name: 'review', component: ReviewView },
    { path: '/vocab', name: 'vocab', component: VocabView },
    { path: '/false-friends', name: 'false-friends', component: FalseFriendsView },
    { path: '/people', name: 'people', component: PeopleView },
  ],
})
