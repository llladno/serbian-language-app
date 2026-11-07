<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { X } from 'lucide-vue-next'
import ModalShell from './ModalShell.vue'
import SeedIcon from './SeedIcon.vue'
import { ApiError } from '../api'
import { useWalletStore } from '../stores/wallet'
import { useCourseStore } from '../stores/course'
import { useTariffsStore } from '../stores/tariffs'
import { grouped, offerForPhase, shortDate } from '../lib/tariffs'
import { useToasts } from '../lib/toasts'
import { useUnlockModal } from '../lib/unlockModal'
import { useLevelChoiceModal } from '../lib/levelChoiceModal'
import heart from '../assets/donate/heart.png'
import { trackTariff } from '../lib/tariffEvents'
import { claimVariantFor } from '../lib/claimVariant'
import { useSessionStore } from '../stores/session'
import scene from '../assets/tariffs-promo-scene.webp'

const wallet = useWalletStore()
const session = useSessionStore()
const courseStore = useCourseStore()
const tariffs = useTariffsStore()
const { phaseId, closeModal } = useUnlockModal()
const choiceModal = useLevelChoiceModal()

const phase = computed(() => courseStore.course?.phases.find((p) => p.id === phaseId.value) ?? null)
const product = computed(() => (phaseId.value ? wallet.productForPhase(phaseId.value) : null))
const price = computed(() => product.value?.price_effective ?? phase.value?.price_effective ?? 0)
const fullPrice = computed(() => product.value?.price ?? phase.value?.price ?? 0)
const missing = computed(() => Math.max(0, price.value - wallet.balance))
const canAfford = computed(() => missing.value === 0)

// Short of зёрнышки, the way forward is to pay: «Уровень», the first tariff, for a level of the
// learner's choice (this one included). While the monthly limit holds a level back it is still the gold
// button, with its date and not pressable. When a paid level is still waiting to be spent, the way forward
// is to choose it. The year is not offered here at all (the owner's decision); it is sold on the tariffs
// screen. A picture of the swallow with a seed tops the card, as in the window that follows a payment. The words are those of the tariffs screen and the
// Telegram channel («вы», warm, a little adventure); the pay button is gold, like the one that
// collects a reward.
const offer = computed(() =>
  phaseId.value && tariffs.ready
    ? offerForPhase(courseStore.course, phaseId.value, {
        levelPrice: tariffs.levelNow,
        levelFrom: tariffs.levelFrom,
        hasCredit: tariffs.hasCredit,
      })
    : null,
)
// The admin's «Воронка тарифов» counts a showing when the payment is on offer (not when the window only opens,
// and not when the level can be bought with зёрнышки or a paid level just waits to be chosen): each time it appears.
const sellsNow = computed(() => !!phaseId.value && !!phase.value && !canAfford.value && offer.value?.kind === 'level')
// Each event names the level the window was opened for, so the admin can tell which door people stop at.
const about = computed(() => (phaseId.value ? `phase:${phaseId.value}` : ''))
watch(sellsNow, (selling) => selling && trackTariff('view', 'unlock_modal', about.value), { immediate: true })

function payClicked() {
  trackTariff('click', 'unlock_modal', about.value)
  closeModal()
}

// Зёрнышки earned and never taken: quests finished and still waiting for «Забрать». Short of зёрнышки, the learner is
// told so here, in one of two wordings (lib/claimVariant.ts: a button, or a line in place of «Заработать зёрнышки»);
// taking them can be all it needs to open the level.
const waiting = computed(() => wallet.quests.filter((q) => q.done && !q.claimed))
const waitingSeeds = computed(() => waiting.value.reduce((sum, q) => sum + q.reward, 0))
const claimVariant = computed(() => claimVariantFor(session.user?.id ?? ''))
const claimOffered = computed(() => sellsNow.value && waitingSeeds.value > 0)
// It comes a second after the window opens (the owner's wish): the first second is for the level and the gold button.
// «Заработать зёрнышки» does not show meanwhile either, since there are зёрнышки to take and nothing to earn.
const CLAIM_APPEARS_AFTER_MS = 1000
const claimReady = ref(false)
let claimTimer: ReturnType<typeof setTimeout> | undefined
watch(
  phaseId,
  (id) => {
    clearTimeout(claimTimer)
    claimReady.value = false
    if (id) claimTimer = setTimeout(() => (claimReady.value = true), CLAIM_APPEARS_AFTER_MS)
  },
  { immediate: true },
)
onBeforeUnmount(() => clearTimeout(claimTimer))
const claimShown = computed(() => claimOffered.value && claimReady.value)
const claimDetail = (n: number) => `v=${claimVariant.value},n=${n}`
// Counted when the learner can see it, not when the window opens.
watch(claimShown, (shown) => shown && trackTariff('claim_view', 'unlock_modal', claimDetail(waitingSeeds.value)), {
  immediate: true,
})

