<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import AuthShell from './AuthShell.vue'
import { useSessionStore } from '../stores/session'
import { authErrorMessage } from '../lib/authErrors'
import mascotForgot from '../assets/mascot-forgot.webp'

const session = useSessionStore()
const email = ref('')
const busy = ref(false)
const sent = ref(false)
const error = ref<string | null>(null)

async function submit() {
  if (busy.value) return
  busy.value = true
  error.value = null
  try {
    await session.forgot(email.value.trim())
    sent.value = true
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="Восстановление пароля" :mascot="mascotForgot">
    <div v-if="sent" class="text-center">
      <p>Если такой аккаунт существует, письмо со ссылкой для сброса пароля отправлено.</p>
    </div>
    <form v-else class="space-y-3" @submit.prevent="submit">
      <input v-model="email" type="email" class="field w-full" placeholder="email" required autofocus />
      <button class="btn btn-primary w-full" :disabled="busy">Отправить ссылку</button>
    </form>
    <p v-if="error" class="mt-2 text-sm text-[var(--bad)]">{{ error }}</p>
    <p class="mt-4 text-center text-sm">
      <RouterLink to="/login" class="text-[var(--accent)]">вернуться ко входу</RouterLink>
    </p>
  </AuthShell>
</template>
