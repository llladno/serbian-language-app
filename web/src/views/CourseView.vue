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

function badgeClass(lesson?: LessonRef) {
  if (lesson?.status === 'done') return 'bg-[color-mix(in_srgb,var(--good)_16%,transparent)] text-[var(--good)]'
  if (lesson?.status === 'in_progress') return 'bg-[var(--accent)] text-white'
  return 'bg-[var(--bg-soft)] text-[var(--muted)]'
}
</script>

<template>
  <h1 class="mb-4 text-2xl font-extrabold">Курс</h1>

  <p v-if="loading" class="text-[var(--muted)]">Загрузка…</p>
  <p v-else-if="error" class="card p-4 text-[var(--bad)]">
    {{ error }} <button class="font-semibold text-[var(--accent)]" @click="store.load(true)">повторить</button>
  </p>

  <div v-else-if="course" class="space-y-8">
    <section v-for="phase in course.phases" :key="phase.id">
      <h2 class="mb-3 text-sm font-bold uppercase tracking-wide text-[var(--muted)]">{{ phase.title }}</h2>
      <div class="space-y-3">
        <RouterLink
          v-for="lid in phase.lessons"
          :key="lid"
          :to="`/lesson/${lid}`"
          class="card flex items-center gap-3.5 p-4 transition active:scale-[0.99] hover:-translate-y-0.5"
          :class="byId.get(lid)?.status === 'in_progress' ? 'ring-2 ring-[var(--accent)]' : ''"
        >
          <span
            class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-sm font-bold tabular-nums"
            :class="badgeClass(byId.get(lid))"
          >
            {{ lid }}
          </span>
          <span class="min-w-0 flex-1">
            <span class="block font-semibold">{{ byId.get(lid)?.title || 'Урок ' + lid }}</span>
            <span class="block truncate text-sm text-[var(--muted)]">{{ byId.get(lid)?.subtitle }}</span>
          </span>
          <StatusBadge
            v-if="byId.get(lid)"
            :status="byId.get(lid)!.status"
            :planned="byId.get(lid)!.planned"
          />
        </RouterLink>
      </div>
    </section>
  </div>
</template>
