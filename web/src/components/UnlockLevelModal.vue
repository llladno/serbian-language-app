<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { X } from 'lucide-vue-next'
import ModalShell from './ModalShell.vue'
import SeedIcon from './SeedIcon.vue'
import { ApiError } from '../api'
import { useWalletStore } from '../stores/wallet'
import { useCourseStore } from '../stores/course'
import { useToasts } from '../lib/toasts'
import { useUnlockModal } from '../lib/unlockModal'

const wallet = useWalletStore()
const courseStore = useCourseStore()
const { phaseId, closeModal } = useUnlockModal()

const phase = computed(() => courseStore.course?.phases.find((p) => p.id === phaseId.value) ?? null)
const product = computed(() => (phaseId.value ? wallet.productForPhase(phaseId.value) : null))
const price = computed(() => product.value?.price_effective ?? phase.value?.price_effective ?? 0)
const fullPrice = computed(() => product.value?.price ?? phase.value?.price ?? 0)
const missing = computed(() => Math.max(0, price.value - wallet.balance))
const canAfford = computed(() => missing.value === 0)

const buying = ref(false)
const error = ref('')
watch(phaseId, () => {
  error.value = ''
  buying.value = false
})

async function buy() {
  if (!product.value || buying.value) return
  buying.value = true
  error.value = ''
  try {
    await wallet.buy(product.value.id)
    await courseStore.load(true)
    useToasts().push({ title: 'Уровень открыт', text: phase.value?.title ?? '' })
    closeModal()
  } catch (e) {
    if (e instanceof ApiError && e.message === 'insufficient_funds') {
      // The balance moved since the screen loaded; show the honest number.
      await wallet.refresh()
    } else if (e instanceof ApiError && e.message === 'already_owned') {
      await courseStore.load(true)
      closeModal()
    } else {
      error.value = 'Не получилось открыть уровень. Попробуй ещё раз.'
    }
  } finally {
    buying.value = false
  }
}
</script>

<template>
  <ModalShell :open="phaseId !== null && phase !== null" size="sm" panel-class="p-5 text-center" @close="closeModal">
    <button class="icon-btn absolute right-3 top-3" title="Закрыть" aria-label="Закрыть" @click="closeModal">
      <X :size="19" :stroke-width="2.25" />
    </button>

    <div v-if="phase" class="space-y-3" data-test="unlock-modal">
      <p class="pr-6 text-lg font-extrabold">{{ phase.title }}</p>

      <p class="flex items-center justify-center gap-2 text-3xl font-extrabold tabular-nums" data-test="unlock-price">
        <SeedIcon :size="26" />
        {{ price }}
        <span v-if="fullPrice > price" class="text-base font-semibold text-[var(--muted)] line-through">{{
          fullPrice
        }}</span>
      </p>

      <template v-if="canAfford">
        <p class="text-sm text-[var(--muted)]" data-test="unlock-enough">
          Этот уровень открывается за {{ wallet.amount(price) }}. У тебя {{ wallet.amount(wallet.balance) }}, после
          покупки останется {{ wallet.amount(wallet.balance - price) }}.
        </p>
        <p v-if="error" class="text-sm text-[var(--bad)]" data-test="unlock-error">{{ error }}</p>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="btn btn-primary flex-1" :disabled="buying" data-test="unlock-buy" @click="buy">
            Открыть за {{ price }}
          </button>
          <button class="btn btn-ghost flex-1" @click="closeModal">Не сейчас</button>
        </div>
      </template>

      <template v-else>
        <p class="text-sm text-[var(--muted)]" data-test="unlock-missing">
          Не хватает {{ wallet.amount(missing) }}: у тебя {{ wallet.amount(wallet.balance) }} из {{ price }}. Зёрнышки
          дают за уроки, повторения, серию занятий и задания.
        </p>
        <div class="flex flex-col gap-2 sm:flex-row">
          <RouterLink to="/quests" class="btn btn-primary flex-1" data-test="unlock-quests" @click="closeModal">
            Как заработать
          </RouterLink>
          <button class="btn btn-ghost flex-1" @click="closeModal">Понятно</button>
        </div>
      </template>
    </div>
  </ModalShell>
</template>