// The quests are usually loaded by now (the header's dot needs them); if not, they are fetched when a level is tapped.
watch(
  phaseId,
  async (id) => {
    if (!id || wallet.questsLoaded) return
    try {
      await wallet.loadQuests(true)
    } catch {
      // offline or signed out: the window just has nothing waiting to offer
    }
  },
  { immediate: true },
)

// The offer unfolds into the window: it grows from nothing to its height and rises a little (quick, then settling), while
// it fades in more slowly and evenly, so the fade is seen and not over before the card has grown. No animation where the
// browser has none (tests) or the learner asked for none.
function unfold(el: Element, done: () => void) {
  const node = el as HTMLElement
  const calm = typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (typeof node.animate !== 'function' || calm) return done()
  const height = node.offsetHeight
  node.style.overflow = 'hidden'
  let running = 2
  const end = () => {
    if (--running > 0) return
    node.style.overflow = ''
    done()
  }
  const grow = node.animate(
    [
      { height: '0px', transform: 'translateY(8px)' },
      { height: `${height}px`, transform: 'translateY(0)' },
    ],
    { duration: 450, easing: 'cubic-bezier(0.22, 1, 0.36, 1)' },
  )
  const fade = node.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 700, easing: 'ease-in-out' })
  for (const motion of [grow, fade]) {
    motion.onfinish = end
    motion.oncancel = end
  }
}

const claiming = ref(false)
// Takes every finished quest, a few at a time: there can be dozens of small ones, and one after another would take a
// while. The balance and the list are read again at the end, so the window shows what is true: «Не хватает» may be gone.
async function claimWaiting() {
  if (claiming.value || !waiting.value.length) return
  claiming.value = true
  trackTariff('claim', 'unlock_modal', claimDetail(waitingSeeds.value))
  const due = waiting.value.slice()
  let paid = 0
  try {
    for (let i = 0; i < due.length; i += 6) {
      const results = await Promise.allSettled(due.slice(i, i + 6).map((q) => wallet.claim(q.id)))
      for (const r of results) if (r.status === 'fulfilled') paid += r.value
    }
  } finally {
    await Promise.allSettled([wallet.refresh(), wallet.loadQuests(true)])
    claiming.value = false
  }
  if (paid > 0) useToasts().push({ title: 'Зёрнышки получены', text: 'Они ждали в заданиях', reward: paid })
}

// Closing without going on (the cross, the backdrop, «Позже»), and the free way out, are the other two things a
// learner does here: the admin compares them with the press on the gold button. Only while a payment is on offer.
function dismiss() {
  if (sellsNow.value) trackTariff('dismiss', 'unlock_modal', about.value)
  closeModal()
}
function earnClicked() {
  if (sellsNow.value) trackTariff('earn', 'unlock_modal', about.value)
  closeModal()
}

// Hands over to the window that spends the paid level.
function chooseCredit() {
  closeModal()
  choiceModal.openFor()
}
// "1 000 зёрнышек", in the currency's own name and plural.
function seeds(n: number): string {
  return `${grouped(n)} ${wallet.currencyWord(n)}`.trim()
}
// The prices are fetched once a level is tapped, not on every screen.
watch(phaseId, (id) => {
  if (id) tariffs.load()
})

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
      error.value = 'Не получилось открыть уровень. Попробуйте ещё раз.'
    }
  } finally {
    buying.value = false
  }
}
</script>

