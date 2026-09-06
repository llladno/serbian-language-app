<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../../api'

const props = defineProps<{
  lesson: string
  exerciseId: string
  prompt: string
}>()

const emit = defineEmits<{ graded: [ok: boolean] }>()

const answer = ref('')
const sample = ref<string | null>(null)
const done = ref(false)

async function reveal() {
  const res = await api.check(props.lesson, props.exerciseId, { answer: answer.value, self: true })
  sample.value = res.sample ?? ''
}

async function selfGrade(ok: boolean) {
  await api.check(props.lesson, props.exerciseId, { answer: answer.value, self: ok })
  done.value = true
  emit('graded', ok)
}
</script>

<template>
  <div class="rounded-lg border border-stone-200 p-3 dark:border-stone-700">
    <p class="mb-2 whitespace-pre-wrap">{{ prompt }}</p>

    <textarea
      v-model="answer"
      rows="2"
      :disabled="done"
      class="w-full rounded border border-stone-300 bg-transparent px-2 py-1 dark:border-stone-600"
      placeholder="твой вариант…"
    />

    <div v-if="sample === null" class="mt-2">
      <button class="rounded bg-amber-600 px-3 py-1 text-white" @click="reveal">Показать образец</button>
    </div>

    <div v-else-if="!done" class="mt-2">
      <p class="mb-2 rounded bg-stone-100 p-2 text-sm dark:bg-stone-800">
        <span class="text-stone-500">образец: </span>{{ sample }}
      </p>
      <div class="flex gap-2">
        <button class="rounded bg-emerald-600 px-3 py-1 text-white" @click="selfGrade(true)">Справился</button>
        <button class="rounded bg-red-600 px-3 py-1 text-white" @click="selfGrade(false)">Не справился</button>
      </div>
    </div>

    <p v-else class="mt-2 text-sm text-stone-500">Отмечено.</p>
  </div>
</template>
