<script setup lang="ts">
// Задания: the full quest list, the screen the header balance leads to.
// Quests are paid out here and nowhere else, so this is also the one place
// that opens the reward modal.
//
// Design note: everything else in the app is rounded, soft and quiet. Here one
// thing is loud — the quest that can be claimed right now — and the devices
// that carry it are the brand's own: gold means money (the seed palette,
// the same in every theme), the accent means progress, and progress is drawn
// as hard pixel blocks rather than a smooth bar.
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { Check, Send, UserPlus } from 'lucide-vue-next'
import { useWalletStore } from '../stores/wallet'
import { useRewardModal } from '../lib/rewardModal'
import { authErrorMessage } from '../lib/authErrors'
import { pileFor } from '../lib/seeds'
import { pluralRu } from '../lib/plural'
import { openTelegramLink } from '../telegram'
import { useTelegramStart } from '../lib/telegramStart'
import { useToasts } from '../lib/toasts'
import { useInviteModal } from '../lib/inviteModal'
import { groupQuests } from '../lib/questGroups'
import SeedIcon from '../components/SeedIcon.vue'
import SegmentBar from '../components/SegmentBar.vue'
import InviteModal from '../components/InviteModal.vue'
import AnimatedNumber from '../components/AnimatedNumber.vue'
import type { Quest } from '../types'

const wallet = useWalletStore()
const { celebrate } = useRewardModal()
const invite = useInviteModal()

// The invite quest pays the same for everyone; the modal quotes the number from
// the quest itself, so changing the reward in the admin panel needs no deploy.
const inviteReward = computed(() => wallet.quests.find((q) => q.kind === 'friends_invited')?.reward ?? 0)

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
onMounted(() => {
  load()
  document.addEventListener('visibilitychange', onBackToTheApp)
})
onBeforeUnmount(() => document.removeEventListener('visibilitychange', onBackToTheApp))

// Quests that happen outside the app (subscribing to the channel) are only seen
// by the server, so coming back from Telegram is the moment to ask again. This
// one is not silent: a quest that finished while the learner was away says so.
function onBackToTheApp() {
  if (document.visibilityState === 'visible') wallet.loadQuests().catch(() => {})
}

// The channel button has three ways to go, depending on who is pressing it:
//   - Telegram is linked: straight to the channel, then a "check" button here;
//   - already subscribed: the quest is simply done, there is no button;
//   - Telegram is not linked: to the bot, which links it and then offers the
//     subscribe and check buttons itself. Coming back, the list asks again.
const toasts = useToasts()
const visited = reactive<Record<number, boolean>>({})
const rechecking = ref<number | null>(null)
const tg = useTelegramStart('link', { intent: 'channel' })

function viaTheBot() {
  tg.start(() => {
    toasts.push({
      title: 'Telegram привязан',
      text: 'Подпишись на канал в боте и нажми «Проверить подписку»',
    })
    wallet.loadQuests(true).catch(() => {})
  })
}

function goToChannel(q: Quest, e: Event) {
  visited[q.id] = true
  // Inside Telegram the link opens in Telegram; elsewhere the anchor does it.
  if (q.url && openTelegramLink(q.url)) e.preventDefault()
}

