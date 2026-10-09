<script setup lang="ts">
import { computed, ref } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { api } from '../../api'
import { useWrongGoesToEnd } from '../../lib/lessonRules'
import type { CheckResult, LessonAttempt } from '../../types'
import HintButton from '../HintButton.vue'
import PhraseSpeak from '../PhraseSpeak.vue'
import PromptSpeak from '../PromptSpeak.vue'

const props = defineProps<{
  lesson: string
  exerciseId: string
  prompt: string
  options: string[]
  explain?: string
  prior?: LessonAttempt
}>()
const emit = defineEmits<{ graded: [ok: boolean, result: CheckResult, answer: string]; ungraded: [] }>()

const shuffled = ref([...props.options].sort(() => Math.random() - 0.5))
const picked = ref<string | null>(props.prior?.answer ?? null)
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)

// After a mistake the way forward is the lesson's own button (the exercise
// comes back at the end of the step), not an instant redo.
const wrongGoesToEnd = useWrongGoesToEnd()
const showRetry = computed(
  () => (fromPrior.value || !!result.value) && !(wrongGoesToEnd.value && result.value && !result.value.ok),
)
const pending = ref(false)

async function choose(opt: string) {
  if (pending.value || result.value || fromPrior.value) return
  picked.value = opt
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { answer: opt })
    emit('graded', !!result.value.ok, result.value, opt)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  fromPrior.value = false
  picked.value = null
  emit('ungraded')
}
</script>

<template>
  <div class="relative text-center">
    <button
      v-if="showRetry"
      class="icon-btn absolute right-0 top-0"
      title="Переделать"
      @click="retry"
    >
      <RefreshCw :size="15" :stroke-width="2.25" />
    </button>

    <p class="mb-5 flex flex-wrap items-center justify-center gap-1.5 whitespace-pre-wrap px-8 text-xl font-medium">
      {{ prompt }}
      <PromptSpeak :prompt="prompt" />
      <HintButton v-if="explain && !result && !fromPrior" :text="explain" />
    </p>

    <div class="flex flex-wrap justify-center gap-2.5">
      <button
        v-for="o in shuffled"
        :key="o"
        class="btn btn-ghost serbian"
        :class="{
          'ring-2 ring-[var(--good)]': (result?.ok || (fromPrior && prior?.correct)) && picked === o,
          'ring-2 ring-[var(--bad)]': ((result && !result.ok) || (fromPrior && !prior?.correct)) && picked === o,
        }"
        :disabled="!!result || fromPrior"
        @click="choose(o)"
      >
        {{ o }}
      </button>
    </div>

    <div v-if="fromPrior" class="mt-4 text-sm">
      <p class="font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ prior!.correct ? '✓ Отвечено верно' : '✗ Был неверный ответ' }}
      </p>
      <p v-if="prior!.correct" class="mt-1 flex justify-center"><PhraseSpeak :text="prior!.answer" /></p>
    </div>

    <div v-else-if="result" class="pop mt-4 text-sm">
      <p class="font-semibold" :class="result.ok ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ result.ok ? '✓ Верно' : '✗ Не то' }}
      </p>
      <p v-if="!result.ok && result.expected">
        Правильно: <span class="serbian font-semibold">{{ result.expected }}</span>
      </p>
      <!-- only ever the right phrase: the options themselves stay silent, a wrong one must not be voiced -->
      <p class="mt-1 flex justify-center">
        <PhraseSpeak :text="result.ok ? picked! : result.expected" />
      </p>
      <p v-if="result.explain" class="mt-1 text-[var(--muted)]">{{ result.explain }}</p>
    </div>
  </div>
</template>
