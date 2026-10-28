<script setup lang="ts">
// Asks where the receipt goes. It opens when an account with no email of its own (it came in through
// Telegram) presses a pay button: the bank makes a receipt (54-ФЗ) only for an email or a phone, and an
// account with an email never sees this window. The field starts with what was typed last time, so a
// returning buyer confirms with one tap. Tone of the tariffs screen: «вы».
import { nextTick, ref, watch } from 'vue'
import { X } from 'lucide-vue-next'
import ModalShell from './ModalShell.vue'
import { looksLikeContact } from '../lib/tariffs'

const props = defineProps<{ open: boolean; initial: string }>()
const emit = defineEmits<{ close: []; submit: [contact: string] }>()

const value = ref('')
const wrong = ref(false)
const field = ref<HTMLInputElement | null>(null)

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    value.value = props.initial
    wrong.value = false
    await nextTick()
    field.value?.focus()
  },
  { immediate: true },
)

function send() {
  const contact = value.value.trim()
  if (!looksLikeContact(contact)) {
    wrong.value = true
    field.value?.focus()
    return
  }
  emit('submit', contact)
}
</script>

<template>
  <ModalShell :open="open" size="sm" @close="emit('close')">
    <form
      v-if="open"
      class="space-y-4 p-5"
      role="dialog"
      aria-modal="true"
      aria-labelledby="receipt-title"
      data-test="receipt-contact-modal"
      @submit.prevent="send"
    >
      <button
        type="button"
        class="icon-btn absolute right-3 top-3"
        title="Закрыть"
        aria-label="Закрыть"
        data-test="receipt-contact-close"
        @click="emit('close')"
      >
        <X :size="19" :stroke-width="2.25" />
      </button>

      <div class="pr-8">
        <h2 id="receipt-title" class="text-xl font-extrabold leading-snug">Куда прислать чек?</h2>
        <p class="mt-1 text-sm text-[var(--muted)]">Туда придёт чек об оплате. Доступ откроется на этом аккаунте.</p>
      </div>

      <div>
        <label for="receipt-contact" class="block text-sm font-bold">Почта или телефон</label>
        <input
          id="receipt-contact"
          ref="field"
          v-model="value"
          type="text"
          inputmode="email"
          autocomplete="email"
          autocapitalize="off"
          placeholder="почта или телефон"
          class="field mt-2 w-full"
          :aria-invalid="wrong"
          data-test="receipt-contact-input"
          @input="wrong = false"
        />
        <p v-if="wrong" class="mt-1.5 text-xs text-[var(--bad)]" role="alert" data-test="receipt-contact-error">
          Похоже на опечатку. Впишите почту или телефон.
        </p>
      </div>

      <button type="submit" class="btn btn-primary w-full" data-test="receipt-contact-submit">Перейти к оплате</button>
    </form>
  </ModalShell>
</template>
