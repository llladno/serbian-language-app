<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../../api'
import type { CheckResult } from '../../types'

const props = defineProps<{
  lesson: string
  exerciseId: string
  prompt: string
  forms: string[]
  meta?: string
}>()

const emit = defineEmits<{ graded: [ok: boolean] }>()

const answers = ref<string[]>(props.forms.map(() => ''))
const result = ref<CheckResult | null>(null)
const pending = ref(false)

async function submit() {
  if (pending.value) return
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { answers: answers.value })
    emit('graded', !!result.value.ok)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  answers.value = props.forms.map(() => '')
}
</script>

<template>
  <div class="rounded-lg border border-stone-200 p-3 dark:border-stone-700">
    <p class="mb-2">
      <span class="font-medium">{{ prompt }}</span>
      <span v-if="meta" class="ml-2 text-sm text-stone-500">{{ meta }}</span>
    </p>

    <form class="grid grid-cols-1 gap-2 sm:grid-cols-2" @submit.prevent="submit">
      <label v-for="(f, i) in forms" :key="f" class="flex items-center gap-2 text-sm">
        <span class="w-24 shrink-0 text-stone-500">{{ f }}</span>
        <input
          v-model="answers[i]"
          type="text"
          autocomplete="off"
          autocapitalize="off"
          spellcheck="false"
          :disabled="!!result"
          class="min-w-0 flex-1 rounded border border-stone-300 bg-transparent px-2 py-1 dark:border-stone-600"
          :class="result?.forms?.[i] ? (result.forms[i].ok ? 'border-emerald-500' : 'border-red-500') : ''"
        />
        <span v-if="result?.forms?.[i]" class="w-4">{{ result.forms[i].ok ? '✓' : '✗' }}</span>
      </label>

      <button
        v-if="!result"
        type="submit"
        :disabled="pending"
        class="col-span-full mt-1 justify-self-start rounded bg-amber-600 px-3 py-1 text-white disabled:opacity-50"
      >
        Проверить
      </button>
    </form>

    <div v-if="result" class="mt-2 text-sm">
      <p v-for="(r, i) in result.forms" :key="i" v-show="!r.ok" class="text-stone-500">
        {{ forms[i] }} → <span class="font-medium">{{ r.expected }}</span>
      </p>
      <button class="mt-1 underline" @click="retry">Ещё раз</button>
    </div>
  </div>
</template>
