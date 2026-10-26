<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { X } from 'lucide-vue-next'
import ModalShell from './ModalShell.vue'
import SeedIcon from './SeedIcon.vue'
import { ApiError } from '../api'
import { useWalletStore } from '../stores/wallet'
import { useCourseStore } from '../stores/course'
import { useTariffsStore } from '../stores/tariffs'
import { grouped, offerForPhase, rub, shortDate } from '../lib/tariffs'
import { useToasts } from '../lib/toasts'
import { useUnlockModal } from '../lib/unlockModal'
import { useLevelChoiceModal } from '../lib/levelChoiceModal'
import heart from '../assets/donate/heart.png'
import { trackTariff } from '../lib/tariffEvents'
import scene from '../assets/tariffs-promo-scene.webp'

const wallet = useWalletStore()
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
watch(sellsNow, (selling) => selling && trackTariff('view', 'unlock_modal'), { immediate: true })

function payClicked() {
  trackTariff('click', 'unlock_modal')
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
  <ModalShell :open="phaseId !== null && phase !== null" size="sm" panel-class="overflow-hidden text-center" @close="closeModal">
    <button class="icon-btn close absolute right-3 top-3 z-10" title="Закрыть" aria-label="Закрыть" @click="closeModal">
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
            <button v-if="offer.from" class="btn btn-seed w-full" disabled data-test="unlock-pay">
              Хочу уровень — {{ rub(offer.price) }}
            </button>
            <RouterLink v-else to="/tariffs" class="btn btn-seed w-full" data-test="unlock-pay" @click="payClicked">
              Хочу уровень — {{ rub(offer.price) }}
              <s v-if="tariffs.levelDiscount" class="was" data-test="unlock-was">{{ rub(tariffs.levelRegular) }}</s>
            </RouterLink>
            <p v-if="!offer.from" class="text-xs text-[var(--muted)]" data-test="unlock-thanks">
              Навсегда, без подписки. Hvala lepo <img :src="heart" alt="" aria-hidden="true" class="heart" />
            </p>
          </template>
          <RouterLink to="/quests" class="quiet-link" data-test="unlock-quests" @click="closeModal">
            Заработать зёрнышки
          </RouterLink>
        </template>

        <!-- No payment on offer (tariffs closed for this account): earning is the way. -->
        <div v-else class="flex flex-col gap-2 sm:flex-row">
          <RouterLink to="/quests" class="btn btn-primary flex-1" data-test="unlock-quests" @click="closeModal">
            Заработать зёрнышки
          </RouterLink>
          <button class="btn btn-ghost flex-1" @click="closeModal">Позже</button>
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
/* The regular price, struck through inside the button, next to the first-purchase price. */
.was {
  margin-left: 0.4rem;
  font-size: 0.8em;
  font-weight: 600;
  opacity: 0.65;
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
