<script setup lang="ts">
import { reactive, computed } from 'vue'
import type { ExerciseBlock } from '../../types'
import ExerciseItem from './ExerciseItem.vue'

const props = defineProps<{ lesson: string; block: ExerciseBlock }>()

const graded = reactive<Record<string, boolean>>({})

function onGraded(id: string, ok: boolean) {
  graded[id] = ok
}

const score = computed(() => {
  const ids = Object.keys(graded)
  return { done: ids.length, ok: ids.filter((i) => graded[i]).length, total: props.block.exercises.length }
})
</script>

<template>
  <section class="space-y-3">
    <div class="flex items-baseline justify-between">
      <h3 class="text-lg font-semibold">{{ block.title }}</h3>
      <span v-if="score.done" class="text-sm text-stone-500">{{ score.ok }} / {{ score.total }}</span>
    </div>
    <p v-if="block.instruction" class="text-sm text-stone-500">{{ block.instruction }}</p>
    <ExerciseItem
      v-for="ex in block.exercises"
      :key="ex.id"
      :lesson="lesson"
      :exercise="ex"
      @graded="onGraded(ex.id, $event)"
    />
  </section>
</template>
