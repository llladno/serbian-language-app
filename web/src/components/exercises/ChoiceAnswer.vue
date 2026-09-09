<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../../api'
import type { CheckResult, LessonAttempt } from '../../types'

const props = defineProps<{
  lesson: string
  exerciseId: string
  prompt: string
  options: string[]
  prior?: LessonAttempt
}>()
const emit = defineEmits<{ graded: [ok: boolean, result: CheckResult] }>()

const shuffled = ref([...props.options].sort(() => Math.random() - 0.5))
const picked = ref<string | null>(props.prior?.answer ?? null)
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)
const pending = ref(false)

async function choose(opt: string) {
  if (pending.value || result.value || fromPrior.value) return
  picked.value = opt
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { answer: opt })
    emit('graded', !!result.value.ok, result.value)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  fromPrior.value = false
  picked.value = null
}
</script>

<template>
  <div class="card p-3.5">
    <p class="mb-2.5 whitespace-pre-wrap">{{ prompt }}</p>

    <div class="flex flex-wrap gap-2">
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

    <div v-if="fromPrior" class="mt-2 text-sm">
      <p class="font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ prior!.correct ? '✓ Отвечено верно' : '✗ Был неверный ответ' }}
      </p>
      <button class="mt-1 font-medium text-[var(--accent)]" @click="retry">Переделать</button>
    </div>

    <div v-else-if="result" class="pop mt-2 text-sm">
      <p class="font-semibold" :class="result.ok ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ result.ok ? '✓ Верно' : '✗ Не то' }}
      </p>
      <p v-if="!result.ok && result.expected">
        Правильно: <span class="serbian font-semibold">{{ result.expected }}</span>
      </p>
      <p v-if="result.explain" class="mt-1 text-[var(--muted)]">{{ result.explain }}</p>
      <button class="mt-1 font-medium text-[var(--accent)]" @click="retry">Ещё раз</button>
    </div>
  </div>
</template>
