<script setup lang="ts">
import { computed, ref } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { api } from '../../api'
import { useWrongGoesToEnd } from '../../lib/lessonRules'
import { useScript } from '../../lib/useScript'
import type { CheckResult, LessonAttempt } from '../../types'
import HintButton from '../HintButton.vue'
import SerbianKeys from '../SerbianKeys.vue'
import BottomBar from '../BottomBar.vue'

const props = defineProps<{
  lesson: string
  exerciseId: string
  prompt: string
  forms: string[]
  meta?: string
  hint?: string
  prior?: LessonAttempt
}>()

const emit = defineEmits<{ graded: [ok: boolean]; ungraded: [] }>()
// Display only: the forms sent to the server stay as typed.
const { show } = useScript()

const priorAnswers = props.prior ? props.prior.answer.split(' | ') : []
const answers = ref<string[]>(props.forms.map((_, i) => priorAnswers[i] ?? ''))
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)

// After a mistake the way forward is the lesson's own button (the exercise
// comes back at the end of the step), not an instant redo.
const wrongGoesToEnd = useWrongGoesToEnd()
const showRetry = computed(
  () => (fromPrior.value || !!result.value) && !(wrongGoesToEnd.value && result.value && !result.value.ok),
)
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
  emit('ungraded')
}
</script>

<template>
  <div class="relative">
    <button
      v-if="showRetry"
      class="icon-btn absolute right-0 top-0"
      title="Переделать"
      @click="retry"
    >
      <RefreshCw :size="15" :stroke-width="2.25" />
    </button>

    <div class="mb-5 px-8 text-center">
      <p class="mb-1.5 text-[10px] uppercase tracking-widest text-[var(--accent)]">напиши все формы</p>
      <p>
        <span class="serbian text-xl font-semibold">{{ show(prompt) }}</span>
        <span v-if="meta" class="ml-2 rounded bg-[var(--bg-soft)] px-1.5 py-0.5 text-xs text-[var(--muted)]">{{ show(meta) }}</span>
        <HintButton v-if="hint && !fromPrior && !result" :text="show(hint)" class="ml-1.5 align-middle" />
      </p>
    </div>

    <div v-if="fromPrior" class="text-center text-sm">
      <p class="mb-1 font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ prior!.correct ? '✓ Отвечено верно' : '✗ Был ответ с ошибкой' }}
      </p>
      <p class="serbian text-[var(--muted)]">{{ show(answers.filter(Boolean).join(', ')) }}</p>
    </div>

    <form v-else class="mx-auto grid max-w-sm grid-cols-1 gap-2" @submit.prevent="submit">
      <label v-for="(f, i) in forms" :key="f" class="flex items-center gap-2 text-sm">
        <span class="w-24 shrink-0 text-[var(--muted)]">{{ show(f) }}</span>
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
              ? { borderColor: result.forms[i].ok ? 'var(--good)' : result.partial ? 'var(--warning)' : 'var(--bad)' }
              : {}
          "
        />
        <span v-if="result?.forms?.[i]" class="w-4" :class="{ 'text-[var(--warning)]': result.partial && !result.forms[i].ok }">{{
          result.forms[i].ok ? '✓' : '✗'
        }}</span>
      </label>

      <SerbianKeys v-if="!result" class="col-span-full justify-center" />
    </form>
    <BottomBar v-if="!fromPrior && !result">
      <button class="btn btn-primary w-full" :disabled="pending" @click="submit">
        Проверить
      </button>
    </BottomBar>

    <div v-if="result" class="mx-auto mt-4 max-w-sm text-sm pop">
      <p v-if="result.partial" class="mb-1 font-semibold text-[var(--warning)]">
        Почти! Одна форма не та — засчитано, но запомни её
      </p>
      <p v-for="(r, i) in result.forms" :key="'t' + i" v-show="r.ok && r.typo" class="text-[var(--warning)]">
        {{ show(forms[i]) }}: засчитано, но пишется <span class="serbian font-semibold">{{ show(r.expected) }}</span>
      </p>
      <p v-for="(r, i) in result.forms" :key="i" v-show="!r.ok" class="text-[var(--muted)]">
        {{ show(forms[i]) }} → <span class="serbian font-semibold text-[var(--fg)]">{{ show(r.expected) }}</span>
      </p>
      <p v-if="result.explain && (!result.ok || result.partial)" class="mt-2 text-[var(--muted)]">{{ show(result.explain) }}</p>
    </div>
  </div>
</template>
