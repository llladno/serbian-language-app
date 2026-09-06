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
  <h1 class="mb-1 text-2xl font-extrabold">Ловушки</h1>
  <p class="mb-4 text-sm text-[var(--muted)]">Слова, которые звучат знакомо, но значат другое.</p>

  <p v-if="error" class="card mb-3 p-3 text-[var(--bad)]">{{ error }}</p>

  <div class="mb-3 flex flex-wrap gap-2">
    <input v-model="q" placeholder="поиск…" class="field flex-1" />
    <select v-model="group" class="field">
      <option v-for="g in GROUPS" :key="g.v" :value="g.v">{{ g.label }}</option>
    </select>
  </div>

  <p class="mb-2 text-sm text-[var(--muted)]">{{ rows.length }} записей</p>

  <div class="space-y-2">
    <div v-for="f in rows" :key="f.id" class="card p-3">
      <div class="flex items-baseline justify-between gap-2">
        <span class="serbian text-lg font-semibold">{{ f.sr }}</span>
        <span v-if="f.not" class="shrink-0 text-sm text-[var(--bad)]">≠ {{ f.not }}</span>
      </div>
      <p class="text-sm">{{ f.means }}</p>
      <p v-if="f.correct" class="mt-0.5 text-sm text-[var(--muted)]">
        «то самое» → <span class="serbian">{{ f.correct }}</span>
      </p>
    </div>
  </div>
</template>
