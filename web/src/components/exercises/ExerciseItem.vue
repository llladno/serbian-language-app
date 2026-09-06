<script setup lang="ts">
import type { Exercise } from '../../types'
import TextAnswer from './TextAnswer.vue'
import ConjugateAnswer from './ConjugateAnswer.vue'
import FreeAnswer from './FreeAnswer.vue'

const props = defineProps<{ lesson: string; exercise: Exercise }>()
defineEmits<{ graded: [ok: boolean] }>()

const textTypes = ['translate', 'fill_blank', 'fix_error']
void props
</script>

<template>
  <ConjugateAnswer
    v-if="exercise.type === 'conjugate'"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :prompt="exercise.prompt"
    :forms="exercise.forms ?? []"
    :meta="exercise.meta"
    @graded="$emit('graded', $event)"
  />
  <FreeAnswer
    v-else-if="exercise.type === 'free'"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :prompt="exercise.prompt"
    @graded="$emit('graded', $event)"
  />
  <TextAnswer
    v-else-if="textTypes.includes(exercise.type)"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :type="exercise.type as 'translate' | 'fill_blank' | 'fix_error'"
    :prompt="exercise.prompt"
    @graded="$emit('graded', $event)"
  />
</template>
