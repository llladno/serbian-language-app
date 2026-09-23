<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { useDonateModal, TRIBUTE_TELEGRAM_LINK, TRIBUTE_WEB_LINK } from '../lib/donateModal'
import supportBird from '../assets/donate/support-bird.png'

const { open, closeModal } = useDonateModal()
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

          <div class="space-y-3">
            <img :src="supportBird" alt="" class="mx-auto h-24 w-24 object-contain" />
            <p class="text-lg font-extrabold">Šoljica kafe ☕ (Чашечка кофе)</p>
            <p class="text-sm text-[var(--muted)]">
              Hvala što si tu! 💛 Ваша поддержка помогает добавлять новые уроки, слова и улучшать приложение.
            </p>

            <div class="flex flex-col gap-2 sm:flex-row">
              <a
                :href="TRIBUTE_TELEGRAM_LINK"
                target="_blank"
                rel="noopener"
                class="btn btn-primary flex-1"
                data-test="donate-telegram"
              >
                Поддержать через телеграм
              </a>
              <a
                :href="TRIBUTE_WEB_LINK"
                target="_blank"
                rel="noopener"
                class="btn btn-ghost flex-1"
                data-test="donate-web"
              >
                Поддержать
              </a>
            </div>
          </div>
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
</style>
