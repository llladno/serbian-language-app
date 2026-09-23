<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AuthShell from './AuthShell.vue'
import { useSessionStore } from '../stores/session'
import { api } from '../api'
import { authErrorMessage } from '../lib/authErrors'
import mascotOk from '../assets/mascot-verify-ok.webp'
import mascotErr from '../assets/mascot-verify-err.webp'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()

const mode = computed<'ok' | 'err' | 'gate'>(() => {
  if (route.query.ok === '1') return 'ok'
  if (route.query.err === '1') return 'err'
  return 'gate'
})

const mascot = computed(() => (mode.value === 'err' ? mascotErr : mascotOk))

const targetEmail = computed(
  () => session.user?.email || (typeof route.query.email === 'string' ? route.query.email : ''),
)

const emailInput = ref(targetEmail.value)
const busy = ref(false)
const sent = ref(false)
const error = ref<string | null>(null)
const cooldown = ref(0)
let timer: ReturnType<typeof setInterval> | undefined

onMounted(() => {
  if (mode.value === 'gate' && session.user?.email_verified) {
    router.push('/profile')
  }
})

function startCooldown() {
  cooldown.value = 30
  timer = setInterval(() => {
    cooldown.value -= 1
    if (cooldown.value <= 0) clearInterval(timer)
  }, 1000)
}

async function resend() {
  const email = (targetEmail.value || emailInput.value).trim()
  if (!email || busy.value || cooldown.value > 0) return
  busy.value = true
  error.value = null
  try {
    await api.resendVerification(email)
    sent.value = true
    startCooldown()
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthShell title="Почта" :mascot="mascot">
    <div v-if="mode === 'ok'" class="text-center">
      <p class="mb-4">Почта подтверждена!</p>
      <RouterLink to="/login" class="btn btn-primary">Войти</RouterLink>
    </div>

    <div v-else-if="mode === 'err'" class="space-y-3">
      <p>Ссылка устарела или уже использована.</p>
      <input v-model="emailInput" type="email" class="field w-full" placeholder="email" />
      <button class="btn btn-primary w-full" :disabled="busy || cooldown > 0" @click="resend">
        {{ cooldown > 0 ? `отправить снова (${cooldown})` : 'отправить снова' }}
      </button>
      <p v-if="sent" class="text-sm text-[var(--good)]">Письмо отправлено.</p>
      <p v-if="error" class="text-sm text-[var(--bad)]">{{ error }}</p>
      <p class="text-center text-sm">
        <RouterLink to="/login" class="text-[var(--accent)]">вернуться к другим способам входа</RouterLink>
      </p>
    </div>

    <div v-else class="space-y-3">
      <template v-if="targetEmail">
        <p>Письмо для подтверждения отправлено на <b>{{ targetEmail }}</b>.</p>
        <button class="btn btn-ghost w-full" :disabled="busy || cooldown > 0" @click="resend">
          {{ cooldown > 0 ? `отправить снова (${cooldown})` : 'отправить снова' }}
        </button>
      </template>
      <template v-else>
        <p>Введите email, чтобы отправить письмо для подтверждения.</p>
        <input v-model="emailInput" type="email" class="field w-full" placeholder="email" />
        <button class="btn btn-primary w-full" :disabled="busy || cooldown > 0" @click="resend">
          {{ cooldown > 0 ? `отправить снова (${cooldown})` : 'отправить письмо' }}
        </button>
      </template>
      <p v-if="sent" class="text-sm text-[var(--good)]">Письмо отправлено.</p>
      <p v-if="error" class="text-sm text-[var(--bad)]">{{ error }}</p>
      <p class="text-center text-sm">
        <RouterLink to="/login" class="text-[var(--accent)]">вернуться к другим способам входа</RouterLink>
      </p>
    </div>
  </AuthShell>
</template>
