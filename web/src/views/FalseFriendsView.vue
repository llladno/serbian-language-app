<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import type { FalseFriend } from '../types'

const all = ref<FalseFriend[]>([])
const q = ref('')
const group = ref('all')
const error = ref<string | null>(null)

const GROUPS = [
  { v: 'all', label: 'все' },
  { v: 'top', label: 'бытовые' },
  { v: 'shop', label: 'магазин / еда' },
  { v: 'small', label: 'мелкие частые' },
]

let t: ReturnType<typeof setTimeout>
watch(q, () => {
  clearTimeout(t)
  t = setTimeout(load, 200)
})

async function load() {
  try {
    all.value = await api.falseFriends({ q: q.value || undefined })
    error.value = null
  } catch (e) {
    error.value = (e as Error).message
  }
}
onMounted(load)

const rows = computed(() => all.value.filter((f) => group.value === 'all' || f.group === group.value))
</script>

<template>
  <h1 class="mb-4 text-2xl font-bold">Ложные друзья</h1>
  <p class="mb-3 text-sm text-stone-500">Слова, которые звучат знакомо, но значат другое.</p>

  <p v-if="error" class="mb-3 rounded bg-red-100 p-3 text-red-800 dark:bg-red-950 dark:text-red-200">{{ error }}</p>

  <div class="mb-3 flex flex-wrap gap-2">
    <input
      v-model="q"
      placeholder="поиск…"
      class="flex-1 rounded border border-stone-300 bg-transparent px-2 py-1 dark:border-stone-600"
    />
    <select v-model="group" class="rounded border border-stone-300 bg-transparent px-2 py-1 dark:border-stone-600">
      <option v-for="g in GROUPS" :key="g.v" :value="g.v">{{ g.label }}</option>
    </select>
  </div>

  <p class="mb-2 text-sm text-stone-500">{{ rows.length }} записей</p>

  <div class="overflow-x-auto">
    <table class="w-full border-collapse text-sm">
      <thead>
        <tr class="border-b border-stone-300 text-left dark:border-stone-600">
          <th class="py-2 pr-3">сербское</th>
          <th class="py-2 pr-3">значит</th>
          <th class="py-2 pr-3">а НЕ</th>
          <th class="py-2">«то самое»</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="f in rows" :key="f.id" class="border-b border-stone-100 dark:border-stone-800">
          <td class="py-2 pr-3 font-medium">{{ f.sr }}</td>
          <td class="py-2 pr-3">{{ f.means }}</td>
          <td class="py-2 pr-3 text-red-600 dark:text-red-400">{{ f.not }}</td>
          <td class="py-2 text-stone-500">{{ f.correct }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
