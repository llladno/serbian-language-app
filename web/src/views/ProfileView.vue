<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Check, Settings, X } from 'lucide-vue-next'
import { api } from '../api'
import { palette, setPalette, PALETTE_META, type Palette } from '../palette'
import { useSessionStore } from '../stores/session'
import { authErrorMessage } from '../lib/authErrors'
import { useTelegramStart } from '../lib/telegramStart'
import ProgressDashboard from '../components/ProgressDashboard.vue'
import PhaseProgressCard from '../components/PhaseProgressCard.vue'
import LeaderboardCard from '../components/LeaderboardCard.vue'
import SupportCard from '../components/SupportCard.vue'
import DonateCard from '../components/DonateCard.vue'
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

const showSettings = ref(false)

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
</script>

<template>
  <div class="space-y-4">
    <p v-if="loadError" class="card p-4 text-[var(--bad)]">{{ loadError }}</p>

    <div v-else-if="!me" class="grid grid-cols-1 items-start gap-4 lg:grid-cols-3">
      <div class="space-y-4 lg:col-span-2 lg:order-1">
        <div class="card space-y-3 p-5">
          <div class="skel h-20 w-full rounded-2xl"></div>
          <div class="skel h-12 w-full rounded-2xl"></div>
        </div>
        <div class="card p-5">
          <div class="skel mb-3 h-4 w-40"></div>
          <div class="flex flex-wrap justify-around gap-4">
            <div v-for="i in 3" :key="i" class="skel h-[72px] w-[72px] rounded-full"></div>
          </div>
        </div>
      </div>
      <div class="space-y-4 lg:sticky lg:top-20 lg:order-2 lg:col-span-1">
        <div class="card space-y-3 p-5">
          <div class="skel h-6 w-32"></div>
          <div class="skel h-3.5 w-40"></div>
          <div class="skel h-3.5 w-28"></div>
        </div>
        <div class="card space-y-2 p-5">
          <div class="skel h-4 w-20"></div>
          <div class="skel h-3.5 w-full"></div>
          <div class="skel h-3.5 w-full"></div>
        </div>
      </div>
    </div>

    <div v-else-if="me" class="grid grid-cols-1 items-start gap-4 lg:grid-cols-3">
    <div class="space-y-4 lg:col-span-2 lg:order-1">
      <ProgressDashboard v-if="me" />
      <PhaseProgressCard />
    </div>

    <div class="space-y-4 lg:sticky lg:top-20 lg:order-2 lg:col-span-1">
      <div class="card space-y-3 p-5">
        <div class="flex items-center justify-between gap-2">
          <p class="text-xl font-extrabold">{{ me.name }}</p>
          <button class="icon-btn shrink-0" title="Настройки" aria-label="Настройки" @click="showSettings = true">
            <Settings :size="19" :stroke-width="2.25" />
          </button>
        </div>

        <div class="space-y-1.5 text-sm">
          <div class="flex items-center gap-2">
            <span class="text-[var(--muted)]">Telegram:</span>
            <span v-if="me.telegram.linked">@{{ me.telegram.username || '—' }}</span>
            <span v-else class="text-[var(--muted)]">не привязан</span>
          </div>
          <div class="flex items-center gap-2">
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
        </div>
      </div>

      <LeaderboardCard :name="me.name" />
    </div>
    </div>

    <SupportCard v-if="me" />
    <DonateCard v-if="me" />

    <Teleport to="body">
      <Transition name="modal-overlay">
        <div
          v-if="me && showSettings"
          class="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto p-4 sm:items-center"
        >
        <div class="fixed inset-0 bg-black/40" @click="showSettings = false" />
        <div class="card modal-panel relative z-10 w-full max-w-md space-y-4 p-5">
          <div class="flex items-center justify-between">
            <p class="text-lg font-extrabold">Настройки</p>
            <button class="icon-btn" title="Закрыть" aria-label="Закрыть" @click="showSettings = false">
              <X :size="19" :stroke-width="2.25" />
            </button>
          </div>

          <div>
            <div class="flex items-center justify-between gap-2">
              <template v-if="!editingName">
                <p class="font-semibold">{{ me.name }}</p>
                <button class="text-sm text-[var(--accent)]" @click="startEditName">изменить имя</button>
              </template>
              <form v-else class="flex flex-1 gap-2" @submit.prevent="saveName">
                <input v-model="nameDraft" class="field flex-1" maxlength="40" autofocus />
                <button class="btn btn-primary" :disabled="nameBusy">Сохранить</button>
                <button type="button" class="btn btn-ghost" @click="editingName = false">Отмена</button>
              </form>
            </div>
            <p v-if="nameError" class="mt-1 text-sm text-[var(--bad)]">{{ nameError }}</p>
          </div>

          <div class="border-t border-[var(--border)] pt-3">
            <p class="mb-2 text-sm text-[var(--muted)]">Цвет темы</p>
            <div class="flex gap-3">
              <button
                v-for="(meta, key) in PALETTE_META"
                :key="key"
                type="button"
                class="flex h-9 w-9 items-center justify-center rounded-full border-2 transition"
                :style="{
                  background: meta.swatch,
                  borderColor: palette === key ? meta.swatch : 'transparent',
                  boxShadow: palette === key ? `0 0 0 2px var(--card), 0 0 0 4px ${meta.swatch}` : 'none',
                }"
                :title="meta.label"
                :aria-label="meta.label"
                @click="setPalette(key as Palette)"
              >
                <Check v-if="palette === key" :size="16" :stroke-width="3" color="#fff" />
              </button>
            </div>
          </div>

          <div class="border-t border-[var(--border)] pt-3">
            <div class="flex items-center gap-2 text-sm">
              <span class="text-[var(--muted)]">Telegram:</span>
              <span v-if="me.telegram.linked">@{{ me.telegram.username || '—' }}</span>
              <button v-else class="text-[var(--accent)]" :disabled="tg.busy.value" @click="linkTelegram">
                {{ tg.busy.value ? 'ждём подтверждения в Telegram…' : 'привязать' }}
              </button>
            </div>
            <p v-if="tg.error.value" class="mt-1 text-sm text-[var(--bad)]">{{ tg.error.value }}</p>
          </div>

          <div class="border-t border-[var(--border)] pt-3">
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

          <div class="flex flex-wrap gap-2 border-t border-[var(--border)] pt-3">
            <button class="btn btn-ghost" @click="logout">Выйти</button>
          </div>
          <p v-if="sessionActionError" class="text-sm text-[var(--bad)]">{{ sessionActionError }}</p>
        </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.modal-overlay-enter-active,
.modal-overlay-leave-active {
  transition: opacity 0.18s ease;
}
.modal-overlay-enter-from,
.modal-overlay-leave-to {
  opacity: 0;
}
.modal-overlay-enter-active .modal-panel,
.modal-overlay-leave-active .modal-panel {
  transition: transform 0.18s ease, opacity 0.18s ease;
}
.modal-overlay-enter-from .modal-panel,
.modal-overlay-leave-to .modal-panel {
  opacity: 0;
  transform: scale(0.95) translateY(6px);
}
</style>
