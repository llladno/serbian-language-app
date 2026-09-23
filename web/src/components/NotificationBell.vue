<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Bell, X } from 'lucide-vue-next'
import { useNotifications, renderNotificationText } from '../lib/notifications'

const { items, unreadCount, markRead, start } = useNotifications()
const open = ref(false)

function toggle() {
  open.value = !open.value
  if (open.value) markRead()
}
function close() {
  open.value = false
}

onMounted(start)
</script>

<template>
  <div class="relative">
    <button
      class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-[var(--muted)] transition hover:bg-[var(--bg-soft)] hover:text-[var(--fg)]"
      title="Уведомления"
      aria-label="Уведомления"
      data-test="open-notifications"
      @click="toggle"
    >
      <Bell :size="17" :stroke-width="2.25" />
      <span
        v-if="unreadCount > 0"
        class="absolute right-1.5 top-1.5 h-2 w-2 rounded-full"
        style="background: var(--accent)"
        data-test="notification-dot"
      />
    </button>

    <Transition name="notification-dropdown">
      <div v-if="open">
        <div class="fixed inset-0 z-10" @click="close" />

        <div
          class="notification-panel card fixed inset-x-3 top-[calc(env(safe-area-inset-top)+4.25rem)] z-20 max-h-[70vh] overflow-y-auto p-2 sm:absolute sm:inset-x-auto sm:right-0 sm:top-full sm:mt-2 sm:max-h-96 sm:w-80 sm:max-w-[90vw]"
        >
          <div class="mb-1 flex items-center justify-between px-1 pt-1 sm:hidden">
            <p class="text-sm font-semibold">Уведомления</p>
            <button
              class="icon-btn h-9 w-9"
              title="Закрыть"
              aria-label="Закрыть"
              data-test="close-notifications"
              @click="close"
            >
              <X :size="18" :stroke-width="2.25" />
            </button>
          </div>

          <p v-if="items.length === 0" class="p-3 text-center text-sm text-[var(--muted)]">Пока пусто</p>
          <div
            v-for="n in items"
            :key="n.id"
            class="rounded-2xl p-3 text-sm"
            :class="n.read ? '' : 'bg-[var(--accent-soft)]'"
            v-html="renderNotificationText(n.text)"
          />
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.notification-dropdown-enter-active,
.notification-dropdown-leave-active {
  transition: opacity 0.15s ease;
}
.notification-dropdown-enter-from,
.notification-dropdown-leave-to {
  opacity: 0;
}
.notification-dropdown-enter-active .notification-panel,
.notification-dropdown-leave-active .notification-panel {
  transition: transform 0.15s ease, opacity 0.15s ease;
}
.notification-dropdown-enter-from .notification-panel,
.notification-dropdown-leave-to .notification-panel {
  opacity: 0;
  transform: scale(0.97) translateY(-6px);
}
</style>
