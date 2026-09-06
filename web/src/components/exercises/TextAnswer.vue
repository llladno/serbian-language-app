<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../../api'
import type { CheckResult } from '../../types'

const props = defineProps<{
  lesson: string
  exerciseId: string
  type: 'translate' | 'fill_blank' | 'fix_error'
  prompt: string
}>()

const emit = defineEmits<{ graded: [ok: boolean] }>()

const answer = ref('')
const result = ref<CheckResult | null>(null)
const pending = ref(false)

async function submit() {
  if (pending.value || !answer.value.trim()) return
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { answer: answer.value })
    emit('graded', !!result.value.ok)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  answer.value = ''
}
</script>

<template>
  <div class="card p-3.5">
    <p class="mb-2 whitespace-pre-wrap">{{ prompt }}</p>

    <form v-if="!result" class="flex gap-2" @submit.prevent="submit">
      <input
        v-model="answer"
        type="text"
        autocomplete="off"
        autocapitalize="off"
        autocorrect="off"
        spellcheck="false"
        class="field flex-1"
        placeholder="ответ…"
      />
      <button type="submit" :disabled="pending" class="btn btn-primary disabled:opacity-50">Проверить</button>
    </form>

    <div v-else class="pop">
      <p
        class="mb-1 flex items-center gap-1.5 font-semibold"
        :class="result.ok ? 'text-[var(--good)]' : 'text-[var(--bad)]'"
      >
        <span>{{ result.ok ? '✓' : '✗' }}</span>
        <span>{{ result.ok ? 'Верно' : result.near_miss ? 'Почти — опечатка?' : 'Не совсем' }}</span>
      </p>
      <p v-if="result.diff?.length && !result.ok" class="mb-1">
        <span v-for="(c, i) in result.diff" :key="i" :class="c.ok ? '' : 'chunk-wrong'">{{ c.text
        }}{{ i < result.diff.length - 1 ? ' ' : '' }}</span>
      </p>
      <p v-if="!result.ok && result.expected" class="text-sm">
        Правильно: <span class="serbian font-semibold">{{ result.expected }}</span>
      </p>
      <p v-if="result.explain" class="mt-1 text-sm text-[var(--muted)]">{{ result.explain }}</p>
      <button class="mt-2 text-sm font-medium text-[var(--accent)]" @click="retry">Ещё раз</button>
    </div>
  </div>
</template>
