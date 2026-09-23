<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { useSupportModal, SUPPORT_TELEGRAM_URL } from '../lib/supportModal'
import supportMascot from '../assets/support-mascot.webp'

const { open, message, busy, error, sent, closeModal, send } = useSupportModal()
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-overlay">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto p-4 sm:items-center"
      >
        <div class="fixed inset-0 bg-black/40" @click="closeModal" />
        <div class="card modal-panel relative z-10 w-full max-w-md p-5 text-center">
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
        </div>
      </div>
    </Transition>
  </Teleport>
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
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
