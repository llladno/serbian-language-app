<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AuthShell from './AuthShell.vue'
import { useSessionStore } from '../stores/session'
import { authErrorMessage } from '../lib/authErrors'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()

const token = typeof route.query.token === 'string' ? route.query.token : ''
const password = ref('')
const confirm = ref('')
const busy = ref(false)
const error = ref<string | null>(null)

async function submit() {
  if (busy.value) return
  if (password.value !== confirm.value) {
    error.value = 'Пароли не совпадают'
    return
  }
  busy.value = true
  error.value = null
  try {
    await session.reset(token, password.value)
    router.push('/profile')
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="Новый пароль">
    <p v-if="!token" class="text-[var(--bad)]">Ссылка недействительна.</p>
    <form v-else class="space-y-3" @submit.prevent="submit">
      <input
        v-model="password"
        type="password"
        class="field w-full"
        placeholder="новый пароль"
        required
        minlength="8"
        maxlength="128"
        autofocus
      />
      <input
        v-model="confirm"
        type="password"
        class="field w-full"
        placeholder="повторите пароль"
        required
        minlength="8"
        maxlength="128"
      />
      <button class="btn btn-primary w-full" :disabled="busy">Сохранить</button>
    </form>
    <p v-if="error" class="mt-2 text-sm text-[var(--bad)]">{{ error }}</p>
    <p class="mt-4 text-center text-sm">
      <RouterLink to="/login" class="text-[var(--accent)]">вернуться ко входу</RouterLink>
    </p>
  </AuthShell>
</template>
