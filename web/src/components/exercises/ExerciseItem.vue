<script setup lang="ts">
import type { Exercise, LessonAttempt } from '../../types'
import TextAnswer from './TextAnswer.vue'
import ConjugateAnswer from './ConjugateAnswer.vue'
import FreeAnswer from './FreeAnswer.vue'
import ChoiceAnswer from './ChoiceAnswer.vue'
import WordBankAnswer from './WordBankAnswer.vue'
import MatchAnswer from './MatchAnswer.vue'

defineProps<{ lesson: string; exercise: Exercise; prior?: LessonAttempt }>()
defineEmits<{ graded: [ok: boolean] }>()

const textTypes = ['translate', 'fill_blank', 'fix_error', 'listen']
</script>

<template>
  <ConjugateAnswer
    v-if="exercise.type === 'conjugate'"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :prompt="exercise.prompt"
    :forms="exercise.forms ?? []"
    :meta="exercise.meta"
    :prior="prior"
    @graded="$emit('graded', $event)"
  />
  <FreeAnswer
    v-else-if="exercise.type === 'free'"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :prompt="exercise.prompt"
    :prior="prior"
    @graded="$emit('graded', $event)"
  />
  <ChoiceAnswer
    v-else-if="exercise.type === 'choice'"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :prompt="exercise.prompt"
    :options="exercise.options ?? []"
    :prior="prior"
    @graded="$emit('graded', $event)"
  />
  <WordBankAnswer
    v-else-if="exercise.type === 'word_bank'"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :prompt="exercise.prompt"
    :bank="exercise.bank ?? []"
    :prior="prior"
    @graded="$emit('graded', $event)"
  />
  <MatchAnswer
    v-else-if="exercise.type === 'match'"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :prompt="exercise.prompt"
    :left="exercise.left ?? []"
    :right="exercise.right ?? []"
    :prior="prior"
    @graded="$emit('graded', $event)"
  />
  <TextAnswer
    v-else-if="textTypes.includes(exercise.type)"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :type="exercise.type as 'translate' | 'fill_blank' | 'fix_error' | 'listen'"
    :prompt="exercise.prompt"
    :audio="exercise.audio"
    :prior="prior"
    @graded="$emit('graded', $event)"
  />
</template>
