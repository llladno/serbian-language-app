<script setup lang="ts">
import { reactive } from 'vue'
import type { ExerciseBlock, LessonAttempts } from '../../types'
import ExerciseItem from './ExerciseItem.vue'

const props = defineProps<{ lesson: string; block: ExerciseBlock; exIdx: number; priors?: LessonAttempts }>()
const emit = defineEmits<{ graded: [id: string, ok: boolean]; ungraded: [id: string] }>()

const graded = reactive<Record<string, boolean>>({})
for (const ex of props.block.exercises) {
  const p = props.priors?.[ex.id]
  if (p) graded[ex.id] = p.correct
}
function onGraded(id: string, ok: boolean) {
  graded[id] = ok
  emit('graded', id, ok)
}
function onUngraded(id: string) {
  delete graded[id]
  emit('ungraded', id)
}
</script>

<template>
  <section class="space-y-3">
    <p v-if="block.instruction" class="text-sm text-[var(--muted)]">{{ block.instruction }}</p>
    <!-- All exercises stay mounted (v-show, not v-for+v-if) so paging back
         and forth within the block keeps each exercise's own answered/result
         state instead of losing it to a remount. -->
    <div
      v-for="(ex, i) in block.exercises"
      v-show="i === exIdx"
      :key="ex.id"
      :data-ex="ex.id"
      :class="{ 'ex-fade-in': i === exIdx }"
    >
      <ExerciseItem
        :lesson="lesson"
        :exercise="ex"
        :prior="priors?.[ex.id]"
        @graded="onGraded(ex.id, $event)"
        @ungraded="onUngraded(ex.id)"
      />
    </div>
  </section>
</template>

<style scoped>
/* v-show keeps every exercise mounted (see the comment above) so a Vue
   Transition — which only fires on mount/unmount — can't animate the swap.
   A plain CSS animation still restarts whenever the class is (re)applied,
   which happens exactly when this exercise becomes the active one (i ===
   exIdx flips from false to true), v-show'd-display or not. */
@keyframes exFadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
.ex-fade-in {
  animation: exFadeIn 0.15s ease-out;
}
</style>
