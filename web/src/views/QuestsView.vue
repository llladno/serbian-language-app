<script setup lang="ts">
// Задания: the full quest list, the screen the header balance leads to.
// Quests are paid out here and nowhere else, so this is also the one place
// that opens the reward modal.
//
// Design note: everything else in the app is rounded, soft and quiet. Here one
// thing is loud — the quest that can be claimed right now — and the devices
// that carry it are the brand's own: gold means money (the feather palette,
// the same in every theme), the accent means progress, and progress is drawn
// as hard pixel blocks rather than a smooth bar.
import { computed, onMounted, ref } from 'vue'
import { Check } from 'lucide-vue-next'
import { useWalletStore } from '../stores/wallet'
import { useRewardModal } from '../lib/rewardModal'
import { authErrorMessage } from '../lib/authErrors'
import { pileFor } from '../lib/feathers'
import { pluralRu } from '../lib/plural'
import { groupQuests } from '../lib/questGroups'
import FeatherIcon from '../components/FeatherIcon.vue'
import SegmentBar from '../components/SegmentBar.vue'
import AnimatedNumber from '../components/AnimatedNumber.vue'
import type { Quest } from '../types'

const wallet = useWalletStore()
const { celebrate } = useRewardModal()

const loadError = ref<string | null>(null)
const claimError = ref<string | null>(null)
const busyId = ref<number | null>(null)

async function load() {
  loadError.value = null
  try {
    await Promise.all([wallet.refresh(), wallet.loadQuests(true), wallet.loadShop()])
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
const left = computed(() =>
  wallet.goal ? Math.max(0, wallet.goal.price_effective - wallet.balance) : 0,
)

function claimTitle(q: Quest): string {
  return busyId.value === q.id ? 'Забираем…' : 'Забрать'
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
    await wallet.loadQuests(true)
  } finally {
    busyId.value = null
  }
}
</script>

<template>
  <div class="space-y-6">
    <section class="card p-5">
      <div class="flex items-center gap-4">
        <img :src="pile" alt="" aria-hidden="true" class="pile" />
        <div class="min-w-0">
          <AnimatedNumber
            class="serbian block text-[1.75rem] font-bold leading-none sm:text-3xl"
            data-test="wallet-balance"
            :value="wallet.balance"
            :format="wallet.amount"
          />
          <p v-if="wallet.streakDays > 0" class="mt-1.5 text-sm text-[var(--muted)]">
            🔥 {{ streakText }}
          </p>
          <p v-else class="mt-1.5 text-sm text-[var(--muted)]">
            Занимайся каждый день — за серию тоже платят
          </p>
        </div>
      </div>

      <template v-if="wallet.goal">
        <SegmentBar
          class="mt-4"
          :value="wallet.balance"
          :max="wallet.goal.price_effective"
          :segments="20"
          tone="feather"
        />
        <p class="mt-2 text-sm text-[var(--muted)]">
          <template v-if="left > 0">
            До «{{ wallet.goal.title }}» ещё
            <span class="font-semibold text-[var(--fg)]">{{ wallet.amount(left) }}</span>
          </template>
          <template v-else>Хватает на «{{ wallet.goal.title }}»</template>
        </p>
      </template>
    </section>

    <p v-if="loadError" class="card p-5 text-sm text-[var(--bad)]">{{ loadError }}</p>
    <p v-if="claimError" class="card p-5 text-sm text-[var(--bad)]" data-test="claim-error">
      {{ claimError }}
    </p>

    <div v-if="!wallet.questsLoaded && !loadError" class="space-y-3">
      <div v-for="i in 3" :key="i" class="card space-y-2 p-4">
        <div class="skel h-4 w-40"></div>
        <div class="skel h-3.5 w-full"></div>
      </div>
    </div>

    <template v-else>
      <section v-if="ready.length" class="space-y-3">
        <h2 class="px-1 font-bold">Можно забрать</h2>
        <div v-for="q in ready" :key="q.id" class="plate p-4" data-test="quest-ready">
          <div class="flex items-baseline justify-between gap-3">
            <p class="font-bold">{{ q.title }}</p>
            <span class="flex shrink-0 items-center gap-1 font-extrabold">
              +{{ q.reward }}<FeatherIcon :size="16" />
            </span>
          </div>
          <p v-if="q.description" class="mt-0.5 text-sm text-[var(--muted)]">{{ q.description }}</p>
          <button
            class="btn btn-feather mt-3 w-full"
            :disabled="busyId === q.id"
            :data-test="'claim-' + q.id"
            @click="claim(q)"
          >
            {{ claimTitle(q) }}
          </button>
        </div>
      </section>

      <section v-if="active.length" class="space-y-3">
        <h2 class="px-1 font-bold">В процессе</h2>
        <div class="card overflow-hidden">
          <div
            v-for="(q, i) in active"
            :key="q.id"
            class="px-4 py-3.5"
            :class="i > 0 ? 'border-t border-[var(--border)]' : ''"
          >
            <div class="flex items-baseline justify-between gap-3">
              <p class="font-semibold">{{ q.title }}</p>
              <span class="flex shrink-0 items-center gap-1 text-sm font-bold text-[var(--muted)]">
                {{ q.reward }}<FeatherIcon :size="13" />
              </span>
            </div>
            <p v-if="q.description" class="mt-0.5 text-sm text-[var(--muted)]">
              {{ q.description }}
            </p>
            <!-- A yes/no quest has nothing to count, and a row of empty blocks
                 for "not subscribed yet" would be a progress bar about nothing. -->
            <div v-if="q.target > 1" class="mt-2.5 flex items-center gap-3">
              <SegmentBar class="flex-1" :value="q.value" :max="q.target" />
              <span class="shrink-0 text-xs font-semibold tabular-nums text-[var(--muted)]">
                {{ Math.min(q.value, q.target) }}/{{ q.target }}
              </span>
            </div>
          </div>
        </div>
      </section>

      <section v-if="claimed.length" class="space-y-3">
        <h2 class="px-1 font-bold">Получено</h2>
        <div class="flex flex-wrap gap-2">
          <span
            v-for="q in claimed"
            :key="q.id"
            class="flex items-center gap-1.5 rounded-full bg-[var(--bg-soft)] px-3 py-1.5 text-sm text-[var(--muted)]"
          >
            <Check :size="14" :stroke-width="3" />{{ q.title }}
            <span class="flex items-center gap-0.5 font-semibold">
              {{ q.reward }}<FeatherIcon :size="12" />
            </span>
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
  width: 64px;
  image-rendering: pixelated;
  flex: none;
}
@media (min-width: 640px) {
  .pile {
    width: 84px;
  }
}

/* The one loud element on the screen: a quest that can be claimed right now.
   The gold wash is the same colour as the currency, so the plate, its reward
   and its button all say the same thing without a second device. */
.plate {
  background: linear-gradient(0deg, var(--feather-wash), var(--feather-wash)), var(--card);
  border-radius: 20px;
  box-shadow: var(--shadow);
}
.btn-feather {
  background: var(--feather);
  color: var(--feather-ink);
}
.btn-feather:not(:disabled):hover {
  background: var(--feather-hi);
}
.btn-feather:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}
</style>
