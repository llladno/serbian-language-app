<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { useSessionStore } from '../stores/session'
import { authErrorMessage } from '../lib/authErrors'
import { useTelegramStart } from '../lib/telegramStart'
import ProgressDashboard from '../components/ProgressDashboard.vue'
import type { Me } from '../types'

const router = useRouter()
const session = useSessionStore()

const me = ref<Me | null>(null)
const loadError = ref<string | null>(null)

async function loadMe() {
  try {
    me.value = await api.getMe()
  } catch (e) {
    loadError.value = authErrorMessage(e)
  }
}
onMounted(loadMe)

const hasPassword = computed(() => !!me.value?.email)

// -- rename --
const editingName = ref(false)
const nameDraft = ref('')
const nameBusy = ref(false)
const nameError = ref<string | null>(null)
function startEditName() {
  nameDraft.value = me.value?.name ?? ''
  nameError.value = null
  editingName.value = true
}
async function saveName() {
  const trimmed = nameDraft.value.trim()
  if (!trimmed) return
  nameBusy.value = true
  nameError.value = null
  try {
    const updated = await api.patchMe(trimmed)
    if (me.value) me.value.name = updated.name
    if (session.user) session.user.name = updated.name
    editingName.value = false
  } catch (e) {
    nameError.value = authErrorMessage(e)
  } finally {
    nameBusy.value = false
  }
}

// -- password --
const showPasswordForm = ref(false)
const currentPassword = ref('')
const newPassword = ref('')
const pwEmail = ref('')
const pwBusy = ref(false)
const pwError = ref<string | null>(null)
const pwStatus = ref<string | null>(null)

async function submitPassword() {
  if (newPassword.value.length < 8) {
    pwError.value = 'Пароль должен быть от 8 до 128 символов'
    return
  }
  pwBusy.value = true
  pwError.value = null
  try {
    const res = await api.setPassword({
      current: hasPassword.value ? currentPassword.value : undefined,
      new: newPassword.value,
      email: hasPassword.value ? undefined : pwEmail.value,
    })
    pwStatus.value =
      res.status === 'verify_sent' ? 'Проверьте почту — письмо для подтверждения отправлено' : 'Пароль изменён'
    showPasswordForm.value = false
    currentPassword.value = ''
    newPassword.value = ''
  } catch (e) {
    pwError.value = authErrorMessage(e)
  } finally {
    pwBusy.value = false
  }
}

// -- telegram --
const tg = useTelegramStart('link')
function linkTelegram() {
  tg.start(() => {
    loadMe()
  })
}

// -- devices --
const devicesError = ref<string | null>(null)
async function revokeSession(id: string) {
  if (!confirm('Выйти на этом устройстве?')) return
  devicesError.value = null
  try {
    await api.deleteSession(id)
    await loadMe()
  } catch (e) {
    devicesError.value = authErrorMessage(e)
  }
}

const sessionActionError = ref<string | null>(null)
async function logout() {
  sessionActionError.value = null
  try {
    await session.logout()
    router.push('/login')
  } catch (e) {
    sessionActionError.value = authErrorMessage(e)
  }
}
async function logoutAll() {
  sessionActionError.value = null
  try {
    await session.logoutAll()
    router.push('/login')
  } catch (e) {
    sessionActionError.value = authErrorMessage(e)
  }
}

// -- danger zone --
const resetError = ref<string | null>(null)
async function resetExercises() {
  if (!confirm('Сбросить весь прогресс по заданиям (ответы и отметки уроков)? Карточки слов останутся.')) return
  resetError.value = null
  try {
    await api.resetExercises()
    location.reload()
  } catch (e) {
    resetError.value = authErrorMessage(e)
  }
}

