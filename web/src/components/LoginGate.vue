<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { setAccount } from '../account'

const name = ref('')
const existing = ref<string[]>([])
const busy = ref(false)
const error = ref<string | null>(null)

onMounted(async () => {
  try {
    existing.value = await api.listAccounts()
  } catch {
    /* first run, no accounts yet */
  }
})

async function enter(n: string) {
  const trimmed = n.trim()
  if (!trimmed || busy.value) return
  busy.value = true
  error.value = null
  try {
    const res = await api.createAccount(trimmed)
    setAccount(res.name) // reloads the app
  } catch (e) {
    error.value = (e as Error).message
    busy.value = false
  }
}
</script>

<template>
  <div class="mx-auto mt-16 max-w-sm px-4 text-center">
    <p class="serbian text-4xl font-semibold">Српски</p>
    <p class="mt-1 text-[var(--muted)]">Кто занимается?</p>

    <form class="mt-6 flex gap-2" @submit.prevent="enter(name)">
      <input
        v-model="name"
        class="field flex-1 text-center"
        placeholder="имя, например Гриша"
        autofocus
        maxlength="40"
      />
      <button class="btn btn-primary" :disabled="busy || !name.trim()">Войти</button>
    </form>
    <p v-if="error" class="mt-2 text-sm text-[var(--bad)]">{{ error }}</p>

    <div v-if="existing.length" class="mt-6">
      <p class="mb-2 text-xs uppercase tracking-wide text-[var(--muted)]">продолжить как</p>
      <div class="flex flex-wrap justify-center gap-2">
        <button
          v-for="u in existing"
          :key="u"
          class="btn btn-ghost"
          :disabled="busy"
          @click="enter(u)"
        >
          {{ u }}
        </button>
      </div>
    </div>

    <p class="mt-8 text-xs text-[var(--muted)]">
      Без пароля. Прогресс хранится по имени — на этом устройстве и в базе.
    </p>
  </div>
</template>
