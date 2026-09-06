<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import type { Vocab } from '../types'

const all = ref<Vocab[]>([])
const q = ref('')
const lesson = ref('')
const tag = ref('')
const script = ref<'latin' | 'cyrillic'>('latin')
const error = ref<string | null>(null)

let t: ReturnType<typeof setTimeout>
watch(q, () => {
  clearTimeout(t)
  t = setTimeout(load, 200)
})

async function load() {
  try {
    all.value = await api.vocab({ q: q.value || undefined })
    error.value = null
  } catch (e) {
    error.value = (e as Error).message
  }
}
onMounted(load)

const lessons = computed(() => [...new Set(all.value.map((v) => v.lesson).filter(Boolean))].sort())
const tags = computed(() => [...new Set(all.value.flatMap((v) => v.tags ?? []))].sort())

const rows = computed(() =>
  all.value.filter(
    (v) => (!lesson.value || v.lesson === lesson.value) && (!tag.value || (v.tags ?? []).includes(tag.value)),
  ),
)

// --- quick flashcard mode over the current selection ---
const drilling = ref(false)
const drillIdx = ref(0)
const drillShown = ref(false)
function startDrill() {
  drilling.value = true
  drillIdx.value = 0
  drillShown.value = false
}
function nextCard() {
  drillShown.value = false
  drillIdx.value++
  if (drillIdx.value >= rows.value.length) drilling.value = false
}
</script>

<template>
  <div class="mb-4 flex items-baseline justify-between">
    <h1 class="text-2xl font-bold">Словарь</h1>
    <button
      class="rounded border border-stone-300 px-2 py-1 text-sm dark:border-stone-600"
      @click="script = script === 'latin' ? 'cyrillic' : 'latin'"
    >
      {{ script === 'latin' ? 'латиница' : 'кириллица' }}
    </button>
  </div>

  <p v-if="error" class="mb-3 rounded bg-red-100 p-3 text-red-800 dark:bg-red-950 dark:text-red-200">{{ error }}</p>

  <div class="mb-3 flex flex-wrap gap-2">
    <input
      v-model="q"
      placeholder="поиск…"
      class="flex-1 rounded border border-stone-300 bg-transparent px-2 py-1 dark:border-stone-600"
    />
    <select v-model="lesson" class="rounded border border-stone-300 bg-transparent px-2 py-1 dark:border-stone-600">
      <option value="">урок: все</option>
      <option v-for="l in lessons" :key="l" :value="l">урок {{ l }}</option>
    </select>
    <select v-model="tag" class="rounded border border-stone-300 bg-transparent px-2 py-1 dark:border-stone-600">
      <option value="">тег: все</option>
      <option v-for="tg in tags" :key="tg" :value="tg">{{ tg }}</option>
    </select>
  </div>

  <div class="mb-3 flex items-center gap-3 text-sm text-stone-500">
    <span>{{ rows.length }} слов</span>
    <button v-if="rows.length" class="text-amber-600 underline" @click="startDrill">Учить выборку →</button>
  </div>

  <!-- flashcard drill -->
  <div
    v-if="drilling && rows[drillIdx]"
    class="mb-4 rounded-xl border border-stone-200 p-6 text-center dark:border-stone-700"
    @click="drillShown = true"
  >
    <p class="text-xs text-stone-400">{{ drillIdx + 1 }} / {{ rows.length }}</p>
    <p class="mt-2 text-2xl font-semibold">
      {{ script === 'latin' ? rows[drillIdx].latin : rows[drillIdx].cyrillic }}
    </p>
    <template v-if="drillShown">
      <p class="mt-2 text-lg">{{ rows[drillIdx].ru }}</p>
      <p v-if="rows[drillIdx].note" class="text-sm text-stone-500">{{ rows[drillIdx].note }}</p>
      <button class="mt-3 rounded bg-amber-600 px-3 py-1 text-white" @click.stop="nextCard">Дальше</button>
    </template>
    <p v-else class="mt-3 text-xs text-stone-400">нажми, чтобы показать перевод</p>
  </div>

  <div class="overflow-x-auto">
    <table class="w-full border-collapse text-sm">
      <thead>
        <tr class="border-b border-stone-300 text-left dark:border-stone-600">
          <th class="py-2 pr-3">слово</th>
          <th class="py-2 pr-3">перевод</th>
          <th class="py-2 pr-3">заметка</th>
          <th class="py-2">урок</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="v in rows" :key="v.id" class="border-b border-stone-100 dark:border-stone-800">
          <td class="py-2 pr-3 font-medium">{{ script === 'latin' ? v.latin : v.cyrillic }}</td>
          <td class="py-2 pr-3">{{ v.ru }}</td>
          <td class="py-2 pr-3 text-stone-500">{{ v.note }}</td>
          <td class="py-2 text-stone-400">{{ v.lesson }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
