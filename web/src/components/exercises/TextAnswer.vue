<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '../../api'
import type { CheckResult, LessonAttempt } from '../../types'
import SerbianKeys from '../SerbianKeys.vue'
import GlossedText from '../GlossedText.vue'

const props = defineProps<{
  lesson: string
  exerciseId: string
  type: 'translate' | 'fill_blank' | 'fix_error'
  prompt: string
  prior?: LessonAttempt
}>()

// fill_blank / fix_error prompts are Serbian sentences — make their words
// clickable. A translate prompt is Russian (and its Serbian answer must not be
// spoiled), so it stays plain.
const glossPrompt = computed(() => props.type === 'fill_blank' || props.type === 'fix_error')

const emit = defineEmits<{ graded: [ok: boolean] }>()

const answer = ref(props.prior?.answer ?? '')
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)
const pending = ref(false)

async function submit() {
  if (pending.value || !answer.value.trim()) return
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { answer: answer.value })
    fromPrior.value = false
    emit('graded', !!result.value.ok)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  fromPrior.value = false
  answer.value = ''
}
</script>

<template>
  <div class="card p-3.5">
    <p class="mb-2 whitespace-pre-wrap">
      <GlossedText v-if="glossPrompt" :text="prompt" />
      <template v-else>{{ prompt }}</template>
    </p>

    <!-- previously answered -->
    <div v-if="fromPrior" class="text-sm">
      <p class="mb-1 flex items-center gap-1.5 font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        <span>{{ prior!.correct ? '✓' : '✗' }}</span>
        <span>{{ prior!.correct ? 'Отвечено верно' : 'Был ответ с ошибкой' }}</span>
      </p>
      <p class="text-[var(--muted)]">ты писал: <span class="serbian text-[var(--fg)]">{{ prior!.answer }}</span></p>
      <button class="mt-1.5 font-medium text-[var(--accent)]" @click="retry">Переделать</button>
    </div>

    <form v-else-if="!result" @submit.prevent="submit">
      <div class="flex gap-2">
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
      </div>
      <SerbianKeys class="mt-1.5" />
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
