<script setup lang="ts">
import { computed, ref } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { api } from '../../api'
import type { CheckResult, LessonAttempt } from '../../types'
import SerbianKeys from '../SerbianKeys.vue'
import SpeakButton from '../SpeakButton.vue'
import GlossedText from '../GlossedText.vue'
import BottomBar from '../BottomBar.vue'

const props = defineProps<{
  lesson: string
  exerciseId: string
  type: 'translate' | 'fill_blank' | 'fix_error' | 'listen'
  prompt: string
  audio?: string
  prior?: LessonAttempt
}>()

// fill_blank / fix_error prompts are Serbian sentences — make their words
// clickable. A translate prompt is Russian (and its Serbian answer must not be
// spoiled), so it stays plain. A listen prompt is just an instruction; the
// sentence to transcribe lives only in the audio.
const glossPrompt = computed(() => props.type === 'fill_blank' || props.type === 'fix_error')

const emit = defineEmits<{ graded: [ok: boolean, result: CheckResult]; ungraded: [] }>()

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
    emit('graded', !!result.value.ok, result.value)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  fromPrior.value = false
  answer.value = ''
  emit('ungraded')
}
</script>

<template>
  <div class="relative text-center">
    <button
      v-if="fromPrior || result"
      class="icon-btn absolute right-0 top-0"
      title="Переделать"
      @click="retry"
    >
      <RefreshCw :size="15" :stroke-width="2.25" />
    </button>

    <div v-if="type === 'listen'" class="mb-5 flex flex-col items-center gap-3 px-8">
      <SpeakButton :src="audio" :size="52" />
      <span class="text-sm text-[var(--muted)]">{{ prompt || 'Напиши, что слышишь' }}</span>
    </div>
    <p v-else class="mb-5 whitespace-pre-wrap px-8 text-xl font-medium">
      <GlossedText v-if="glossPrompt" :text="prompt" />
      <template v-else>{{ prompt }}</template>
    </p>

    <!-- previously answered -->
    <div v-if="fromPrior" class="text-sm">
      <p class="mb-1 flex items-center justify-center gap-1.5 font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        <span>{{ prior!.correct ? '✓' : '✗' }}</span>
        <span>{{ prior!.correct ? 'Отвечено верно' : 'Был ответ с ошибкой' }}</span>
      </p>
      <p class="text-[var(--muted)]">ты писал: <span class="serbian text-[var(--fg)]">{{ prior!.answer }}</span></p>
    </div>

    <form v-if="!fromPrior && !result" class="mx-auto max-w-sm" @submit.prevent="submit">
      <input
        v-model="answer"
        type="text"
        autocomplete="off"
        autocapitalize="off"
        autocorrect="off"
        spellcheck="false"
        class="field w-full text-center"
        placeholder="ответ…"
      />
      <SerbianKeys class="mt-1.5 justify-center" />
    </form>
    <BottomBar v-if="!fromPrior && !result">
      <button class="btn btn-primary w-full disabled:opacity-50" :disabled="pending || !answer.trim()" @click="submit">
        Проверить
      </button>
    </BottomBar>

    <div v-if="result" class="pop mt-4">
      <p
        class="mb-1 flex items-center justify-center gap-1.5 font-semibold"
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
    </div>
  </div>
</template>
