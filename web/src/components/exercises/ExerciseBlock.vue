<script setup lang="ts">
import type { ExerciseBlock, LessonAttempts } from '../../types'
import ExerciseItem from './ExerciseItem.vue'

// The lesson decides which exercise is on screen (`activeId`) and in what
// order they come — a wrong answer sends one to the end — so this only draws.
// `rounds` counts how many times an exercise was sent back: a new round is a
// fresh component (empty field, reshuffled options) and its earlier attempt is
// no longer shown as if it were this one.
const props = defineProps<{
  lesson: string
  block: ExerciseBlock
  activeId?: string
  priors?: LessonAttempts
  rounds?: Record<string, number>
}>()
const emit = defineEmits<{ graded: [id: string, ok: boolean]; ungraded: [id: string]; cantListen: [] }>()

const roundOf = (id: string) => props.rounds?.[id] ?? 0
</script>

<template>
  <section class="space-y-3">
    <p v-if="block.instruction" class="text-sm text-[var(--muted)]">{{ block.instruction }}</p>
    <!-- All exercises stay mounted (v-show, not v-for+v-if) so paging back
         and forth within the block keeps each exercise's own answered/result
         state instead of losing it to a remount. -->
    <div
      v-for="ex in block.exercises"
      v-show="ex.id === activeId"
      :key="ex.id + ':' + roundOf(ex.id)"
      :data-ex="ex.id"
      :class="{ 'ex-fade-in': ex.id === activeId }"
    >
      <ExerciseItem
        :lesson="lesson"
        :exercise="ex"
        :prior="roundOf(ex.id) ? undefined : priors?.[ex.id]"
        @graded="emit('graded', ex.id, $event)"
        @ungraded="emit('ungraded', ex.id)"
        @cant-listen="emit('cantListen')"
      />
    </div>
  </section>
</template>

<style scoped>
/* v-show keeps every exercise mounted (see the comment above) so a Vue
   Transition — which only fires on mount/unmount — can't animate the swap.
   A plain CSS animation still restarts whenever the class is (re)applied,
   which happens exactly when this exercise becomes the active one (the
   `ex.id === activeId` flips from false to true), v-show'd-display or not. */
@keyframes exFadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
.ex-fade-in {
  animation: exFadeIn 0.15s ease-out;
}
</style>
