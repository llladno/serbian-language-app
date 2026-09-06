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
  <div class="rounded-lg border border-stone-200 p-3 dark:border-stone-700">
    <p class="mb-2 whitespace-pre-wrap">{{ prompt }}</p>

    <form v-if="!result" class="flex gap-2" @submit.prevent="submit">
      <input
        v-model="answer"
        type="text"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        class="flex-1 rounded border border-stone-300 bg-transparent px-2 py-1 dark:border-stone-600"
        placeholder="ответ…"
      />
      <button
        type="submit"
        :disabled="pending"
        class="rounded bg-amber-600 px-3 py-1 text-white disabled:opacity-50"
      >
        Проверить
      </button>
    </form>

    <div v-else>
      <p class="mb-1 font-medium" :class="result.ok ? 'text-emerald-600' : 'text-red-600'">
        {{ result.ok ? '✓ Верно' : '✗ Не совсем' }}
      </p>
      <p v-if="result.diff?.length" class="mb-1">
        <span
          v-for="(c, i) in result.diff"
          :key="i"
          :class="c.ok ? '' : 'chunk-wrong'"
        >{{ c.text }}{{ i < result.diff.length - 1 ? ' ' : '' }}</span>
      </p>
      <p v-if="!result.ok && result.expected" class="text-sm">
        Правильно: <span class="font-medium">{{ result.expected }}</span>
      </p>
      <p v-if="result.explain" class="mt-1 text-sm text-stone-500">{{ result.explain }}</p>
      <button class="mt-2 text-sm underline" @click="retry">Ещё раз</button>
    </div>
  </div>
</template>
