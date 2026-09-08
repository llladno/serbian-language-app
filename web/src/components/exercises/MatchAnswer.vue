<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '../../api'
import type { CheckResult, LessonAttempt } from '../../types'

const props = defineProps<{
  lesson: string
  exerciseId: string
  prompt: string
  left: string[]
  right: string[]
  prior?: LessonAttempt
}>()
const emit = defineEmits<{ graded: [ok: boolean] }>()

const choices = ref([...props.right].sort(() => Math.random() - 0.5))
const picks = ref<Record<string, string>>({})
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)
const pending = ref(false)

const complete = computed(() => props.left.every((l) => picks.value[l]))

async function submit() {
  if (pending.value || !complete.value) return
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { pairs: { ...picks.value } })
    emit('graded', !!result.value.ok)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  fromPrior.value = false
  picks.value = {}
}
</script>

<template>
  <div class="card p-3.5">
    <p class="mb-2.5 whitespace-pre-wrap">{{ prompt }}</p>

    <div v-if="fromPrior" class="text-sm">
      <p class="mb-1 font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ prior!.correct ? '✓ Отвечено верно' : '✗ Был неверный ответ' }}
      </p>
      <button class="mt-1 font-medium text-[var(--accent)]" @click="retry">Переделать</button>
    </div>

    <template v-else>
      <div class="space-y-2">
        <div v-for="l in left" :key="l" class="flex items-center gap-2">
          <span class="serbian w-2/5 shrink-0 font-medium">{{ l }}</span>
          <select
            v-model="picks[l]"
            class="field flex-1"
            :disabled="!!result"
            :class="{
              'ring-2 ring-[var(--good)]': result?.match?.[l] === true,
              'ring-2 ring-[var(--bad)]': result?.match?.[l] === false,
            }"
          >
            <option value="" disabled>— выбери —</option>
            <option v-for="r in choices" :key="r" :value="r">{{ r }}</option>
          </select>
        </div>
      </div>

      <div v-if="!result" class="mt-2.5">
        <button class="btn btn-primary disabled:opacity-50" :disabled="pending || !complete" @click="submit">
          Проверить
        </button>
      </div>

      <div v-else class="pop mt-2 text-sm">
        <p class="font-semibold" :class="result.ok ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
          {{ result.ok ? '✓ Верно' : '✗ Есть ошибки' }}
        </p>
        <button class="mt-1 font-medium text-[var(--accent)]" @click="retry">Ещё раз</button>
      </div>
    </template>
  </div>
</template>
