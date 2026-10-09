<script setup lang="ts">
import type { Exercise, LessonAttempt } from '../../types'
import TextAnswer from './TextAnswer.vue'
import ConjugateAnswer from './ConjugateAnswer.vue'
import ChoiceAnswer from './ChoiceAnswer.vue'
import WordBankAnswer from './WordBankAnswer.vue'
import MatchAnswer from './MatchAnswer.vue'

defineProps<{ lesson: string; exercise: Exercise; prior?: LessonAttempt }>()
defineEmits<{ graded: [ok: boolean]; ungraded: []; cantListen: [] }>()

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
    @ungraded="$emit('ungraded')"
  />
  <ChoiceAnswer
    v-else-if="exercise.type === 'choice'"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :prompt="exercise.prompt"
    :options="exercise.options ?? []"
    :explain="exercise.explain"
    :prior="prior"
    @graded="$emit('graded', $event)"
    @ungraded="$emit('ungraded')"
  />
  <WordBankAnswer
    v-else-if="exercise.type === 'word_bank'"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :prompt="exercise.prompt"
    :bank="exercise.bank ?? []"
    :explain="exercise.explain"
    :prior="prior"
    @graded="$emit('graded', $event)"
    @ungraded="$emit('ungraded')"
  />
  <MatchAnswer
    v-else-if="exercise.type === 'match'"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :prompt="exercise.prompt"
    :left="exercise.left ?? []"
    :right="exercise.right ?? []"
    :explain="exercise.explain"
    :prior="prior"
    @graded="$emit('graded', $event)"
    @ungraded="$emit('ungraded')"
  />
  <TextAnswer
    v-else-if="textTypes.includes(exercise.type)"
    :lesson="lesson"
    :exercise-id="exercise.id"
    :type="exercise.type as 'translate' | 'fill_blank' | 'fix_error' | 'listen'"
    :prompt="exercise.prompt"
    :audio="exercise.audio"
    :text="exercise.text"
    :explain="exercise.explain"
    :prior="prior"
    @graded="$emit('graded', $event)"
    @ungraded="$emit('ungraded')"
    @cant-listen="$emit('cantListen')"
  />
</template>