const deletePassword = ref('')
const deleteBusy = ref(false)
const deleteError = ref<string | null>(null)
async function deleteAccount() {
  if (!confirm('Удалить аккаунт безвозвратно? Это нельзя отменить.')) return
  deleteBusy.value = true
  deleteError.value = null
  try {
    await api.deleteMe(deletePassword.value || undefined)
    session.user = null
    router.push('/login')
  } catch (e) {
    deleteError.value = authErrorMessage(e)
  } finally {
    deleteBusy.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <p v-if="loadError" class="card p-4 text-[var(--bad)]">{{ loadError }}</p>

    <div v-else-if="me" class="card space-y-4 p-5">
      <div class="flex items-center justify-between gap-2">
        <template v-if="!editingName">
          <p class="text-xl font-extrabold">{{ me.name }}</p>
          <button class="text-sm text-[var(--accent)]" @click="startEditName">изменить</button>
        </template>
        <form v-else class="flex flex-1 gap-2" @submit.prevent="saveName">
          <input v-model="nameDraft" class="field flex-1" maxlength="40" autofocus />
          <button class="btn btn-primary" :disabled="nameBusy">Сохранить</button>
          <button type="button" class="btn btn-ghost" @click="editingName = false">Отмена</button>
        </form>
      </div>
      <p v-if="nameError" class="text-sm text-[var(--bad)]">{{ nameError }}</p>

      <div class="flex items-center gap-2 text-sm">
        <span class="text-[var(--muted)]">Почта:</span>
        <span v-if="me.email">{{ me.email }}</span>
        <span v-else class="text-[var(--muted)]">не указана</span>
        <span
          v-if="me.email"
          class="rounded-full px-2 py-0.5 text-xs font-semibold"
          :style="
            me.email_verified
              ? { background: 'color-mix(in srgb, var(--good) 16%, transparent)', color: 'var(--good)' }
              : { background: 'var(--bg-soft)', color: 'var(--muted)' }
          "
        >
          {{ me.email_verified ? 'подтверждена' : 'не подтверждена' }}
        </span>
      </div>

      <div>
        <div class="flex items-center gap-2 text-sm">
          <span class="text-[var(--muted)]">Telegram:</span>
          <span v-if="me.telegram.linked">@{{ me.telegram.username || '—' }}</span>
          <button v-else class="text-[var(--accent)]" :disabled="tg.busy.value" @click="linkTelegram">
            {{ tg.busy.value ? 'ждём подтверждения в Telegram…' : 'привязать' }}
          </button>
        </div>
        <p v-if="tg.error.value" class="mt-1 text-sm text-[var(--bad)]">{{ tg.error.value }}</p>
      </div>

      <div>
        <button v-if="!showPasswordForm" class="btn btn-ghost" @click="showPasswordForm = true">
          {{ hasPassword ? 'Сменить пароль' : 'Задать пароль' }}
        </button>
        <form v-else class="mt-2 space-y-2" @submit.prevent="submitPassword">
          <input
            v-if="hasPassword"
            v-model="currentPassword"
            type="password"
            class="field w-full"
            placeholder="текущий пароль"
          />
          <input
            v-if="!hasPassword"
            v-model="pwEmail"
            type="email"
            class="field w-full"
            placeholder="email для входа по паролю"
          />
          <input
            v-model="newPassword"
            type="password"
            class="field w-full"
            placeholder="новый пароль"
            minlength="8"
            maxlength="128"
          />
          <div class="flex gap-2">
            <button class="btn btn-primary" :disabled="pwBusy">Сохранить</button>
            <button type="button" class="btn btn-ghost" @click="showPasswordForm = false">Отмена</button>
          </div>
        </form>
        <p v-if="pwError" class="mt-1 text-sm text-[var(--bad)]">{{ pwError }}</p>
        <p v-if="pwStatus" class="mt-1 text-sm text-[var(--good)]">{{ pwStatus }}</p>
      </div>

      <div>
        <p class="mb-2 text-sm font-bold">Устройства</p>
        <ul class="space-y-1.5">
          <li v-for="d in me.sessions" :key="d.id" class="flex items-center justify-between gap-2 text-sm">
            <span class="min-w-0 truncate text-[var(--muted)]">
              {{ d.user_agent || 'неизвестное устройство' }}
              <span v-if="d.current" class="text-[var(--accent)]">— это устройство</span>
            </span>
            <button class="shrink-0 text-xs text-[var(--bad)]" @click="revokeSession(d.id)">выйти</button>
          </li>
        </ul>
        <p v-if="devicesError" class="mt-1 text-sm text-[var(--bad)]">{{ devicesError }}</p>
      </div>

      <div class="flex flex-wrap gap-2 border-t border-[var(--border)] pt-3">
        <button class="btn btn-ghost" @click="logout">Выйти</button>
        <button class="btn btn-ghost" @click="logoutAll">Выйти везде</button>
      </div>
      <p v-if="sessionActionError" class="text-sm text-[var(--bad)]">{{ sessionActionError }}</p>
    </div>

    <ProgressDashboard v-if="me" :name="me.name" />

    <div v-if="me" class="card space-y-3 p-5">
      <p class="font-bold text-[var(--bad)]">Опасная зона</p>
      <button class="text-sm text-[var(--muted)] hover:text-[var(--bad)]" @click="resetExercises">
        сбросить прогресс по заданиям
      </button>
      <p v-if="resetError" class="text-sm text-[var(--bad)]">{{ resetError }}</p>
      <div class="space-y-2 border-t border-[var(--border)] pt-3">
        <input
          v-if="hasPassword"
          v-model="deletePassword"
          type="password"
          class="field w-full"
          placeholder="пароль для подтверждения"
        />
        <button
          class="btn"
          style="background: var(--bad); color: #fff"
          :disabled="deleteBusy"
          @click="deleteAccount"
        >
          Удалить аккаунт
        </button>
        <p v-if="deleteError" class="text-sm text-[var(--bad)]">{{ deleteError }}</p>
      </div>
    </div>
  </div>
</template>
