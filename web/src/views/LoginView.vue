<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AuthShell from './AuthShell.vue'
import { useSessionStore } from '../stores/session'
import { authErrorMessage, isEmailUnverified } from '../lib/authErrors'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()

const email = ref('')
const password = ref('')
const busy = ref(false)
const error = ref<string | null>(null)

function goNext() {
  const next = typeof route.query.next === 'string' ? route.query.next : '/profile'
  router.push(next)
}

async function submit() {
  if (busy.value) return
  busy.value = true
  error.value = null
  try {
    await session.login(email.value.trim(), password.value)
    goNext()
  } catch (e) {
    if (isEmailUnverified(e)) {
      router.push({ path: '/verify', query: { email: email.value.trim() } })
      return
    }
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="Вход">
    <form class="space-y-3" @submit.prevent="submit">
      <input v-model="email" type="email" class="field w-full" placeholder="email" required autofocus />
      <input
        v-model="password"
        type="password"
        class="field w-full"
        placeholder="пароль"
        required
        minlength="8"
        maxlength="128"
      />
      <button class="btn btn-primary w-full" :disabled="busy">Войти</button>
    </form>
    <p v-if="error" class="mt-2 text-sm text-[var(--bad)]">{{ error }}</p>

    <div class="mt-4 flex justify-between text-sm">
      <RouterLink to="/forgot" class="text-[var(--accent)]">забыли пароль?</RouterLink>
      <RouterLink to="/register" class="text-[var(--accent)]">регистрация</RouterLink>
    </div>
  </AuthShell>
</template>
