<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '../../api'
import type { CheckResult, LessonAttempt } from '../../types'

const props = defineProps<{
  lesson: string
  exerciseId: string
  prompt: string
  bank: string[]
  prior?: LessonAttempt
}>()
const emit = defineEmits<{ graded: [ok: boolean] }>()

// chips carry a stable index so duplicates ("se", "se") stay distinct
const chips = ref(
  props.bank.map((text, i) => ({ id: i, text })).sort(() => Math.random() - 0.5),
)
const picked = ref<number[]>([])
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)
const pending = ref(false)

const available = computed(() => chips.value.filter((c) => !picked.value.includes(c.id)))
const sentence = computed(() =>
  picked.value.map((id) => chips.value.find((c) => c.id === id)!.text).join(' '),
)

function add(id: number) {
  if (result.value) return
  picked.value.push(id)
}
function removeAt(i: number) {
  if (result.value) return
  picked.value.splice(i, 1)
}

async function submit() {
  if (pending.value || picked.value.length === 0) return
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { answer: sentence.value })
    emit('graded', !!result.value.ok)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  fromPrior.value = false
  picked.value = []
}
</script>

<template>
  <div class="card p-3.5">
    <p class="mb-2.5 whitespace-pre-wrap">{{ prompt }}</p>

    <div v-if="fromPrior" class="text-sm">
      <p class="mb-1 font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ prior!.correct ? '✓ Отвечено верно' : '✗ Был неверный ответ' }}
      </p>
      <p class="text-[var(--muted)]">ты собрал: <span class="serbian text-[var(--fg)]">{{ prior!.answer }}</span></p>
      <button class="mt-1.5 font-medium text-[var(--accent)]" @click="retry">Переделать</button>
    </div>

    <template v-else>
      <div class="mb-2 flex min-h-11 flex-wrap items-center gap-1.5 rounded-lg border border-dashed border-[var(--border)] p-2">
        <button
          v-for="(id, i) in picked"
          :key="id"
          class="rounded-md bg-[var(--accent-soft)] px-2 py-1 text-sm serbian text-[var(--fg)]"
          :disabled="!!result"
          @click="removeAt(i)"
        >
          {{ chips.find((c) => c.id === id)!.text }}
        </button>
        <span v-if="!picked.length" class="text-sm text-[var(--muted)]">нажимай слова по порядку</span>
      </div>

      <div class="flex flex-wrap gap-1.5">
        <button
          v-for="c in available"
          :key="c.id"
          class="btn btn-ghost serbian !py-1"
          :disabled="!!result"
          @click="add(c.id)"
        >
          {{ c.text }}
        </button>
      </div>

      <div v-if="!result" class="mt-2.5">
        <button class="btn btn-primary disabled:opacity-50" :disabled="pending || !picked.length" @click="submit">
          Проверить
        </button>
      </div>

      <div v-else class="pop mt-2 text-sm">
        <p class="mb-1 font-semibold" :class="result.ok ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
          {{ result.ok ? '✓ Верно' : result.near_miss ? 'Почти — опечатка?' : '✗ Не совсем' }}
        </p>
        <p v-if="!result.ok && result.expected">
          Правильно: <span class="serbian font-semibold">{{ result.expected }}</span>
        </p>
        <p v-if="result.explain" class="mt-1 text-[var(--muted)]">{{ result.explain }}</p>
        <button class="mt-1 font-medium text-[var(--accent)]" @click="retry">Ещё раз</button>
      </div>
    </template>
  </div>
</template>