<template>
  <ModalShell :open="phaseId !== null && phase !== null" size="sm" panel-class="overflow-hidden text-center" @close="dismiss">
    <button class="icon-btn close absolute right-3 top-3 z-10" title="Закрыть" aria-label="Закрыть" @click="dismiss">
      <X :size="19" :stroke-width="2.25" />
    </button>

    <div class="art" aria-hidden="true"><img :src="scene" alt="" data-test="unlock-scene" /></div>

    <div v-if="phase" class="space-y-3 px-5 pb-5 pt-1" data-test="unlock-modal">
      <p class="text-lg font-extrabold">{{ phase.title }}</p>

      <p class="flex items-center justify-center gap-2 text-3xl font-extrabold tabular-nums" data-test="unlock-price">
        <SeedIcon :size="26" />
        {{ price }}
        <span v-if="fullPrice > price" class="text-base font-semibold text-[var(--muted)] line-through">{{
          fullPrice
        }}</span>
      </p>

      <template v-if="canAfford">
        <p class="text-sm text-[var(--muted)]" data-test="unlock-enough">
          После покупки останется {{ wallet.amount(wallet.balance - price) }}.
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
        <p class="text-sm text-[var(--muted)]" data-test="unlock-missing">Не хватает {{ wallet.amount(missing) }}</p>

        <template v-if="offer">
          <!-- A paid level is waiting: nothing to sell, only to choose. -->
          <template v-if="offer.kind === 'credit'">
            <p class="font-semibold" data-test="unlock-credit">Один уровень у вас уже оплачен: осталось выбрать</p>
            <button class="btn btn-seed w-full" data-test="unlock-credit-choose" @click="chooseCredit">
              Выбрать уровень
            </button>
          </template>
          <template v-else>
            <!-- The hook, or while the monthly limit holds, the date. -->
            <p v-if="offer.from" class="text-sm text-[var(--muted)]" data-test="unlock-cooldown">
              Уровень за рубли можно взять с {{ shortDate(offer.from) }}
            </p>
            <p v-else-if="tariffs.levelBonus" class="font-semibold" data-test="unlock-bonus">
              Откройте сразу — и {{ seeds(tariffs.levelBonus) }} в подарок
            </p>
            <!-- No price in this window, neither on the button nor under it (the owner, 2026-11-07): the button is about
                 what the learner gets, and the price is on the tariffs screen it leads to. -->
            <button v-if="offer.from" class="btn btn-seed w-full" disabled data-test="unlock-pay">Открыть уровень</button>
            <RouterLink v-else to="/tariffs" class="btn btn-seed w-full" data-test="unlock-pay" @click="payClicked">
              Открыть уровень
            </RouterLink>
            <p v-if="!offer.from" class="text-xs text-[var(--muted)]" data-test="unlock-thanks">
              Навсегда, без подписки. Hvala lepo <img :src="heart" alt="" aria-hidden="true" class="heart" />
            </p>
          </template>
          <!-- Зёрнышки waiting in the quests, in one of two wordings (lib/claimVariant.ts). While there are any, earning
               more is not on offer (the owner: nothing to earn when earned ones lie untaken); «Заработать зёрнышки»
               is back once nothing is left to take. The offer comes a second after the window opens, unfolding. -->
          <div class="claim-zone">
            <Transition :css="false" @enter="unfold">
              <div v-if="(claimShown || claiming) && claimVariant === 'button'" class="claim-item">
                <button
                  type="button"
                  class="btn btn-ghost w-full"
                  :disabled="claiming"
                  data-test="unlock-claim-button"
                  @click="claimWaiting"
                >
                  {{ claiming ? 'Забираем…' : 'Забрать' }}
                  <template v-if="!claiming">
                    {{ waitingSeeds }} <SeedIcon :size="18" /><span class="sr-only">{{ wallet.currencyWord(waitingSeeds) }}</span>
                  </template>
                </button>
              </div>
            </Transition>
            <Transition :css="false" @enter="unfold">
              <div v-if="(claimShown || claiming) && claimVariant === 'link'" class="claim-item">
                <button
                  type="button"
                  class="quiet-link"
                  :disabled="claiming"
                  data-test="unlock-claim-link"
                  @click="claimWaiting"
                >
                  <template v-if="claiming">Забираем…</template>
                  <template v-else>
                    Вас ждут {{ waitingSeeds }} <SeedIcon :size="15" class="seed-in-text" /><span class="sr-only">{{ wallet.currencyWord(waitingSeeds) }}</span> — забрать
                  </template>
                </button>
              </div>
            </Transition>
            <RouterLink
              v-if="!claimOffered && !claiming"
              to="/quests"
              class="quiet-link"
              data-test="unlock-quests"
              @click="earnClicked"
            >
              Заработать зёрнышки
            </RouterLink>
          </div>
        </template>

        <!-- No payment on offer (tariffs closed for this account): earning is the way. -->
        <div v-else class="flex flex-col gap-2 sm:flex-row">
          <RouterLink to="/quests" class="btn btn-primary flex-1" data-test="unlock-quests" @click="earnClicked">
            Заработать зёрнышки
          </RouterLink>
          <button class="btn btn-ghost flex-1" @click="dismiss">Позже</button>
        </div>
      </template>
    </div>
  </ModalShell>
</template>

<style scoped>
/* The picture melts into the card instead of ending in an edge. */
.art {
  height: 8.5rem;
  overflow: hidden;
  background: linear-gradient(135deg, var(--accent-soft), var(--card));
}
.art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 50% 20%;
  mask-image: linear-gradient(to bottom, #000 62%, transparent);
  -webkit-mask-image: linear-gradient(to bottom, #000 62%, transparent);
}
/* The cross sits on the picture, which can be bright in a dark theme: give it its own plate. */
.close {
  background: color-mix(in srgb, var(--card) 72%, transparent);
  backdrop-filter: blur(4px);
}
/* Earning is a quiet way out under the gold button, not a second button. */
.quiet-link {
  display: block;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--muted);
  text-decoration: underline;
  text-underline-offset: 3px;
}
.quiet-link:hover {
  color: var(--fg);
}
/* One place under the gold button for whichever of the three is shown, so the gap above it is the same either way. */
.claim-zone {
  display: block;
}
.claim-item {
  display: block;
}
/* As a button the quiet link is the same line: no box, centred, one seed in the text. */
button.quiet-link {
  margin: 0 auto;
  background: none;
  cursor: pointer;
}
button.quiet-link:disabled {
  opacity: 0.7;
  cursor: wait;
}
.seed-in-text {
  display: inline-block;
  vertical-align: -0.2em;
  margin-right: 0.3em;
}
/* The pixel heart sits on the text line like a letter would. */
.heart {
  display: inline-block;
  height: 0.95em;
  width: auto;
  vertical-align: -0.14em;
  image-rendering: pixelated;
}
</style>
