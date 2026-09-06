<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useCourseStore } from '../stores/course'
import StatusBadge from '../components/StatusBadge.vue'
import type { LessonRef } from '../types'

const store = useCourseStore()
const { course, loading, error } = storeToRefs(store)

onMounted(() => store.load())

const byId = computed(() => {
  const m = new Map<string, LessonRef>()
  course.value?.lessons.forEach((l) => m.set(l.id, l))
  return m
})
</script>

<template>
  <h1 class="mb-4 text-2xl font-bold">Курс</h1>

  <p v-if="loading" class="text-stone-500">Загрузка…</p>
  <p v-else-if="error" class="rounded bg-red-100 p-3 text-red-800 dark:bg-red-950 dark:text-red-200">
    {{ error }} <button class="underline" @click="store.load(true)">повторить</button>
  </p>

  <div v-else-if="course" class="space-y-8">
    <section v-for="phase in course.phases" :key="phase.id">
      <h2 class="mb-2 text-lg font-semibold text-stone-600 dark:text-stone-300">{{ phase.title }}</h2>
      <ul class="divide-y divide-stone-200 overflow-hidden rounded-lg border border-stone-200 dark:divide-stone-700 dark:border-stone-700">
        <li v-for="lid in phase.lessons" :key="lid">
          <RouterLink
            :to="`/lesson/${lid}`"
            class="flex items-baseline gap-3 px-4 py-3 hover:bg-stone-100 dark:hover:bg-stone-800"
          >
            <span class="w-6 shrink-0 font-mono text-sm text-stone-400">{{ lid }}</span>
            <span class="min-w-0 flex-1">
              <span class="font-medium">{{ byId.get(lid)?.title || 'Урок ' + lid }}</span>
              <span class="block truncate text-sm text-stone-500">{{ byId.get(lid)?.subtitle }}</span>
            </span>
            <StatusBadge
              v-if="byId.get(lid)"
              :status="byId.get(lid)!.status"
              :planned="byId.get(lid)!.planned"
            />
          </RouterLink>
        </li>
      </ul>
    </section>
  </div>
</template>
