<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../../api'
import type { LessonAttempt } from '../../types'
import SerbianKeys from '../SerbianKeys.vue'

const props = defineProps<{
  lesson: string
  exerciseId: string
  prompt: string
  prior?: LessonAttempt
}>()

const emit = defineEmits<{ graded: [ok: boolean] }>()

const answer = ref(props.prior?.answer ?? '')
const sample = ref<string | null>(null)
const done = ref(false)
const fromPrior = ref(!!props.prior)

async function reveal() {
  const res = await api.check(props.lesson, props.exerciseId, { answer: answer.value, self: true })
  sample.value = res.sample ?? ''
}

async function selfGrade(ok: boolean) {
  await api.check(props.lesson, props.exerciseId, { answer: answer.value, self: ok })
  done.value = true
  emit('graded', ok)
}

function retry() {
  fromPrior.value = false
  sample.value = null
  done.value = false
}
</script>

<template>
  <div class="card p-3.5">
    <p class="mb-2 whitespace-pre-wrap">{{ prompt }}</p>

    <div v-if="fromPrior" class="text-sm">
      <p class="mb-1 font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ prior!.correct ? '✓ Отмечено как «справился»' : '✗ Отмечено как «не справился»' }}
      </p>
      <p v-if="prior!.answer" class="text-[var(--muted)]">
        твой вариант: <span class="serbian text-[var(--fg)]">{{ prior!.answer }}</span>
      </p>
      <button class="mt-1.5 font-medium text-[var(--accent)]" @click="retry">Переделать</button>
    </div>

    <template v-else>
      <textarea
        v-model="answer"
        rows="2"
        :disabled="done"
        class="field w-full serbian"
        placeholder="твой вариант…"
      />
      <SerbianKeys v-if="!done" class="mt-1.5" />

      <div v-if="sample === null" class="mt-2">
        <button class="btn btn-primary" @click="reveal">Показать образец</button>
      </div>

      <div v-else-if="!done" class="mt-2 pop">
        <p class="mb-2 rounded-lg bg-[var(--bg-soft)] p-2.5 text-sm">
          <span class="text-[var(--muted)]">образец: </span><span class="serbian">{{ sample }}</span>
        </p>
        <div class="flex gap-2">
          <button class="btn text-white" style="background: var(--good)" @click="selfGrade(true)">Справился</button>
          <button class="btn text-white" style="background: var(--bad)" @click="selfGrade(false)">Не справился</button>
        </div>
      </div>

      <p v-else class="mt-2 text-sm text-[var(--muted)]">Отмечено.</p>
    </template>
  </div>
</template>
