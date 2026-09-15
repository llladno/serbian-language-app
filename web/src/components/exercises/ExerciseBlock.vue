<script setup lang="ts">
import { reactive } from 'vue'
import type { ExerciseBlock, LessonAttempts } from '../../types'
import ExerciseItem from './ExerciseItem.vue'

const props = defineProps<{ lesson: string; block: ExerciseBlock; exIdx: number; priors?: LessonAttempts }>()
const emit = defineEmits<{ graded: [id: string, ok: boolean] }>()

const graded = reactive<Record<string, boolean>>({})
for (const ex of props.block.exercises) {
  const p = props.priors?.[ex.id]
  if (p) graded[ex.id] = p.correct
}
function onGraded(id: string, ok: boolean) {
  graded[id] = ok
  emit('graded', id, ok)
}
</script>

<template>
  <section class="space-y-3">
    <p v-if="block.instruction" class="text-sm text-[var(--muted)]">{{ block.instruction }}</p>
    <!-- All exercises stay mounted (v-show, not v-for+v-if) so paging back
         and forth within the block keeps each exercise's own answered/result
         state instead of losing it to a remount. -->
    <div v-for="(ex, i) in block.exercises" v-show="i === exIdx" :key="ex.id" :data-ex="ex.id">
      <ExerciseItem :lesson="lesson" :exercise="ex" :prior="priors?.[ex.id]" @graded="onGraded(ex.id, $event)" />
    </div>
  </section>
</template>
