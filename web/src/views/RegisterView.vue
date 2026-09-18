<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AuthShell from './AuthShell.vue'
import { useSessionStore } from '../stores/session'
import { authErrorMessage } from '../lib/authErrors'

const router = useRouter()
const session = useSessionStore()

const name = ref('')
const email = ref('')
const password = ref('')
const consentPrivacy = ref(false)
const consentTerms = ref(false)
const busy = ref(false)
const error = ref<string | null>(null)

const strength = computed(() => {
  const n = password.value.length
  if (n === 0) return null
  if (n < 8) return { label: 'слишком короткий', color: 'var(--bad)' }
  if (n < 12) return { label: 'нормальный', color: 'var(--muted)' }
  return { label: 'надёжный', color: 'var(--good)' }
})

async function submit() {
  if (busy.value) return
  busy.value = true
  error.value = null
  try {
    await session.register(email.value.trim(), password.value, name.value.trim())
    router.push({ path: '/verify', query: { email: email.value.trim() } })
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="Регистрация">
    <form class="space-y-3" @submit.prevent="submit">
      <input v-model="name" class="field w-full" placeholder="имя" required maxlength="40" autofocus />
      <input v-model="email" type="email" class="field w-full" placeholder="email" required />
      <div>
        <input
          v-model="password"
          type="password"
          class="field w-full"
          placeholder="пароль"
          required
          minlength="8"
          maxlength="128"
        />
        <p v-if="strength" class="mt-1 text-xs" :style="{ color: strength.color }">{{ strength.label }}</p>
      </div>
      <label class="flex items-start gap-2 text-xs text-[var(--muted)]">
        <input v-model="consentPrivacy" type="checkbox" required class="mt-0.5" />
        <span>
          Я даю согласие на обработку персональных данных в соответствии с
          <a href="/privacy" target="_blank" rel="noopener" class="text-[var(--accent)]"
            >Политикой конфиденциальности</a
          >
        </span>
      </label>
      <label class="flex items-start gap-2 text-xs text-[var(--muted)]">
        <input v-model="consentTerms" type="checkbox" required class="mt-0.5" />
        <span>
          Я принимаю
          <a href="/terms" target="_blank" rel="noopener" class="text-[var(--accent)]"
            >Условия использования</a
          >
        </span>
      </label>
      <button class="btn btn-primary w-full" :disabled="busy">Зарегистрироваться</button>
    </form>
    <p v-if="error" class="mt-2 text-sm text-[var(--bad)]">{{ error }}</p>

    <p class="mt-4 text-center text-sm">
      Уже есть аккаунт? <RouterLink to="/login" class="text-[var(--accent)]">войти</RouterLink>
    </p>
  </AuthShell>
</template>
