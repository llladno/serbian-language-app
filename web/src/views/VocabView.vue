<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import type { Vocab } from '../types'

const all = ref<Vocab[]>([])
const q = ref('')
const lesson = ref('')
const tag = ref('')
const scriptMode = ref<'latin' | 'cyrillic'>('latin')
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
    (v) =>
      (!lesson.value || v.lesson === lesson.value) && (!tag.value || (v.tags ?? []).includes(tag.value)),
  ),
)

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
    <h1 class="text-2xl font-extrabold">Словарь</h1>
    <button
      class="btn btn-ghost px-2.5 py-1 text-sm"
      @click="scriptMode = scriptMode === 'latin' ? 'cyrillic' : 'latin'"
    >
      {{ scriptMode === 'latin' ? 'латиница' : 'кириллица' }}
    </button>
  </div>

  <p v-if="error" class="card mb-3 p-3 text-[var(--bad)]">{{ error }}</p>

  <div class="mb-3 flex flex-wrap gap-2">
    <input v-model="q" placeholder="поиск…" class="field flex-1" />
    <select v-model="lesson" class="field">
      <option value="">урок: все</option>
      <option v-for="l in lessons" :key="l" :value="l">урок {{ l }}</option>
    </select>
    <select v-model="tag" class="field">
      <option value="">тег: все</option>
      <option v-for="tg in tags" :key="tg" :value="tg">{{ tg }}</option>
    </select>
  </div>

  <div class="mb-3 flex items-center gap-3 text-sm text-[var(--muted)]">
    <span>{{ rows.length }} слов</span>
    <button v-if="rows.length" class="font-semibold text-[var(--accent)]" @click="startDrill">
      Учить выборку →
    </button>
  </div>

  <div
    v-if="drilling && rows[drillIdx]"
    class="card mb-4 p-6 text-center"
    @click="drillShown = true"
  >
    <p class="text-xs text-[var(--muted)]">{{ drillIdx + 1 }} / {{ rows.length }}</p>
    <p class="serbian mt-2 text-3xl font-semibold">
      {{ scriptMode === 'latin' ? rows[drillIdx].latin : rows[drillIdx].cyrillic }}
    </p>
    <template v-if="drillShown">
      <p class="mt-2 text-lg">{{ rows[drillIdx].ru }}</p>
      <p v-if="rows[drillIdx].note" class="text-sm text-[var(--muted)]">{{ rows[drillIdx].note }}</p>
      <button class="btn btn-primary mt-3" @click.stop="nextCard">Дальше</button>
    </template>
    <p v-else class="mt-3 text-xs text-[var(--muted)]">нажми, чтобы показать перевод</p>
  </div>

  <div class="card overflow-x-auto p-1">
    <table class="w-full border-collapse text-sm">
      <thead>
        <tr class="text-left text-[var(--muted)]">
          <th class="px-3 py-2 font-semibold">слово</th>
          <th class="px-3 py-2 font-semibold">перевод</th>
          <th class="px-3 py-2 font-semibold">заметка</th>
          <th class="px-3 py-2 font-semibold">урок</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="v in rows" :key="v.id" class="border-t border-[var(--border)]">
          <td class="serbian px-3 py-2 font-semibold">
            {{ scriptMode === 'latin' ? v.latin : v.cyrillic }}
          </td>
          <td class="px-3 py-2">{{ v.ru }}</td>
          <td class="px-3 py-2 text-[var(--muted)]">{{ v.note }}</td>
          <td class="px-3 py-2 text-[var(--muted)]">{{ v.lesson }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
