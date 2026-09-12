<script setup lang="ts">
import { onMounted, ref } from 'vue'

const props = defineProps<{ botId: string }>()
const emit = defineEmits<{ auth: [payload: Record<string, unknown>] }>()
const ready = ref(false)

declare global {
  interface Window {
    Telegram?: {
      Login?: {
        auth: (
          opts: { bot_id: string; request_access?: boolean },
          cb: (data: Record<string, unknown> | false) => void,
        ) => void
      }
    }
  }
}

function loadScript(): Promise<void> {
  if (window.Telegram?.Login) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const s = document.createElement('script')
    s.src = 'https://telegram.org/js/telegram-widget.js?22'
    s.async = true
    s.onload = () => resolve()
    s.onerror = () => reject(new Error('telegram widget script failed to load'))
    document.head.appendChild(s)
  })
}

onMounted(async () => {
  try {
    await loadScript()
    ready.value = !!window.Telegram?.Login
  } catch {
    ready.value = false
  }
})

function click() {
  if (!window.Telegram?.Login) return
  window.Telegram.Login.auth({ bot_id: props.botId, request_access: true }, (data) => {
    if (data) emit('auth', data)
  })
}
</script>

<template>
  <button v-if="ready" type="button" class="btn btn-ghost w-full" @click="click">
    Войти через Telegram
  </button>
</template>