async function recheck(q: Quest) {
  rechecking.value = q.id
  try {
    await wallet.loadQuests(true)
    const done = !!wallet.quests.find((x) => x.id === q.id)?.done
    toasts.push(
      done
        ? { title: 'Подписка найдена', text: 'Задание выполнено, награду можно забрать', to: '/quests' }
        : { title: 'Подписки пока не видно', text: 'Подпишись на канал и проверь ещё раз' },
    )
  } catch (e) {
    claimError.value = authErrorMessage(e)
  } finally {
    rechecking.value = null
  }
}

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
          tone="seed"
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
        <div v-for="q in ready" :key="q.id" class="plate flex flex-wrap items-center gap-x-3 gap-y-3 p-4 sm:flex-nowrap" data-test="quest-ready">
          <div class="reward reward-ready" data-test="quest-reward">
            <SeedIcon :size="20" />
            <span>+{{ q.reward }}</span>
          </div>
          <div class="min-w-0 flex-1">
            <p class="font-bold">{{ q.title }}</p>
            <p v-if="q.description" class="mt-0.5 text-sm text-[var(--muted)]">{{ q.description }}</p>
          </div>
          <button
            class="btn btn-seed btn-side w-full sm:w-auto"
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
          <!-- Reward on the left, the thing to do on the right: the eye reads
               "what do I get" first and the button is always under the thumb. -->
          <div
            v-for="(q, i) in active"
            :key="q.id"
            class="flex flex-wrap items-center gap-x-3 gap-y-3 px-4 py-3.5 sm:flex-nowrap"
            :class="i > 0 ? 'border-t border-[var(--border)]' : ''"
            data-test="quest-active"
          >
            <div class="reward" data-test="quest-reward">
              <SeedIcon :size="18" />
              <span>{{ q.reward }}</span>
            </div>

            <div class="min-w-0 flex-1 break-words">
              <p class="font-semibold">{{ q.title }}</p>
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

            <!-- Subscribing happens in Telegram, so the row sends the learner there. -->
            <div
              v-if="q.url"
              class="flex w-full shrink-0 gap-2 sm:w-auto sm:flex-col sm:items-stretch"
              data-test="quest-link"
            >
              <button
                v-if="q.needs_telegram"
                type="button"
                class="btn btn-primary btn-side flex-1 sm:flex-none"
                data-test="quest-go-bot"
                :disabled="tg.busy.value"
                @click="viaTheBot"
              >
                <Send :size="15" :stroke-width="2.25" />{{ tg.busy.value ? 'Ждём Telegram…' : 'Подписаться' }}
              </button>
              <template v-else>
                <a
                  :href="q.url"
                  target="_blank"
                  rel="noopener"
                  class="btn btn-primary btn-side flex-1 sm:flex-none"
                  data-test="quest-go"
                  @click="goToChannel(q, $event)"
                >
                  <Send :size="15" :stroke-width="2.25" />Подписаться
                </a>
                <button
                  v-if="visited[q.id]"
                  type="button"
                  class="btn btn-ghost btn-side flex-1 sm:flex-none"
                  data-test="quest-recheck"
                  :disabled="rechecking === q.id"
                  @click="recheck(q)"
                >
                  {{ rechecking === q.id ? 'Проверяем…' : 'Проверить' }}
                </button>
              </template>
            </div>
            <div v-else-if="q.kind === 'friends_invited'" class="w-full shrink-0 sm:w-auto">
              <button
                type="button"
                class="btn btn-primary btn-side w-full sm:w-auto"
                data-test="quest-invite"
                @click="invite.openModal()"
              >
                <UserPlus :size="15" :stroke-width="2.25" />Пригласить
              </button>
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
              {{ q.reward }}<SeedIcon :size="12" />
            </span>
          </span>
        </div>
      </section>

      <p v-if="!wallet.quests.length && !loadError" class="card p-5 text-sm text-[var(--muted)]">
        Заданий пока нет — загляни позже.
      </p>
    </template>

    <InviteModal :reward="inviteReward" />
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

/* The currency column: the same width on every row, so the buttons on the right
   and the titles in the middle line up down the whole list. */
.reward {
  display: flex;
  flex: none;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  width: 44px;
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--muted);
}
.reward-ready {
  color: var(--fg);
  font-size: 1rem;
}
.btn-side {
  padding: 0.5rem 0.85rem;
  font-size: 0.875rem;
  white-space: nowrap;
  min-width: 8rem;
}

/* The one loud element on the screen: a quest that can be claimed right now.
   The gold wash is the same colour as the currency, so the plate, its reward
   and its button all say the same thing without a second device. */
.plate {
  background: linear-gradient(0deg, var(--seed-wash), var(--seed-wash)), var(--card);
  border-radius: 20px;
  box-shadow: var(--shadow);
}
.btn-seed {
  background: var(--seed);
  color: var(--seed-ink);
}
.btn-seed:not(:disabled):hover {
  background: var(--seed-hi);
}
.btn-seed:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}
</style>
