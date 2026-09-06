<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../../api'
import type { CheckResult, LessonAttempt } from '../../types'
import SerbianKeys from '../SerbianKeys.vue'

const props = defineProps<{
  lesson: string
  exerciseId: string
  prompt: string
  forms: string[]
  meta?: string
  prior?: LessonAttempt
}>()

const emit = defineEmits<{ graded: [ok: boolean] }>()

const priorAnswers = props.prior ? props.prior.answer.split(' | ') : []
const answers = ref<string[]>(props.forms.map((_, i) => priorAnswers[i] ?? ''))
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)
const pending = ref(false)

async function submit() {
  if (pending.value) return
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { answers: answers.value })
    fromPrior.value = false
    emit('graded', !!result.value.ok)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  fromPrior.value = false
  answers.value = props.forms.map(() => '')
}
</script>

<template>
  <div class="card p-3.5">
    <p class="mb-2.5">
      <span class="serbian font-semibold">{{ prompt }}</span>
      <span v-if="meta" class="ml-2 rounded bg-[var(--bg-soft)] px-1.5 py-0.5 text-xs text-[var(--muted)]">{{ meta }}</span>
    </p>

    <div v-if="fromPrior" class="text-sm">
      <p class="mb-1 font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ prior!.correct ? '✓ Отвечено верно' : '✗ Был ответ с ошибкой' }}
      </p>
      <p class="serbian text-[var(--muted)]">{{ answers.filter(Boolean).join(', ') }}</p>
      <button class="mt-1.5 font-medium text-[var(--accent)]" @click="retry">Переделать</button>
    </div>

    <form v-else class="grid grid-cols-1 gap-2 sm:grid-cols-2" @submit.prevent="submit">
      <label v-for="(f, i) in forms" :key="f" class="flex items-center gap-2 text-sm">
        <span class="w-24 shrink-0 text-[var(--muted)]">{{ f }}</span>
        <input
          v-model="answers[i]"
          type="text"
          autocomplete="off"
          autocapitalize="off"
          autocorrect="off"
          spellcheck="false"
          :disabled="!!result"
          class="field min-w-0 flex-1 serbian"
          :style="
            result?.forms?.[i]
              ? { borderColor: result.forms[i].ok ? 'var(--good)' : 'var(--bad)' }
              : {}
          "
        />
        <span v-if="result?.forms?.[i]" class="w-4">{{ result.forms[i].ok ? '✓' : '✗' }}</span>
      </label>

      <SerbianKeys v-if="!result" class="col-span-full" />

      <button
        v-if="!result"
        type="submit"
        :disabled="pending"
        class="btn btn-primary col-span-full mt-1 justify-self-start disabled:opacity-50"
      >
        Проверить
      </button>
    </form>

    <div v-if="result" class="mt-2 text-sm pop">
      <p v-for="(r, i) in result.forms" :key="i" v-show="!r.ok" class="text-[var(--muted)]">
        {{ forms[i] }} → <span class="serbian font-semibold text-[var(--fg)]">{{ r.expected }}</span>
      </p>
      <button class="mt-1 font-medium text-[var(--accent)]" @click="retry">Ещё раз</button>
    </div>
  </div>
</template>
