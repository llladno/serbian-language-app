<script setup lang="ts">
// Which Telegram messages the bot may send. This is the "unsubscribe": every
// optional kind has its own switch, and the same choices can be made from the
// bot itself (a button under each reminder, /stop, /settings). Sign-in and
// payment messages answer something the person just did and are not here.
import { computed, onMounted } from 'vue'
import { useSessionStore } from '../stores/session'
import { useBotPrefsStore } from '../stores/botPrefs'
import type { BotPrefs } from '../types'

const props = withDefaults(defineProps<{ only?: (keyof BotPrefs)[] }>(), { only: undefined })

const session = useSessionStore()
const store = useBotPrefsStore()

const linked = computed(() => !!session.user?.telegram.linked)

const ROWS: { key: keyof BotPrefs; title: string; hint: string }[] = [
  { key: 'learning', title: 'Напоминания об учёбе', hint: 'Когда слова на сегодня сделаны и когда давно не заглядывал' },
  { key: 'billing', title: 'Про тариф и оплату', hint: 'Например, что год подходит к концу' },
  { key: 'news', title: 'Новые уровни и предложения', hint: 'Редко и только по делу' },
]
const rows = computed(() => ROWS.filter((r) => !props.only || props.only.includes(r.key)))

onMounted(() => {
  if (linked.value) store.load()
})
</script>

<template>
  <div data-test="bot-prefs">
    <p class="mb-2 text-sm text-[var(--muted)]">Сообщения в Telegram</p>

    <p v-if="!linked" class="text-sm text-[var(--muted)]" data-test="bot-prefs-unlinked">
      Привяжи Telegram, и бот сможет напоминать об учёбе. Отключить напоминания можно будет здесь.
    </p>

    <template v-else-if="store.prefs">
      <ul class="space-y-1">
        <li v-for="r in rows" :key="r.key">
          <label class="row">
            <span class="min-w-0 flex-1">
              <span class="block text-sm font-semibold">{{ r.title }}</span>
              <span class="block text-xs text-[var(--muted)]">{{ r.hint }}</span>
            </span>
            <input
              type="checkbox"
              role="switch"
              class="switch"
              :checked="store.prefs[r.key]"
              :aria-label="r.title"
              :data-test="'pref-' + r.key"
              @change="store.set(r.key, ($event.target as HTMLInputElement).checked)"
            />
          </label>
        </li>
      </ul>
      <p class="mt-2 text-xs text-[var(--muted)]">
        Вход и оплату бот пишет всегда. Остановить всё сразу можно командой /stop в боте.
      </p>
    </template>

    <div v-else-if="!store.error" class="space-y-2">
      <div class="skel h-9 w-full"></div>
      <div class="skel h-9 w-full"></div>
    </div>

    <p v-if="store.error" class="mt-1 text-sm text-[var(--bad)]" data-test="bot-prefs-error">{{ store.error }}</p>
  </div>
</template>

<style scoped>
.row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  border-radius: 14px;
  padding: 0.5rem 0.25rem;
  cursor: pointer;
}
/* A plain checkbox drawn as a switch: no border, the track is a fill. */
.switch {
  appearance: none;
  flex: none;
  width: 2.5rem;
  height: 1.5rem;
  border-radius: 999px;
  background: var(--ring-track);
  position: relative;
  cursor: pointer;
  transition: background-color 0.15s ease;
}
.switch::after {
  content: '';
  position: absolute;
  top: 0.2rem;
  left: 0.2rem;
  width: 1.1rem;
  height: 1.1rem;
  border-radius: 50%;
  background: #fff;
  transition: transform 0.15s ease;
}
.switch:checked {
  background: var(--accent);
}
.switch:checked::after {
  transform: translateX(1rem);
}
.switch:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
</style>
