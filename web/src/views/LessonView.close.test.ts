import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createRouter, createMemoryHistory, RouterView } from 'vue-router'
import { defineComponent, h } from 'vue'
import LessonView from './LessonView.vue'
import { api } from '../api'
import type { Lesson, ExerciseBlock } from '../types'

// The real router, not a stub: closing a lesson is a navigation, and the lesson screen watches the
// route (it reloads when ?step= changes), so only a real route change shows what happens while leaving.
const lesson: Lesson = {
  id: '08d',
  title: 'Повторение',
  subtitle: '',
  planned: false,
  markdown: '',
  status: 'in_progress',
  steps: [{ id: '08d.1', kind: 'practice', title: 'Диктант', exercise_ids: ['08d.1.1'], status: 'in_progress' }],
}
const blocks: ExerciseBlock[] = [
  { id: '08d.1', title: 'Диктант', exercises: [{ id: '08d.1.1', type: 'translate', prompt: 'Привет' }] },
]

beforeEach(() => setActivePinia(createPinia()))
afterEach(() => vi.restoreAllMocks())

describe('closing a lesson with the cross', () => {
  it('goes to the course and does not try to load a lesson that is not there', async () => {
    const loadLesson = vi.spyOn(api, 'lesson').mockResolvedValue(structuredClone(lesson))
    vi.spyOn(api, 'exercises').mockResolvedValue(structuredClone(blocks))
    vi.spyOn(api, 'lessonAttempts').mockResolvedValue({})
    const Course = defineComponent({ render: () => h('div', { 'data-test': 'course' }, 'Курс') })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/lesson/:id', name: 'lesson', component: LessonView },
        { path: '/course', name: 'course', component: Course },
      ],
    })
    router.push('/lesson/08d')
    await router.isReady()

    const w = mount(defineComponent({ render: () => h(RouterView) }), { global: { plugins: [router] } })
    await flushPromises()
    expect(w.text()).toContain('Диктант')

    await w.get('button[aria-label="Закрыть урок"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/course')
    expect(w.find('[data-test="course"]').exists()).toBe(true)
    // The screen left the lesson: it must not go and fetch "lesson undefined" on the way out.
    expect(loadLesson.mock.calls.map((c) => c[0])).toEqual(['08d'])
  })
})
