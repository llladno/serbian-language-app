<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Bell } from 'lucide-vue-next'
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

    <div v-if="open" class="fixed inset-0 z-10" @click="close" />

    <div v-if="open" class="card absolute right-0 top-full z-20 mt-2 max-h-96 w-80 max-w-[90vw] overflow-y-auto p-2">
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
</template>
