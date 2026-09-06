import { createRouter, createWebHistory } from 'vue-router'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: () => import('./views/DashboardView.vue') },
    { path: '/course', name: 'course', component: () => import('./views/CourseView.vue') },
    { path: '/lesson/:id', name: 'lesson', component: () => import('./views/LessonView.vue') },
    { path: '/review', name: 'review', component: () => import('./views/ReviewView.vue') },
    { path: '/vocab', name: 'vocab', component: () => import('./views/VocabView.vue') },
    { path: '/false-friends', name: 'false-friends', component: () => import('./views/FalseFriendsView.vue') },
    { path: '/people', name: 'people', component: () => import('./views/PeopleView.vue') },
  ],
})
