<script setup lang="ts">
// Tap a word in either column, then its partner in the other column — the
// pair locks in place. Tapping a locked pair again frees it up for a redo.
import { computed, reactive, ref } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { api } from '../../api'
import type { CheckResult, LessonAttempt } from '../../types'
import BottomBar from '../BottomBar.vue'

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
// sparse: only left words that are currently paired have an entry
const picks = reactive<Record<string, string>>({})
const selectedLeft = ref<string | null>(null)
const selectedRight = ref<string | null>(null)
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)
const pending = ref(false)

const complete = computed(() => props.left.every((l) => l in picks))

function isRightUsed(r: string) {
  return Object.values(picks).includes(r)
}
function leftFor(r: string) {
  return props.left.find((l) => picks[l] === r)
}

function tapLeft(l: string) {
  if (result.value) return
  if (l in picks) {
    delete picks[l]
    return
  }
  if (selectedRight.value) {
    picks[l] = selectedRight.value
    selectedRight.value = null
    return
  }
  selectedLeft.value = selectedLeft.value === l ? null : l
}
function tapRight(r: string) {
  if (result.value) return
  if (isRightUsed(r)) {
    const l = leftFor(r)
    if (l) delete picks[l]
    return
  }
  if (selectedLeft.value) {
    picks[selectedLeft.value] = r
    selectedLeft.value = null
    return
  }
  selectedRight.value = selectedRight.value === r ? null : r
}

function leftClass(l: string) {
  if (result.value) {
    if (!(l in picks)) return 'btn-ghost opacity-50'
    return result.value.match?.[l] ? 'btn-ghost ring-2 ring-[var(--good)]' : 'btn-ghost ring-2 ring-[var(--bad)]'
  }
  if (l in picks) return 'border border-[var(--accent)] bg-[var(--accent-soft)] text-[var(--accent)]'
  if (selectedLeft.value === l) return 'btn-ghost ring-2 ring-[var(--accent)]'
  return 'btn-ghost'
}
function rightClass(r: string) {
  if (result.value) {
    const l = leftFor(r)
    if (!l) return 'btn-ghost opacity-50'
    return result.value.match?.[l] ? 'btn-ghost ring-2 ring-[var(--good)]' : 'btn-ghost ring-2 ring-[var(--bad)]'
  }
  if (isRightUsed(r)) return 'border border-[var(--accent)] bg-[var(--accent-soft)] text-[var(--accent)]'
  if (selectedRight.value === r) return 'btn-ghost ring-2 ring-[var(--accent)]'
  return 'btn-ghost'
}

async function submit() {
  if (pending.value || !complete.value) return
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { pairs: { ...picks } })
    emit('graded', !!result.value.ok)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  fromPrior.value = false
  for (const k of Object.keys(picks)) delete picks[k]
  selectedLeft.value = null
  selectedRight.value = null
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

    <p class="mb-5 whitespace-pre-wrap px-8 text-xl font-medium">{{ prompt }}</p>

    <div v-if="fromPrior" class="text-sm">
      <p class="font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ prior!.correct ? '✓ Отвечено верно' : '✗ Был неверный ответ' }}
      </p>
    </div>

    <template v-else>
      <div class="mx-auto grid max-w-sm grid-cols-2 gap-2.5">
        <div class="flex flex-col gap-2">
          <button
            v-for="l in left"
            :key="l"
            type="button"
            class="btn serbian w-full"
            :class="leftClass(l)"
            :disabled="!!result"
            @click="tapLeft(l)"
          >
            {{ l }}
          </button>
        </div>
        <div class="flex flex-col gap-2">
          <button
            v-for="r in choices"
            :key="r"
            type="button"
            class="btn serbian w-full"
            :class="rightClass(r)"
            :disabled="!!result"
            @click="tapRight(r)"
          >
            {{ r }}
          </button>
        </div>
      </div>

      <BottomBar v-if="!result">
        <button class="btn btn-primary w-full disabled:opacity-50" :disabled="pending || !complete" @click="submit">
          Проверить
        </button>
      </BottomBar>

      <div v-else class="mt-4 text-sm pop">
        <p class="font-semibold" :class="result.ok ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
          {{ result.ok ? '✓ Верно' : '✗ Есть ошибки' }}
        </p>
      </div>
    </template>
  </div>
</template>
