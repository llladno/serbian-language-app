<script setup lang="ts">
// The stand-in cash desk (PAYMENTS_DEV): a page for a developer's machine that plays the bank. The server
// sends the buyer here instead of to Т-Касса; two buttons say how the payment ended, and the page returns
// to /tariffs?order=N exactly as the bank would, so the real return flow (settling, the choice of a
// level, the thank-you) is what gets tried. Where the desk is off the server answers 404 and the page
// only explains. Server side: server/internal/api/devpay.go.
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import { rub } from '../lib/tariffs'
import type { DevOrder } from '../types'

const orderId = computed(() => Number(new URL(window.location.href).searchParams.get('order')))
const order = ref<DevOrder | null>(null)
const loading = ref(true)
const unavailable = ref(false)
const busy = ref(false)
const error = ref('')

function back() {
  window.location.assign(`/tariffs?order=${orderId.value}`)
}

async function press(outcome: 'paid' | 'failed') {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await api.devPay(orderId.value, outcome)
    back()
  } catch {
    error.value = 'Касса не ответила. Попробуйте ещё раз.'
    busy.value = false
  }
}

onMounted(async () => {
  if (!(orderId.value > 0)) {
    unavailable.value = true
    loading.value = false
    return
  }
  try {
    order.value = await api.devOrder(orderId.value)
  } catch {
    unavailable.value = true
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="mx-auto max-w-sm py-10">
    <div v-if="loading" class="text-center text-sm text-[var(--muted)]">Загрузка…</div>

    <section v-else-if="unavailable || !order" class="card p-5 text-center" data-test="dev-unavailable">
      <h1 class="text-lg font-extrabold">Здесь нечего оплачивать</h1>
      <p class="mt-2 text-sm text-[var(--muted)]">
        Тестовая касса включается переменной PAYMENTS_DEV=1 и работает только на вашем компьютере, с адресом на http.
      </p>
    </section>

    <section v-else class="card space-y-4 p-5 text-center" data-test="dev-desk">
      <p class="text-xs font-bold uppercase tracking-wider text-[var(--muted)]">Тестовая касса</p>
      <div>
        <h1 class="text-lg font-extrabold">{{ order.title }}</h1>
        <p class="mt-1 text-3xl font-extrabold tabular-nums" data-test="dev-amount">{{ rub(order.amount) }}</p>
      </div>
      <p class="text-sm text-[var(--muted)]">Деньги не списываются. Выберите, чем закончилась оплата.</p>

      <p v-if="error" class="text-sm text-[var(--bad)]" data-test="dev-error">{{ error }}</p>

      <div v-if="order.status === 'pending'" class="flex flex-col gap-2">
        <button class="btn btn-primary w-full" :disabled="busy" data-test="dev-pay" @click="press('paid')">Оплатить</button>
        <button class="btn btn-ghost w-full" :disabled="busy" data-test="dev-fail" @click="press('failed')">Отказаться</button>
      </div>
      <div v-else class="space-y-2">
        <p class="text-sm font-semibold">
          {{ order.status === 'succeeded' ? 'Заказ уже оплачен' : 'Заказ уже закрыт' }}
        </p>
        <button class="btn btn-primary w-full" data-test="dev-back" @click="back">Вернуться к тарифам</button>
      </div>
    </section>
  </div>
</template>
