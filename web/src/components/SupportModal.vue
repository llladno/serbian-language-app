<script setup lang="ts">
import { X } from 'lucide-vue-next'
import ModalShell from './ModalShell.vue'
import { useSupportModal, SUPPORT_TELEGRAM_URL } from '../lib/supportModal'
import supportMascot from '../assets/support-mascot.webp'

const { open, message, busy, error, sent, closeModal, send } = useSupportModal()
</script>

<template>
  <ModalShell :open="open" panel-class="p-5 text-center" @close="closeModal">
    <button class="icon-btn absolute right-3 top-3" title="Закрыть" aria-label="Закрыть" @click="closeModal">
      <X :size="19" :stroke-width="2.25" />
    </button>

    <Transition name="fade" mode="out-in">
      <div v-if="sent" key="thanks" class="space-y-2 py-4">
        <img :src="supportMascot" alt="" class="mx-auto h-20 w-20 object-contain" />
        <p class="text-lg font-extrabold">Спасибо, что написали! 💙</p>
        <p class="text-sm text-[var(--muted)]">Мы обязательно прочитаем.</p>
      </div>
      <div v-else key="form" class="space-y-3">
        <img :src="supportMascot" alt="" class="mx-auto h-24 w-24 object-contain" />
        <p class="text-lg font-extrabold">Есть вопрос, пожелание или что-то не работает?</p>
        <p class="text-sm text-[var(--muted)]">Напишите в любой момент — мы читаем каждое сообщение.</p>

        <textarea
          v-model="message"
          class="field w-full"
          rows="4"
          placeholder="Что случилось, чего не хватает, что понравилось..."
          maxlength="2000"
        />
        <p v-if="error" class="text-sm text-[var(--bad)]">{{ error }}</p>

        <div class="flex flex-col gap-2 sm:flex-row">
          <button
            class="btn btn-primary flex-1"
            data-test="send-support"
            :disabled="busy || !message.trim()"
            @click="send"
          >
            {{ busy ? 'Отправляем…' : 'Отправить' }}
          </button>
          <a
            :href="SUPPORT_TELEGRAM_URL"
            target="_blank"
            rel="noopener"
            class="btn btn-ghost flex-1"
            data-test="support-telegram"
          >
            Написать в Telegram
          </a>
        </div>
      </div>
    </Transition>
  </ModalShell>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
