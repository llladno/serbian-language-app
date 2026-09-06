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
  <h1 class="mb-4 text-2xl font-extrabold">Курс</h1>

  <p v-if="loading" class="text-[var(--muted)]">Загрузка…</p>
  <p v-else-if="error" class="card p-4 text-[var(--bad)]">
    {{ error }} <button class="font-semibold text-[var(--accent)]" @click="store.load(true)">повторить</button>
  </p>

  <div v-else-if="course" class="space-y-7">
    <section v-for="phase in course.phases" :key="phase.id">
      <h2 class="mb-2 text-sm font-bold uppercase tracking-wide text-[var(--muted)]">{{ phase.title }}</h2>
      <ul class="card divide-y divide-[var(--border)] overflow-hidden">
        <li v-for="lid in phase.lessons" :key="lid">
          <RouterLink
            :to="`/lesson/${lid}`"
            class="flex items-baseline gap-3 px-4 py-3 transition hover:bg-[var(--bg-soft)]"
          >
            <span class="w-6 shrink-0 font-mono text-sm text-[var(--muted)]">{{ lid }}</span>
            <span class="min-w-0 flex-1">
              <span class="font-semibold">{{ byId.get(lid)?.title || 'Урок ' + lid }}</span>
              <span class="block truncate text-sm text-[var(--muted)]">{{ byId.get(lid)?.subtitle }}</span>
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
