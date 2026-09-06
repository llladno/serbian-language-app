import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'
import type { Course, LessonStatus } from '../types'

export const useCourseStore = defineStore('course', () => {
  const course = ref<Course | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function load(force = false) {
    if (course.value && !force) return
    loading.value = true
    error.value = null
    try {
      course.value = await api.course()
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      loading.value = false
    }
  }

  function setStatus(id: string, status: LessonStatus) {
    const l = course.value?.lessons.find((x) => x.id === id)
    if (l) l.status = status
  }

  async function markDone(id: string) {
    await api.completeLesson(id)
    setStatus(id, 'done')
  }

  return { course, loading, error, load, markDone, setStatus }
})
