<script setup lang="ts">
// Задания: the full quest list, the screen the header balance leads to.
// Quests are paid out here and nowhere else, so this is also the one place
// that opens the reward modal.
import { computed, onMounted, ref } from 'vue'
import { Check } from 'lucide-vue-next'
import { useWalletStore } from '../stores/wallet'
import { useRewardModal } from '../lib/rewardModal'
import { authErrorMessage } from '../lib/authErrors'
import { pileFor } from '../lib/feathers'
import { pluralRu } from '../lib/plural'
import { groupQuests } from '../lib/questGroups'
import FeatherIcon from '../components/FeatherIcon.vue'
import type { Quest } from '../types'

const wallet = useWalletStore()
const { celebrate } = useRewardModal()

const loadError = ref<string | null>(null)
const claimError = ref<string | null>(null)
const busyId = ref<number | null>(null)

async function load() {
  loadError.value = null
  try {
    await Promise.all([wallet.refresh(), wallet.loadQuests(true)])
  } catch (e) {
    loadError.value = authErrorMessage(e)
  }
}
onMounted(load)

// A ladder (5 → 10 → 20 уроков) shows only the rung in progress, so finishing
// one visibly turns the card into the next.
const groups = computed(() => groupQuests(wallet.quests))
const ready = computed(() => groups.value.ready)
const active = computed(() => groups.value.active)
const claimed = computed(() => groups.value.done)

const pile = computed(() => pileFor(wallet.balance))
const streakText = computed(
  () => `${wallet.streakDays} ${pluralRu(wallet.streakDays, 'день', 'дня', 'дней')} подряд`,
)

function percent(q: Quest): number {
  if (q.target <= 0) return 0
  return Math.min(100, Math.round((q.value / q.target) * 100))
}

async function claim(q: Quest) {
  busyId.value = q.id
  claimError.value = null
  try {
    const reward = await wallet.claim(q.id)
    celebrate(reward, q.title)
  } catch (e) {
    // A claim can legitimately fail — another tab got there first, or the
    // counter moved — so the list is refetched instead of being patched up
    // from a guess about what went wrong.
    claimError.value = authErrorMessage(e)
    await wallet.loadQuests()
  } finally {
    busyId.value = null
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="card flex items-center gap-4 p-5">
      <img :src="pile" alt="" aria-hidden="true" class="pile" />
      <div>
        <p class="text-2xl font-extrabold" data-test="wallet-balance">{{ wallet.amount(wallet.balance) }}</p>
        <p v-if="wallet.streakDays > 0" class="text-sm text-[var(--muted)]">🔥 {{ streakText }}</p>
        <p v-else class="text-sm text-[var(--muted)]">Занимайся каждый день — за серию тоже платят</p>
      </div>
    </div>

    <p v-if="loadError" class="card p-5 text-sm text-[var(--bad)]">{{ loadError }}</p>
    <p v-if="claimError" class="card p-5 text-sm text-[var(--bad)]" data-test="claim-error">{{ claimError }}</p>

    <div v-if="!wallet.questsLoaded && !loadError" class="space-y-3">
      <div v-for="i in 3" :key="i" class="card space-y-2 p-4">
        <div class="skel h-4 w-40"></div>
        <div class="skel h-3.5 w-full"></div>
      </div>
    </div>

    <template v-else>
      <section v-if="ready.length" class="space-y-2">
        <h2 class="px-1 text-sm font-bold text-[var(--muted)]">Можно забрать</h2>
        <div v-for="q in ready" :key="q.id" class="card space-y-3 p-4" data-test="quest-ready">
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="font-bold">{{ q.title }}</p>
              <p v-if="q.description" class="text-sm text-[var(--muted)]">{{ q.description }}</p>
            </div>
            <span class="flex shrink-0 items-center gap-1 font-extrabold">
              <FeatherIcon :size="16" />{{ q.reward }}
            </span>
          </div>
          <button
            class="btn btn-primary w-full"
            :disabled="busyId === q.id"
            :data-test="'claim-' + q.id"
            @click="claim(q)"
          >
            {{ busyId === q.id ? 'Забираем…' : 'Забрать' }}
          </button>
        </div>
      </section>

      <section v-if="active.length" class="space-y-2">
        <h2 class="px-1 text-sm font-bold text-[var(--muted)]">В процессе</h2>
        <div v-for="q in active" :key="q.id" class="card space-y-2 p-4">
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="font-bold">{{ q.title }}</p>
              <p v-if="q.description" class="text-sm text-[var(--muted)]">{{ q.description }}</p>
            </div>
            <span class="flex shrink-0 items-center gap-1 font-extrabold text-[var(--muted)]">
              <FeatherIcon :size="16" />{{ q.reward }}
            </span>
          </div>
          <div v-if="q.target > 1" class="flex items-center gap-2">
            <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-[var(--bg-soft)]">
              <div class="h-full rounded-full bg-[var(--accent)]" :style="{ width: percent(q) + '%' }"></div>
            </div>
            <span class="text-xs tabular-nums text-[var(--muted)]">
              {{ Math.min(q.value, q.target) }} / {{ q.target }}
            </span>
          </div>
        </div>
      </section>

      <section v-if="claimed.length" class="space-y-2">
        <h2 class="px-1 text-sm font-bold text-[var(--muted)]">Получено</h2>
        <div v-for="q in claimed" :key="q.id" class="card flex items-center justify-between gap-3 p-4">
          <p class="font-semibold text-[var(--muted)]">{{ q.title }}</p>
          <span class="flex shrink-0 items-center gap-1 text-sm font-semibold text-[var(--muted)]">
            <Check :size="15" :stroke-width="3" />
            <FeatherIcon :size="14" />{{ q.reward }}
          </span>
        </div>
      </section>

      <p v-if="!wallet.quests.length && !loadError" class="card p-5 text-sm text-[var(--muted)]">
        Заданий пока нет — загляни позже.
      </p>
    </template>
  </div>
</template>

<style scoped>
.pile {
  width: 72px;
  image-rendering: pixelated;
  flex: none;
}
</style>
